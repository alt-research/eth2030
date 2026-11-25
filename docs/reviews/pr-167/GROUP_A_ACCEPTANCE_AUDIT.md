# Group A acceptance audit

Audited on 2026-11-25 against merged master `89b3b1f2d12352777903de9803f3c93285bdcdf1`, whose tree equals PR #167 head `a86473ea6c04fa33a2fd488c33506380d2b96a8f`. The PR is merged and its 15 hosted checks passed. This is an acceptance review of the seven Group A deliverables, not a requirement to implement all later groups before completing A.

**Conclusion: Group A cannot currently be accepted as fully complete.** The task register's seven `verified` flags predate this audit. Existing code and qualified tests should be reused; unresolved specification combinations remain isolated.

## Task-by-task disposition

| Task | Acceptance assessment | Evidence and remaining work |
| --- | --- | --- |
| HGT-001 | Delivered with a metadata inconsistency | Fourteen repository pins, 108 file hashes, six profiles and nine decisions exist. `profiles.json` identifies the active historical reference as Geth v1.17.1, while `pkg/go.mod`, the runner and current evidence use v1.17.10. Reconcile active reference provenance while retaining historical pins as history. |
| HGT-004 | Runtime code exercised; current route table incomplete | The native identity, versioned Bogota dispatcher and actual FCU/head-change tests pass. The registered route document is explicitly historical and still names deleted tests and unsupported APIs that master now implements. Publish a current route/test matrix and link it from the task record. |
| HGT-005 | Incomplete: reproduced runtime defect | Legacy isolation, aliases and FCU/reorg checks pass, but canonical prefix simulation discards the head number. A valid Frame can fail public admission when historical fork blocks are nonzero. See finding A1. |
| HGT-006 | Minimum baseline deliverable verified; historical gap inventory needs maintenance | The mandatory native/direct-Geth runner executes and passes all 12 cases with zero skips. Current skip/build evidence exists. Fifteen prerequisite records explicitly require requalification on integrated master; they are a gap inventory, not proof that implementations are absent. No full Glamsterdam or Hegota conformance claim follows from this minimal baseline. |
| HGT-007 | Partially delivered | Schema, six ADRs, seven encoding goldens, two arithmetic goldens and one root-domain golden exist. The remaining nine approval/rollback/nonce/root scenarios are natural-language trace seeds without concrete runnable fixtures. See finding A2. |
| HGT-008 | Scoped safety checks verified; historical caller table needs synchronization | Targeted BLS, PQ, unauthenticated bid, missing-verifier, native backend and activation checks pass. The historical audit still mentions deleted builder APIs and obsolete snap stubs; the integration audit explains their replacement. Consolidate the current caller table without reviving removed implementations. This is not a whole-repository security certification. |
| HGT-009 | Delivered within the pinned scope | All 72 proposals are present exactly once: three implement, 29 conditional, three watch and 37 excluded. Source/admission validation and 12 update-check regressions pass. No conditional proposal is thereby admitted to the release profile. |

## A1: canonical prefix simulation loses the actual block number

Priority: high. Caller: `pkg/txpool/frame_8141.go:74`.

`validateFrame8141` constructs the simulation header with `Number: new(big.Int)`, even after `Reset(db, head)` has stored the real height in `pool.frameNumber`. `ConfigureBlockEVM` selects fork rules from that simulated number; `ChainConfig.Rules` disables merge-dependent rules if London is not active at that number. A timestamp alone cannot recover the missing block-fork context. Public validation prohibiting the NUMBER opcode does not remove the need to select the correct VM rules.

Reproduced through exported production APIs, without changing source or using a mock executor:

1. Copy `config.TestConfig`, set `LondonBlock=10` and `HogotaTime=0`.
2. Fund sender `0x0000000000000000000000000000000000000091` with 100,000,000 wei and install `0x60035f5faa` (combined APPROVE with empty return data).
3. Construct a canonical Frame with this sender, chain ID 1337, nonce 0, no signatures, one VERIFY frame with flags 3 and execution limit 30,000; max fee 1,000,000 and priority cap 2. A coded account can define its own validation; this is a rule-selection counterexample, not an account-security recommendation.
4. Use a head with number 10, timestamp 100, gas limit 30,000,000 and base fee 1.
5. `execution.ValidateFrame8141Prefix` with the real head succeeds. `pool.SetForkRules`, `pool.Reset(db, head)` and `pool.AddRemote` with the same inputs fail:

```text
direct_correct_head=<nil>
pool_same_head=EIP-8141 verify frame 0: invalid opcode
```

The current tests largely activate historical forks at genesis and do not catch this case. Closure requires propagating the stored head number into simulation and regression coverage for nonzero block-fork prerequisites and head changes, preserving live-state isolation. Base this work on the pinned EIP-8141 behavior/public-prefix rules and existing shared executor; do not build a second validation engine.

## A2: semantic trace seeds are not complete goldens

`specs/hegota/semantic-goldens.json:81` contains nine natural-language inputs/expectations. `tools/hegota/semantic_goldens.py:26` checks only their count; it validates arithmetic/domain vectors but does not evaluate the trace descriptions. In particular, the same gate would pass if an approval or rollback expectation were replaced with an incorrect string.

This is an A-level fixture-delivery gap, separate from later native implementation qualification. Approval, rollback and nonce/window scenarios need concrete inputs and machine-checkable expected outputs, or an explicit per-scenario specification blocker with an agreed partial acceptance scope. Reuse canonical Frames already on master when qualifying executable base-profile cases. Keep keyed-nonce and recent-root extensions conditional under their recorded decisions; do not combine incompatible drafts merely to make a test run. Existing `not_implemented_for_current_spec` labels are historical assertions and must not be treated as a fresh audit of master.

## A3: completion checking trusts completion labels

`tools/hegota/check.py:140` requires every A task's status to equal `verified` and checks that artifacts exist. It does not establish that each acceptance criterion has runnable evidence. Source fingerprints protect provenance, not semantic completeness. This explains how a green check coexists with A1/A2.

Closure requires task-specific acceptance evidence and a truthful register that can represent review/incomplete work without pretending all seven tasks are verified. Add meaningful negative coverage for missing or inconsistent acceptance evidence. Preserve old verification records at their original revisions instead of rewriting historical outcomes.

## Fresh verification

The following passed on the audited source tree:

- `python3 tools/hegota/check.py`: 72 proposals, 99 checked local source hashes, 108 pinned source files, six profiles, nine decisions and seven codec goldens.
- `python3 -m unittest discover -s tools/hegota -p 'test_*.py'`: 12 tests.
- `python3 tools/hegota/semantic_goldens.py`: two arithmetic oracles, one root-domain oracle, nine explicitly unexecuted trace seeds.
- `env -u GOROOT GOFLAGS=-p=4 go build ./...` from `pkg`.
- `env -u GOROOT GOFLAGS=-p=4 go run ./cmd/eth2030-baseline`: 12 executed, 12 passed, zero failed/skipped; direct reference Geth v1.17.10.
- Focused Go tests covering configuration/aliases, codec goldens, legacy processor/VM/pool boundaries, canonical pool behavior, native runtime/FCU/reorg, unauthenticated bids, BLS backend/vectors and PQ rejection. Packages with no matching tests are not counted as executed cases.

These are fresh local checks, not a rerun of the full suite, race matrix, containers, CL interoperability or hosted real-proof workflows. The earlier hosted results remain bound to `a86473ea`. No production code, source lock, oracle or existing verification output was changed for this audit.
