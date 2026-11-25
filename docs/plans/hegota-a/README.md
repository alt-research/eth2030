# Hegota Group A delivery

The seven A tasks establish pinned sources/admission, current runtime routes, activation and safety boundaries, prerequisite inventory, a strict minimum differential baseline and reproducible semantic fixtures. [Criterion-level acceptance](../../../specs/hegota/acceptance.json) and [source-bound verification](../../../evidence/hegota-a/verification.json) govern completion; task labels alone are insufficient.

| Task | Current deliverables |
| --- | --- |
| HGT-001 | [Profiles/source roles](specification-profiles.md): 15 repository records, 108 file hashes, six profiles, nine decisions; active Geth v1.17.10 separately identified from historical v1.17.1. |
| HGT-004 | [Current runtime matrix](runtime-and-activation.md), actual native/wrapper identities, wire and FCU/restart tests. |
| HGT-005 | Configuration aliases/compatibility, legacy isolation, canonical prefix using actual head height, before/at/after fork and reorg regressions. |
| HGT-006 | [Baseline/inventory](baseline-and-gaps.md): 12 mandatory native/direct-Geth cases, zero skips; all 16 prerequisite findings reassessed with current sources/tests and remaining consumer scope. |
| HGT-007 | [Six ADRs](semantic-adrs.md), schema, seven codec goldens, five gas and one root-domain oracle; all nine former trace seeds replaced by concrete fixtures with seven native and two core conditional-rule consumers, plus 25 concrete conditional boundary vectors. |
| HGT-008 | [Current caller audit](validation-audit.md): bid/BLS/PQ/KZG, native backend, unavailable IL evidence and experimental defaults/provider boundaries; retired implementations stay retired. |
| HGT-009 | [Admission](../../../specs/hegota/admission.json): all 72 proposals, source/discussion/disposition and append-only impact decisions. |

The [current seven-task review](../../reviews/pr-168/FINAL_REVIEW.md) maps the original A deliverables to live checks and the 26-criterion matrix. The [case-level baseline and gas audit](../../reviews/pr-168/CASE_LEVEL_AUDIT.md) records the latest implementation/vector closures. The [earlier adverse-input recheck](../../reviews/pr-168/AUDIT.md), [initial completion report](../../reviews/pr-167/ACCEPTANCE_COMPLETION.md), [original audit](../../reviews/pr-167/GROUP_A_ACCEPTANCE_AUDIT.md), [merge audit](master-integration-167.md) and [77-task reuse inventory](../../reviews/pr-167/DUPLICATION_REVIEW.md) remain dated history. Existing master canonical Frames, Bogota APIs, FCU, real snap and test infrastructure are reused.

Run `python3 tools/hegota/check.py`, both golden checkers and `python3 -m unittest discover -s tools/hegota -p 'test_*.py'` from the root. From `pkg`, run `env -u GOROOT go run ./cmd/eth2030-baseline`, build/vet and Go tests. Full/race skip inventories are distinct from the mandatory zero-skip baseline. The current gate validates all seven tasks' individual criteria, executed test events, recomputed source/artifact hashes, nonempty successful full/race runs and consistent reference identity; review/blocked statuses can be represented truthfully and fail the completion gate.

Completion is limited to A. Full selected-spec qualification, conditional extension production integration, independent CL/network agreement, containers/RC and long-duration/resource acceptance remain later tasks. `hegota-release` stays inactive; open composition decisions do not become implementation defaults.
