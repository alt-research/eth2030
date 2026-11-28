#!/usr/bin/env python3
"""Run current native ELs with an independently pinned Lighthouse Heze network.

The proxy can withhold a builder view, but cannot manufacture a validity or
satisfaction response. PQ measurements concern canonical bytes carried in BLS
signed ILs; they do not assign a transport for omitted dependency witnesses.
"""
import argparse
import importlib.util
import json
import math
import os
from pathlib import Path
import signal
import subprocess
import threading
import time
from urllib.request import urlopen
from uuid import uuid4

ROOT = Path(__file__).resolve().parents[2]
SCRIPTS = ROOT / "pkg/devnet/kurtosis/scripts"
CL_REVISION = "bc56927fdb0c31f908286ec39013e4452864dc18"


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


base = load("hegota_e_control_base", SCRIPTS / "run-track4-control.py")
focil = load("hegota_e_focil_control", SCRIPTS / "verify-heze-focil.py")


def containers(enclave):
    rows = base.run("docker", "ps", "--filter", "network=kt-" + enclave,
                    "--format", "{{.ID}} {{.Names}}").splitlines()
    values = {row.split()[1].split("--", 1)[0]: row.split()[0] for row in rows}
    return [values[name] for name in base.EL]


def private_profile():
    # ethereum-package requires unauthenticated admin_nodeInfo for its owned
    # bridge-network health checks. Make that private-only compatibility choice
    # explicit instead of weakening the native node's default guard or JWT.
    profile = (SCRIPTS.parent / "configs/frame-heze-control.yaml").read_text()
    marker = '"--rpc.rate_limit=0"'
    if profile.count(marker) != 2:
        raise ValueError("unexpected private-profile layout")
    return profile.replace(marker, marker + ', "--rpc.allow-insecure"')


def configure(ids, value):
    program = "import sys; from pathlib import Path; p=Path('/tmp/hegota-e-control.new'); p.write_bytes(sys.stdin.buffer.read()); p.replace('/tmp/hegota-e-control.json')"
    for identifier in ids:
        subprocess.run(["docker", "exec", "-i", identifier, "python3", "-c", program],
                       input=json.dumps(value).encode(), check=True, timeout=10)


def journals(ids):
    return [[json.loads(line) for line in base.run("docker", "exec", identifier,
             "cat", "/tmp/hegota-e-engine.jsonl").splitlines()] for identifier in ids]


def retain_service_logs(enclave, services, output):
    """Keep actual peer diagnostics before cleanup, without masking acceptance."""
    errors = {}
    for name in sorted(services):
        try:
            text = base.run("kurtosis", "service", "logs", "--all", enclave, name,
                            timeout=20, stderr=subprocess.STDOUT)
            (output / (name + ".log")).write_text(text + "\n")
        except (OSError, subprocess.SubprocessError) as error:
            errors[name] = type(error).__name__ + ": " + str(error)
    return errors


def nodes(value):
    if isinstance(value, dict):
        if "root" in value and "execution_status" in value:
            yield value
        for child in value.values():
            yield from nodes(child)
    elif isinstance(value, list):
        for child in value:
            yield from nodes(child)


def valid_execution(value):
    return isinstance(value, dict) and len(value) == 1 and next(iter(value)).lower() == "valid"


def valid_root(snapshot, execution_hash, satisfied):
    view = snapshot["data"]
    unsatisfied = set(view["unsatisfied_inclusion_lists"])
    for node in nodes(view["nodes"]):
        status = node["execution_status"]
        if valid_execution(status) and next(iter(status.values())) == execution_hash and (node["root"] not in unsatisfied) == satisfied:
            return node["root"]
    return None


def unsatisfied_valid_root(snapshot, execution_hash):
    return valid_root(snapshot, execution_hash, False)


