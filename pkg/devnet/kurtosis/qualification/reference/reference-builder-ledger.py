import argparse
import io, json, sys
from pathlib import Path
from frozendict import frozendict
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.utils import bls

sys.path.insert(0, str(Path(__file__).resolve().parent))
from ssz_rest import parsed

parser = argparse.ArgumentParser(
    description="Replay the authentic signed payment transition and reconcile independent EL proof results"
)
parser.add_argument("--artifact-dir", required=True, type=Path)
args = parser.parse_args()
root = args.artifact_dir
bundle = json.loads(
    (root / "attempt4-builder-ledger-fresh-20261119T234834Z.json").read_text()
)
out = root / "replay-reference-builder-ledger.json"
assert not out.exists()
report = {
    "passed": False,
    "spec_revision": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "reference_source_unmodified": True,
    "bls_active": bls.bls_active,
}


def save():
    out.write_text(json.dumps(report, indent=2) + "\n")


try:
    assert bls.bls_active
    network = json.loads(
        (root / "actual-builder-ledger-network-spec.json").read_text()
    )["data"]
    updates = {}
    for name, current in spec.config._asdict().items():
        if name not in network:
            continue
        value = network[name]
        if isinstance(current, tuple):
            updates[name] = tuple(
                frozendict(
                    {
                        key: int(v) if isinstance(v, str) and v.isdigit() else v
                        for key, v in item.items()
                    }
                )
                for item in value
            )
        elif isinstance(current, str):
            updates[name] = value
        else:
            updates[name] = parsed(value, type(current))
    spec.config = spec.config._replace(**updates)
    report["applied_config_fields"] = sorted(updates)
    report["blob_schedule"] = [dict(x) for x in spec.config.BLOB_SCHEDULE]
    save()
    states = []
    for label in ["before", "after"]:
        entry = bundle[label]
        raw = (root / entry["state_file"]).read_bytes()
        state = spec.BeaconState.deserialize(io.BytesIO(raw), len(raw))
        block = parsed(entry["block"]["data"], spec.SignedBeaconBlock)
        assert spec.hash_tree_root(state) == block.message.state_root
        assert (
            "0x" + bytes(spec.hash_tree_root(block.message)).hex()
            == entry["beacon_root"]
        )
        states.append((state, block))
    (before, before_block), (after, after_block) = states
    assert spec.get_current_epoch(before) == spec.get_current_epoch(after)
    assert len(before.builders) == len(after.builders) == 1
    assert (
        not before.pending_deposits
        and not after_block.message.body.parent_execution_requests.builder_deposits
    )
    replay = before.copy()
    spec.state_transition(replay, after_block, validate_result=True)
    assert spec.hash_tree_root(replay) == spec.hash_tree_root(after)
    report["full_signed_state_transition_root_match"] = True
    report["slots"] = [int(before.slot), int(after.slot)]
    report["state_roots"] = [
        "0x" + bytes(spec.hash_tree_root(s)).hex() for s in [before, after]
    ]
    save()
    withdrawals = list(after.payload_expected_withdrawals)
    assert withdrawals and all(
        spec.is_builder_index(w.validator_index) for w in withdrawals
    )
    amount = sum(int(w.amount) for w in withdrawals)
    debit = sum(int(b.balance) for b in before.builders) - sum(
        int(b.balance) for b in after.builders
    )
    assert debit == amount and amount > 0
    el = json.loads((root / "replay-independent-el-ledger.json").read_text())
    assert el["passed"] and el["execution_hashes"] == bundle["execution_hashes"]
    assert len(el["observations"]) == 3
    for obs in el["observations"]:
        assert int(obs["paid_wei"]) == amount * 10**9
    report["payment"] = {
        "builder_debit_gwei": debit,
        "execution_credit_wei": str(amount * 10**9),
        "withdrawal_indices": [int(w.index) for w in withdrawals],
        "builder_indices": [
            int(spec.convert_validator_index_to_builder_index(w.validator_index))
            for w in withdrawals
        ],
        "execution_endpoints_verified": 3,
        "supply_scope": "Exact matched builder/recipient transfer; validator epoch issuance and whole-chain total supply are not inferred.",
    }
    controls = []
    payment_index = int(spec.SLOTS_PER_EPOCH) + int(
        before.latest_block_header.slot
    ) % int(spec.SLOTS_PER_EPOCH)
    assert (
        int(before.builder_pending_payments[payment_index].withdrawal.amount) == amount
    )
    for name in [
        "duplicate_pending_payment",
        "missing_pending_payment",
        "duplicate_builder_debit",
    ]:
        altered_state = before.copy()
        if name == "duplicate_pending_payment":
            withdrawal = withdrawals[0]
            altered_state.builder_pending_withdrawals.append(
                spec.BuilderPendingWithdrawal(
                    fee_recipient=withdrawal.address,
                    amount=withdrawal.amount,
                    builder_index=spec.convert_validator_index_to_builder_index(
                        withdrawal.validator_index
                    ),
                )
            )
        elif name == "missing_pending_payment":
            altered_state.builder_pending_payments[payment_index] = (
                spec.BuilderPendingPayment()
            )
        else:
            altered_state.builders[0].balance -= spec.Gwei(amount)
        try:
            spec.state_transition(altered_state, after_block, validate_result=True)
        except AssertionError:
            controls.append(name)
        else:
            raise AssertionError("reference accepted state/payment mutation " + name)
    altered = after_block.copy()
    altered.message.body.signed_execution_payload_bid.message.value += 1
    try:
        spec.state_transition(before.copy(), altered, validate_result=True)
    except AssertionError:
        controls.append("modified_signed_payment_bid")
    else:
        raise AssertionError("reference accepted modified signed payment bid")
    finality = json.loads((root / "replay-reference-period1-finality.json").read_text())
    assert (
        finality["passed"]
        and finality["actual_payment_finalized_ancestor"]["root"]
        == bundle["after"]["beacon_root"]
    )
    report["authenticated_finalized_ancestor"] = finality[
        "actual_payment_finalized_ancestor"
    ]
    report["rejected_controls"] = controls
    report["passed"] = True
    save()
    print(
        json.dumps(
            {
                "passed": True,
                "full_signed_state_transition_root_match": True,
                "slots": report["slots"],
                "payment": report["payment"],
                "rejected_controls": controls,
            }
        ),
        flush=True,
    )
except Exception as e:
    report["failure"] = {"type": type(e).__name__, "message": str(e)}
    save()
    raise
