# Group A completeness and adverse-input recheck

> Historical review at `5fb8c998`. The [case-level baseline and fee audit](CASE_LEVEL_AUDIT.md) records the subsequent gaps, fixes and current qualification.

Review date: 2026-11-26. Starting revision: `eaa13674987d2371e3e038a3f360c106cff50697`, current PR #168; canonical master is `89b3b1f2d12352777903de9803f3c93285bdcdf1`. This review retains the earlier production head-number fix and all 25 original acceptance criteria. It tests the acceptance mechanisms as well as the features they describe.

## Findings and closures

| Task | Recheck / closure |
| --- | --- |
| HGT-001 | Existing normative pins, source hashes, six profiles and nine decision gates are retained. Direct Geth v1.17.10 identity remains consistent. No specification bytes were edited or new release scope inferred. |
| HGT-004 | Native, helper, wrapper, RPC, FCU and container routes remain distinguished. Current task branch/date and follow-up evidence links replace stale original-branch metadata. |
| HGT-005 | Existing block-height test remains required. New independent canonical processor/pool tests cover before/at/after the time fork, reject before mutation and exercise reorg eviction/recovery. Pool tests call the same `SetForkRules` then `Reset` sequence as native head updates. |
| HGT-006 | Run validation now rejects empty runs, nonzero exits, named/package failures, inconsistent package totals, passing claims exceeding execution counts, duplicate/unexplained skips and failed race runs. All 16 exact prerequisite IDs are mandatory. Source fingerprints are recomputed and must match runtime, summary and test evidence; executable artifacts cannot disappear from the source manifest. |
| HGT-007 | Empty gas/domain families, swapped case kinds/EIPs, unknown fixture fields/types and Python optimization bypasses now reject explicitly. Seven native cases consume declared pool-restoration flags, explicit zero escrow and complete balances. Two core conditional rule oracles plus 23 new boundary vectors cover duplicate/unsorted/overflow keys, sequence/legacy exhaustion, budget/rollback, slot window, source/salt/root domain and orphaned entries. Root-domain hashes were frozen with independent upstream Geth Keccak. |
| HGT-008 | Existing fail-closed bid/BLS/PQ/KZG/IL/native-backend/experimental boundaries and negative tests remain required. Skip/run checks cannot hide a failed safety qualification. No removed production stub or test bypass was restored. |
| HGT-009 | All 72 proposal records, dispositions, source/discussion mappings and append-only histories remain unchanged. Source/admission regression tests continue to run alongside the stricter acceptance checks. |

Previously reproduced bypasses were accepted by the earlier helper: deleting both oracle families, corrupting expected gas under `python -O`, and supplying failed race metadata with zero skips. New regression tests reject each. Ordinary and optimized Python runs execute the same checks; completion does not depend on `assert` statements.

New conditional fixtures are isolated specification oracles for the named transitions. They do not activate keyed-nonce/recent-root production code or resolve D01–D09. In the nonce oracle, `valid` is nonce pre-state validity; payment commitment is separately represented by `approval_applied`. Existing native implementation and later independent-client/release qualification remain distinct.

## Verification

Final results and current source/artifact fingerprints are recorded in `evidence/hegota-a/verification.json`; exact-head hosted results are recorded on the English PR #168. The preceding `eaa13674` results are preserved at that immutable revision, not presented as fresh executions.

Fresh local results: full Go 23,531 named pass events, zero failures, 124 separately inventoried skips across 240 completed packages; focused race 3,504 pass events, zero failures, six skips across 31 completed packages. Both standard and optimized Python modes pass the same 49 regressions. Mandatory native/direct-Geth baseline: 12/12, zero failures/skips. Build/vet and both host binaries pass. All seven A tasks and 25 criteria are backed by 57 required executed Go test records. Native negative overlays reject the old block-zero implementation, incorrect nonce, false pool-restoration flag, explicit wrong zero escrow and unknown expected errors. None of these controls changes working-tree production code.
