#!/usr/bin/env python3
"""Verify compulsory inclusion on the two-pair private Heze devnet."""

import argparse
import json
from pathlib import Path
import sys
import queue
import subprocess
import threading
import time
from urllib.request import Request, urlopen


PAIR_VALIDATORS = 128
FIXTURE_CHAIN = 3151908
FIXTURE_KEY = "0x" + "0" * 63 + "1"
RECIPIENT = "0x0000000000000000000000000000000000000801"


def command(*args):
    return subprocess.check_output(args, text=True).strip()


def endpoint(enclave, service, port):
    address = command("kurtosis", "port", "print", enclave, service, port)
    return address if address.startswith("http") else "http://" + address


def fetch(url, payload=None):
    body = None if payload is None else json.dumps(payload).encode()
    with urlopen(Request(url, data=body, headers={"Content-Type": "application/json"}), timeout=5) as response:
        return json.load(response)


def rpc(url, method, *params):
    reply = fetch(url, {"id": 1, "jsonrpc": "2.0", "method": method, "params": params})
    if "error" in reply:
        raise ValueError(reply["error"])
    return reply["result"]


def wait_for(check, deadline, interval=0.25):
    while time.time() < deadline:
        try:
            result = check()
            if result:
                return result
        except OSError:
            pass
        time.sleep(interval)
    raise TimeoutError(check.__name__)


def listen_for_lists(beacon, events):
    try:
        with urlopen(beacon + "/eth/v1/events?topics=inclusion_list", timeout=90) as stream:
            for line in stream:
                if line.startswith(b"data:"):
                    events.put(json.loads(line[5:])["data"])
    except Exception as error:
        events.put(error)


def frame_readback(execution, tx_hash):
    receipt = rpc(execution, "eth_getTransactionReceipt", tx_hash)
    if not receipt or receipt.get("type") != "0x6" or receipt.get("status") != "0x1":
        raise ValueError("missing successful canonical Frame receipt")
    frames = receipt.get("frameResults", [])
    if len(frames) != 3 or any(frame["status"] != "0x1" for frame in frames):
        raise ValueError("Frame execution did not complete all three frames")
    if int(frames[2].get("stateGasUsed", "0x0"), 16) != 97920:
        raise ValueError("Frame did not charge the expected new storage slot")
    block = rpc(execution, "eth_getBlockByNumber", receipt["blockNumber"], False)
    parent = hex(int(receipt["blockNumber"], 16) - 1)
    sender = command("cast", "wallet", "address", "--private-key", FIXTURE_KEY)
    payer = command("cast", "wallet", "address", "--private-key", "0x" + "0" * 63 + "2")
    if receipt.get("payer", "").lower() != payer.lower():
        raise ValueError("wrong Frame payer")
    balances = {}
    for name, address in (("sender", sender), ("payer", payer)):
        balances[name] = [int(rpc(execution, "eth_getBalance", address, at), 16)
                          for at in (parent, receipt["blockNumber"])]
    expected_charge = int(receipt["gasUsed"], 16) * (int(block["baseFeePerGas"], 16) + 1000000000)
    if balances["payer"][0] - balances["payer"][1] != expected_charge or balances["sender"][0] != balances["sender"][1]:
        raise ValueError("Frame sponsor charge or sender balance mismatch")
    storage = [rpc(execution, "eth_getStorageAt", "0x0000000000000000000000000000000000007100", "0x0", at)
               for at in ("0x0", receipt["blockNumber"])]
    if [int(value, 16) for value in storage] != [0, 1]:
        raise ValueError("Frame historical state mismatch")
    return {"receipt": receipt, "balances": balances, "storage": storage,
            "state_root": block["stateRoot"], "payer_charge": str(expected_charge)}


