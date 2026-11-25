#!/usr/bin/env python3
"""Replay actual signed registry sweep/reuse states and the retired-key process control."""

import argparse, base64, io, json, sys
from pathlib import Path
from frozendict import frozendict
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.utils import bls

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "qualification/reference"))
from ssz_rest import parsed

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--capture", type=Path, required=True)
parser.add_argument("--output", type=Path, required=True)
parser.add_argument(
    "--registry-only",
    action="store_true",
    help="Replay sweep/reuse only; do not count the separate signature guard",
)
args = parser.parse_args()
root = args.capture
assert not args.output.exists()
assert bls.bls_active
network = json.loads((root / "canonical-states/network-spec.json").read_text())
changes = {}
for name, current in spec.config._asdict().items():
    if name not in network:
        continue
    value = network[name]
    if isinstance(current, tuple):
        changes[name] = tuple(
            frozendict(
                {
                    key: int(v) if isinstance(v, str) and v.isdigit() else v
                    for key, v in item.items()
                }
            )
            for item in value
        )
    elif isinstance(current, str):
        changes[name] = value
    else:
        changes[name] = parsed(value, type(current))
spec.config = spec.config._replace(**changes)


def hx(value):
    return "0x" + bytes(value).hex()


profile_input = json.loads((root / "profile.json").read_text())
withdrawable_epoch = profile_input.get("withdrawable_epoch", 766)
old_pub = bytes.fromhex(
    profile_input.get(
        "retired_pubkey",
        "0xb49b274d74d615523a5591a22a37b4e9cdfb1fbaab95df7e8aae393d5823493011965f67213574ffcd7138c9689aa96e",
    ).removeprefix("0x")
)
blocks = {}
states = {}
for folder in [root / "canonical-states", root / "reuse-capture"]:
    for file in sorted(folder.glob("cl-1-lighthouse-geth-block-*.json")):
        block = parsed(json.loads(file.read_text())["data"], spec.SignedBeaconBlock)
        identity = hx(spec.hash_tree_root(block.message))
        path = folder / ("state-" + str(int(block.message.slot)) + ".ssz")
        if not path.exists():
            continue
        raw = path.read_bytes()
        state = spec.BeaconState.deserialize(io.BytesIO(raw), len(raw))
        assert spec.hash_tree_root(state) == block.message.state_root
        blocks[identity] = block
        states[identity] = state
transitions = []
withdrawals = []
reuses = []
for identity, block in sorted(
    blocks.items(), key=lambda item: int(item[1].message.slot)
):
    parent = hx(block.message.parent_root)
    if parent not in states:
        continue
    before, after = states[parent], states[identity]
    replayed = before.copy()
    spec.state_transition(replayed, block, validate_result=True)
    assert spec.hash_tree_root(replayed) == spec.hash_tree_root(after)
    a, b = before.builders[1], after.builders[1]
    row = {
        "slot": int(after.slot),
        "beacon_root": identity,
        "before_slot": int(before.slot),
        "before_root": hx(spec.hash_tree_root(before)),
        "after_root": hx(spec.hash_tree_root(after)),
    }
    transitions.append(row)
    if (
        bytes(a.pubkey) == old_pub
        and bytes(b.pubkey) == old_pub
        and int(a.balance) == 50000000000
        and int(b.balance) == 0
    ):
        assert (
            int(a.withdrawable_epoch) == withdrawable_epoch
            and int(after.slot) // 32 >= withdrawable_epoch
        )
        paid = [
            w
            for w in after.payload_expected_withdrawals
            if int(w.validator_index) == (1 << 40) | 1
        ]
        assert sum(int(w.amount) for w in paid) == 50000000000
        withdrawals.append(
            row
            | {
                "builder_index": 1,
                "debit_gwei": 50000000000,
                "withdrawable_epoch": withdrawable_epoch,
                "execution_parent": hx(
                    block.message.body.signed_execution_payload_bid.message.parent_block_hash
                ),
                "execution_payload": hx(
                    block.message.body.signed_execution_payload_bid.message.block_hash
                ),
            }
        )
    if bytes(a.pubkey) == old_pub and bytes(b.pubkey) != old_pub:
        assert (
            int(a.balance) == 0
            and int(b.balance) == 50000000000
            and int(b.withdrawable_epoch) == 2**64 - 1
            and int(b.deposit_epoch) >= withdrawable_epoch
        )
        reuses.append(
            row
            | {
                "builder_index": 1,
                "old_pubkey": hx(a.pubkey),
                "new_pubkey": hx(b.pubkey),
                "new_balance_gwei": int(b.balance),
                "new_deposit_epoch": int(b.deposit_epoch),
            }
        )
