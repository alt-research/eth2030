#!/usr/bin/env python3
"""Independently close the frozen 48-hour run; prefix audits never claim completion."""

import argparse
import hashlib
import json
import math
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import urlsplit

SPEC = "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a"
VERIFIER_SHA = "c36bdfcc2169a3be3ca667ea19a4855bc8efd1c31513b63b77d308cfd76072a9"
LEDGER_SHA = "1410810997dec76e777fd541f0883ba79bacb1b808e0c81d06755020db8ac123"
REFERENCE_SHAS = {
    "eth_consensus_specs.gloas.mainnet": "533f62dc20e90272a314365ce900e0bef0a70769dc71a5eba97d2fae262a3cf3",
    "eth_consensus_specs.fulu.mainnet": "f0c74b75cb302065686673cc582531194f06b1001b991788a0c78ebc3c4560f8",
    "eth_consensus_specs.utils.bls": "4f16de600dc05f6c9ff917dbbd8c577a1136c5acacc903323715c462c614c904",
}
ARKWORKS_SHA = "e70210171261b0e435acaf09dc68d6c9fe35d1bc170aa3dce7fbe3ac65cda3db"
EXECUTABLES = {
    "el": "4e0f7a54765ce0992d0477d65565fcb41c27a8bac6525bb6f2497721939a9043",
    "cl": "3158ae6dc22db03e406b710bd9188ade72d9aad1c0d8cc4a493e4a0d09d6f142",
}
SOURCE = "2c0df12ff4c89d6407c5f27c8de8ea56517d7a4e"
LABEL = "remaining-code-20261122"
MEMORY = 4096000000
DURATION = 172800
MAX_SAMPLE_GAP = 60
ADDRESSES = (
    "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf",
    "0xee9ede56b61f213e69233a001b17fd6414c93675",
)


class EvidenceError(ValueError):
    pass


def require(condition, message):
    # Qualification must remain fail-closed even when the caller uses python -O.
    if not condition:
        raise EvidenceError(message)


def digest(body):
    return hashlib.sha256(body).hexdigest()


def utc():
    return datetime.now(timezone.utc).isoformat()


def timestamp(value):
    result = datetime.fromisoformat(value.replace("Z", "+00:00"))
    require(result.tzinfo is not None, "evidence timestamp has no timezone")
    return result.timestamp()


def read_json(path):
    # The legacy collector writes its small result in place. Retry a concurrent
    # partial write, but never silently accept a permanently truncated result.
    for attempt in range(5):
        try:
            return json.loads(Path(path).read_bytes())
        except (json.JSONDecodeError, FileNotFoundError):
            if attempt == 4:
                raise EvidenceError(f"missing or incomplete JSON: {path}") from None
            time.sleep(0.05)


def write_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".tmp")
    temporary.write_text(json.dumps(value, indent=2) + "\n")
    temporary.replace(path)


def run_json(command, timeout):
    result = subprocess.run(command, capture_output=True, text=True, timeout=timeout)
    require(result.returncode == 0, f"consumer failed: {command[0]}: {result.stderr[-1000:]}")
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as error:
        raise EvidenceError("consumer did not return a JSON result") from error


def root_bytes(value):
    require(isinstance(value, str) and value.startswith("0x") and len(value) == 66,
            "invalid 32-byte root")
    try:
        return bytes.fromhex(value[2:])
    except ValueError as error:
        raise EvidenceError("non-hexadecimal root") from error


def beacon_header_root(message):
    # Phase0 BeaconBlockHeader container: two uint64 chunks and three Roots,
    # padded to eight leaves. This hashes ancestry; BLS remains the unchanged
    # independently regenerated consensus consumer, never a native EL helper.
    leaves = []
    for key in ("slot", "proposer_index"):
        value = int(message[key])
        require(0 <= value < 2**64, "header uint64 out of range")
        leaves.append(value.to_bytes(8, "little") + bytes(24))
    leaves += [root_bytes(message[key]) for key in ("parent_root", "state_root", "body_root")]
    leaves += [bytes(32)] * 3
    while len(leaves) > 1:
        leaves = [hashlib.sha256(leaves[i] + leaves[i + 1]).digest()
                  for i in range(0, len(leaves), 2)]
    return "0x" + leaves[0].hex()


