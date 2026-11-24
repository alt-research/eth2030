"""Export independent EIP-8045/8061 expectations; never imports client code.

Use the unchanged generated consensus reference venv. Arguments: Lighthouse
fixture directory, the pinned EIP-8061 Markdown source, output directory.
"""
import gzip
import inspect
import io
import json
from pathlib import Path
import re
import subprocess
import sys

from eth_consensus_specs.gloas import mainnet as spec

source = next(p for p in Path(inspect.getfile(spec)).parents if (p / ".git").exists())
assert subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip() == "d2021d2719235f0df4726a1d330502cb15aa8ea3"
assert not subprocess.check_output(["git", "-C", str(source), "diff", "--", "specs", "pysetup", "tests/core/pyspec/eth_consensus_specs/utils"], text=True)
fixture = Path(sys.argv[1])
raw = gzip.decompress((fixture / "gloas-384-state.ssz.gz").read_bytes())
expected = json.loads((fixture / "gloas-384-proofs.json").read_text())
def state_from_capture():
    state = spec.BeaconState.deserialize(io.BytesIO(raw), len(raw))
    assert bytes(spec.hash_tree_root(state)).hex() == expected["state_root"].removeprefix("0x")
    return state

state = state_from_capture()
epoch = int(spec.get_current_epoch(state)) + 2
decision_slot = int(spec.compute_start_slot_at_epoch(spec.get_current_epoch(state) + 1)) - 1
proposers = []
for stride in [0, 2, 3]:
    state = state_from_capture()
    state.slot = spec.Slot(decision_slot)
    if stride:
        for index in range(0, len(state.validators), stride):
            state.validators[index].slashed = True
    indices = [int(x) for x in spec.get_beacon_proposer_indices(state, spec.Epoch(epoch))]
    assert len(indices) == 32 and not any(state.validators[x].slashed for x in indices)
    proposers.append({"slash_stride": stride, "epoch": epoch, "expected": indices})

# Execute the literal normative functions with only their primitive state-read
# and type environment supplied. Synthetic aggregate totals test rounding/caps;
# they do not assert an actual 36M-ETH captured state or validator population.
ns = {"BeaconState": dict, "Gwei": int, "uint64": int,
      "get_total_active_balance": lambda s: s["total"],
      "MIN_PER_EPOCH_CHURN_LIMIT_ELECTRA": int(spec.config.MIN_PER_EPOCH_CHURN_LIMIT_ELECTRA),
      "EFFECTIVE_BALANCE_INCREMENT": int(spec.EFFECTIVE_BALANCE_INCREMENT),
      "CHURN_LIMIT_QUOTIENT_GLOAS": 32768,
      "CONSOLIDATION_CHURN_LIMIT_QUOTIENT": 65536,
      "MAX_PER_EPOCH_ACTIVATION_CHURN_LIMIT": 256_000_000_000,
      "MIN_VALIDATOR_WITHDRAWABILITY_DELAY": int(spec.config.MIN_VALIDATOR_WITHDRAWABILITY_DELAY),
      "SAFETY_DECAY": int(spec.SAFETY_DECAY)}
eip_bytes = Path(sys.argv[2]).read_bytes()
if sys.argv[2].endswith(".gz"):
    eip_bytes = gzip.decompress(eip_bytes)
for block in re.findall(r"```python\n(.*?)```", eip_bytes.decode(), re.S):
    if "def get_activation_churn_limit(" in block or "def compute_weak_subjectivity_period(" in block:
        exec(compile(block, "EIP-8061 normative functions", "exec"), ns)
churn = []
for eth in [1, 32, 4_194_303, 4_194_304, 4_194_305, 8_388_607,
            8_388_608, 8_388_609, 36_000_000, 120_000_000]:
    state = {"total": eth * 1_000_000_000}
    churn.append({"total": state["total"], "activation": ns["get_activation_churn_limit"](state),
                  "exit": ns["get_exit_churn_limit"](state),
                  "consolidation": ns["get_consolidation_churn_limit"](state),
                  "weak_subjectivity": ns["compute_weak_subjectivity_period"](state)})
topups = []
for prefix in [0xb0, 0xff]:
    captured = state_from_capture()
    builder = captured.builders[0]
    before = int(builder.balance)
    recipient = bytes(builder.execution_address)
    request = spec.BuilderDepositRequest(pubkey=builder.pubkey,
        withdrawal_credentials=bytes([prefix]) + bytes(31), amount=1_000_000_000,
        signature=bytes(96))
    spec.process_builder_deposit_request(captured, request)
    topups.append({"prefix": prefix, "before": before, "after": int(captured.builders[0].balance),
        "credited": int(captured.builders[0].balance) - before,
        "execution_address_unchanged": bytes(captured.builders[0].execution_address) == recipient})
assert topups[0]["credited"] == 1_000_000_000 and topups[1]["credited"] == 0
out = Path(sys.argv[3])
out.mkdir(parents=True, exist_ok=True)
(out / "cl-captured-state.ssz").write_bytes(raw)
(out / "BUILDER_TOPUP_REFERENCE.json").write_text(json.dumps({"reference": "d2021d2719235f0df4726a1d330502cb15aa8ea3", "amount": 1_000_000_000, "cases": topups}, indent=2) + "\n")
(out / "CL_ORACLE.json").write_text(json.dumps({"reference": "ethereum/consensus-specs@d2021d2719235f0df4726a1d330502cb15aa8ea3",
    "state_root": expected["state_root"], "captured_validators": 384,
    "proposer_cases": proposers, "decision_slot": decision_slot, "lookahead": list(map(int, state_from_capture().proposer_lookahead)),
    "churn_input_scope": "Synthetic cached aggregate totals for actual helper-contract tests, not network-scale admission.",
    "churn_cases": churn}, indent=2) + "\n")
print(json.dumps({"proposer_cases": len(proposers), "churn_cases": len(churn)}))
