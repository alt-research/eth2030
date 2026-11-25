# Group A acceptance completion

This follow-up closes the concrete omissions in [the post-merge audit](GROUP_A_ACCEPTANCE_AUDIT.md). Its baseline is merged master `89b3b1f2d12352777903de9803f3c93285bdcdf1`; the original audit remains a dated record. Current acceptance is governed by [25 criteria across all seven tasks](../../../specs/hegota/acceptance.json), the [task register](../../../specs/hegota/tasks.json) and [source-bound evidence](../../../evidence/hegota-a/verification.json).

| Finding / task | Closure |
| --- | --- |
| A1 / HGT-005 | Prefix simulation copies the stored real head number. A production pool regression covers nonzero London height, 9/10/11, rewind and recovery, with live-state isolation. Replacing the implementation with the old source through a Go overlay makes this regression fail at active head 10. |
| A2 / HGT-007 | All nine prose seeds are concrete input/output fixtures: seven execute the existing canonical VM/executor, two conditional rule oracles execute independently. Native cases assert approval/payment/nonce/balances/gas and VERIFY/batch/refill rollback. An overlay with an incorrect expected nonce is rejected by the native fixture consumer. No oracle is regenerated from the executor. |
| A3 / all A tasks | Completion is checked per criterion against current artifacts, actual named passing test events, source fingerprint and command checks. Missing/duplicate criteria, removed critical regressions, skipped tests, incomplete/stale runs and inconsistent references fail negative tests. Review/blocked states remain representable and do not pass completion. |
| HGT-001 | Active Geth v1.17.10 module origin/revision/checksum, profile and runner agree. Historical v1.17.1 provenance is retained with an explicit role and dated adoption history; normative EIP/CL/fixture sources are unchanged. |
| HGT-004 | Current native/wrapper/helper/RPC/pool/FCU/container routes replace the historical table; tests identify their actual route. Bogota methods and native selection are implemented, rather than incorrectly labeled unsupported. |
| HGT-006 | All 16 prerequisite records now identify current sources, scoped executed tests and remaining consumers. Deposit-log collection, direct canonical envelope dispatch and native FCU historical findings are resolved within their specific scope. Other existing implementations retain explicit independent qualification tasks rather than duplicate development work. |
| HGT-008 | Current safety inventory distinguishes real native backends, legacy helpers, unavailable IL evidence, KZG/BLS/PQ and experimental defaults/provider gates. Retired builder APIs and snap stubs remain retired. |
| HGT-009 | All 72 entries and their prior history remain intact. Reference metadata adoption retains its dated impact decision; immutable EIP/source bytes were not changed. |

## Fresh validation

- Build/vet and both host binaries passed.
- Full Go suite: 23,529 named test/subtest pass events, zero failures, 124 inventoried skips; all 240 packages completed (221 pass, 19 no-tests/skip).
- Focused race suite: 3,502 named pass events, zero failures, six inventoried skips; all 31 packages completed (29 pass, two no-tests/skip). Scope: configuration, execution, pool, node, Engine, FOCIL, BLS and PQ.
- Mandatory historical differential runner: 12 executed/passed, zero failures/skips, direct Geth v1.17.10.
- Python regressions: 33 passing tests, including acceptance/reference/skip/semantic mutation controls. Seven codec goldens, two gas oracles, one domain oracle, seven native semantic cases and two isolated conditional rule oracles are checked.
- Negative Go overlays reject the original head-number implementation and an incorrect semantic expected nonce without modifying working-tree source.

The verification file binds current source and artifact hashes and names every required executed Go test. Earlier broad/tagged/hosted results remain pinned to their original revisions. Exact-head hosted outcomes for this follow-up are recorded on its English PR; local results do not claim new hosted real-proof runs.

## Acceptance boundary

All Group A deliverables are complete within the stated criteria. This does not mean every implementation/qualification task in groups B–J is complete. D01–D09 and the inactive combined release profile remain explicit. Conditional nonce/root rule oracles do not claim production extension activation, independent CL agreement, official devnet operation, containers/RC or long-duration/resource acceptance. Those were not omissions in the bounded A deliverable and are not silently marked complete.
