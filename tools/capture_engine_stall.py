#!/usr/bin/env python3
"""Capture an owned Engine timeout before cleanup; never emit acceptance.

Use with an owned diagnostic record, including actual executable hashes and
resource declarations. The optional CL debugger runs only after a timeout and
is recorded as interference. It does not belong in clean operating samples.
"""
import argparse
import hashlib
import json
import os
import select
import signal
import subprocess
import time
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
from engine_storage_snapshot import snapshot as storage_snapshot, start_ticks


def utc():
    return datetime.now(timezone.utc).isoformat()


def pipe_lines(stream, stop=lambda: False, poll_seconds=1, max_line=1 << 20):
    """Read raw bytes, draining a whole ready chunk before selecting again.

    select cannot observe TextIOWrapper's prefetched bytes. In particular, a
    final timeout line in a burst must arrive without requiring another write.
    None is a heartbeat; an incomplete final line is retained at EOF.
    """
    pending = bytearray()
    while not stop():
        if not select.select([stream.fileno()], [], [], poll_seconds)[0]:
            yield None
            continue
        chunk = os.read(stream.fileno(), 65536)
        if not chunk:
            if pending:
                yield bytes(pending)
            return
        pending.extend(chunk)
        while True:
            end = pending.find(b"\n")
            if end < 0:
                break
            if end > max_line:
                raise ValueError("diagnostic log line exceeds declared bound")
            line = bytes(pending[:end])
            del pending[:end + 1]
            yield line
        if len(pending) > max_line:
            raise ValueError("diagnostic log line exceeds declared bound")


def docker(*args, timeout=15):
    return subprocess.check_output(["docker", *args], text=True, timeout=timeout).strip()


def inspect(cid):
    return json.loads(docker("inspect", cid))[0]


def identity(cid, expected):
    info = inspect(cid)
    if info["Config"]["Labels"].get("eth2030.qualification") != "remaining-code-20261122":
        raise ValueError("target is not an owned qualification container")
    actual = docker("exec", cid, "sha256sum", "/proc/1/exe").split()[0]
    if actual != expected:
        raise ValueError("actual diagnostic executable differs from record")
    result = {"container": cid, "started": info["State"]["StartedAt"],
              "host_pid": info["State"]["Pid"], "image": info["Image"], "binary_sha256": actual}
    try:
        result["start_ticks"] = start_ticks(result["host_pid"])
    except (OSError, IndexError) as error:
        result["storage_identity_error"] = repr(error)
    return result


def namespace_storage(cid, expected_start, image):
    """Read an owned target through its PID namespace, without attaching a debugger."""
    if expected_start is None:
        return {"error": "target process start identity is unavailable"}
    name = "storage-wait-" + str(time.time_ns())
    module = Path(__file__).with_name("engine_storage_snapshot.py").resolve()
    command = ["docker", "run", "--name", name, "--rm", "--network", "none",
               "--pid", "container:" + cid, "--read-only", "--cap-drop", "ALL",
               "--cap-add", "SYS_PTRACE", "--user", "0", "--cpus", "0.25",
               "--memory", "134217728", "--mount",
               "type=bind,src=" + str(module) + ",dst=/engine_storage_snapshot.py,readonly",
               "--entrypoint", "python3", image, "-B", "-c",
               "import sys,json;sys.path.insert(0,'/');from engine_storage_snapshot import snapshot;print(json.dumps(snapshot(1,sys.argv[1])))",
               str(expected_start)]
    result = {"pid_namespace": "container:" + cid, "image": image,
              "access": "Read-only procfs with SYS_PTRACE permission; no debugger attachment.",
              "started_at": utc()}
    try:
        process = subprocess.run(command, capture_output=True, timeout=8)
        result["exit_code"] = process.returncode
        if process.returncode:
            result["error"] = process.stderr[:4096].decode(errors="replace")
        elif len(process.stdout) > 32 << 20:
            result["error"] = "namespace storage output exceeded declared32MiB bound"
        else:
            observation = json.loads(process.stdout)
            observation["namespace_pid"] = observation.pop("host_pid")
            result["observation"] = observation
    except Exception as error:
        result["error"] = repr(error)
    finally:
        # A timed-out docker CLI does not stop its daemon-managed helper.
        try:
            cleanup = subprocess.run(["docker", "rm", "-f", name], capture_output=True, timeout=10)
            result["helper_remove_exit"] = cleanup.returncode
        except Exception as error:
            result["cleanup_error"] = repr(error)
        result["finished_at"] = utc()
    return result