def check_samples(state, samples, complete=False, now=None):
    require(state.get("status") != "failed", "operating controller failed")
    profile = state["profile"]
    require(profile["duration_seconds"] == DURATION and profile["sampling_seconds"] == 30,
            "duration or sampling profile changed")
    require(profile["source"] == SOURCE and profile["binary_sha256"] == EXECUTABLES["el"]
            and profile["cl_binary_sha256"] == EXECUTABLES["cl"], "runtime source changed")
    require(profile["memory_each_bytes"] == MEMORY and profile["cpus_each"] == 2,
            "resource profile changed")
    require(type(state["samples"]) is int and state["samples"] == len(samples) and samples,
            "sample count does not describe the complete prefix")
    initial_sync = state["initial"]["sync"]
    require(state["initial"]["ready"] is True and not any(initial_sync[key] for key in
            ("is_syncing", "is_optimistic", "el_offline")), "unverified initial execution state")
    start = timestamp(state["accepted_started_at"])
    previous_elapsed, previous_wall = 0.0, start
    previous_height = state["initial"]["finalized_number"]
    advanced_at = 0.0
    max_gap = 0.0
    for row in samples:
        elapsed, wall = row["elapsed_seconds"], timestamp(row["at"])
        require(isinstance(elapsed, (int, float)) and math.isfinite(elapsed), "non-finite elapsed time")
        gap = elapsed - previous_elapsed
        require(0 < gap <= MAX_SAMPLE_GAP, "missing or excessively delayed sample")
        require(0 < wall - previous_wall <= MAX_SAMPLE_GAP, "wall-clock sampling gap")
        max_gap = max(max_gap, gap, wall - previous_wall)
        sync = row["sync"]
        require(row["ready"] is True and not any(sync[k] for k in
                ("is_syncing", "is_optimistic", "el_offline")), "unverified execution sample")
        require(0 <= row["reference_lag"] <= 128, "independent finalized execution lag")
        root_bytes(row["finalized_hash"])
        root_bytes(row["finalized_root"])
        height = row["finalized_number"]
        require(height >= previous_height, "finalized execution regressed")
        if height > previous_height:
            previous_height, advanced_at = height, elapsed
        require(elapsed - advanced_at < 600, "finalized execution stalled")
        require(set(row["memory"]) == {"el", "cl"}, "missing daemon memory evidence")
        for memory in row["memory"].values():
            require(memory["memory_max"] == MEMORY and 0 < memory["process_hwm"] < MEMORY,
                    "memory limit or process high-water mark failed")
            require(memory["events"]["oom"] == memory["events"]["oom_kill"] == 0, "OOM observed")
        previous_elapsed, previous_wall = elapsed, wall
    require(state["elapsed_seconds"] == samples[-1]["elapsed_seconds"], "elapsed result disagrees")
    require(state["last"] == {key: value for key, value in samples[-1].items()
                             if key not in ("at", "elapsed_seconds")},
            "result does not name the final sample")
    if complete:
        require(state.get("passed") is True and previous_elapsed >= DURATION,
                "actual 48-hour operating duration is incomplete")
        require(timestamp(state["finished_at"]) - start >= DURATION,
                "less than 48 real hours between acceptance and finish")
        require(0 <= timestamp(state["finished_at"]) - previous_wall <= MAX_SAMPLE_GAP,
                "unobserved terminal interval")
    elif now is not None:
        require(-1 <= now - previous_wall <= MAX_SAMPLE_GAP, "operating evidence is stale")
    return {"samples": len(samples), "max_sample_gap_seconds": max_gap,
            "observed_seconds": previous_elapsed, "finalized_progress":
            samples[-1]["finalized_number"] - state["initial"]["finalized_number"]}