def native_verdicts(events, raw, satisfied, execution_hash=None):
    return [event for event in events if event.get("method") == "engine_newPayloadV6"
            and raw in event.get("inclusion_list", [])
            and isinstance(event.get("result"), dict)
            and event["result"].get("status") == "VALID"
            and (execution_hash is None or event.get("block_hash") == execution_hash)
            and event["result"].get("inclusionListSatisfied") is satisfied]


def native_omissions(events, raw, execution_hash=None):
    return native_verdicts(events, raw, False, execution_hash)


def matching_satisfied(enclave, ids, raw, predicate=None):
    cls = [base.endpoint(enclave, service, "http") for service in base.CL]
    records = journals(ids)
    for event in native_verdicts(records[0], raw, True):
        execution_hash = event["block_hash"]
        matches = [native_verdicts(events, raw, True, execution_hash) for events in records]
        if not all(matches):
            continue
        if predicate is not None and not predicate(execution_hash):
            continue
        snapshots = [focil.fetch(cl + "/lighthouse/proto_array") for cl in cls]
        roots = [valid_root(snapshot, execution_hash, True) for snapshot in snapshots]
        if all(roots) and roots[0] == roots[1]:
            return {"execution_hash": execution_hash, "beacon_root": roots[0],
                    "native_responses": matches, "cl_snapshots": snapshots}
    return None


def invalid_exemptions(enclave, fixture, output):
    ids = containers(enclave)
    el = base.endpoint(enclave, base.EL[0], "rpc")
    sender = json.loads(base.run(str(fixture)))["sender"]
    nonce = int(focil.rpc(el, "eth_getTransactionCount", sender, "latest"), 16)
    future = json.loads(base.run(str(fixture), "--nonce", str(nonce + 100)))
    results = {}
    for name, raw in (("malformed", "0xff"), ("nonce", future["raw"])):
        configure(ids, {"mode": "inject", "transactions": [raw]})
        try:
            result = base.wait_for(lambda: matching_satisfied(enclave, ids, raw), 90,
                                   "invalid IL was not tolerated independently by both CL/EL pairs")
        finally:
            configure(ids, {"mode": "off"})
        result["inclusion_transaction"] = raw
        results[name] = result
    (output / "invalid-exemptions.json").write_text(json.dumps(results, indent=2) + "\n")
    return {"malformed": True, "nonce": True, "both_execution_valid": True,
            "both_inclusion_satisfied": True}