def capture_storage(cid, expected, destination, proc_image=None):
    current = inspect(cid)
    if (current["State"]["StartedAt"] != expected["started"]
            or current["State"]["Pid"] != expected["host_pid"]):
        raise ValueError("storage target restarted before capture")
    result = storage_snapshot(expected["host_pid"], expected.get("start_ticks"), current.get("Mounts", []))
    if proc_image:
        result["namespace_proc"] = namespace_storage(cid, expected.get("start_ticks"), proc_image)
        observation = result["namespace_proc"].get("observation", {})
        result["namespace_identity_matches"] = observation.get("identity_matches")
        if result["namespace_identity_matches"] is False:
            result["identity_matches"] = False
    after = inspect(cid)["State"]
    if after["StartedAt"] != expected["started"] or after["Pid"] != expected["host_pid"]:
        result.update(identity_matches=False, target_restarted_during_capture=True)
    destination.write_text(json.dumps(result, indent=2) + "\n")
    return {"at": result["at"], "path": destination.name,
            "identity_matches": result["identity_matches"],
            "sha256": hashlib.sha256(destination.read_bytes()).hexdigest()}


def native_profile(url, destination):
    with urllib.request.urlopen(url, timeout=5) as response:
        body = response.read((8 << 20) + 1)
    destination.write_bytes(body[:8 << 20])
    return {"at": utc(), "bytes": len(body), "truncated": len(body) > (8 << 20),
            "sha256": hashlib.sha256(body[:8 << 20]).hexdigest()}