def check_wire(records, state, samples, geth_url):
    """Every metadata sample must have its own actual CL/EL/Geth exchange group."""
    groups = []
    for record in records:
        if record["method"] is None and urlsplit(record["url"]).path == "/eth/v1/node/syncing":
            groups.append([])
        if groups:
            groups[-1].append(record)
    complete_groups = []
    for group in groups:
        sync = group[0]["response"]["data"]
        if any(sync[key] for key in ("is_syncing", "is_optimistic", "el_offline")):
            continue
        native = [row for row in group if row["method"] == "eth_getBlockByNumber"
                  and row["url"] != geth_url and row["params"] == ["finalized", False]]
        if not native:  # An in-flight or failed catchup group is not accepted evidence.
            continue
        header = native[0]["response"].get("result")
        if not header:
            continue
        canonical = [row for row in group if row["url"] == geth_url
                     and row["method"] == "eth_getBlockByNumber"
                     and row["params"] == [header["number"], False]]
        tips = [row for row in group if row["url"] == geth_url
                and row["method"] == "eth_getBlockByNumber" and row["params"] == ["finalized", False]]
        if not canonical or not tips:
            continue
        require(len(native) == len(canonical) == len(tips) == 1, "ambiguous comparison exchange group")
        reference = canonical[0]["response"].get("result")
        require(reference and reference["hash"] == header["hash"]
                and reference["stateRoot"] == header["stateRoot"], "actual EL/Geth commitment disagreement")
        lag = int(tips[0]["response"]["result"]["number"], 16) - int(header["number"], 16)
        require(0 <= lag <= 128, "actual Geth finalized lag exceeded")
        complete_groups.append({"begin": timestamp(group[0]["at"]), "end": timestamp(group[-1]["at"]),
                                "sync": sync, "header": header, "lag": lag})
    start = timestamp(state["accepted_started_at"])
    initial = [group for group in complete_groups if group["end"] <= start]
    require(initial and start - initial[-1]["end"] < MAX_SAMPLE_GAP, "initial wire exchanges are absent")
    selected = [initial[-1]]
    cursor = complete_groups.index(initial[-1]) + 1
    previous = start
    for sample in samples:
        end = timestamp(sample["at"])
        candidates = []
        while cursor < len(complete_groups) and complete_groups[cursor]["begin"] <= end:
            group = complete_groups[cursor]
            require(previous < group["begin"] <= group["end"] <= end,
                    "wire exchange lies outside its observation interval")
            candidates.append(group)
            cursor += 1
        require(len(candidates) == 1, "sample lacks its unique actual wire exchanges")
        selected.append(candidates[0])
        previous = end
    for sample, group in zip([state["initial"], *samples], selected):
        header = group["header"]
        require(sample["finalized_hash"] == header["hash"]
                and sample["finalized_root"] == header["stateRoot"]
                and sample["finalized_number"] == int(header["number"], 16)
                and sample["sync"] == group["sync"] and sample["reference_lag"] == group["lag"],
                "metadata sample differs from actual CL/EL/Geth bytes")
    if state.get("passed") is True:
        end = timestamp(state["finished_at"])
        closing = [group for group in complete_groups[cursor:]
                   if previous < group["begin"] <= group["end"] <= end]
        require(len(closing) == 1, "missing final post-duration proof exchange group")
        selected.append(closing[0])
    return selected


