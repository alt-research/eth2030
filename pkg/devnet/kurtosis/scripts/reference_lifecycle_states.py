#!/usr/bin/env python3
import sys, json, io, copy, argparse, base64
from pathlib import Path
from frozendict import frozendict
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.utils import bls

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "qualification/reference"))
from ssz_rest import parsed

parser = argparse.ArgumentParser(
    description="Replay complete original signed state transitions, without native client code."
)
parser.add_argument("--capture", type=Path, required=True)
parser.add_argument("--output", type=Path, required=True)
parser.add_argument("--invalid-slot", type=int)
parser.add_argument("--invalid-engine", type=Path)
args = parser.parse_args()
root = args.capture
out = args.output
assert not out.exists()
report = {
    "passed": False,
    "reference_source": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "bls_active": bls.bls_active,
    "transitions": [],
    "missing_parent_captures": [],
}
assert bls.bls_active
network = json.loads((root / "network-spec.json").read_text())
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


blocks = {}
states = {}
for path in sorted(root.glob("cl-1-lighthouse-geth-block-*.json")):
    value = parsed(json.loads(path.read_text())["data"], spec.SignedBeaconBlock)
    block_root = hx(spec.hash_tree_root(value.message))
    blocks[block_root] = value
    state_path = root / ("state-" + str(int(value.message.slot)) + ".ssz")
    if state_path.exists():
        raw = state_path.read_bytes()
        state = spec.BeaconState.deserialize(io.BytesIO(raw), len(raw))
        assert spec.hash_tree_root(state) == value.message.state_root
        states[block_root] = state
