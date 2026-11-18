#!/usr/bin/env python3
"""Qualify coded sponsorship, blob Frames and a new CL's historical envelopes."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import time
from uuid import uuid4

import importlib.util
SCRIPTS = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("frame_control", SCRIPTS / "run-frame-control.py")
base = importlib.util.module_from_spec(spec)
spec.loader.exec_module(base)
control, verifier = base.control, base.verifier
PAYMASTER = "0x0000000000000000000000000000000000007300"
BLOB_TARGET = "0x0000000000000000000000000000000000007101"
KEY1, KEY2 = "0x" + "0" * 63 + "1", "0x" + "0" * 63 + "2"


def payload(record):
    return record["data"]["message"]["payload"]


def check_historical_envelope(record, block_hash, tx_hash, expected_commitments, beacon_root):
    if record.get("execution_optimistic") is not False or record.get("finalized") is not True:
        raise ValueError("historical envelope is optimistic or unfinalized")
    message = payload(record)
    if record["data"]["message"]["beacon_block_root"] != beacon_root:
        raise ValueError("historical envelope is bound to a different beacon block")
    if message["block_hash"] != block_hash:
        raise ValueError("historical execution hash differs")
    transactions = message["transactions"]
    matches = [raw for raw in transactions if verifier.command("cast", "keccak", raw) == tx_hash]
    if len(matches) != 1:
        raise ValueError("historical envelope lost the original Frame bytes")
    return {"envelope": record, "raw_transaction": matches[0], "commitments": expected_commitments}


def find_proposal(beacon, block_hash):
    head = verifier.fetch(beacon + "/eth/v2/beacon/blocks/head")
    current = int(head["data"]["message"]["slot"])
    for slot in range(current, max(-1, current - 12), -1):
        try:
            block = verifier.fetch(beacon + f"/eth/v2/beacon/blocks/{slot}")
        except OSError:
            continue
        bid = block["data"]["message"]["body"].get("signed_execution_payload_bid", {}).get("message", {})
        if bid.get("block_hash") == block_hash:
            return slot, block
    return None


def verify_blob(rpcs, beacons, frame_tool, output):
    blob = output / "blob.bin"
    # Nonzero valid field elements ensure proof-index tests are not degenerate.
    data = bytearray(131072)
    data[31] = 1
    blob.write_bytes(data)
    report = json.loads(control.run(str(frame_tool), "--rpc", rpcs[0], "--key", KEY1,
        "--sponsor-key", KEY2, "--paymaster", PAYMASTER, "--to", BLOB_TARGET,
        "--gas", "30000", "--state-gas", "97920", "--blob", str(blob), timeout=150))
    receipt = report["receipt"]
    tx_hash = report["transaction"]
    (output / "blob-submission.json").write_text(json.dumps(report, indent=2) + "\n")
    if receipt.get("payer", "").lower() != PAYMASTER or int(receipt.get("blobGasUsed", "0x0"), 16) != 131072:
        raise ValueError("blob Frame payer or blob charge missing")
    if len(receipt.get("frameResults", [])) != 3 or any(frame["status"] != "0x1" for frame in receipt["frameResults"]):
        raise ValueError("blob Frame execution failed")
    if int(receipt["frameResults"][2].get("stateGasUsed", "0x0"), 16) != 97920:
        raise ValueError("blob application did not create the expected slot")
    control.wait_for(lambda: control.rpc(rpcs[1], "eth_getTransactionReceipt", tx_hash) == receipt, 90, "second EL did not import blob Frame")
    proposal = control.wait_for(lambda: find_proposal(beacons[0], receipt["blockHash"]), 30, "blob proposal missing")
    slot, block = proposal
    commitments = block["data"]["message"]["body"]["signed_execution_payload_bid"]["message"]["blob_kzg_commitments"]
    tx = control.rpc(rpcs[0], "eth_getTransactionByHash", tx_hash)
    hashes = []
    for commitment in commitments:
        value = bytearray(hashlib.sha256(bytes.fromhex(commitment[2:])).digest())
        value[0] = 1
        hashes.append("0x" + value.hex())
    if len(hashes) != 1 or tx.get("blobVersionedHashes") != hashes:
        raise ValueError("Frame versioned hashes differ from ordered CL commitments")
    execution_block = control.rpc(rpcs[0], "eth_getBlockByNumber", receipt["blockNumber"], False)
    parent = hex(int(receipt["blockNumber"], 16) - 1)
    balances = [int(control.rpc(rpcs[0], "eth_getBalance", PAYMASTER, at), 16) for at in (parent, receipt["blockNumber"])]
    charge = int(receipt["gasUsed"], 16) * (int(execution_block["baseFeePerGas"], 16) + 1000000000)
    charge += int(receipt["blobGasUsed"], 16) * int(receipt["blobGasPrice"], 16)
    if balances[0] - balances[1] != charge:
        raise ValueError("blob sponsor balance does not reconcile with both resource fees")
    result = {"transaction": tx_hash, "slot": slot, "block_hash": receipt["blockHash"],
              "receipt": receipt, "commitments": commitments, "balances": balances, "charge": str(charge)}
    (output / "blob-inclusion.json").write_text(json.dumps(result, indent=2) + "\n")
    def finalized():
        for beacon in beacons:
            checkpoint = verifier.fetch(beacon + "/eth/v2/beacon/blocks/finalized")
            if int(checkpoint["data"]["message"]["slot"]) < slot or checkpoint["execution_optimistic"]:
                return None
        return True
    control.wait_for(finalized, 600, "blob Frame did not reach common finality")
    return result


def start_observer(enclave, beacon, output):
    joins = verifier.fetch(beacon + "/eth/v2/beacon/blocks/head")
    checkpoint = verifier.fetch(beacon + "/eth/v1/beacon/headers/finalized")
    bootnodes = [verifier.fetch(control.endpoint(enclave, name, "http") + "/eth/v1/node/identity")["data"]["enr"] for name in control.CL]
    with (output / "observer-launch.log").open("w") as log:
        control.run_job(["kurtosis", "run", str(SCRIPTS / "frame-history-observer.star"), "--enclave", enclave,
                         "--args", json.dumps({"bootnodes": ",".join(bootnodes)})], log, 360)
    observer = control.endpoint(enclave, "frame-history-observer", "http")
    ids = control.run("docker", "ps", "-q", "--filter", "network=kt-" + enclave,
                      "--filter", "name=frame-history-observer--").splitlines()
    if len(ids) != 1:
        raise RuntimeError("observer resource ownership missing")
    item = json.loads(control.run("docker", "inspect", ids[0]))[0]
    memory = item["HostConfig"]["Memory"]
    if not 0 < memory <= 2048 * 1024 * 1024:
        raise RuntimeError("observer memory cap missing")
    return observer, {"join_slot": joins["data"]["message"]["slot"], "checkpoint": checkpoint,
                      "image_id": item["Image"], "memory": memory}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--frame-tool", type=Path, required=True)
    args = parser.parse_args()
    if control.run("git", "-C", str(args.package), "rev-parse", "HEAD") != control.PACKAGE_REV:
        raise ValueError("history control requires the pinned package")
    control.run("git", "-C", str(args.package), "diff", "--quiet", "HEAD")
    if control.enclave_networks():
        raise RuntimeError("finish other enclaves before running this control")
    args.output.mkdir(parents=True, exist_ok=False)
    enclave = "eth2030-paymaster-history-" + uuid4().hex[:8]
    was_running = control.engine_running()
    signal.signal(signal.SIGTERM, control.interrupt)
    evidence = {"enclave": enclave, "source": control.run("git", "-C", str(base.REPO), "rev-parse", "HEAD"), "package": control.PACKAGE_REV}
    try:
        with (args.output / "launch.log").open("w") as log:
            control.run_job(["bash", str(SCRIPTS / "run-devnet.sh"), "frame-paymaster-history", enclave], log, 420,
                env={**os.environ, "KURTOSIS_PACKAGE": str(args.package.resolve()), "SKIP_EL_IMAGE_BUILD": "1"})
        evidence["enclave_id"] = control.enclave_id("kt-" + enclave)
        evidence["nodes"] = control.collect_images(enclave)
        evidence["execution_source"] = control.run("docker", "image", "inspect", "eth2030:frame-control", "--format", '{{index .Config.Labels "commit"}}')
        evidence["consensus_source"] = control.run("docker", "image", "inspect", "eth2030cl:frame-control", "--format", '{{index .Config.Labels "source_commit"}}')
        rpcs = [control.endpoint(enclave, name, "rpc") for name in control.EL]
        beacons = [control.endpoint(enclave, name, "http") for name in control.CL]
        evidence["sponsored"] = verifier.verify(enclave, 900, args.frame_tool.resolve(), args.output, PAYMASTER)
        evidence["blob"] = verify_blob(rpcs, beacons, args.frame_tool.resolve(), args.output)
        observer, evidence["observer"] = start_observer(enclave, beacons[0], args.output)
        histories = []
        for item in (evidence["sponsored"], evidence["blob"]):
            slot = item.get("proposal_slot", item.get("slot"))
            def history():
                response = verifier.fetch(observer + f"/eth/v1/beacon/execution_payload_envelopes/{slot}")
                block = verifier.fetch(observer + f"/eth/v2/beacon/blocks/{slot}")
                bid = block["data"]["message"]["body"]["signed_execution_payload_bid"]["message"]
                if bid["blob_kzg_commitments"] != item.get("commitments", []):
                    raise ValueError("observer historical commitment ordering changed")
                original_header = verifier.fetch(beacons[0] + f"/eth/v1/beacon/headers/{slot}")
                historical_header = verifier.fetch(observer + f"/eth/v1/beacon/headers/{slot}")
                if original_header["data"]["root"] != historical_header["data"]["root"]:
                    raise ValueError("observer returned a competing historical beacon block")
                return check_historical_envelope(response, item["block_hash"], item["transaction"], bid["blob_kzg_commitments"], original_header["data"]["root"])
            histories.append(control.wait_for(history, 300, "new CL did not backfill original Frame envelope"))
        evidence["observer"]["historical_envelopes"] = histories
        control.run("kurtosis", "service", "stop", enclave, control.EL[1], timeout=75)
        control.run("kurtosis", "service", "start", enclave, control.EL[1], timeout=75)
        evidence["after_restart"] = control.wait_for(lambda: verifier.frame_readback(control.endpoint(enclave, control.EL[1], "rpc"), evidence["sponsored"]["transaction"], PAYMASTER), 120, "paymaster history missing after restart")
        if evidence["after_restart"] != evidence["sponsored"]["frame"]:
            raise RuntimeError("paymaster history changed across restart")
    finally:
        try:
            control.cleanup(enclave, evidence, was_running)
        finally:
            (args.output / "evidence.json").write_text(json.dumps(evidence, indent=2) + "\n")
    print("Verified paymaster, blob and observer history; removed", enclave, flush=True)


if __name__ == "__main__":
    main()
