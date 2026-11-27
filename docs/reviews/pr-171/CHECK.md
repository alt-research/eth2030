# Group D development checklist

Baseline: merged PR #170, master `81e63f586642a1406d56db9fa7cbf00941feb7bf`.
Branch: `codex/hegota-d-integration`. References and independent A/B/C oracles
are read-only. Every implementation must identify its normative revision and
consumers; open specification questions must not become invented consensus.

- [x] Re-read latest EIP-8504/#12672, 8288, 8298, 8151, 7906 and consumed dependencies, including histories/discussions and unresolved composition decisions.
- [x] Inventory existing native implementations and tests, separating reusable primitives, incomplete prototypes and incompatible or duplicate paths.
- [x] HGT-030: pin scheme/dependency/witness/hash boundaries, selected profile and aggregation/admission ADR.
- [x] HGT-031: implement all determinate proof-frame/dependency encoding, zero substitution, ordering, deduplication, binding and independent vectors.
- [x] HGT-032: implement the pinned real verifier and independently checked vectors, exact parameters, bounded CPU/gas and explicit provisional activation.
- [x] HGT-033: implement all normatively specified witness availability, validation, caching, replacement/eviction and transport; gate undefined wire formats.
- [x] HGT-034: implement all normatively specified block dependency union/commitment/import checks with native invalid-block controls; distinguish direct verification from recursive aggregation.
- [x] HGT-035: implement SETCODEFROM activation, execution/initcode/reentrancy/rollback, code-hash persistence, migration and exact BAL changes.
- [x] HGT-036: reuse existing EIP-8151 protection, add migration/public-pool dependencies and compatibility/admission report without disabling safety checks.
- [x] HGT-037: implement ASSERT semantics and complete assertion/paymaster/body rollback boundaries, including token, retry and batch consumers.
- [x] Reconcile every criterion with source, production code, positive/negative consumers and explicit remaining upstream gates.
- [x] Run required builds/vet/tests, full and affected race suites, A/B/C regression checks and source-bound D evidence.
- [x] Inspect staged changes for conflict markers, duplicate work and baseline drift; preserve root trackers, refs and independent A/B/C oracles.
- [x] Commit/push and publish [English PR #171](https://github.com/alt-research/eth2030/pull/171).
- [x] Verify the hosted CI workflow and independent snap2 interoperability on exact source `335466499783ceed25de596e2ba0fb8b3e5fb67d`; retain the PR suite as authority for final-head reruns and actual proof acceptance.

## Latest-EIP completeness re-audit

- [x] Recheck current EIP heads, every normative clause and discussion changes.
- [x] Audit witness transport, blob composition, payload/import/restart and all attachment-copy paths.
- [x] Audit adoption origins, nested creation/destruction, BAL consumers and migration/dependency invalidation.
- [x] Audit ASSERT trace completeness, cache lifetime, subtree/static boundaries and whole-body settlement.
- [x] Add native regression consumers for every reproduced gap and reconcile the complete criterion register.
- [x] Run current-source builds, vet, full/race and A/B/C/D evidence checks; preserve independent oracles.
- [x] Inspect conflicts/duplication and prepare committed source-bound evidence for the existing English PR; record final-head hosted checks in the live PR after this offline freeze.

Final local qualification: source `d7fa00f0`; A/B/C/D mandatory consumers
62/79/102/55, full 24,176 and affected race 11,535 named passes; nine compiled
semantic-removal controls fail as intended. Exact-head hosted publication/results
are authoritative in [PR #171](https://github.com/alt-research/eth2030/pull/171),
not inferred from predecessor workflow runs.
