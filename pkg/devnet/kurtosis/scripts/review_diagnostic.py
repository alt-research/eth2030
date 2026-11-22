#!/usr/bin/env python3
"""Review bounded native diagnostics without accepting a legacy no-stall label.

This reviews operational evidence and reported cryptographic windows. It does
not verify BLS signatures itself, certify 48 hours, or close an earlier fault.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import subprocess
import time
import urllib.request

READY_THRESHOLD = 5.5 * 3600
MAX_SAMPLE_GAP = 60
FINALITY_STALL = 600
MEMORY = 4096000000


def timestamp(value):
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    if parsed.tzinfo is None:
        raise ValueError("timestamp has no timezone")
    return parsed.timestamp()


def review(state, rows, bls, now):
    """Fail closed on operational evidence; keep time and BLS gates separate."""
    errors = []
    report = {
        "status": "pending", "errors": errors,
        "actual48h_passed": False, "original_fault_resolved": False,
        "stopped_head_closure_verified": False,
        "bls_signatures_verified_by_this_tool": False,
    }
    try:
        start = timestamp(state["started_at"])
        finished = timestamp(state["finished_at"]) if "finished_at" in state else None
        end = finished if finished is not None else now
        if end < start or end > now + 5:
            raise ValueError("invalid runtime clock")
        report["runtime_seconds"] = end - start
        if not rows:
            raise ValueError("no observation samples")
        first_healthy = next((r for r in rows if r.get("ready") is True and all(
            r.get("sync", {}).get(key) is False
            for key in ("is_syncing", "is_optimistic", "el_offline")
        )), None)
        if first_healthy:
            threshold = timestamp(first_healthy["at"]) + READY_THRESHOLD
            report["threshold_at"] = datetime.fromtimestamp(threshold, timezone.utc).isoformat()
            report["threshold_time_reached"] = now >= threshold
        if state["samples"] != len(rows):
            raise ValueError("summary count differs from observation ledger")
        ready = None
        previous = start
        height = advanced = None
        ready_rows = []
        max_gap = 0
        for index, row in enumerate(rows):
            at = timestamp(row["at"])
            gap = at - previous
            if not 0 <= gap <= MAX_SAMPLE_GAP or (index > 0 and at == previous):
                raise ValueError("missing, duplicate, regressed or excessively delayed sample")
            if at > now + 5 or (finished is not None and at > finished):
                raise ValueError("sample outside runtime")
            max_gap = max(max_gap, gap)
            previous = at
            healthy = row.get("ready") is True and all(
                row.get("sync", {}).get(key) is False
                for key in ("is_syncing", "is_optimistic", "el_offline")
            )
            if ready is None and healthy:
                ready, height, advanced = at, row["finalized_number"], at
            if ready is None:
                continue
            if not healthy:
                raise ValueError("execution lost readiness after first ready sample")
            if not 0 <= row["reference_lag"] <= 128:
                raise ValueError("reference finalized lag exceeds bound")
            current = row["finalized_number"]
            if type(current) is not int or current < height:
                raise ValueError("finalized height regressed or is malformed")
            if current > height:
                height, advanced = current, at
            if at - advanced >= FINALITY_STALL:
                raise ValueError("finalized execution stalled")
            if set(row["memory"]) != {"el", "cl"}:
                raise ValueError("missing daemon memory evidence")
            for memory in row["memory"].values():
                if memory["memory_max"] != MEMORY or not 0 < memory["process_hwm"] < MEMORY:
                    raise ValueError("resource profile or process high-water mark failed")
                if memory["events"]["oom"] or memory["events"]["oom_kill"]:
                    raise ValueError("OOM observed")
            ready_rows.append(row)
        if end - previous > MAX_SAMPLE_GAP:
            raise ValueError("observation ledger is stale")
        if state["last"] != {k: v for k, v in rows[-1].items() if k != "at"}:
            raise ValueError("summary last sample differs from observation ledger")
        report["max_sample_gap_seconds"] = max_gap
        report["memory_limit_events"] = {role: max(r["memory"][role]["events"].get("max", 0) for r in ready_rows)
                                         for role in ("el", "cl")} if ready_rows else {}
        report["ready_seconds"] = 0 if ready is None else previous - ready
        report["first_ready_at"] = None if ready is None else datetime.fromtimestamp(ready, timezone.utc).isoformat()
        threshold = None if ready is None else ready + READY_THRESHOLD
        report["threshold_at"] = None if threshold is None else datetime.fromtimestamp(threshold, timezone.utc).isoformat()
        report["post_threshold_samples"] = sum(timestamp(r["at"]) >= threshold for r in ready_rows) if threshold else 0
        report["ready_5_5h_covered"] = bool(report["post_threshold_samples"])
        windows = bls.get("windows", [])
        if bls.get("status") == "verification_failed":
            raise ValueError("BLS window controller failed")
        first_slot = int(ready_rows[0]["sync"]["head_slot"]) if ready_rows else None
        through = first_slot
        missing = set()
        for window in windows:
            # An active verification is pending, never a pass.
            if window.get("passed") is not True:
                report["bls_window_pending"] = True
                continue
            value = window["beacon_window"]
            if through is None or window["first"] != through or value["first"] != through:
                raise ValueError("disconnected or overlapping BLS window boundary")
            if value["stop"] != window["through"] + 1 or window["through"] < through:
                raise ValueError("invalid BLS window endpoint")
            if value["signature_participants"] < 1 or not 0 < value["parent_hash_links"] < 4096:
                raise ValueError("invalid reported BLS participants or ancestry bound")
            missing.update(value["missing_slots"])
            through = window["through"]
        report["reported_bls_windows"] = sum(w.get("passed") is True for w in windows)
        report["reported_bls_through_slot"] = through
        report["reported_missing_slots"] = sorted(missing)
        report["bls_window_reports_cover_observed_slots"] = bool(
            ready_rows and windows and through == int(ready_rows[-1]["sync"]["head_slot"])
            and all(w.get("passed") is True for w in windows)
        )
        if finished is not None:
            if state["status"] != "bounded_diagnostic_finished_without_stall":
                raise ValueError("runtime controller ended with a failure")
            if report["runtime_seconds"] < 24300:
                raise ValueError("runtime ended before the declared 6h45m duration")
            cleanup = state["cleanup"]
            if set(cleanup) != {"el", "cl"} or any(
                x["stop_exit"] != 0 or x["state"]["ExitCode"] != 0 or x["state"]["OOMKilled"]
                for x in cleanup.values()
            ):
                raise ValueError("runtime cleanup failed")
            report["status"] = "runtime_finished_stopped_head_and_crypto_closure_pending"
        else:
            report["status"] = "observing_after_5_5h" if report["ready_5_5h_covered"] else "observing_before_5_5h"
    except (KeyError, TypeError, ValueError, OverflowError) as error:
        errors.append(str(error))
        report["status"] = "failed"
    return report


def load_review(run, now):
    # Controller writes summary after appending the ledger. Retry a concurrent
    # publication rather than persisting a torn view as acceptance evidence.
    for _ in range(3):
        try:
            state = json.loads((run / "result.json").read_text())
            rows = [json.loads(line) for line in (run / "observations/samples.jsonl").read_text().splitlines()]
        except json.JSONDecodeError:
            time.sleep(0.05)
            continue
        if state["samples"] == len(rows):
            break
        time.sleep(0.05)
    else:
        raise ValueError("could not obtain a coherent controller/ledger publication")
    profile = json.loads((run / "containers.json").read_text())["profile"]
    build = json.loads((run / "BUILD.json").read_text())
    for key, build_key in (("source", "source"), ("binary_sha256", "sha256"),
                           ("cl_source", "cl_source"), ("cl_binary_sha256", "cl_sha256")):
        if profile[key] != build[build_key]:
            raise ValueError("runtime/build identity mismatch: " + key)
    if profile["duration_seconds"] != 24300:
        raise ValueError("bounded diagnostic duration changed")
    path = run / "bounded-bls-windows/result.json"
    bls = json.loads(path.read_text()) if path.exists() else {}
    return review(state, rows, bls, now), state


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True, timeout=15)


def capture(run, output, record):
    """Read-only point snapshot; never signal, pause or restart either node."""
    at = datetime.now(timezone.utc).isoformat()
    directory = output / ("capture-" + str(time.time_ns()))
    directory.mkdir()
    receipt = {"at": at, "errors": [], "scope": "Passive snapshot, not fault closure"}
    for role in ("el", "cl"):
        try:
            info = json.loads(docker("inspect", record[role]))[0]
            receipt[role] = {"state": info["State"], "restart_count": info["RestartCount"]}
            if info["RestartCount"] or info["State"]["StartedAt"] != record[role + "_started"]:
                raise ValueError("runtime restarted or identity changed")
            if info["HostConfig"]["Memory"] != MEMORY or info["HostConfig"]["NanoCpus"] != 2000000000:
                raise ValueError("runtime resource limits changed")
            if not info["State"]["Running"]:
                raise ValueError("owned runtime stopped")
            expected = record["profile"]["binary_sha256" if role == "el" else "cl_binary_sha256"]
            verified = record.setdefault("verified_executables", {})
            if role not in verified or time.monotonic() - verified[role] >= 3600:
                actual = docker("exec", record[role], "sha256sum", "/proc/1/exe").split()[0]
                if actual != expected:
                    raise ValueError("runtime executable changed")
                verified[role] = time.monotonic()
            receipt[role]["pinned_binary_sha256"] = expected
            pressure = docker("exec", record[role], "sh", "-c", "cat /sys/fs/cgroup/cpu.pressure /sys/fs/cgroup/io.pressure /sys/fs/cgroup/memory.pressure /sys/fs/cgroup/memory.events")
            (directory / (role + "-pressure.txt")).write_text(pressure)
            if role == "el":
                port = info["NetworkSettings"]["Ports"]["9001/tcp"][0]["HostPort"]
                with urllib.request.urlopen("http://127.0.0.1:" + port + "/debug/pprof/goroutine?debug=2", timeout=5) as response:
                    body = response.read((8 << 20) + 1)
                if len(body) > 8 << 20:
                    raise ValueError("goroutine profile exceeded bound")
                (directory / "goroutines.txt").write_bytes(body)
                receipt["stack_sha256"] = hashlib.sha256(body).hexdigest()
        except Exception as error:
            receipt["errors"].append(role + ": " + repr(error))
    (directory / "receipt.json").write_text(json.dumps(receipt, indent=2) + "\n")
    return receipt, directory.name


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--watch", action="store_true", help="Append passive minute snapshots alongside an existing run")
    args = parser.parse_args()
    args.output.mkdir(mode=0o700, exist_ok=False)
    record = json.loads((args.run / "containers.json").read_text())
    identities = json.loads((args.run / "observations/identities.json").read_text())
    for role in ("el", "cl"):
        record[role + "_started"] = identities[role]["started"]
    metadata = {"attached_at": datetime.now(timezone.utc).isoformat(), "collector_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), "run": str(args.run), "watch": args.watch, "interval_seconds": 60, "scope": "Periodic evidence begins at attachment; earlier overwritten ring profiles are not recovered"}
    (args.output / "profile.json").write_text(json.dumps(metadata, indent=2) + "\n")
    threshold_saved = False
    failures = []
    while True:
        before = time.monotonic()
        try:
            result, state = load_review(args.run, time.time())
            if args.watch and "finished_at" not in state:
                receipt, directory = capture(args.run, args.output, record)
                result["capture"] = directory
                if receipt["errors"]:
                    failures.extend(receipt["errors"])
                # Capture at the time threshold even if readiness was lost or
                # the ledger stalled: that is exactly when a stack matters.
                if result.get("threshold_time_reached") and receipt.get("stack_sha256") and not threshold_saved:
                    (args.output / "THRESHOLD_5_5H.json").write_text(json.dumps({"review": result, "capture": directory, "at": receipt["at"]}, indent=2) + "\n")
                    threshold_saved = True
            failures.extend(result["errors"])
            result["retained_guard_failures"] = failures
            result["evidence_status"] = "failed" if failures else "pending_stopped_head_and_crypto_closure"
            result["reviewed_at"] = datetime.now(timezone.utc).isoformat()
            with (args.output / "reviews.jsonl").open("a") as stream:
                stream.write(json.dumps(result) + "\n")
            temporary = args.output / "result.tmp"
            temporary.write_text(json.dumps(result, indent=2) + "\n")
            temporary.replace(args.output / "result.json")
            if not args.watch or "finished_at" in state:
                return 1 if result["errors"] or failures else 0
        except Exception as error:
            failures.append(repr(error))
            with (args.output / "errors.jsonl").open("a") as stream:
                stream.write(json.dumps({"at": datetime.now(timezone.utc).isoformat(), "error": repr(error)}) + "\n")
            if not args.watch:
                return 1
        time.sleep(max(0, 60 - (time.monotonic() - before)))


if __name__ == "__main__":
    raise SystemExit(main())
