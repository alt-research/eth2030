#!/usr/bin/env python3
"""Qualify the standard CL FCR -> Engine safeBlockHash -> native RPC consumer."""
import argparse
import collections
import json
import subprocess
import threading
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def now():
    return datetime.now(timezone.utc).isoformat()


def run(*args):
    return subprocess.check_output(args, text=True).strip()


def profile(el_image, cl_image):
    base = Path(__file__).parents[1] / "configs/gloas-genesis.yaml"
    contracts = next(line.split("additional_preloaded_contracts:", 1)[1] for line in base.read_text().splitlines() if "additional_preloaded_contracts:" in line)
    return {
        "participants": [{"el_type": "geth", "el_image": el_image, "el_max_mem": 8192, "cl_max_mem": 2048, "vc_max_mem": 1024, "el_extra_params": ["--rpc.rate_limit=0", "--http.api=admin,eth,net,web3,zkvm", "--ws.api=eth,net,web3,zkvm", "--rpc.allow-insecure"], "cl_type": "lighthouse", "cl_image": cl_image, "cl_extra_params": ["--subscribe-all-subnets", "--target-peers=1", "--disable-peer-scoring", "--enable-fast-confirmation"], "count": 2}],
        "network_params": {"additional_preloaded_contracts": json.loads(contracts), "network": "kurtosis", "seconds_per_slot": 6, "genesis_delay": 60, "electra_fork_epoch": 0, "fulu_fork_epoch": 0, "gloas_fork_epoch": 0, "num_validator_keys_per_node": 128},
        "ethereum_genesis_generator_params": {"image": "ethpandaops/ethereum-genesis-generator@sha256:e1ac93bb064fc00aacf5a6533b8a5a8d9eb1903350d24598a87d3ed186eb6b8a"},
        "additional_services": [],
    }


def endpoint(enclave, service, port):
    value = json.loads(run("kurtosis", "service", "inspect", "--output", "json", enclave, service))
    return "http://127.0.0.1:" + str(value["public_ports"][port]["number"]), value


def get(url):
    with OPENER.open(url, timeout=15) as response:
        return json.load(response)


def rpc(url, method, *params):
    request = urllib.request.Request(url, json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}).encode(), {"Content-Type": "application/json"})
    with OPENER.open(request, timeout=15) as response:
        result = json.load(response)
    if "error" in result:
        raise RuntimeError(result["error"])
    return result["result"]


def selected_container(service, public_port, internal_port):
    candidates = run("docker", "ps", "-q", "--filter", "label=kurtosis_service_name=" + service).split()
    matches = []
    for candidate in candidates:
        item = json.loads(run("docker", "inspect", candidate))[0]
        ports = item["NetworkSettings"]["Ports"].get(internal_port, []) or []
        if any(int(port["HostPort"]) == public_port for port in ports):
            matches.append(item)
    assert len(matches) == 1, (service, public_port, len(matches))
    return matches[0]


def restarted_endpoint(container_id, internal_port="4000/tcp"):
    # Docker may reassign an automatically published host port after start.
    # Kurtosis metadata still contains the original mapping.
    item = json.loads(run("docker", "inspect", container_id))[0]
    ports = item["NetworkSettings"]["Ports"][internal_port]
    local = [port for port in ports if port["HostIp"] in ("127.0.0.1", "0.0.0.0")]
    assert len(local) == 1, (container_id, ports)
    return "http://127.0.0.1:" + local[0]["HostPort"]