controls = []
try:
    for block_root, after_block in sorted(
        blocks.items(), key=lambda kv: int(kv[1].message.slot)
    ):
        parent = hx(after_block.message.parent_root)
        if block_root not in states or parent not in states:
            report["missing_parent_captures"].append(int(after_block.message.slot))
            continue
        before, after = states[parent], states[block_root]
        replay = before.copy()
        spec.state_transition(replay, after_block, validate_result=True)
        assert spec.hash_tree_root(replay) == spec.hash_tree_root(after)
        previous_bid = blocks[parent].message.body.signed_execution_payload_bid.message
        bid = after_block.message.body.signed_execution_payload_bid.message
        withdrawals = [
            w
            for w in after.payload_expected_withdrawals
            if spec.is_builder_index(w.validator_index)
        ]
        amount = sum(int(w.amount) for w in withdrawals)
        debit = sum(int(b.balance) for b in before.builders) - sum(
            int(b.balance) for b in after.builders
        )
        row = {
            "before_slot": int(before.slot),
            "after_slot": int(after.slot),
            "before_root": hx(spec.hash_tree_root(before)),
            "after_root": hx(spec.hash_tree_root(after)),
            "beacon_root": block_root,
            "parent_beacon_root": parent,
            "parent_status": (
                "FULL" if bid.parent_block_hash == previous_bid.block_hash else "EMPTY"
            ),
            "builder_debit_gwei": debit,
            "expected_builder_withdrawals_gwei": amount,
            "after_execution_hash": hx(bid.block_hash),
            "parent_execution_hash": hx(bid.parent_block_hash),
            "withdrawals": [
                {
                    "index": int(w.index),
                    "validator_index": int(w.validator_index),
                    "address": hx(w.address),
                    "amount": int(w.amount),
                }
                for w in withdrawals
            ],
        }
        report["transitions"].append(row)
        # Full signed state-root validation must detect a waived/de-duplicated
        # payment obligation, not merely accept a structurally valid container.
        if not controls and amount > 0:
            changed = before.copy()
            changed.builders[0].balance -= spec.Gwei(1)
            try:
                spec.state_transition(changed, after_block, validate_result=True)
            except AssertionError:
                controls.append("changed_builder_balance")
            else:
                raise AssertionError(
                    "reference accepted changed committed builder balance"
                )
            pending = next(
                (
                    i
                    for i, p in enumerate(before.builder_pending_payments)
                    if int(p.withdrawal.amount) > 0
                ),
                None,
            )
            if pending is not None:
                changed = before.copy()
                changed.builder_pending_payments[pending].withdrawal.amount = spec.Gwei(
                    0
                )
                try:
                    spec.state_transition(changed, after_block, validate_result=True)
                except AssertionError:
                    controls.append("waived_original_pending_payment")
                else:
                    raise AssertionError(
                        "reference accepted waived original pending payment"
                    )
        (root / "reference-progress.json").write_text(
            json.dumps({"transitions": len(report["transitions"])}) + "\n"
        )
    if args.invalid_slot is not None:
        assert args.invalid_engine is not None
        slot = args.invalid_slot
        envelope = parsed(
            json.loads((root / f"{slot}-builder-envelope.json").read_text())["data"],
            spec.SignedExecutionPayloadEnvelope,
        )
        state = max(
            (state for state in states.values() if int(state.slot) < slot),
            key=lambda state: int(state.slot),
        )
        pubkey = state.builders[int(envelope.message.builder_index)].pubkey
        domain = spec.get_domain(state, spec.DOMAIN_BEACON_BUILDER)
        assert bls.Verify(
            pubkey,
            spec.compute_signing_root(envelope.message, domain),
            envelope.signature,
        )
        expected_hash = hx(envelope.message.payload.block_hash)
        expected_root = hx(envelope.message.payload.state_root)
        assert (
            int(envelope.message.payload.slot_number) == slot
            and expected_root == "0x" + "00" * 32
        )
        labels = []
        for line in args.invalid_engine.read_text().splitlines():
            record = json.loads(line)
            request = json.loads(
                base64.b64decode(record["request_body"], validate=True)
            )
            response = json.loads(
                base64.b64decode(record["response_body"], validate=True)
            )
            assert record["complete"] and record["http_status"] == 200
            assert (
                request["method"] == "engine_newPayloadV5"
                and request["params"][0]["blockHash"] == expected_hash
            )
            assert (
                request["params"][0]["stateRoot"] == expected_root
                and response["result"]["status"] == "INVALID"
            )
            labels.append(record["service"])
        assert sorted(labels) == [
            "el-1-geth-lighthouse-engine.jsonl",
            "el-2-geth-lodestar-engine.jsonl",
            "el-3-geth-lighthouse-engine.jsonl",
        ]
        observation = json.loads(
            (root / "invalid-root-process-result.json").read_text()
        )
        assert len(observation["no_imported_invalid_block"]) == 3
        assert all(
            json.loads(rec["response"])["result"] is None
            for rec in observation["no_imported_invalid_block"].values()
        )
        report["invalid_signed_root"] = {
            "slot": slot,
            "original_builder_signature": True,
            "actual_engine_rejections": labels,
            "invalid_block_unavailable_on_three_els": True,
        }
    assert (
        report["transitions"]
        and any(x["parent_status"] == "EMPTY" for x in report["transitions"])
        and any(x["parent_status"] == "FULL" for x in report["transitions"])
    )
    report.update(
        passed=True,
        rejected_controls=controls,
        complete_transitions=len(report["transitions"]),
    )
    out.write_text(json.dumps(report, indent=2) + "\n")
    print(
        json.dumps(
            {
                "passed": True,
                "transitions": len(report["transitions"]),
                "empty": sum(
                    x["parent_status"] == "EMPTY" for x in report["transitions"]
                ),
                "payments": [
                    (
                        x["after_slot"],
                        x["builder_debit_gwei"],
                        x["expected_builder_withdrawals_gwei"],
                    )
                    for x in report["transitions"]
                    if x["expected_builder_withdrawals_gwei"] > 0
                ],
                "rejections": controls,
            }
        ),
        flush=True,
    )
except Exception as error:
    report["failure"] = {"class": type(error).__name__, "message": str(error)}
    out.write_text(json.dumps(report, indent=2) + "\n")
    raise
