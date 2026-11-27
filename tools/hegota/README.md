# Hegotá A verification tools

From the repository root:

```sh
python3 tools/hegota/check.py
python3 tools/hegota/codec_goldens.py
python3 tools/hegota/semantic_goldens.py
python3 -m unittest discover -s tools/hegota -p 'test_*.py'
cd pkg
env -u GOROOT go run ./cmd/eth2030-baseline
```

All Python verification uses only the standard library; it performs no network
calls and does not change locks/oracles. Source check requires the two pinned
reference submodules. `check.py --compare-lock NEW_LOCK --compare-admission
NEW_ADMISSION` prints old/new source and disposition changes plus affected tasks;
changed EIP hashes, revisions or activation decisions require an appended dated impact/admission history entry. New source locks must match admission records; old history cannot be rewritten. Repository SHA changes require dated `source_updates` records naming repo/ref, previous/new SHA, affected tasks and rationale. Immutable source URLs cannot silently change their digest. Additional/removal of non-scope source files requires `file_updates` old/new digests, date, tasks and rationale unless covered by a reviewed repository change. It never
adopts the new files. New scope requires a new explicit baseline version.

`codec_goldens.py --write` and `cd pkg && go run
../tools/hegota/bls_vector.go` are explicit, review-only regeneration commands.
The first uses an independent minimal Python RLP/Keccak oracle. The second signs
a deterministic public test message with the separately pinned blst library;
normal builds test gnark verification against the frozen bytes. The fixture's
key is exclusively a public test vector. Neither command is invoked to update
an oracle during acceptance.

`semantic_goldens.py` checks five hand-worked gas/fee oracles, a root-domain oracle,
structured fixture coverage and two isolated nonce/window rule oracles. All nine
former trace seeds now have concrete inputs/outputs. Seven base cases are executed
by `core/execution.TestHegotaSemanticGoldens` using the existing native VM/executor;
the two core conditional oracles and 25 additional nonce/root boundary vectors do not activate a composed network profile. Semantic checks reject missing oracle families, unknown fields and type drift, including under Python optimization.

`acceptance.py` validates the criterion matrix, actual passing native test events,
current artifact/source hashes, skip counts, prerequisite assessments and active
reference identity. It permits truthful review/blocked task records, while the
completion gate rejects unresolved criteria. Neither `verified` labels nor
existing artifact filenames replace execution evidence. Negative regressions
cover missing criteria/tests, skipped/incomplete/stale runs, version drift and
incorrect conditional semantics. CI runs the source/acceptance checks and strict
baseline before the normal Go checks; Go tests execute the native fixtures.

Persisted differential reports are checked case by case against the pinned manifest and fixture state/log oracles, with exact coverage and native/reference gas agreement. The explicit gas/settlement acceptance criterion requires all five native budget/payment/settlement consumers; passing summary counters or arithmetic-only labels cannot replace them.

Group C is registered in `specs/hegota/group-c.json` with ten tasks and forty
clause/code/native-consumer criteria. Run `python3 tools/hegota/check_c.py` to
verify committed source-bound evidence. To qualify an implementation revision,
first complete `cd pkg && env -u GOROOT GOFLAGS=-p=4 go test -json ./... -count=1`
and then supply its complete successful JSON event log with
`python3 tools/hegota/check_c.py --write --log /absolute/path/to/full.jsonl`.
`test_group_c.py` checks omission, stale-source and public-gate controls in both
normal and optimized Python modes. CI verifies A, B and C independently.

A green C result qualifies selected-profile consumers. It deliberately reports
`public_complete: false`: upstream adoption and fork pinning of the canonical
paymaster remain unresolved. The exact reproducible PR-12041 proposal runtime
is implemented and explicitly selectable; its full native consumers are mandatory.
D01 is resolved for the explicit EIP8250Time
composition by merged delta/base history and preserved independent vectors;
its native public consumers are mandatory, as are conditional EIP-8151 signer
class/gas/share regressions.
The local paymaster v1 and pinned PR-12041 proposal require separate, mutually
exclusive experimental selections. See
`docs/plans/hegota-c/PLAN.md` and `docs/plans/hegota-c/RPC_PROFILE.md` for exact
scope and the authenticated simulation contract. No vector expectation or
reference submodule is regenerated to obtain a green C result.

## Group E: canonical Heze and native Bogotá

`python3 tools/hegota/check_e.py --write --log /absolute/native-go-test.jsonl`
records the complete nine-task/36-criterion register against 26 required native
consumers and all exact source fingerprints. `check_e.py` then rejects stale or
missing evidence. `--validate-only` validates sources and mappings without
claiming test execution. Expected SSZ roots are independently generated by
`generate_e_heze_vectors.py`; normal acceptance never regenerates them.

See `docs/plans/hegota-e/COMPOSITION.md` for selected Heze/Bogotá precedence,
legacy helper retirement, Profile 2 activation prerequisites, and PQ/external
CL boundaries. The informational AA-VOPS reference helper is never a production
eligibility decision. The website retains those gates separately from historical
Glamsterdam completion.
