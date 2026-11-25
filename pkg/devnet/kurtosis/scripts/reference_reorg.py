#!/usr/bin/env python3
"""Verify captured original reorg signatures and branch bindings offline."""

import argparse
import base64
import io
import json
import sys
from pathlib import Path

from eth_consensus_specs.fulu import mainnet as fulu
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.utils import bls

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "qualification/reference"))
from ssz_rest import parsed


def verify(root, genesis_path):
    profile = json.loads((root / "profile.json").read_text())
    switches = json.loads((root / "same-height-reorgs.json").read_text())
    assert len(switches) == 2
    assert switches[0]["old"] == switches[1]["new"]
    assert switches[0]["new"] == switches[1]["old"]
    old, new = switches[0]["old"], switches[0]["new"]
    assert old["hash"] != new["hash"] and old["number"] == new["number"]
    assert old["parentHash"] == new["parentHash"]
    records = [
        json.loads(line)
        for line in (root / "proxy-exchanges.jsonl").read_text().splitlines()
    ]
    selected = [
        record for record in records if record["action"] == "first-envelope-to-observer"
    ]
    assert (
        len(selected) == 1
        and selected[0]["http_status"] == 200
        and selected[0]["complete"]
    )
    contents = base64.b64decode(selected[0]["request_body"], validate=True)
    # Pinned stateless Contents: three variable fields, first signed envelope.
    offsets = [int.from_bytes(contents[i : i + 4], "little") for i in (0, 4, 8)]
    assert (
        offsets[0] == 12 and offsets == sorted(offsets) and offsets[-1] <= len(contents)
    )
    wire = contents[offsets[0] : offsets[1]]
    envelope = spec.SignedExecutionPayloadEnvelope.deserialize(
        io.BytesIO(wire), len(wire)
    )
    raw = genesis_path.read_bytes()
    genesis = fulu.BeaconState.deserialize(io.BytesIO(raw), len(raw))
    gvr = spec.hash_tree_root(genesis.validators)
    assert (
        bls.bls_active
        and bytes(gvr).hex()
        == "0888cde6ee0c51578046793b377d6f6474908f7812b8db4d87a736d73e539d2c"
    )
    fork = spec.Version(bytes.fromhex("80000038"))
    builder_domain = spec.compute_domain(spec.DOMAIN_BEACON_BUILDER, fork, gvr)
    builder_pubkey = bytes.fromhex(
        profile["original_builder_status"]["builder_pubkey"][2:]
    )
    signing_root = spec.compute_signing_root(envelope.message, builder_domain)
    assert bls.Verify(builder_pubkey, signing_root, envelope.signature)
    changed = bytearray(bytes(envelope.signature))
    changed[0] ^= 1
    assert not bls.Verify(builder_pubkey, signing_root, bytes(changed))
    slot = profile["slot"]
    blocks = []
    for number in (slot, slot + 1):
        block = parsed(
            json.loads((root / f"block-{number}.json").read_text())["data"],
            spec.SignedBeaconBlock,
        )
        domain = spec.compute_domain(spec.DOMAIN_BEACON_PROPOSER, fork, gvr)
        assert bls.Verify(
            genesis.validators[block.message.proposer_index].pubkey,
            spec.compute_signing_root(block.message, domain),
            block.signature,
        )
        blocks.append(block.message)
    selected_block, child = blocks
    bid = selected_block.body.signed_execution_payload_bid.message
    assert envelope.message.beacon_block_root == spec.hash_tree_root(selected_block)
    assert envelope.message.parent_beacon_block_root == selected_block.parent_root
    assert envelope.message.builder_index == bid.builder_index == 0
    assert envelope.message.payload.block_hash == bid.block_hash
    assert "0x" + bytes(bid.block_hash).hex() == new["hash"]
    assert child.parent_root == spec.hash_tree_root(selected_block)
    child_bid = child.body.signed_execution_payload_bid.message
    assert child_bid.parent_block_hash == bid.parent_block_hash != bid.block_hash
    assert "0x" + bytes(child_bid.block_hash).hex() == old["hash"]
    recovery = json.loads((root / "canonical-recovery-result.json").read_text())
    assert (
        recovery["passed"]
        and recovery["canonical"] == old["hash"]
        and recovery["orphan"] == new["hash"]
    )
    assert recovery["independent_proofs"]["passed"]
    return {
        "passed": True,
        "spec": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
        "same_height_switches": 2,
        "order": [old["hash"], new["hash"], old["hash"]],
        "original_envelope_bls": True,
        "original_beacon_block_bls": 2,
        "empty_child_parent_binding": True,
        "changed_signature_rejected": True,
        "independent_header_and_mpt_proofs": recovery["independent_proofs"],
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--capture", type=Path, required=True)
    parser.add_argument("--genesis", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    result = verify(args.capture, args.genesis)
    with args.output.open("x") as output:
        json.dump(result, output, indent=2)
        output.write("\n")
    print(json.dumps(result))