def verify(enclave, timeout, frame_tool=None, evidence_dir=None):
    el = [endpoint(enclave, f"el-{i}-geth-lighthouse", "rpc") for i in (1, 2)]
    bn = [endpoint(enclave, f"cl-{i}-lighthouse-geth", "http") for i in (1, 2)]
    deadline = time.time() + timeout
    if any(int(rpc(node, "eth_chainId"), 16) != FIXTURE_CHAIN for node in el):
        raise ValueError("expected the private Kurtosis fixture chain")

    def assert_isolated():
        counts = [int(rpc(node, "net_peerCount"), 16) for node in el]
        if counts != [0, 0]:
            raise ValueError(f"execution nodes must be isolated, got {counts}")

    def heze_head():
        block = fetch(bn[0] + "/eth/v2/beacon/blocks/head")
        return block["version"] == "heze" and not block["execution_optimistic"]

    wait_for(heze_head, deadline)
    assert_isolated()
    spec = fetch(bn[0] + "/eth/v1/config/spec")["data"]
    seconds_per_slot = int(spec["SECONDS_PER_SLOT"])
    slots_per_epoch = int(spec["SLOTS_PER_EPOCH"])
    genesis = int(fetch(bn[0] + "/eth/v1/beacon/genesis")["data"]["genesis_time"])
    sender = command("cast", "wallet", "address", "--private-key", FIXTURE_KEY)
    if any(int(rpc(node, "eth_getBalance", sender, "latest"), 16) == 0 for node in el):
        raise ValueError("the public fixture account must be prefunded on both nodes")

    events = [queue.SimpleQueue(), queue.SimpleQueue()]
    for node, received in zip(bn, events):
        threading.Thread(target=listen_for_lists, args=(node, received), daemon=True).start()

    previous_slot = 0
    epoch_with_duties = None
    while time.time() < deadline:
        now = time.time()
        slot = int((now - genesis) // seconds_per_slot)
        if slot == previous_slot:
            time.sleep(0.05)
            continue
        previous_slot = slot
        epoch = slot // slots_per_epoch
        if slot % slots_per_epoch == slots_per_epoch - 1:
            continue
        if epoch != epoch_with_duties:
            proposer_duties = fetch(bn[0] + f"/eth/v1/validator/duties/proposer/{epoch}")
            list_duties = fetch(
                bn[0] + f"/eth/v1/validator/duties/inclusion_list/{epoch}",
                [str(i) for i in range(2 * PAIR_VALIDATORS)],
            )
            proposers = {int(d["slot"]): int(d["validator_index"]) for d in proposer_duties["data"]}
            members = {(int(d["slot"]), int(d["validator_index"])) for d in list_duties["data"]}
            epoch_with_duties = epoch

        target = proposers[slot + 1] // PAIR_VALIDATORS
        origin = 1 - target
        if proposers[slot] // PAIR_VALIDATORS != target:
            continue
        if not any(s == slot and validator // PAIR_VALIDATORS == origin for s, validator in members):
            continue
        if time.time() - (genesis + slot * seconds_per_slot) > seconds_per_slot / 4:
            continue

        if frame_tool:
            signed = json.loads(command(
                str(frame_tool), "--rpc", el[origin], "--key", FIXTURE_KEY,
                "--sponsor-key", "0x" + "0" * 63 + "2", "--to",
                "0x0000000000000000000000000000000000007100", "--gas", "30000",
                "--state-gas", "97920", "--dry-run",
            ))
            raw, tx_hash = signed["raw"], signed["transaction"]
        else:
            nonce = int(rpc(el[origin], "eth_getTransactionCount", sender, "latest"), 16)
            raw = command(
                "cast", "mktx", "--private-key", FIXTURE_KEY, "--chain", str(FIXTURE_CHAIN),
                "--nonce", str(nonce), "--legacy", "--gas-limit", "21000", "--gas-price",
                "10000000000", "--value", str(slot), RECIPIENT,
            )
            tx_hash = command("cast", "keccak", raw)
        assert_isolated()
        if rpc(el[target], "eth_getTransactionByHash", tx_hash) is not None:
            raise ValueError("the proposing EL already has the fixture transaction")
        if rpc(el[origin], "eth_sendRawTransaction", raw) != tx_hash:
            raise ValueError("the committee EL returned an unexpected transaction hash")
        if rpc(el[target], "eth_getTransactionByHash", tx_hash) is not None:
            raise ValueError("the proposing EL received the fixture transaction through its mempool")

        def proposer_receipt():
            return rpc(el[target], "eth_getTransactionReceipt", tx_hash)

        try:
            receipt = wait_for(proposer_receipt, min(deadline, genesis + (slot + 2) * seconds_per_slot + 1))
            block = fetch(bn[target] + f"/eth/v2/beacon/blocks/{slot + 1}")["data"]["message"]
        except (TimeoutError, OSError):
            if frame_tool:
                raise RuntimeError("Frame missed the selected FOCIL proposal")
            continue
        bid = block["body"]["signed_execution_payload_bid"]["message"]
        if receipt["blockHash"] != bid["block_hash"]:
            if frame_tool:
                raise RuntimeError("Frame appeared outside the selected FOCIL proposal")
            continue

        def matching_list():
            while not events[target].empty():
                event = events[target].get_nowait()
                if isinstance(event, Exception):
                    raise event
                message = event["message"]
                if ((int(message["slot"]), int(message["validator_index"])) in members
                        and int(message["slot"]) == slot
                        and int(message["validator_index"]) // PAIR_VALIDATORS == origin
                        and message["dependent_root"] == list_duties["dependent_root"]
                        and raw in message["transactions"]):
                    return event
            return None

        signed_list = wait_for(matching_list, min(deadline, time.time() + 3))
        if int(block["proposer_index"]) != proposers[slot + 1] or receipt["status"] != "0x1":
            raise ValueError("unexpected proposer or unsuccessful inclusion")

        inclusion = {"list_slot": slot, "proposal_slot": slot + 1,
                     "committee_el": origin + 1, "proposer_el": target + 1,
                     "transaction": tx_hash, "raw_transaction": raw,
                     "signed_inclusion_list": signed_list, "payload_bid": bid,
                     "block_hash": receipt["blockHash"]}
        if evidence_dir is not None:
            Path(evidence_dir, "inclusion.json").write_text(json.dumps(inclusion, indent=2) + "\n")
        if frame_tool:
            print(f"Frame {tx_hash} included at slot {slot + 1}; waiting for finality", file=sys.stderr, flush=True)

        def finalized_on_both_pairs():
            assert_isolated()
            for beacon, execution in zip(bn, el):
                block = fetch(beacon + f"/eth/v2/beacon/blocks/{slot + 1}")
                other = rpc(execution, "eth_getTransactionReceipt", tx_hash)
                if not block["finalized"] or block["execution_optimistic"] or other != receipt:
                    return False
            return True

        wait_for(finalized_on_both_pairs, deadline, interval=seconds_per_slot)
        result = {
            **inclusion,
            "list_slot": slot, "proposal_slot": slot + 1, "proposer": proposers[slot + 1],
            "committee_el": origin + 1, "proposer_el": target + 1, "transaction": tx_hash,
            "block_hash": receipt["blockHash"], "finalized_on_both_pairs": True,
        }
        if frame_tool:
            readbacks = [frame_readback(node, tx_hash) for node in el]
            if readbacks[0] != readbacks[1]:
                raise ValueError("canonical Frame peers disagree on settlement or state")
            result["frame"] = readbacks[0]
        print(json.dumps(result))
        return result

    raise TimeoutError("no ordinary transaction included through FOCIL")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("enclave")
    parser.add_argument("--timeout", type=int, default=600)
    options = parser.parse_args()
    verify(options.enclave, options.timeout)
