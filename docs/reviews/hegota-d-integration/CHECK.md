# Group D development checklist

Baseline: merged PR #170, master `81e63f586642a1406d56db9fa7cbf00941feb7bf`.
Branch: `codex/hegota-d-integration`. References and independent A/B/C oracles
are read-only. Every implementation must identify its normative revision and
consumers; open specification questions must not become invented consensus.

- [ ] Re-read latest EIP-8504/#12672, 8288, 8298, 8151, 7906 and consumed dependencies, including histories/discussions and unresolved composition decisions.
- [ ] Inventory existing native implementations and tests, separating reusable primitives, incomplete prototypes and incompatible or duplicate paths.
- [ ] HGT-030: pin scheme/dependency/witness/hash boundaries, selected profile and aggregation/admission ADR.
- [ ] HGT-031: implement all determinate proof-frame/dependency encoding, zero substitution, ordering, deduplication, binding and independent vectors.
- [ ] HGT-032: implement the pinned real verifier and independently checked vectors, exact parameters, bounded CPU/gas and explicit provisional activation.
- [ ] HGT-033: implement all normatively specified witness availability, validation, caching, replacement/eviction and transport; gate undefined wire formats.
- [ ] HGT-034: implement all normatively specified block dependency union/commitment/import checks with native invalid-block controls; distinguish direct verification from recursive aggregation.
- [ ] HGT-035: implement SETCODEFROM activation, execution/initcode/reentrancy/rollback, code-hash persistence, migration and exact BAL changes.
- [ ] HGT-036: reuse existing EIP-8151 protection, add migration/public-pool dependencies and compatibility/admission report without disabling safety checks.
- [ ] HGT-037: implement POST_TX semantics and complete assertion/paymaster/body rollback boundaries, including token, retry and batch consumers.
- [ ] Reconcile every criterion with source, production code, positive/negative consumers and explicit remaining upstream gates.
- [ ] Run required builds/vet/tests, full and affected race suites, A/B/C regression checks and source-bound D evidence.
- [ ] Inspect staged changes for conflict markers, duplicate work and baseline drift; commit/push, publish an English PR and verify exact-head hosted results.
