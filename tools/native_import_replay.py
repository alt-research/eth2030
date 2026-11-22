#!/usr/bin/env python3
"""Run genuine native imports immediately from an immutable mature snapshot.

Build the node test binary from the candidate revision first. This tool restores
an exclusive disposable database, fetches real reference RLP, and executes the
native/Engine/proof/restart test under the original resource limits. It never
changes source archives, validator nodes, validation rules or elapsed soak time.
"""
import argparse
import hashlib
import json
import subprocess
import time
import urllib.request
from datetime import datetime, timezone
from pathlib import Path


def digest(path):
    value = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b""):
            value.update(chunk)
    return value.hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--snapshot", type=Path, required=True)
    parser.add_argument("--snapshot-sha256", required=True)
    parser.add_argument("--genesis", type=Path, required=True)
    parser.add_argument("--reference-rpc", required=True)
    parser.add_argument("--first", type=int, required=True)
    parser.add_argument("--count", type=int, default=256)
    parser.add_argument("--test-binary", type=Path, required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.first < 1 or args.count < 1:
        parser.error("first and count must be positive")
    out = args.output.resolve()
    out.mkdir(mode=0o700, exist_ok=False)
    report = {"passed": False, "actual48h_passed": False,
              "scope": "Native execution/Engine/MPT/restart event replay; CL BLS and wall-clock stability are independent gates.",
              "source": args.source, "started_at": datetime.now(timezone.utc).isoformat(),
              "first": args.first, "blocks": args.count,
              "memory_bytes": 4096000000, "cpus": 2}
    start = time.monotonic()
    cid = None
    try:
        snapshot, genesis, binary = (p.resolve() for p in (args.snapshot, args.genesis, args.test_binary))
        if digest(snapshot) != args.snapshot_sha256:
            raise ValueError("immutable snapshot digest differs")
        report.update(snapshot_sha256=args.snapshot_sha256, genesis_sha256=digest(genesis), binary_sha256=digest(binary))
        data = out / "snapshot"
        data.mkdir(mode=0o700)
        with (out / "restore.log").open("wb") as log:
            subprocess.run(["tar", "-xf", str(snapshot), "-C", str(data)], stdout=log, stderr=subprocess.STDOUT, check=True)
        if not (data / "execution-data/chaindata").is_dir():
            raise ValueError("snapshot must contain execution-data/chaindata")
        rows = out / "blocks.jsonl"
        with rows.open("x") as stream:
            for number in range(args.first, args.first + args.count):
                request = [{"jsonrpc": "2.0", "id": 1, "method": "debug_getRawBlock", "params": [hex(number)]},
                           {"jsonrpc": "2.0", "id": 2, "method": "eth_getBlockByNumber", "params": [hex(number), False]}]
                req = urllib.request.Request(args.reference_rpc, data=json.dumps(request).encode(), headers={"Content-Type": "application/json"})
                with urllib.request.urlopen(req, timeout=10) as response:
                    raw = response.read()
                with (out / "reference-exchanges.jsonl").open("a") as capture:
                    capture.write(json.dumps({"request": request, "response": raw.decode()}) + "\n")
                replies = json.loads(raw)
                if not isinstance(replies, list) or len(replies) != 2 or {x["id"] for x in replies} != {1, 2}:
                    raise ValueError("reference batch identity differs")
                values = {x["id"]: x for x in replies}
                if any("error" in x or not x.get("result") for x in replies):
                    raise ValueError("reference block missing or RPC failed")
                if int(values[2]["result"]["number"], 16) != number:
                    raise ValueError("reference block number differs")
                stream.write(json.dumps({"hash": values[2]["result"]["hash"], "rlp": values[1]["result"]}) + "\n")
        manifest = out / "manifest.json"
        manifest.write_text(json.dumps({"disposable_clone": True, "datadir": str(data / "execution-data"),
                                       "genesis": str(genesis), "rows": str(rows), "rows_sha256": digest(rows)}, indent=2) + "\n")
        report["rows_sha256"] = digest(rows)
        command = ["docker", "create", "--network", "none", "--cpus", "2", "--memory", "4096000000",
                   "--env", "GOMAXPROCS=2", "--entrypoint", str(binary)]
        for source, readonly in [(out, False), (genesis, True), (binary, True)]:
            command += ["--mount", "type=bind,src=" + str(source) + ",dst=" + str(source) + (",readonly" if readonly else "")]
        command += [args.image, "-test.v", "-test.run=^TestNativeMatureImportReplay$", "-test.timeout=8m",
                    "-native-import-replay=" + str(manifest)]
        cid = subprocess.check_output(command, text=True, timeout=45).strip()
        report["container"] = cid
        execution_start = time.monotonic()
        with (out / "replay.log").open("wb") as log:
            process = subprocess.run(["docker", "start", "-a", cid], stdout=log, stderr=subprocess.STDOUT, timeout=540)
        info = json.loads(subprocess.check_output(["docker", "inspect", cid], text=True, timeout=15))[0]
        report.update(execution_seconds=time.monotonic() - execution_start, exit_code=process.returncode, state=info["State"])
        text = (out / "replay.log").read_text()
        if (process.returncode != 0 or info["State"]["ExitCode"] != 0 or info["State"]["OOMKilled"]
                or "--- PASS: TestNativeMatureImportReplay" not in text
                or "durable native restart passed" not in text or "REPLAY blocks=" + str(args.count) + " " not in text):
            raise RuntimeError("native replay did not fully execute and qualify; retain replay.log and before-cleanup waiting stack")
        report["passed"] = True
    except BaseException as error:
        report["error"] = repr(error)
        if cid:
            subprocess.run(["docker", "stop", "--time", "30", cid], capture_output=True, timeout=45)
        raise
    finally:
        report["elapsed_seconds"] = time.monotonic() - start
        (out / "result.json").write_text(json.dumps(report, indent=2) + "\n")
        print(json.dumps({k: report[k] for k in ["passed", "actual48h_passed", "elapsed_seconds"]}), flush=True)


if __name__ == "__main__":
    main()
