# Group D verification

## Identity and scope

Implementation: `c449112d2e455eef15546ff46418431cd170daaa`.
Baseline: `81e63f586642a1406d56db9fa7cbf00941feb7bf` (merged PR #170).
Reviewed on 2026-11-27, Asia/Shanghai. The final source/fixture fingerprints are
checked by `tools/hegota/check.py`, `check_b.py`, `check_c.py` and `check_d.py`.
Review tracker renames and evidence-only commits do not change the tested code.

The [plan](../../plans/hegota-d/PLAN.md) and
[composition report](../../plans/hegota-d/COMPOSITION.md) define all eight tasks
and their 32 criteria. Source references are immutable; the EIPs master and
EIP-8504 PR heads were rechecked and still match the pinned revisions.

## Executed qualification

All Go commands use `env -u GOROOT GOFLAGS=-p=4`.

| Command or consumer | Observed result |
|---|---|
| `go build ./...` | Exit 0 |
| `go vet ./...` | Exit 0 |
| `gofmt -l .` | No unformatted files |
| Native and geth wrapper host builds plus `--version` | Exit 0; actual commit and binary hashes in runtime manifest |
| `go test -json ./... -count=1` | 243/243 packages completed; 24,166 named pass, 0 fail, 124 optional skips |
| Affected `go test -race -json ... -count=1` | 68/68 packages completed; 10,801 named pass, 0 fail, 78 optional skips |
| `go run ./cmd/eth2030-baseline` | 12 executed, 12 pass, 0 fail, 0 skip against unchanged Geth v1.17.10 |
| Python `unittest discover -s tools/hegota -p 'test_*.py'` | 102 pass in normal and optimized mode |
| Fixed-source and A/B/C/D checks, codec/semantic/keyed goldens | All pass; independent expected values unchanged |
| Mandatory native consumers A / B / C / D | 62 / 79 / 102 / 45; zero omissions or skips |

The race command covers bal, core/{types,config,vm,execution,block,chain,state,
framepay,frameproof,rawdb}, crypto/leansphincs, txpool/..., eth, node/..., rpc/...,
engine/..., p2p/ethproto, sync/statesync, cmd/{eth2030,send-frame} and eiptest.
Every D mandatory consumer appears as an actual pass in both full and race logs.
Optional skip reasons are in `evidence/hegota-a/skipped-tests.json`; missing
optional corpora are never reported as executed native passes.

## Independent and native boundaries

- All 26 immutable EthLib/leanVM verification vectors and every one of 7,740
  signature truncations. Valid verification uses exactly 515 Keccak-f permutations.
- Independent EELS Python signing/verifying plus Python RLP/Keccak produces
  canonical base/keyed PQ-wallet vectors. The wallet checks its own declaration,
  key and zero-message convention; replay-identity mutation fails verification.
- Independent remerkleable Gloas 19/20-field progressive SSZ bytes/roots, plus
  actual native Engine block/BAL/witness conversion to the CL codec.
- Real HTTP simulation/submission, Engine build/import/body queries and restart;
  live peer wrapper admission/exchange and ordinary-eth isolation. Missing,
  tampered, extra and wrongly bound witnesses fail before state contamination.
- Wrapper replacement, eviction, reorg, full byte budgets, expiry indexing and
  bounded draining beyond one request have direct native consumers.
- Adoption gas/static/system/source validity, loaded-code versus reentry,
  CREATE/CREATE2 immediate callback, ignored EF return, persistent bytes and
  hash-only BAL; migration retires old ordinary/redelegation authority while
  Frame control remains usable.
- Existing 8151 signer-class rules remain intact through both D journals.
  Native permit/multisig/fault-proof authorization kernels distinguish 8151
  off/on; complete deployed application qualification remains a release gate.
- All ASSERT selectors and event copying, net prestate tables, actual escrow,
  static subtree approval, paid/unpaid failure, entire-body rollback, BAL read
  retention and exact PR-12041 paymaster/token false-return/retry/atomic cases.

Three Go overlays independently remove witness verification, ASSERT body
restoration or code adoption while leaving tests unchanged. Each compiles and
fails its intended native consumer. Their log and overlay identities are in
`evidence/hegota-d/verification.json`. Earlier A/B/C extended controls remain
immutable predecessor provenance, not newly claimed D executions.

## Conflict and release review

The branch is based on current merged master. The root trackers and refs have
no diff; no old independent expected vectors were changed. No conflict markers
or staged whitespace errors remain. Direct verification does not claim recursive
aggregation. Official transport assignments, provisional parameters, CL/external
devnet qualification and undefined 7807/omitted-proof IL composition remain
closed gates. A bare omitted proof never receives a witness-free validity
verdict. Existing A/B/C release gates remain in force.

The PR's exact-head hosted checks are the live CI authority; this offline record
claims the local runs above, not an unobserved hosted result.

## Hosted observation

On head `335466499783ceed25de596e2ba0fb8b3e5fb67d`, all nine
[CI jobs](https://github.com/alt-research/eth2030/actions/runs/45895680692)
and [independent snap2 interoperability](https://github.com/alt-research/eth2030/actions/runs/45895680579)
completed successfully. This includes the complete selected official native
state/blockchain corpora, published application sequences, pinned EF vectors,
consensus/control fixtures, frozen paymaster rebuild, fixed-source checks,
full Go tests and all four build variants. Actual canonical-proof acceptance
and any final documentation-head reruns are separately reported by the live
PR suite. No pending or canceled run is counted as a pass.