assert len(withdrawals) == len(reuses) == 1 and len(transitions) >= 60, (
    len(transitions),
    withdrawals,
    reuses,
)
if args.registry_only:
    result = {
        "passed": True,
        "scope": "Actual sweep and index reuse only; the separate retired-key node-envelope guard remains unevaluated.",
        "signature_guard_evaluated": False,
        "reference_source": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
        "complete_signed_state_transitions": len(transitions),
        "transitions": transitions,
        "actual_sweep": withdrawals,
        "actual_index_reuse": reuses,
    }
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({k: v for k, v in result.items() if k != "transitions"}))
    raise SystemExit(0)
control = root / "replaced-key"
records = [
    json.loads(line)
    for line in (control / "proxy-exchanges.jsonl").read_text().splitlines()
]
bad = [r for r in records if r["action"] == "replaced-key-signature-probe"]
good = [r for r in records if r["action"] == "first-envelope-to-observer"]
unavailable = [
    r for r in records if r["action"] == "replaced-key-availability-after-rejection"
]
assert len(bad) == len(good) == len(unavailable) == 1
assert (
    bad[0]["complete"]
    and bad[0]["http_status"] == 400
    and b"signature" in base64.b64decode(bad[0]["response_body"]).lower()
)
assert (
    unavailable[0]["complete"]
    and unavailable[0]["http_status"] == 404
    and good[0]["complete"]
    and good[0]["http_status"] == 200
)


def envelope(record):
    raw = base64.b64decode(record["request_body"], validate=True)
    start = int.from_bytes(raw[:4], "little")
    stop = int.from_bytes(raw[4:8], "little")
    assert start == 12 and start < stop <= len(raw)
    wire = raw[start:stop]
    return spec.SignedExecutionPayloadEnvelope.deserialize(io.BytesIO(wire), len(wire))


a, b = envelope(bad[0]), envelope(good[0])
assert (
    spec.hash_tree_root(a.message) == spec.hash_tree_root(b.message)
    and a.signature != b.signature
    and int(b.message.builder_index) == 1
)
latest = max(states.values(), key=lambda state: int(state.slot))
domain = spec.get_domain(latest, spec.DOMAIN_BEACON_BUILDER)
signing = spec.compute_signing_root(b.message, domain)
pub = bytes.fromhex(reuses[0]["new_pubkey"][2:])
assert (
    bls.Verify(old_pub, signing, a.signature)
    and not bls.Verify(pub, signing, a.signature)
    and bls.Verify(pub, signing, b.signature)
)
profile = json.loads((control / "profile.json").read_text())
slot = profile["slot"]
block = parsed(
    json.loads((control / ("block-" + str(slot) + ".json")).read_text())["data"],
    spec.SignedBeaconBlock,
)
assert spec.hash_tree_root(block.message) == b.message.beacon_block_root
bid = block.message.body.signed_execution_payload_bid
assert (
    bid.message.builder_index == 1
    and bid.message.block_hash == b.message.payload.block_hash
)
assert bls.Verify(pub, spec.compute_signing_root(bid.message, domain), bid.signature)
assert bls.Verify(
    latest.validators[block.message.proposer_index].pubkey,
    spec.compute_signing_root(
        block.message, spec.get_domain(latest, spec.DOMAIN_BEACON_PROPOSER)
    ),
    block.signature,
)
result = {
    "passed": True,
    "reference_source": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "bls_active": True,
    "complete_signed_state_transitions": len(transitions),
    "transitions": transitions,
    "actual_sweep": withdrawals,
    "actual_index_reuse": reuses,
    "retired_signature_verified_under_original_key": True,
    "retired_signature_rejected_under_current_key": True,
    "original_new_key_signature_verified": True,
    "actual_rejection_http_status": 400,
    "no_availability_after_rejection_http_status": 404,
    "untouched_original_new_key_delivery_http_status": 200,
    "signed_new_key_bid_and_beacon_block": True,
}
args.output.write_text(json.dumps(result, indent=2) + "\n")
print(json.dumps({k: v for k, v in result.items() if k != "transitions"}))
