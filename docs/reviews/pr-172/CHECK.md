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

- [x] Reconfirm current source revisions/discussions and every Group E clause.
- [x] Verify the EIP-7805 last-transaction state boundary against withdrawals and post-execution system calls.
- [x] Reproduce and fix any import/build disagreement with independent native regressions.
- [x] Recheck request, exact-byte, frame gas, RLP and authenticated cache boundaries for additional missing rules.
- [x] Refresh source-bound evidence and website progress; run required build/full/race/control checks.
- [x] Commit owned source and qualification artifacts; publish to the existing English review.
- Publication and exact-head hosted CI are tracked in the English PR description; the frozen local snapshot does not pre-claim hosted outcomes.

## Pre-execution validity and complete-consumer audit — November 27, 2026

- [x] Reconfirm all current normative sources/discussions and the nine-task register.
- [x] Audit pre-execution Frame nonce, aggregate fee and gas checks before unavailable-witness handling, including build/import agreement.
- [x] Review all remaining Heze wire/store/request, Engine, builder, omission and CL decision boundaries for missing defined rules.
- [x] Reproduce confirmed gaps before correction; share existing execution rules and add positive/negative native consumers.
- [x] Refresh source-bound A–E evidence and website progress, preserving expected oracles and read-only refs.
- [x] Complete local build/vet/full/race/control/website qualification and commit owned artifacts.
- Publication and exact-head hosted results are tracked in the English PR description; this local snapshot does not pre-claim hosted outcomes.

## Independent completion pass — November 27, 2026

- [x] Recheck upstream heads, source content and discussion updates against the published head.
- [x] Review all nine tasks across authenticated store/requests, wire, builds, omission evaluation and fork choice.
- [x] Reproduce any new defined-rule gaps before fixing; retain existing expected oracles and explicit specification gates.
- [x] Run proportionate source/native/build checks, update scoped status and publish owned changes with English PR evidence.

## Remaining Group E delivery - November 28, 2026

- [x] Reconcile latest upstream changes with the remaining activation and integration prerequisites.
- [x] Identify and pin the independent Heze CL source and executable network/test path.
- [x] Implement and execute the selected specification-defined two-pair integration and bounded network resource controls in hosted CI.
- [ ] Validate actual consumers, record unavailable prerequisites precisely, and update progress and English PR evidence.
