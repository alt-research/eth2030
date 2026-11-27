# Group E development checklist

Baseline: merged PR #171 at `cc31afffaa12919bbc08dc854c03af1e1928dc3c`.
Branch: `codex/hegota-e-integration`. Read-only refs and independent A/B/C/D
expected oracles are preserved. Completion requires production consumers and
current-source evidence; unresolved specifications do not become consensus.

- [x] Re-read current EIP-7805/8369, Heze consensus/Engine specifications, dependencies, histories and discussions; reconcile all nine E tasks with current normative clauses.
- [x] Inventory existing FOCIL primitives, prototypes, native/alternate backend wiring, validation, persistence, build/import and CL interfaces; avoid duplicate implementations.
- [x] HGT-040: record exact selected scope, timing, branch identity, execution-versus-satisfaction and Engine responsibilities.
- [x] HGT-041: implement normative IL types, storage, exact bytes, duplicate/equivocation/late/reorg/pruning behavior.
- [x] HGT-042: implement selected Engine methods, versions, capabilities, schemas and unsupported-version rejection.
- [x] HGT-043: implement builder updates/rebuild/cancellation/deadlines and sequential transaction validity without deadlocks.
- [x] HGT-044: implement exact Profile 1 presence/validity/gas/byte rules, including EIP-7934 boundaries and invalid transactions.
- [x] HGT-045: connect actual native node storage/build/import/results and document alternate backend behavior.
- [x] HGT-046 (reference predicates and enforced D04 gate; activation is NOT complete): implement all defined AA-VOPS requirements and only activate a complete normative profile; preserve explicit D04 gates for undefined rules.
- [x] HGT-047 (local native/CL adapter; external HGT-070/080 qualification remains open): qualify gossip-to-fork-choice consumers and all normative omission/full/invalid/late/branch exceptions.
- [x] HGT-048 (bounded local measurements; network tail latency and omitted witnesses remain unqualified): measure selected PQ IL capacity/propagation/verification budgets without inventing protocol limits or proof propagation.
- [x] Add positive/negative native unit/integration consumers and criterion-level immutable-source coverage for every task.
- [x] Run local builds, vet, appropriate full/race tests, A/B/C/D regressions and source-bound E qualification.
- [x] Update website/GitHub Pages roadmap and progress to match demonstrated results and actual release gates; build/check exports and UI.
- [x] Inspect base drift, conflicts and duplicated work; commit/push and publish English PR #172.
- Hosted exact-head results are tracked in [PR #172](https://github.com/alt-research/eth2030/pull/172) check runs and its English description.

## Completion re-audit — November 27, 2026

- [x] Confirm latest EIP/Heze/Bogotá source blobs and discussions against the frozen scope.
- [x] Recheck every HGT-040–048 clause against actual consumers; distinguish implementation from unresolved activation/external prerequisites.
- [x] Exercise untested validity, byte/gas, request, retention, fork-choice, replacement/cancellation and persistence boundaries.
- [x] Fix all reproducible specification deviations and add independent negative/positive regressions.
- [x] Refresh source-bound A–E evidence, website progress and scope limitations; preserve independent expected oracles.
- [x] Run appropriate local build/vet/full/race, source-bound, compiled-removal and website checks.
- Publication and exact-head hosted CI are tracked in the English PR #172 description; frozen local artifacts do not pre-claim hosted outcomes.

## End-of-transaction state audit — November 27, 2026

- [ ] Reconfirm current source revisions/discussions and every Group E clause.
- [ ] Verify the EIP-7805 last-transaction state boundary against withdrawals and post-execution system calls.
- [ ] Reproduce and fix any import/build disagreement with independent native regressions.
- [ ] Recheck request, exact-byte, frame gas, RLP and authenticated cache boundaries for additional missing rules.
- [ ] Refresh source-bound evidence and website progress; run required build/full/race/control checks.
- [ ] Commit/push owned changes to the English review and verify current-head hosted results.
