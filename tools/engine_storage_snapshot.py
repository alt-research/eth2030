"""Bounded, read-only Linux storage evidence for an already-owned diagnostic PID.

Raw syscall/wchan/fd observations are evidence, not an attribution or acceptance.
Permission failures, races, truncation and PID reuse remain explicit.
"""
import os
import stat
from pathlib import Path
from datetime import datetime, timezone

MAX_TEXT = 16384
MAX_THREADS = 256
MAX_FDS = 512


def read_text(path, limit=MAX_TEXT):
    try:
        with path.open("rb") as stream:
            body = stream.read(limit + 1)
        return {"text": body[:limit].decode(errors="replace"),
                "truncated": len(body) > limit}
    except OSError as error:
        return {"error": str(error), "errno": error.errno}


def start_ticks(pid):
    # comm may contain spaces and parentheses. Fields after its final ')' start at3.
    return (Path("/proc") / str(pid) / "stat").read_text().rsplit(")", 1)[1].split()[19]


def entries(path, limit):
    try:
        # Retain only the bounded prefix without materializing an unbounded fd set.
        with os.scandir(path) as stream:
            names = []
            for item in stream:
                if len(names) == limit:
                    return names, True, None
                names.append(item.name)
        return names, False, None
    except OSError as error:
        return [], False, {"error": str(error), "errno": error.errno}


def snapshot(pid, expected_start, mounts=()):
    result = {"at": datetime.now(timezone.utc).isoformat(), "host_pid": pid,
              "expected_start_ticks": expected_start, "machine": os.uname().machine,
              "kernel_release": os.uname().release, "original_fault_resolved": False,
              "scope": "Read-only diagnostic observation; missing evidence is not success."}
    root = Path("/proc") / str(pid)
    try:
        actual = start_ticks(pid)
    except (OSError, IndexError) as error:
        result.update(identity_matches=False, identity_error=repr(error))
        return result
    result.update(start_ticks=actual, identity_matches=actual == expected_start)
    if not result["identity_matches"]:
        return result
    result["process"] = {name: read_text(root / name) for name in ("stat", "io", "status", "mountinfo", "cgroup")}
    threads, clipped, error = entries(root / "task", MAX_THREADS)
    result.update(threads_truncated=clipped, thread_listing_error=error, threads=[])
    for tid in threads:
        task = root / "task" / tid
        result["threads"].append({"tid": tid, **{name: read_text(task / name) for name in ("syscall", "wchan", "stack")}})
    fds, clipped, error = entries(root / "fd", MAX_FDS)
    result.update(fds_truncated=clipped, fd_listing_error=error, fds=[])
    devices = set()
    for fd in fds:
        link = root / "fd" / fd
        item = {"fd": fd, "binding_stable": False, "fdinfo": read_text(root / "fdinfo" / fd)}
        try:
            target = os.readlink(link)
            info = link.stat()
            item.update(target=target, regular_file=stat.S_ISREG(info.st_mode),
                        inode=info.st_ino, device=f"{os.major(info.st_dev)}:{os.minor(info.st_dev)}")
            if item["regular_file"]:
                devices.add(item["device"])
                # Bind mounts preserve the host source device even when the target is
                # shown through a container path. Do not guess an overlay backing disk.
                candidates = [m for m in mounts if m.get("Type") == "bind"
                              and (target == m["Destination"] or target.startswith(m["Destination"].rstrip("/") + "/"))]
                if candidates:
                    mount = max(candidates, key=lambda m: len(m["Destination"]))
                    source = Path(mount["Source"])
                    device = source.stat().st_dev
                    item["bind_mount"] = {"source": str(source), "destination": mount["Destination"],
                                          "device": f"{os.major(device)}:{os.minor(device)}"}
                    devices.add(item["bind_mount"]["device"])
        except OSError as error:
            item.update(error=str(error), errno=error.errno)
        if "target" in item:
            try:
                after = link.stat()
                item["binding_stable"] = (os.readlink(link) == item["target"]
                                          and after.st_ino == item["inode"]
                                          and f"{os.major(after.st_dev)}:{os.minor(after.st_dev)}" == item["device"])
            except OSError as error:
                item.update(binding_stable=False, binding_error=str(error))
        result["fds"].append(item)
    cgroup = result["process"]["cgroup"].get("text", "")
    result["cgroup_v2"] = {}
    for line in cgroup.splitlines():
        if line.startswith("0::"):
            relative = line[3:].lstrip("/")
            path = (Path("/sys/fs/cgroup") / relative).resolve()
            if path.is_relative_to(Path("/sys/fs/cgroup")):
                result["cgroup_v2"] = {name: read_text(path / name) for name in ("io.stat", "io.pressure", "cpu.stat", "memory.events")}
            else:
                result["cgroup_v2"] = {"error": "cgroup path is outside the host cgroup mount"}
    diskstats = read_text(Path("/proc/diskstats"), 1 << 20)
    if "text" in diskstats:
        diskstats["text"] = "\n".join(line for line in diskstats["text"].splitlines()
                                         if len(line.split()) >= 3 and ":".join(line.split()[:2]) in devices)
    result["diskstats"] = diskstats
    result["devices"] = {device: read_text(Path("/sys/dev/block") / device / "uevent") for device in sorted(devices)}
    try:
        result["identity_matches"] = start_ticks(pid) == expected_start
    except (OSError, IndexError) as error:
        result.update(identity_matches=False, identity_error=repr(error))
    result["finished_at"] = datetime.now(timezone.utc).isoformat()
    return result