def observe(args):
    root = args.output
    root.mkdir(parents=True, exist_ok=False)
    cl, cl_config = endpoint(args.enclave, "cl-1-lighthouse-geth", "http")
    el, _ = endpoint(args.enclave, "el-1-geth-lighthouse", "rpc")
    follower, _ = endpoint(args.enclave, "el-2-geth-lighthouse", "rpc")
    assert "--enable-fast-confirmation" in " ".join(cl_config["cmd"])
    container = selected_container("cl-1-lighthouse-geth", int(cl.rsplit(":", 1)[1]), "4000/tcp")
    capture_profile = {"declared_at": now(), "enclave": args.enclave, "cl": cl, "el": el, "follower": follower, "duration_seconds": args.duration, "restart_at_slot": args.restart_at_slot, "proof_requested": args.proof, "cl_container": container["Id"], "cl_image": container["Image"], "scope": "Bounded 256-validator private FCR/native-node integration; not 48h or unrestricted scale acceptance."}
    (root / "profile.json").write_text(json.dumps(capture_profile, indent=2) + "\n")
    report = {"passed": False, "started": now()}
    events = collections.deque(maxlen=4096)
    event_lock = threading.Lock()
    errors = []
    stop = threading.Event()

    def stream():
        with (root / "events.jsonl").open("x") as output:
            while not stop.is_set():
                try:
                    with OPENER.open(cl + "/eth/v1/events?topics=fast_confirmation", timeout=15) as response:
                        for line in response:
                            if stop.is_set():
                                return
                            if line.startswith(b"data:"):
                                value = json.loads(line[5:])
                                record = {"observed_at": now(), "event": value}
                                output.write(json.dumps(record) + "\n"); output.flush()
                                with event_lock:
                                    events.append(value)
                except Exception as error:
                    errors.append({"at": now(), "error": str(error)})
                    stop.wait(1)

    thread = threading.Thread(target=stream, daemon=True)
    thread.start()
    records = []
    restarted = False
    crash_record = None
    precrash_confirmed_slot = None
    proof_job = None
    proof_complete = None
    start = time.monotonic()
    last_head, last_change = None, start
    genesis = get(cl + "/eth/v1/beacon/genesis")["data"]
    (root / "genesis.json").write_text(json.dumps(genesis, indent=2) + "\n")
    try:
        with (root / "observations.jsonl").open("x") as output:
            while time.monotonic() - start < args.duration:
                stamp = time.monotonic()
                try:
                    latest = rpc(el, "eth_getBlockByNumber", "latest", False)
                    safe = rpc(el, "eth_getBlockByNumber", "safe", False)
                    finalized = rpc(el, "eth_getBlockByNumber", "finalized", False)
                    other = rpc(follower, "eth_getBlockByNumber", "latest", False)
                    header = get(cl + "/eth/v1/beacon/headers/head")["data"]["header"]["message"]
                    slot = int(header["slot"])
                    record = {"at": now(), "slot": slot, "head": latest["hash"], "head_number": int(latest["number"], 16), "safe": None if safe is None else safe["hash"], "safe_number": None if safe is None else int(safe["number"], 16), "finalized": None if finalized is None else finalized["hash"], "finalized_number": None if finalized is None else int(finalized["number"], 16), "follower_head": other["hash"], "elapsed_seconds": time.monotonic() - stamp}
                    output.write(json.dumps(record) + "\n"); output.flush(); records.append(record)
                    if latest["hash"] != last_head:
                        last_head, last_change = latest["hash"], time.monotonic()
                    if int(latest["number"], 16) > 0 and time.monotonic() - last_change > 36:
                        raise AssertionError("native execution head stopped progressing for six slots")
                    if args.proof and proof_job is None and slot >= 48:
                        words = [0x00200893, 0x73, 0x00100893, 0x73, 0x00000513, 0x00000893, 0x73]
                        program = "0x" + b"".join(word.to_bytes(4, "little") for word in words).hex()
                        request = {"program": program, "input": "0x2a", "config": {"max_cycles": 100, "memory_pages": 4, "program_base": 0, "entry_pc": 0, "stack_base": 2147483648, "input_base": 0, "load_input": False}}
                        proof_job = rpc(el, "zkvm_proveExecution", request)
                        (root / "proof-request.json").write_text(json.dumps(request, indent=2) + "\n")
                    if proof_job is not None and proof_complete is None:
                        job = rpc(el, "zkvm_getProof", proof_job["id"])
                        if job["status"] == "complete":
                            proof_complete = job
                            (root / "actual-proof.json").write_text(json.dumps(job, indent=2) + "\n")
                            request = json.loads((root / "proof-request.json").read_text()); request["proof"] = job["proof"]
                            assert rpc(follower, "zkvm_verifyExecution", request)["valid"]
                        else:
                            assert job["status"] == "running", job
                    if args.restart_at_slot and not restarted and slot >= args.restart_at_slot:
                        crash_record = dict(record)
                        with event_lock:
                            precrash_confirmed_slot = max((int(event["slot"]) for event in events), default=0)
                        crash_record["confirmed_slot"] = precrash_confirmed_slot
                        (root / "before-crash.json").write_text(json.dumps(crash_record, indent=2) + "\n")
                        run("docker", "kill", "--signal", "KILL", container["Id"])
                        run("docker", "start", container["Id"])
                        old_cl = cl
                        cl = restarted_endpoint(container["Id"])
                        capture_profile["cl"] = cl
                        capture_profile["cl_endpoint_history"] = [old_cl, cl]
                        (root / "profile.json").write_text(json.dumps(capture_profile, indent=2) + "\n")
                        restarted = True
                except (urllib.error.URLError, TimeoutError, ConnectionError) as error:
                    errors.append({"at": now(), "observation_error": str(error)})
                stop.wait(1)
        with event_lock:
            observed_events = list(events)
        assert len(records) >= 30 and len(observed_events) >= 10, "missing actual FCR observations"
        parents = {}
        with (root / "confirmed-blocks.jsonl").open("x") as output:
            for event in observed_events:
                block_root = event["block"]
                if block_root in parents or block_root == "0x" + "00" * 32:
                    continue
                block = get(cl + "/eth/v2/beacon/blocks/" + block_root)
                bid = block["data"]["message"]["body"]["signed_execution_payload_bid"]["message"]
                parents[block_root] = bid["parent_block_hash"]
                output.write(json.dumps({"root": block_root, "block": block}) + "\n")
        matched = [record for record in records if record["safe"] in parents.values() and record["safe_number"] is not None and record["finalized_number"] is not None and record["safe_number"] > record["finalized_number"]]
        assert len(matched) >= 3, "no FCR-confirmed safe block ahead of finality"
        assert records[-1]["slot"] >= 96 and records[-1]["slot"] - records[0]["slot"] >= 96 and records[-1]["head_number"] >= 80, "window did not cover three epochs"
        assert not args.restart_at_slot or restarted, "crash/restart was not executed"
        postcrash = []
        if restarted:
            postcrash = [record for record in records if record["at"] > crash_record["at"]]
            assert len(postcrash) >= 30 and postcrash[-1]["slot"] >= crash_record["slot"] + 32, "insufficient actual post-crash observation"
            postevents = [json.loads(line)["event"] for line in (root / "events.jsonl").read_text().splitlines() if json.loads(line)["observed_at"] > crash_record["at"]]
            assert len(postevents) >= 10, "missing reconnected FCR events"
            assert min(int(event["slot"]) for event in postevents) >= precrash_confirmed_slot, "crash regressed announced FCR confidence"
        assert not args.proof or proof_complete is not None, "actual proving did not complete"
        # Poll for a common native head after the requested crash and proof work.
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            one, two = rpc(el, "eth_getBlockByNumber", "latest", False), rpc(follower, "eth_getBlockByNumber", "latest", False)
            if one["hash"] == two["hash"]:
                break
            time.sleep(1)
        assert one["hash"] == two["hash"], "native peers did not converge after recovery"
        report.update(passed=True, stopped=now(), observations=len(records), fcr_events=len(observed_events), safe_ahead_samples=len(matched), first_slot=records[0]["slot"], final_slot=records[-1]["slot"], common_head=one["hash"], crash_restart_passed=restarted, postcrash_observations=len(postcrash), precrash_confirmed_slot=precrash_confirmed_slot, actual_proof_passed=proof_complete is not None)
    finally:
        stop.set(); thread.join(timeout=16)
        report["transport_reconnects"] = errors
        (root / "result.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report, indent=2))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    prepare = sub.add_parser("prepare")
    prepare.add_argument("--el-image", required=True); prepare.add_argument("--cl-image", required=True); prepare.add_argument("--output", type=Path, required=True)
    monitor = sub.add_parser("observe")
    monitor.add_argument("--enclave", required=True); monitor.add_argument("--output", type=Path, required=True); monitor.add_argument("--duration", type=int, default=720); monitor.add_argument("--restart-at-slot", type=int, default=72); monitor.add_argument("--proof", action="store_true")
    args = parser.parse_args()
    if args.command == "prepare":
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open("x") as output:
            json.dump(profile(args.el_image, args.cl_image), output, indent=2); output.write("\n")
    else:
        observe(args)


if __name__ == "__main__":
    main()