def linked_window(trace, report, previous_anchor=None):
    """Bind each freshly replayed BLS anchor to its complete captured parent path."""
    window = report["beacon_window"]
    first, stop = window["first"], window["stop"]
    require(stop > first and window["slots"] == stop - first, "invalid finality interval")
    base = urlsplit(trace[0]["url"])
    origin = (base.scheme, base.netloc)
    anchor = window["finalized_root"]
    confirmations, parents = [], []
    for record in trace:
        if "original_trusted_bootstrap" in record:
            continue
        url = urlsplit(record["url"])
        if not url.path.startswith("/eth/v1/beacon/headers/"):
            continue
        require(record["http_status"] == 200, "failed ancestor HTTP capture")
        data = json.loads(record["response"])["data"]
        message = data["header"]["message"]
        root = beacon_header_root(message)
        require(data["root"] == root == url.path.rsplit("/", 1)[1] and data["canonical"] is True,
                "ancestor root or canonical mapping mismatch")
        if (url.scheme, url.netloc) == origin:
            parents.append((root, message))
        else:
            require(root == anchor, "external canonical confirmation names another root")
            confirmations.append(((url.scheme, url.netloc), message))
    require(len(confirmations) == 2 and len({x[0] for x in confirmations}) == 2,
            "missing independent canonical anchor confirmations")
    current = confirmations[0][1]
    require(confirmations[1][1] == current and int(current["slot"]) == window["finalized_slot"],
            "finalized anchor header mismatch")
    visited = {anchor}
    present = {int(current["slot"])}
    for root, parent in parents:
        require(root == current["parent_root"] and int(parent["slot"]) < int(current["slot"]),
                "disconnected or non-descending parent path")
        visited.add(root)
        present.add(int(parent["slot"]))
        current = parent
    require(len(parents) == window["parent_hash_links"] < 4096
            and int(current["slot"]) < first, "incomplete or over-bound finality ancestry")
    require(first in present and stop - 1 in present, "missing finality interval endpoint")
    missing = sorted(set(range(first, stop)) - present)
    require(missing == window["missing_slots"], "missing-slot summary disagrees with ancestry")
    if previous_anchor is not None:
        require(previous_anchor in visited, "previous finalized anchor absent from next proof chain")
    return anchor


def reference_identity(python):
    code = """import hashlib, importlib, importlib.metadata, json, sys
from pathlib import Path
from eth_consensus_specs.utils import bls
import py_arkworks_bls12381 as backend
names = ['eth_consensus_specs.gloas.mainnet','eth_consensus_specs.fulu.mainnet','eth_consensus_specs.utils.bls']
files = {name: hashlib.sha256(Path(importlib.import_module(name).__file__).read_bytes()).hexdigest() for name in names}
native = list(Path(backend.__file__).parent.glob('*.so'))
print(json.dumps({'debug':__debug__, 'bls_active':bls.bls_active, 'files':files,
 'backend_version':importlib.metadata.version('py_arkworks_bls12381'),
 'backend_hashes':[hashlib.sha256(p.read_bytes()).hexdigest() for p in native],
 'python':sys.version}))
"""
    value = run_json([str(python), "-c", code], 30)
    require(value["debug"] and value["bls_active"] and value["files"] == REFERENCE_SHAS,
            "reference assertions/BLS disabled or generated consumer changed")
    require(value["backend_version"] == "0.5.0" and value["backend_hashes"] == [ARKWORKS_SHA],
            "BLS backend differs from the frozen independent environment")
    return value


def check_runtime(infos, baseline, complete=False):
    for role in ("el", "cl"):
        info, original = infos[role], baseline[role]
        require(info["Config"]["Labels"].get("eth2030.qualification") == LABEL,
                "runtime ownership label changed")
        require(info["Image"] == original["image"] and info["State"]["StartedAt"] == original["started"]
                and info["RestartCount"] == 0, "runtime image/start identity changed")
        require(original["binary_sha256"] == EXECUTABLES[role], "initial executable identity changed")
        require(info["HostConfig"]["Memory"] == MEMORY
                and info["HostConfig"]["NanoCpus"] == 2000000000, "actual runtime resources changed")
        require(not info["State"]["OOMKilled"], "runtime was OOM-killed")
        if complete:
            require(not info["State"]["Running"] and info["State"]["ExitCode"] == 0,
                    "runtime cleanup failed or daemon exited unsuccessfully")
        else:
            require(info["State"]["Running"], "observation daemon stopped during the accepted prefix")


