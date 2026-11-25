# Hegotá Group A delivery and integration

The seven A tasks (HGT-001, 004, 005, 006, 007, 008 and 009) establish specification sources, runtime identity, activation boundaries, prerequisite gaps, historical differential tests, semantic oracles and the 72-proposal admission register. They do not establish complete Hegotá conformance.

The [PR #167 integration audit](master-integration-167.md) is authoritative for the merge with upstream `98d43177`. The earlier audit documents retain their original findings as historical evidence. Master now implements canonical Frames, Bogotá APIs, native forkchoice selection and real snap persistence; earlier statements that these paths are absent must not be applied to the integrated tree.

| Task | Artifacts |
| --- | --- |
| HGT-001 | [Source profiles](specification-profiles.md), [source lock](../../../specs/hegota/source-lock.json), [composition decisions](../../../specs/hegota/profiles.json): 14 revisions, 108 file hashes, six isolated profiles. |
| HGT-004 | [Runtime audit](runtime-and-activation.md), current binary/source identities and actual native dispatcher tests. |
| HGT-005 | Genesis alias conflict tests; explicit legacy processor/VM/pool scope; canonical Frame isolation and real FCU/reorg regressions. |
| HGT-006 | [Historical baseline](baseline-and-gaps.md), [conformance register](../../../specs/hegota/conformance.json), 12 required native/direct-Geth cases with no skips. Historical repair findings are requalification items unless current evidence closes them. |
| HGT-007 | [Semantic ADRs](semantic-adrs.md), independent codec/hash and arithmetic/domain oracles. Nine semantic seeds remain unexecuted latest-spec oracles. |
| HGT-008 | [Validation audit](validation-audit.md), authenticated BLS and negative acceptance tests. Upstream retired state/builder stubs stay retired. |
| HGT-009 | [Admission register](../../../specs/hegota/admission.json): 72 proposals, pinned sources, discussions, decisions and impact histories. |

Run `python3 tools/hegota/check.py`, both golden checkers and `python3 -m unittest discover -s tools/hegota -p 'test_*.py'` from the repository root. From `pkg`, run `env -u GOROOT go run ./cmd/eth2030-baseline`, `go build ./...`, `go vet ./...` and the full Go suite. [Current verification evidence](../../../evidence/hegota-a/verification.json) separates full-suite skips from the mandatory zero-skip baseline. [Historical evidence](../../../evidence/hegota-a/history/6a17479/verification.json) preserves the pre-merge audit.
