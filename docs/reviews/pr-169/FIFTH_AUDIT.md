# Fifth independent Group B audit, 2026-11-26

This review checked the complete 12-task development-plan scope again against current normative text and actual execution/readback consumers. Upstream EIPs and consensus-specs master remain `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa` and `d2021d2719235f0df4726a1d330502cb15aa8ea3`; the project master baseline remains `960baab3103e0d34435433be0f8314f127521052`. Reference contents were not edited.

## Result: additional acceptance coverage

No additional implementation defect was reproduced in the selected native execution routes. The missing acceptance consumer was an end-to-end comparison across memory, persisted/reopened Trie and persisted/reopened Snap state. Earlier tests covered these rules individually and covered the controlled binary overlay, but did not explicitly require the same canonical Frame receipts and BAL bytes across all three primary native backends after a commit/reopen boundary.

`TestCanonicalFrame8141PersistentBackendJournals` closes that coverage gap using two signed canonical Frame transactions against each backend:

1. The first transaction includes individual failing frames and a failed atomic batch with a skipped target. Independent EIP-2929/8038 entry/slot expectations require execution gas `[100, 3004, 3004, 5104, 3004, 0, 5104, 204]` for its first eight frames. This distinguishes rollback of warm addresses/slots from successful warming.
2. A delegated child overwrites the caller's transient value and reverts. The parent logs 42, while a later Frame of the same account logs zero. EIP-1153 supplies nested-call ownership/rollback semantics; EIP-8141 additionally requires transient storage to clear between Frames. The ordinary DELEGATECALL shares the caller's transient storage context.
3. State is committed and Trie/Snap are opened again before the second transaction. That transaction clears and restores a pre-existing nonzero storage slot. Exact per-frame execution gas `[100, 15107, 207]` and zero state gas require the correct transaction-original value, cold/warm access, net-metered write cost and absence of a new-slot charge. The final slot equals its original value.
4. Both post-state roots and canonical receipt/BAL bytes agree across the three backends at each transaction boundary. The test checks backend read errors and the committed sender nonce after reopening.

The criterion register now requires this consumer under HGT-012, HGT-013, HGT-015 and HGT-016. EIP-1153 is explicitly hashed alongside the existing normative sources. Checker unit tests reject removal of the consumer from any of these four criteria and reject omission of the transient-storage normative source. The same 36 criteria now require 79 native test identities. No criterion or existing regression was removed.

This is a coverage correction, not a newly discovered production fix. The unchanged implementation passes the new scenario; this report does not claim that an original production-file substitution fails it. Previous implementation-defect negative controls remain available at their immutable earlier audit revisions. New expected-value checks are native cross-backend consumers, not an independent Frame-envelope implementation.

## Complete task review

| Task | Fifth audit result |
| --- | --- |
| HGT-002 | Rechecked canonical field/integer encoding, mode/flags, signing-length commitments and independent hash/wire vectors. No additional implementation defect reproduced. |
| HGT-003 | Rechecked approval scope/identity, default accounts, escrow/current nonce, nested effects and RETURN memory boundaries. Existing direct consumers remain mandatory. |
| HGT-010 | Rechecked all selectors, full-width operand/index handling, future structures/completed results, data padding/copy and signature-witness visibility. No additional implementation defect reproduced. |
| HGT-011 | Rechecked explicit execution/state/blob reservations, fees, floor/refund, state ownership and call/create/deposit charging. New reopened-backend cases require exact clear/restore gas and zero new-slot state gas. |
| HGT-012 | Added common-backend failure/warming, nested transient rollback and inter-Frame clearing. Existing state owners, approvals/fees/nonces/logs, batches and refills retain mandatory consumers. |
| HGT-013 | Added identical signed transaction processing across memory/Trie/Snap. Existing production builder/importer and independent expected-value comparisons remain required. |
| HGT-014 | Rechecked canonical payer/frame encoding, failed/skipped effects, logs/indices and disk-derived fields. New backend consumers compare canonical receipt bytes, while existing block tests verify receipt roots. |
| HGT-015 | Added byte-identical published BAL consumption across the three backends. Existing explicit sentry, prefunded reconstruction, final net change, system update and block commitment/size bounds remain required. |
| HGT-016 | Added commit/reopen between canonical Frame transactions on Trie/Snap, retaining exact original-storage gas and roots. Existing FileDB receipt/replay/reorg/history consumers remain mandatory. |
| HGT-017 | Rechecked the complete isolated keyed-nonce state/vector scope, canonical sets, pricing, exhaustion, atomic consumption, BAL non-warming, retry and CREATE introspection. D01 remains open. |
| HGT-018 | Rechecked exact ordinary deployment/runtime, tuple/domain/window/slot/gas rules, overwrite and canonical reorg. No public-pool extension is claimed. |
| HGT-019 | Rechecked protocol signature schemes, default accounts, low-s/length/signer/digest rules and arbitrary witness separation. No additional implementation defect reproduced. |

## Scope and qualification

The qualified primary native backends are memory, Trie and Snap; the existing controlled binary-overlay consumer remains required separately. The witness-only `StatelessStateDB` has no production constructor call site and explicitly rejects Dup. This review does not admit it as a native node backend or claim witness-execution conformance.

No production Go implementation, public wire/storage schema, activation switch or migration is changed in this round. Prior consensus corrections remain documented at their tested revisions. EIP-8250's conflicting fee example remains unresolved; HGT-017 completes only the development plan's allowed isolated state/vector scope. Public composed activation and Group C pool extensions remain separate.

Fresh build/vet, full/focused-race suites, strict historical baseline and A/B source-bound qualification are recorded in the evidence. Hosted checks must be matched to their tested commit on PR #169.

## Fresh local qualification

At test/register revision `35c95f6ea19889e8d3e5be7f4a05bfc08f058e3a`, all 240 packages completed with 23,765 named passes, zero failures and 124 explicit skips. Focused race completed all 36 packages with 8,005 named passes, zero failures and 74 explicit skips. All 79 required B tests and 62 required A tests passed. Build/vet, both CLI builds, the strict 12-case historical baseline and 75 Python tests in normal/optimized modes passed. Individual skips and source/log digests remain in the evidence.

The earlier account-reconstruction control was freshly rerun with current test sources: restoring only the wrapper from `2a092e43` makes its three earlier regression parents fail, with 15 failed test/subtest events. This rechecks the previous correction and is not a claim of a new fifth-audit implementation defect.