def runtime_snapshot(record, baseline, complete=False):
    infos = {role: run_json(["docker", "inspect", record[role]], 15)[0] for role in ("el", "cl")}
    check_runtime(infos, baseline, complete)
    if not complete:
        for role in infos:
            actual = subprocess.run(["docker", "exec", record[role], "sha256sum", "/proc/1/exe"],
                                    capture_output=True, text=True, timeout=15)
            require(actual.returncode == 0 and actual.stdout.split()[0] == EXECUTABLES[role],
                    "actual running executable differs from the frozen source")
    # Preserve only the reviewed runtime fields, never a generic dump of image
    # environment variables or mounted secret contents in a public capture.
    return {role: {"Id": info["Id"], "Image": info["Image"], "State": info["State"],
                   "RestartCount": info["RestartCount"],
                   "Config": {key: info["Config"][key] for key in ("Labels", "Entrypoint", "Cmd")},
                   "HostConfig": {key: info["HostConfig"][key]
                                  for key in ("Memory", "NanoCpus", "LogConfig")}}
            for role, info in infos.items()}


def audit(args, cache=None):
    """Cache only results freshly verified in this process, bound to every input byte."""
    cache = {} if cache is None else cache
    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    state = read_json(args.run / "result.json")
    require("initial" in state, "operating catchup is incomplete")
    count = state["samples"]
    raw_lines = (args.run / "samples.jsonl").read_bytes().splitlines(keepends=True)
    require(len(raw_lines) >= count, "sample prefix is truncated")
    require(not args.complete or len(raw_lines) == count, "unaccounted terminal samples")
    samples = [json.loads(line) for line in raw_lines[:count]]
    operating_complete = state.get("passed") is True
    metrics = check_samples(state, samples, args.complete or operating_complete,
                            None if operating_complete else datetime.now(timezone.utc).timestamp())
    baseline = read_json(args.run / "identities.json")
    record = read_json(args.record)
    require(record["profile"] == state["profile"], "container record and measured profile disagree")
    runtime = runtime_snapshot(record, baseline, operating_complete)
    write_json(output / "runtime.json", runtime)
    write_json(output / "operating.json", state)
    (output / "samples.jsonl").write_bytes(b"".join(raw_lines[:count]))
    wire_lines = (args.run / "exchanges.jsonl").read_bytes().splitlines(keepends=True)
    cutoff = timestamp(state["finished_at"]) if operating_complete else timestamp(samples[-1]["at"])
    selected_wire = [line for line in wire_lines if line.endswith(b"\n")
                     and timestamp(json.loads(line)["at"]) <= cutoff]
    wire = check_wire([json.loads(line) for line in selected_wire], state, samples, record["geth_rpc"])
    (output / "exchanges.jsonl").write_bytes(b"".join(selected_wire))
    require(digest(Path(args.ledger).read_bytes()) == LEDGER_SHA, "independent SDK executable changed")
    identity = reference_identity(args.reference_python)
    write_json(output / "reference-identity.json", identity)
    require(digest(args.verifier.read_bytes()) == VERIFIER_SHA, "reference verifier source changed")
    bootstrap_hash = digest(args.trusted_bootstrap.read_bytes())
    proofs = 0
    regular_pairs = count // 10 + 1
    for index in range(regular_pairs + int(operating_complete)):
        name = f"proofs-{index}.jsonl"
        body = (args.run / name).read_bytes()
        rows = [json.loads(line) for line in body.splitlines()]
        require(len(rows) == 2, "missing complete independent proof pair")
        sample = state["initial"] if index == 0 else samples[index * 10 - 1] if index < regular_pairs else {
            "finalized_hash": wire[-1]["header"]["hash"], "finalized_root": wire[-1]["header"]["stateRoot"],
        }
        for row, address in zip(rows, ADDRESSES):
            params, header = row["request"]["params"], row["header"]
            require(row["expected_hash"] == header["hash"] == sample["finalized_hash"]
                    and header["stateRoot"] == sample["finalized_root"], "proof detached from sampled commitment")
            require(params == [address, [], {"blockHash": header["hash"], "requireCanonical": True}],
                    "proof request no longer binds the declared account/hash")
        captured = output / name
        captured.write_bytes(body)
        key = ("proof", digest(body), LEDGER_SHA)
        if key not in cache:
            cache[key] = run_json([str(args.ledger), "proof-stream", str(captured)], 30)
        require(cache[key].get("passed") is True and cache[key].get("independent_proofs") == 2,
                "independent SDK did not verify the complete proof pair")
        proofs += 2
    finality = read_json(args.finality / "result.json")
    require(finality.get("status") != "failed", "signed finality collector failed")
    expected = int(state["initial"]["sync"]["head_slot"])
    previous_anchor, windows = None, []
    for index, item in enumerate(finality["windows"]):
        require(item["first"] == expected and item["through"] >= expected,
                "gap or overlap between BLS windows")
        directory = args.finality / f"window-{index}"
        live = read_json(directory / "finality-result.json")
        body = (directory / "reference-finality-exchanges.jsonl").read_bytes()
        key = ("bls", digest(body), VERIFIER_SHA, bootstrap_hash)
        if key not in cache:
            destination = output / f"replay-{index}-{key[1]}.json"
            require(not destination.exists(), "fresh replay output already exists")
            subprocess_result = subprocess.run([
                str(args.reference_python), str(args.verifier), "--capture", str(directory),
                "--trusted-bootstrap", str(args.trusted_bootstrap), "--replay", "--output", str(destination),
            ], capture_output=True, text=True, timeout=180)
            (output / f"replay-{index}.log").write_text(subprocess_result.stdout + subprocess_result.stderr)
            require(subprocess_result.returncode == 0, "unchanged BLS replay rejected captured window")
            cache[key] = read_json(destination)
        replay = cache[key]
        for result in (live, replay):
            require(result.get("passed") is True and result.get("bls_active") is True
                    and result.get("spec") == SPEC, "missing independent BLS verification")
            require(set(result["rejected_controls"]) == {"signature", "finality_branch", "genesis_domain"},
                    "BLS negative controls are incomplete")
        require(replay["replay_input_sha256"] == key[1]
                and replay["beacon_window"] == live["beacon_window"], "replay/live bytes disagree")
        window = replay["beacon_window"]
        require((window["first"], window["stop"]) == (item["first"], item["through"] + 1),
                "collector summary differs from verified interval")
        previous_anchor = linked_window([json.loads(line) for line in body.splitlines()], replay, previous_anchor)
        frozen = output / "finality" / f"window-{index}"
        frozen.mkdir(parents=True, exist_ok=True)
        (frozen / "reference-finality-exchanges.jsonl").write_bytes(body)
        (frozen / "profile.json").write_bytes((directory / "profile.json").read_bytes())
        write_json(frozen / "live.json", live)
        write_json(frozen / "replayed.json", replay)
        windows.append({"first": expected, "through": item["through"], "anchor": previous_anchor,
                        "participants": window["signature_participants"], "trace_sha256": key[1]})
        expected = item["through"] + 1
    require(windows and finality["first_slot"] == windows[0]["first"]
            and finality["covered_through"] == expected - 1, "finality coverage summary disagrees")
    if args.complete:
        require(finality.get("passed") is True
                and expected - 1 >= int(state["last"]["sync"]["head_slot"]),
                "full observed slot span lacks contiguous signed finality")
    result = {"passed": bool(args.complete), "prefix_passed": True,
              "status": "evidence_complete_cleanup_required" if args.complete else "valid_prefix_duration_pending",
              "checked_at": utc(), "actual_48h_evidence": bool(args.complete),
              "independent_proofs": proofs, "operating": metrics, "signed_windows": windows,
              "cross_window_anchors_verified": max(0, len(windows) - 1),
              "covered_through": expected - 1,
              "unsigned_observed_slots": max(0, int(state["last"]["sync"]["head_slot"]) - expected + 1),
              "cleanup_verified": operating_complete}
    if args.complete:
        result["status"] = "complete"
    write_json(output / "result.json", result)
    return result


def arguments():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("run", "finality", "record", "output", "ledger", "reference-python", "verifier", "trusted-bootstrap"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--complete", action="store_true", help="Require the full real duration and signed span")
    return parser


if __name__ == "__main__":
    cli = arguments().parse_args()
    try:
        print(json.dumps(audit(cli)))
    except Exception as error:
        write_json(cli.output / "result.json", {"passed": False, "prefix_passed": False,
                   "status": "failed", "failure": {"class": type(error).__name__, "message": str(error)}})
        raise