def full_exemption(enclave, fixture, output):
    ids = containers(enclave)
    els = [base.endpoint(enclave, service, "rpc") for service in base.EL]
    cls = [base.endpoint(enclave, service, "http") for service in base.CL]
    spec = focil.fetch(cls[0] + "/eth/v1/config/spec")["data"]
    duration, epoch_length = int(spec["SECONDS_PER_SLOT"]), int(spec["SLOTS_PER_EPOCH"])
    genesis = int(focil.fetch(cls[0] + "/eth/v1/beacon/genesis")["data"]["genesis_time"])
    accounts = [json.loads(base.run(str(fixture), "--account", str(i)))["sender"] for i in (1, 2)]
    deadline = time.time() + 240
    selected = None
    while time.time() < deadline:
        slot = int((time.time() - genesis) // duration)
        if slot % epoch_length == epoch_length-1 or time.time() - genesis - slot * duration > duration / 5:
            time.sleep(.05)
            continue
        epoch = slot // epoch_length
        duties = focil.fetch(cls[0] + f"/eth/v1/validator/duties/proposer/{epoch}")["data"]
        proposers = {int(d["slot"]): int(d["validator_index"]) // 128 for d in duties}
        target, origin = proposers[slot+1], 1-proposers[slot+1]
        committee = focil.fetch(cls[origin] + f"/eth/v1/validator/duties/inclusion_list/{epoch}",
                                 [str(i) for i in range(256)])["data"]
        if proposers[slot] != target or not any(int(d["slot"]) == slot and int(d["validator_index"]) // 128 == origin for d in committee):
            time.sleep(duration / 2)
            continue
        head = focil.fetch(cls[target] + "/eth/v2/beacon/blocks/head")["data"]["message"]
        header = focil.rpc(els[target], "eth_getBlockByNumber", "latest", False)
        if int(head["slot"]) != slot or head["body"]["signed_execution_payload_bid"]["message"]["block_hash"] != header["hash"]:
            continue
        gas_limit = int(header["gasLimit"], 16) - 32768
        if gas_limit < 250000:
            raise ValueError("full-capacity fixture requires the pinned large-gas private profile")
        nonce = int(focil.rpc(els[origin], "eth_getTransactionCount", accounts[0], "latest"), 16)
        filler_nonce = int(focil.rpc(els[target], "eth_getTransactionCount", accounts[1], "latest"), 16)
        candidate = json.loads(base.run(str(fixture), "--nonce", str(nonce), "--gas-limit", str(gas_limit)))
        filler = json.loads(base.run(str(fixture), "--account", "2", "--nonce", str(filler_nonce)))
        if any(int(focil.rpc(el, "net_peerCount"), 16) != 0 for el in els):
            raise ValueError("full-capacity control requires isolated EL pools")
        configure([ids[origin]], {"mode": "omit", "transaction": candidate["raw"], "transactions": [candidate["raw"]]})
        configure([ids[target]], {"mode": "omit", "transaction": candidate["raw"]})
        if focil.rpc(els[target], "eth_getTransactionByHash", candidate["transaction"]) is not None:
            raise ValueError("full-capacity candidate already in proposing pool")
        if focil.rpc(els[target], "eth_sendRawTransaction", filler["raw"]) != filler["transaction"]:
            raise ValueError("full-capacity filler hash changed")
        if focil.rpc(els[origin], "eth_sendRawTransaction", candidate["raw"]) != candidate["transaction"]:
            raise ValueError("full-capacity candidate hash changed")
        selected = {"slot": slot, "origin": origin, "target": target,
                    "gas_limit": gas_limit, "candidate": candidate, "filler": filler}
        break
    if selected is None:
        raise TimeoutError("no full-capacity includer/proposer opportunity")

    def capacity(execution_hash):
        blocks = [focil.rpc(el, "eth_getBlockByHash", execution_hash, False) for el in els]
        return all(block and selected["filler"]["transaction"] in block["transactions"]
                   and selected["candidate"]["transaction"] not in block["transactions"]
                   and selected["gas_limit"] > int(block["gasLimit"], 16) - int(block["gasUsed"], 16)
                   for block in blocks)
    try:
        result = base.wait_for(lambda: matching_satisfied(enclave, ids, selected["candidate"]["raw"], capacity),
                               90, "no actual full-capacity omission accepted by both CL/EL pairs")
    finally:
        configure(ids, {"mode": "off"})
    result["fixture"] = selected
    result["blocks"] = [focil.rpc(el, "eth_getBlockByHash", result["execution_hash"], False) for el in els]
    (output / "full-exemption.json").write_text(json.dumps(result, indent=2) + "\n")
    return {"both_execution_valid": True, "both_inclusion_satisfied": True,
            "actual_declared_gas_does_not_fit": True, "candidate_omitted": True,
            "filler_included": True, "execution_hash": result["execution_hash"]}


def omission(enclave, fixture, output):
    ids = containers(enclave)
    els = [base.endpoint(enclave, service, "rpc") for service in base.EL]
    cls = [base.endpoint(enclave, service, "http") for service in base.CL]
    spec = focil.fetch(cls[0] + "/eth/v1/config/spec")["data"]
    duration, epoch_length = int(spec["SECONDS_PER_SLOT"]), int(spec["SLOTS_PER_EPOCH"])
    genesis = int(focil.fetch(cls[0] + "/eth/v1/beacon/genesis")["data"]["genesis_time"])
    sender = json.loads(base.run(str(fixture)))["sender"]
    deadline = time.time() + 240
    selected = None
    while time.time() < deadline:
        slot = int((time.time() - genesis) // duration)
        epoch = slot // epoch_length
        if slot % epoch_length == epoch_length - 1 or time.time() - genesis - slot * duration > duration / 5:
            time.sleep(0.05)
            continue
        duties = focil.fetch(cls[0] + f"/eth/v1/validator/duties/proposer/{epoch}")["data"]
        proposers = {int(d["slot"]): int(d["validator_index"]) // 128 for d in duties}
        target = proposers[slot+1]
        origin = 1 - target
        committee = focil.fetch(cls[origin] + f"/eth/v1/validator/duties/inclusion_list/{epoch}",
                                 [str(i) for i in range(256)])["data"]
        if proposers[slot] != target or not any(int(d["slot"]) == slot and int(d["validator_index"]) // 128 == origin for d in committee):
            time.sleep(duration / 2)
            continue
        if any(int(focil.rpc(el, "net_peerCount"), 16) != 0 for el in els):
            raise ValueError("omission control requires isolated EL pools")
        nonce = int(focil.rpc(els[origin], "eth_getTransactionCount", sender, "latest"), 16)
        signed = json.loads(base.run(str(fixture), "--nonce", str(nonce), "--value", str(slot)))
        raw = signed["raw"]
        configure(ids, {"mode": "omit", "transaction": raw})
        if focil.rpc(els[target], "eth_getTransactionByHash", signed["transaction"]) is not None:
            raise ValueError("omission fixture already in proposing pool")
        if focil.rpc(els[origin], "eth_sendRawTransaction", raw) != signed["transaction"]:
            raise ValueError("unexpected fixture transaction hash")
        selected = {"slot": slot, "origin": origin, "target": target, **signed}
        break
    if selected is None:
        raise TimeoutError("no isolated includer/proposer opportunity")

    def observed():
        records = journals(ids)
        candidates = native_omissions(records[selected["target"]], selected["raw"])
        for event in candidates:
            execution_hash = event["block_hash"]
            snapshots = [focil.fetch(cl + "/lighthouse/proto_array") for cl in cls]
            roots = [unsatisfied_valid_root(snapshot, execution_hash) for snapshot in snapshots]
            responses = [native_omissions(events, selected["raw"], execution_hash) for events in records]
            if all(roots) and roots[0] == roots[1] and all(responses):
                return {"execution_hash": execution_hash, "beacon_root": roots[0],
                        "native_responses": responses,
                        "cl_snapshots": snapshots}
        return None

    result = base.wait_for(observed, 90, "CL did not retain an actual VALID/false omission")
    if not all(result["native_responses"]):
        raise ValueError("both native nodes must independently return VALID/false")
    # A later beacon block may use the EMPTY parent. It must not build on the
    # omitted execution payload as a FULL parent.
    def not_extended():
        blocks = [focil.fetch(cl + "/eth/v2/beacon/blocks/head") for cl in cls]
        if any(int(block["data"]["message"]["slot"]) <= selected["slot"] + 1 for block in blocks):
            return None
        bids = [block["data"]["message"]["body"]["signed_execution_payload_bid"]["message"] for block in blocks]
        if any(bid["parent_block_hash"] == result["execution_hash"] for bid in bids):
            raise ValueError("CL extended the unsatisfied execution payload")
        return blocks
    result["later_blocks"] = base.wait_for(not_extended, 30, "no later CL fork-choice observation")
    configure(ids, {"mode": "off"})
    base.run("kurtosis", "service", "stop", enclave, base.CL[0], timeout=75)
    base.run("kurtosis", "service", "start", enclave, base.CL[0], timeout=75)
    cls[0] = base.endpoint(enclave, base.CL[0], "http")
    result["after_cl_restart"] = base.wait_for(
        lambda: unsatisfied_valid_root(focil.fetch(cls[0] + "/lighthouse/proto_array"), result["execution_hash"]),
        90, "CL lost its valid-but-unsatisfied decision after restart")
    result["fixture"] = selected
    (output / "omission.json").write_text(json.dumps(result, indent=2) + "\n")
    return {"execution_hash": result["execution_hash"], "beacon_root": result["beacon_root"],
            "both_execution_valid": True, "both_inclusion_unsatisfied": True,
            "not_extended_full": True, "cl_restart_retained": True}


def percentile(values, ratio):
    return sorted(values)[max(0, math.ceil(len(values) * ratio) - 1)]


def network_measurements(observations, engine_events, transactions, genesis, duration):
    expected = [tx.lower() for tx in transactions]
    first = {}
    for origin, events in enumerate(engine_events):
        for event in events:
            if event.get("test_il_injected") and event.get("transactions") == transactions:
                slot = int((event["unix_ns"] / 1e9 - genesis) // duration)
                key = (origin, slot)
                first[key] = min(first.get(key, event["end_ns"]), event["end_ns"])
    received = {}
    for peer, at, signed in observations:
        message = signed["message"]
        if [tx.lower() for tx in message["transactions"]] != expected:
            continue
        key = (int(message["slot"]), message["dependent_root"], int(message["validator_index"]))
        peers = received.setdefault(key, {})
        if peer in peers and peers[peer][1] != signed:
            raise ValueError("conflicting signed IL in transport fixture")
        if peer not in peers or at < peers[peer][0]:
            peers[peer] = (at, signed)
    latencies = []
    directions = [0, 0]
    for key, peers in received.items():
        origin = key[2] // 128
        if origin not in (0, 1) or origin not in peers or (origin, key[0]) not in first:
            continue
        if 1-origin not in peers:
            raise ValueError("an authenticated source IL did not reach its peer")
        if peers[origin][1] != peers[1-origin][1]:
            raise ValueError("peer IL bytes/signature differ")
        elapsed = (peers[1-origin][0] - first[origin, key[0]]) / 1e6
        if elapsed < 0:
            raise ValueError("measurement clocks do not share one monotonic domain")
        directions[origin] += 1
        latencies.append(elapsed)
    if min(directions) < 8 or len(latencies) < 32:
        raise ValueError("insufficient actual bidirectional signed-IL samples")
    size = sum(len(bytes.fromhex(tx[2:])) for tx in transactions)
    if size > 8192:
        raise ValueError("measurement exceeded the normative IL bound")
    return {"samples": len(latencies), "producer_directions": directions,
            "genesis_time": genesis, "seconds_per_slot": duration,
            "transaction_occurrences": len(transactions), "transaction_bytes": size,
            "signed_ssz_bytes": 152 + 4 * len(transactions) + size,
            "latency_ms": {"p50": percentile(latencies, .5), "p95": percentile(latencies, .95),
                           "p99": percentile(latencies, .99), "max": max(latencies)},
            "scope": "Engine callback availability through signing and actual peer CL gossip validation/SSE on one Docker host; canonical PQ transaction bytes only, not witness transport or full PQ validity",
            "public_network_tail_qualified": False}


def resources(enclave, fixture, output):
    ids = containers(enclave)
    cls = [base.endpoint(enclave, service, "http") for service in base.CL]
    spec = focil.fetch(cls[0] + "/eth/v1/config/spec")["data"]
    genesis = int(focil.fetch(cls[0] + "/eth/v1/beacon/genesis")["data"]["genesis_time"])
    duration = int(spec["SECONDS_PER_SLOT"])
    sample = json.loads(base.run(str(fixture), "--kind", "pq-transport"))
    raw = sample["raw"]
    count = 8192 // len(bytes.fromhex(raw[2:]))
    transactions = [raw] * count
    observations, errors = [], []
    stop = threading.Event()
    def listen(peer, endpoint):
        try:
            with urlopen(endpoint + "/eth/v1/events?topics=inclusion_list", timeout=15) as stream:
                for line in stream:
                    if stop.is_set():
                        return
                    if line.startswith(b"data:"):
                        observations.append((peer, time.monotonic_ns(), json.loads(line[5:])["data"]))
        except Exception as error:
            if not stop.is_set():
                errors.append(str(error))
    for i, endpoint in enumerate(cls):
        threading.Thread(target=listen, args=(i, endpoint), daemon=True).start()
    time.sleep(1)
    configure(ids, {"mode": "inject", "transactions": transactions})
    try:
        # Ignore the two partial boundary slots and measure six complete slots.
        deadline = time.monotonic() + duration * 8
        while time.monotonic() < deadline:
            if errors:
                raise ValueError("CL event stream failed: " + str(errors))
            time.sleep(.2)
    finally:
        configure(ids, {"mode": "off"})
        time.sleep(duration)
        stop.set()
    events = journals(ids)
    result = network_measurements(observations, events, transactions, genesis, duration)
    (output / "pq-gossip-observations.json").write_text(json.dumps(observations) + "\n")
    (output / "pq-gossip-engine.json").write_text(json.dumps(events) + "\n")
    (output / "pq-gossip-measurements.json").write_text(json.dumps(result, indent=2) + "\n")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, required=True)
    parser.add_argument("--fixture-tool", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    package, fixture, output = args.package.resolve(), args.fixture_tool.resolve(), args.output.resolve()
    if base.run("git", "-C", str(package), "rev-parse", "HEAD") != base.PACKAGE_REV:
        raise ValueError("unqualified ethereum-package source")
    base.run("git", "-C", str(package), "diff", "--quiet", "HEAD")
    if base.enclave_networks():
        raise RuntimeError("another devnet enclave is active")
    output.mkdir(parents=True, exist_ok=False)
    profile = output / "private-profile.yaml"
    profile.write_text(private_profile())
    enclave = "hegota-e-" + uuid4().hex[:8]
    was_running = base.engine_running()
    signal.signal(signal.SIGTERM, base.interrupt)
    evidence = {"source": base.run("git", "-C", str(ROOT), "rev-parse", "HEAD"),
                "consensus_source": CL_REVISION, "package_source": base.PACKAGE_REV,
                "enclave": enclave, "public_complete": False}
    try:
        for name, expected in (("eth2030:frame-control", evidence["source"]), ("eth2030cl:frame-control", CL_REVISION)):
            field = "commit" if name.startswith("eth2030:") else "source_commit"
            actual = base.run("docker", "image", "inspect", name, "--format", '{{index .Config.Labels "' + field + '"}}')
            if actual != expected:
                raise ValueError("image does not match the selected source: " + name)
        with (output / "launch.log").open("w") as log:
            base.run_job(["kurtosis", "run", str(package), "--args-file", str(profile), "--enclave", enclave], log, 600,
                         env=dict(os.environ))
        evidence["enclave_id"] = base.enclave_id("kt-" + enclave)
        evidence["nodes"] = base.collect_images(enclave)
        evidence["positive"] = focil.verify(enclave, 900, evidence_dir=output, fixture_tool=fixture)
        evidence["negative"] = omission(enclave, fixture, output)
        evidence["invalid_exemptions"] = invalid_exemptions(enclave, fixture, output)
        evidence["full_exemption"] = full_exemption(enclave, fixture, output)
        evidence["pq_transport"] = resources(enclave, fixture, output)
        evidence["result"] = "pass"
    except BaseException as error:
        evidence["result"] = "fail"
        evidence["error"] = type(error).__name__ + ": " + str(error)
        raise
    finally:
        try:
            evidence["service_log_errors"] = retain_service_logs(enclave, evidence.get("nodes", {}), output)
            if "enclave_id" in evidence:
                for i, records in enumerate(journals(containers(enclave))):
                    (output / f"engine-{i}.json").write_text(json.dumps(records) + "\n")
        finally:
            try:
                base.cleanup(enclave, evidence, was_running)
            finally:
                (output / "evidence.json").write_text(json.dumps(evidence, indent=2) + "\n")
    if evidence.get("result") != "pass" or evidence.get("enclave_removed") is not True:
        raise RuntimeError("incomplete external Heze qualification")
    print(json.dumps(evidence))


if __name__ == "__main__":
    main()