def cl_backtrace(cid, image, destination):
    # A timed-out docker CLI alone does not stop a daemon-managed debugger.
    # Always remove the owned helper so it cannot keep the target traced.
    name = "engine-stall-debug-" + destination.parent.name
    command = ["docker", "run", "--name", name, "--rm", "--network", "none",
               "--pid", "container:" + cid, "--cpus", "1", "--memory", "1073741824",
               "--cap-add", "SYS_PTRACE", "--security-opt", "seccomp=unconfined",
               "--entrypoint", "gdb", image, "-batch", "-q",
               "-ex", "set debuginfod enabled off", "-ex", "set pagination off",
               "-ex", "file /proc/1/exe", "-ex", "set sysroot /proc/1/root",
               "-ex", "thread apply all bt 45", "-ex", "detach", "-p", "1"]
    result = {"interference": "Debugger attachment on an owned diagnostic; consult caller trigger and timestamps; never clean acceptance time.",
              "started_at": utc()}
    try:
        with destination.open("wb") as log:
            process = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=20)
        result["exit_code"] = process.returncode
    except Exception as error:
        result["error"] = repr(error)
    finally:
        cleanup = subprocess.run(["docker", "rm", "-f", name], capture_output=True, timeout=10)
        result.update(finished_at=utc(), helper_remove_exit=cleanup.returncode)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--record", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--cl-debug-image", help="Optional locally built GDB image; attaches only after a timeout")
    parser.add_argument("--storage-proc-image", help="Optional local Python3 image for read-only owned PID-namespace procfs access; no debugger attachment")
    args = parser.parse_args()
    out = args.output
    out.mkdir(mode=0o700, exist_ok=False)
    report = {"status": "initializing", "started_at": utc(), "captures": 0,
              "collector_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              "storage_collector_sha256": hashlib.sha256(Path(__file__).with_name("engine_storage_snapshot.py").read_bytes()).hexdigest(),
              "actual48h_passed": False,
              "scope": "Read-only diagnostic capture, with explicitly recorded optional post-timeout debugger interference. No operating acceptance."}
    def save():
        report["updated_at"] = utc()
        temporary = out / "result.json.tmp"
        temporary.write_text(json.dumps(report, indent=2) + "\n")
        temporary.replace(out / "result.json")
    save()
    try:
        record = json.loads(args.record.read_text())
        profile = record["profile"]
        el, cl = record["el"], record["cl"]
        identities = {"el": identity(el, profile["binary_sha256"]),
                      "cl": identity(cl, profile["cl_binary_sha256"])}
        port = inspect(el)["NetworkSettings"]["Ports"]["9001/tcp"][0]["HostPort"]
        url = "http://127.0.0.1:" + port + "/debug/pprof/goroutine?debug=2"
        storage_image = None
        if args.storage_proc_image:
            storage_image = json.loads(docker("image", "inspect", args.storage_proc_image))[0]["Id"]
            report["storage_proc_image"] = storage_image
        if args.cl_debug_image:
            docker("image", "inspect", args.cl_debug_image)
        report.update(status="watching", identities=identities,
                      source=profile["source"], cl_source=profile["cl_source"],
                      baseline=native_profile(url, out / "baseline-goroutines.txt"))
        save()
    except BaseException as error:
        report.update(status="collector_failed", error=repr(error), finished_at=utc())
        save()
        raise
    stopped = False
    def stop(_signal, _frame):
        nonlocal stopped
        stopped = True
    signal.signal(signal.SIGINT, stop)
    signal.signal(signal.SIGTERM, stop)
    process = None
    try:
        process = subprocess.Popen(["docker", "logs", "--follow", "--since", utc(), el],
                                   stdout=subprocess.PIPE, stderr=subprocess.STDOUT, bufsize=0)
        for raw in pipe_lines(process.stdout, lambda: stopped):
            if raw is None:
                if not inspect(el)["State"]["Running"]:
                    break
                save()
                continue
            if b"engine_request_timeout" not in raw:
                continue
            directory = out / ("stack-" + datetime.now(timezone.utc).strftime("%H%M%S%f"))
            directory.mkdir()
            capture = {"at": utc(), "trigger": raw.decode(errors="replace"), "profiles": []}
            for role, cid in [("el", el), ("cl", cl)]:
                current = inspect(cid)
                if current["State"]["StartedAt"] != identities[role]["started"]:
                    raise ValueError("diagnostic target restarted before capture")
            for index, delay in enumerate([0, .5, 2, 5]):
                if delay:
                    time.sleep(delay)
                # Start both storage observations and the Go profile together: a slow
                # profile endpoint must not defer the first syscall/fd observation.
                with ThreadPoolExecutor(max_workers=3) as pool:
                    pending = {role: pool.submit(capture_storage, cid, identities[role],
                               directory / (role + "-storage-" + str(index) + ".json"), storage_image)
                               for role, cid in [("el", el), ("cl", cl)]}
                    profile = pool.submit(native_profile, url, directory / ("goroutines-" + str(index) + ".txt"))
                    for role, future in pending.items():
                        try:
                            frame = future.result()
                        except Exception as error:
                            frame = {"at": utc(), "error": repr(error), "identity_matches": False}
                        capture.setdefault("storage_frames", []).append({"role": role, "frame": index, **frame})
                    try:
                        capture["profiles"].append(profile.result())
                    except Exception as error:
                        capture["profiles"].append({"at": utc(), "error": repr(error)})
            if args.cl_debug_image:
                capture["cl_debugger"] = cl_backtrace(cl, args.cl_debug_image, directory / "cl-threads.txt")
            for role, cid in [("el", el), ("cl", cl)]:
                capture[role + "_state"] = inspect(cid)["State"]
                for name, command in [("pressure", ["exec", cid, "sh", "-c", "cat /sys/fs/cgroup/cpu.stat /sys/fs/cgroup/cpu.pressure /sys/fs/cgroup/io.pressure /sys/fs/cgroup/memory.pressure"]),
                                      ("tail", ["logs", "--tail", "120", cid])]:
                    try:
                        value = subprocess.run(["docker", *command], capture_output=True, timeout=10)
                        (directory / (role + "-" + name + ".txt")).write_bytes(value.stdout + value.stderr)
                        capture[role + "_" + name + "_exit"] = value.returncode
                    except Exception as error:
                        capture[role + "_" + name + "_error"] = repr(error)
            (directory / "capture.json").write_text(json.dumps(capture, indent=2) + "\n")
            report.update(captures=report["captures"] + 1, last_capture=str(directory))
            save()
            print("Captured Engine timeout", directory, flush=True)
        report["status"] = "stopped_by_owner" if stopped else "owned_log_stream_ended"
        if not stopped and inspect(el)["State"]["Running"]:
            raise RuntimeError("Docker log stream ended while the diagnostic is still running")
    except BaseException as error:
        report.update(status="collector_failed", error=repr(error))
        raise
    finally:
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
            process.stdout.close()
        report["finished_at"] = utc()
        save()


if __name__ == "__main__":
    main()
