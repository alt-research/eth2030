# PR #167 overlap review status

## Group A gaps completed locally; hosted publication tracked on follow-up PR

All 25 acceptance criteria across the seven A tasks now have current artifacts and executable evidence. The head-number defect is fixed with nonzero-height/reorg controls; all nine semantic seeds have concrete consumers; reference, runtime, safety and prerequisite records are reconciled. [Completion report](ACCEPTANCE_COMPLETION.md) documents every closure and the unchanged later qualification gates.

Fresh build/vet, full Go (23,529 pass events, zero failures), focused race (3,502 pass events, zero failures), 33 Python regressions and 12/12 zero-skip historical differential cases passed. Both negative Go overlays reject the old defect or incorrect semantic expectations. The source-bound completion gate verifies current hashes and individually executed tests. Commit/push and new-head hosted results are tracked on the English follow-up PR.

## Group A acceptance audit: not fully complete

PR #167 merged as `89b3b1f2d12352777903de9803f3c93285bdcdf1`. Its source tree equals qualified head `a86473ea6c04fa33a2fd488c33506380d2b96a8f`; all 15 checks passed on that PR head. A fresh acceptance audit nevertheless reproduced a canonical Frame pool/head-context defect and found incomplete semantic goldens, inconsistent reference-version metadata, and historical runtime tables still referenced as current task artifacts. The seven `verified` flags are therefore not sufficient evidence of completion.

The [seven-task audit](GROUP_A_ACCEPTANCE_AUDIT.md) records the findings, reproduction, verified scope and closure criteria. Fresh local build, focused Go checks, 12 Python regressions, source checks and all 12 mandatory native/direct-Geth baseline cases passed. The counterexample fails despite those green checks. This audit documents the remaining work; it does not claim the findings are repaired or that B–J implementation has been completed.

## Local cleanup verified; publication and hosted results tracked on PR #167

Compared canonical master `98d4317784b755ab75eefbec1ae7d8dca813ae96` with integrated head `dca42829c9bc059244d10efc0a835b220944a6aa`. Master was an ancestor, no upstream commits were missing, and all 15 hosted checks passed on that head. Review approval, not merge conflicts or failed CI, blocked merge.

The [complete duplicate-work review](DUPLICATION_REVIEW.md) covers all 117 initial diff paths and reconciles all 77 planning task IDs. EIPs 8141, 8037 and 8038 are byte-identical at the declared master implementation and research pins. Existing implementations become reuse/qualification work, rather than parallel development from an obsolete baseline.

Removed the duplicate pool chain/time fields and test-only reset API; startup, simulator and regression tests use the existing production setters/full-header reset. Removed the unused two-state IL wrapper and moved the legacy PQ forgery digest into test-only code. Restored master's already-correct compatibility algorithm while preserving the distinct experimental legacy guard. Kept canonical/legacy format boundaries and unique fail-closed validation/source/differential checks.

Fresh validation: build/vet, full suite (23,520 named pass events; zero failures; 124 inventoried skips across 240 completed packages), focused race (2,615 pass events; zero failures; six skips), 12/12 mandatory baseline cases with zero skips, source locks/goldens and 12 Python checker regressions. [Current evidence](../../../evidence/hegota-a/verification.json) records the exact source fingerprint; previous broader runs remain pinned to `dca42829`. Publication and new-head hosted results are recorded in the English [PR #167 description](https://github.com/alt-research/eth2030/pull/167).
