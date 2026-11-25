## Completion audit and remaining-work inventory (2026-11-19)

- [x] Freeze the live PR head, Draft/review state and current master; preserve unrelated work.
- [x] Trace pinned gas requirements through VM, transaction/block accounting, pool, Engine, startup/recovery and tests.
- [x] Distinguish implementation defects, missing qualification, separate method-consumer PRs and governance/release gates.
- [x] Write an independent TODO document with evidence, scope and acceptance conditions; update status and commit documentation only.

## Sequential Draft gas merge/CI repair (2026-11-19)

- [x] Preserve Draft status and the user-selected merged LIFO specification baseline; do not adopt open alternatives.
- [x] Merge current master while retaining canonical EIP-7778 activation and pre-mutation initcode validation.
- [x] Reconcile the repaired existing system-transition foundation (84) without changing gas policy.
- [x] Run full tests, integration, gas/state/Node race, formatting/vet/build; prepare publication and require exact-head CI before completion.

# PR 83 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `bd0f33577c0d343abb7057894d9702a27d7ff157`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [x] Verify merged source, local build and affected tests.
- [x] Prepare the existing PR branch for fast-forward publication; require live conflict-state verification before reporting completion.

## Preserved feature tracking

## Current gas signed and system consumer follow up (2026-11-19)

- [x] Qualify ten signed authorization boundaries including late pre-execution failure after successful authorization writes.
- [x] Observe LIFO refill/ancestor repayment through GAS-dependent on-chain storage.
- [x] Preserve Prague pricing controls; migrate current-fork budgets and exact fees/header expectations.
- [x] Check actual system state margin at 8/16/17 new slots and execution/state block capacity independently.
- [x] Check both actual HTTP blocks' sender/miner settlement, advertised block value and SDK next-base-fee formula.
- [x] Detect ten compiling gas guard mutations; retain the initial escaped rollback mutation and repaired fixture.
- [x] Repeat final full/tagged/affected race/vet/build; archive the clean signed checkpoint separately. RPC/application/final release gates remain separate.

## Canonical execution system transitions (2026-11-19)

- [x] Read complete pinned EIPs 2935/4788 and the selected 8037 system-call margin; compare canonical SDK assets.
- [x] Reproduce wrong history address/window and missing/custom-runtime behavior through actual block processing.
- [x] Share pre-block EVM system-call execution between builder/importer; preserve caller exclusion, ordering, isolation, actual BAL changes and no block/user fee debit.
- [x] Supply explicit canonical assets before owned genesis roots without mutating supplied network state or fork-time injection.
- [x] Qualify independent runtime/storage/root consumers, wrap/missing-slot/query bounds and native Node build/import/restart; preserve 8282/6110/7002/7251 coverage.
- [x] Run full/tagged/affected race/vet/build and sensitive mutations; prepare a separate feature PR.

- [ ] Publish the separate review and record composed qualification.

## eth version 72 sparse blob relay (2026-11-19)

- [x] Pin full EIP-8070/7594 and devp2p schemas plus Geth merged flat request-id fix.
- [x] Add actual negotiated version 72 announcements, elided pooled metadata and correlated cell exchange.
- [x] Stage signed sparse metadata/cells under resource bounds; authenticate every cell and reconstruct before ordinary pool promotion.
- [x] Wire provider/sampler selection, two-provider threshold, custody plus random extra, custody expansion and eager building.
- [x] Consume actual FCU custody updates and implement getBlobsV4 cell/proof readback.
- [x] Verify encrypted SDK/Node consumers, invalid/unrequested data, recovery and independent reconstruction; build/test/mutate and prepare a separate PR.

- [x] Publish the independent feature review as PR 82; composed qualification is tracked separately.

## snap version 2 unchanged storage account proof binding (2026-11-19)

- [x] Reproduce a moving pivot with partial storage whose account body/storage root is unchanged.
- [x] Authenticate that account against the fresh pivot before using its storage root.
- [x] Reject a damaged proof for that unchanged account without checkpoint/adoption.
- [x] Qualify all encrypted Node scenarios, affected race/vet/build and update PR 81.

## snap/2 BAL state catch-up (2026-11-18)

- [x] Pin EIP-8189/7928 and trace real snap peer, state scheduler/writer and CL target adoption.
- [x] Negotiate snap/2 separately, retain unchanged range/code packets and reject removed trie-node messages.
- [x] Serve and authenticate ordered raw BAL responses with missing entries, budgets and cancellation.
- [x] Rebase downloaded state and outstanding ranges through verified BALs, including moving pivots and reorg handling.
- [x] Qualify actual Node sync/restart and independent wire/root consumers; run builds/tests and publish the separate PR.

- [x] Read EIP-7975/8159 and current devp2p status/message schemas; inspect the actual node registration.
- [x] Add independent canonical BAL-wire, versioned handshake and correlated tamper tests.
- [x] Register negotiated versions, implement status/range and version-aware receipt serving.
- [x] Correlate and validate received BALs before restoring verified historical data.
- [x] Verify affected network/chain/node/consumer paths and race/build; prepare the separate feature review.


## EIP-8282 builder execution requests (2026-11-18)

- [x] Read the complete pinned EIP, canonical bytecode assets and selected CL processing.
- [x] Add exact runtime/constructor assets and request types/SSZ encoding with provenance.
- [x] Wire mandatory Amsterdam system queues, rollback and commitment validation.
- [x] Supply explicit pre-fork deployment/genesis fixtures without fork-time code injection.
- [x] Qualify fee/queue/endian/cap/failure and actual Engine/CL consumer paths.
- [x] Run affected builds/tests and publish a dedicated feature PR.

## Complete ordinary blob sidecar validation (2026-11-18)

- [x] Read pinned EIP-4844/7594 networking and proof requirements.
- [x] Reproduce invalid ordinary-cell-proof admission through real node HTTP.
- [x] Validate complete sidecar binding/proofs and preserve Frame regression behavior.
- [x] Require full data for pool-based builders and return genuine legacy blob proofs.
- [x] Qualify actual wire/admission/Engine consumers, affected checks/build and regression mutations.
- [x] Publish the isolated blob-validation PR (PR 76).

## Fork-aware blob capacity, fees and configuration (2026-11-18)

- [x] Base on PR 79 configuration authority and compose PR 76 full-data/proof consumers; retain the 21-blob race timeout as evidence until the current wrapper is qualified.

- [x] Read EIPs 7840/7892/7691/7918 and trace actual builder/import/pool/VM/RPC consumers.
- [x] Reproduce hardcoded capacity/fee and ignored configuration defects against independent reference inputs.
- [x] Wire one selected schedule through canonical block admission, fee calculation, payloads and RPC.
- [x] Qualify actual signed blob traffic, fork transitions, invalid blocks and configuration/restart boundaries.
- [x] Run affected/full builds/tests and publish the isolated feature review.

## Persisted fork configuration and restart authority (2026-11-18)

- [x] Trace actual Node startup before history replay and reuse the existing persisted config key.
- [x] Detect incompatible active schedules and immutable parameter changes before state/metadata adoption.
- [x] Qualify future updates, either-side activation, missing/corrupt metadata and failed persistence.
- [x] Verify actual node restart preserves heads/state/configuration on rejection and does not mutate network globals.
- [x] Run affected/full tests, build and race/vet; publish a dedicated PR.

## Canonical fork schedule and Engine version boundaries (2026-11-18)

- [x] Read the current Engine Amsterdam/Osaka/Prague/Cancun rules and trace actual startup/HTTP consumers.
- [x] Reject incoherent canonical fork schedules before genesis/state adoption; preserve explicit private extensions.
- [x] Enforce canonical Engine version/required-field boundaries with exact activation and presence cases.
- [x] Qualify actual HTTP failures and valid forkchoice adoption independently from attribute rejection.
- [x] Run affected/full tests, build, race/vet and publish the isolated review.

## Mandatory builder-request feature composition (2026-11-18)

- [x] Allocate canonical contracts in blob/factory/slot owned fixtures before root construction.
- [x] Qualify all combined Go tests, tagged Core/EIP, vet and local build.

## Complete ordinary blob sidecar validation (2026-11-18)

- [x] Read pinned EIP-4844/7594 networking and proof requirements.
- [x] Reproduce invalid ordinary-cell-proof admission through real node HTTP.
- [x] Validate complete sidecar binding/proofs and preserve Frame regression behavior.
- [x] Require full data for pool-based builders and return genuine legacy blob proofs.
- [x] Qualify actual wire/admission/Engine consumers, affected checks/build and regression mutations.
- [ ] Publish the isolated blob-validation PR.

## Glamsterdam independent current-text gas corpus (2026-11-18)

- [x] Freeze signed transactions, complete pre-state, fork/config and reference-client version.
- [x] Execute identical inputs through native transaction-level execution and the unmodified Geth Amsterdam transition.
- [x] Record invalid versus included failure, gas dimensions, logs and exact final account/state results.
- [x] Verify pre-Amsterdam controls, archive candidate mismatches and build locally.
- [x] Publish the separate gas-corpus review (PR 75).

## Glamsterdam header and slot independent qualification (2026-11-18)

- [x] Compare native canonical header RLP/hash against go-ethereum v1.17.10.
- [x] Drive nondefault slots through HTTP FCU/build/import and observe SLOTNUM state and block RPC.
- [x] Match historical Engine BAL bytes to the independent decoded header commitment.
- [ ] Run affected tests and local build; submit the isolated qualification PR.

- [x] Revalidate merged canonical body wire/storage and read the current EIP-7928 retention requirement.
- [x] Reproduce the obsolete availability window through the real Engine body consumer.
- [x] Use the 14,299-epoch slot window, preserve conservative fallback and body availability.
- [x] Verify affected consumers, storage regressions, build and prepare a separate PR.


## Empty committed historical BAL recovery (2026-11-18)

- [x] Reproduce an empty BAL cache miss without hiding its execution body.
- [x] Regenerate only the header-committed empty list inside retention, for hash and range APIs.
- [x] Verify affected suites and local build, then update PR 66.

## Glamsterdam EIP-8024 current byte encoding (2026-11-18)

- [x] Pin the current EIP and reproduce published execution vectors against the native VM.
- [x] Correct legacy-bytecode immediate decoding and its invalid ranges.
- [x] Verify every valid depth/pair, stack boundaries, code-end immediates, jumps and native transactions.
- [ ] Run affected tests, local build and vet; publish an isolated implementation PR.

## Glamsterdam EIP-8246 transaction lifecycle (2026-11-18)

- [x] Pin EIP-8246 and read EIP-6780/161; reproduce missing native destruction cleanup.
- [x] Journal same-transaction creations with nested rollback and preserve balances through finalization.
- [x] Verify native CREATE/CREATE2, pre-funded targets, late credits, old contracts and reverted creation.
- [x] Qualify BAL final-state changes and persistence; run affected tests and local build.
- [x] Publish the isolated implementation branch and update program evidence (PR 72).

- [x] Read both local legacy and pinned current EIP-7708, including fee/withdrawal exclusions and 8246 dependency.
- [x] Add failing native fork-boundary, fees/self/zero-value, creation ordering and rollback regressions.
- [x] Correct consensus emission and canonical Amsterdam activation without a new toggle.
- [x] Verify affected consumers/roots and build; prepare a separate PR with the 8246 lifecycle gate explicit.


- [x] Read EIP-7975/8159 and current devp2p status/message schemas; inspect the actual node registration.
- [x] Add independent canonical BAL-wire, versioned handshake and correlated tamper tests.
- [x] Register negotiated versions, implement status/range and version-aware receipt serving.
- [x] Correlate and validate received BALs before restoring verified historical data.
- [x] Verify affected network/chain/node/consumer paths and race/build; prepare the separate feature review.


- [x] Inspect the production HTTP/backend route and merged execution-apis #950 semantics.
- [x] Add failing real-backend HTTP storage-discovery and non-mutation regressions.
- [x] Replace the empty placeholder with bounded discovery/convergence and gas readback.
- [x] Verify returned access items, reverted touches, creation, historical state and affected packages/build.
- [x] Prepare this major RPC feature for its separate PR.


- [x] Read old local and pinned current EIP-7997, including deployment and no-fork-check rules.
- [x] Add canonical address/runtime/nonce and actual CREATE2/value tests plus a no-fork-injection regression.
- [x] Replace the obsolete factory fixture and remove execution/builder fork-time state mutation.
- [x] Qualify affected paths and prepare the separate feature review.


- [x] Revalidate merged canonical body wire/storage and read the current EIP-7928 retention requirement.
- [x] Reproduce the obsolete availability window through the real Engine body consumer.
- [x] Use the 14,299-epoch slot window, preserve conservative fallback and body availability.
- [x] Verify affected consumers, storage regressions, build and prepare a separate PR.


- [x] Read local draft and pinned current EIP-7954; record the 64/128 KiB revision difference.
- [x] Add failing CREATE/CREATE2 and transaction admission boundary tests.
- [x] Correct fork-gated limits and reject oversized creation transactions before mutation.
- [x] Verify affected packages, build and prepare the separate feature review.


- [x] Reproduce nonzero-log consensus receipt roundtrip failure.
- [x] Recompute omitted wire blooms from decoded logs, not from an empty receipt bloom.
- [x] Verify codecs/consumers/build and prepare the scoped fix review.


- [x] Isolate the current-mainline program worktree and preserve the original dirty checkout.
- [ ] Revalidate the 33-task plan against origin/master and freeze specification/decision inputs.
- [ ] Publish the baseline/acceptance program in its own review branch.
- [ ] Deliver each major feature in a separate scoped branch and PR, retaining test evidence.
- [ ] Close native, consumer, independent-client, recovery and release gates with observed evidence.


## Glamsterdam combined feature qualification (2026-11-18)

- [x] Merge only the isolated Glamsterdam review branches into a local integration branch.
- [x] Resolve owned overlapping changes and retain every verification boundary.
- [x] Run the combined Go suite, tagged integration tests, build and affected race/vet checks.
- [x] Record exact branch heads, review links and remaining program gates without claiming release completion.

## Current merged state gas model (2026-11-19)

- [x] Re-read the complete chosen 3db38a3 EIPs, current-text gas corpus and actual production validation/runtime/settlement consumers; exclude older 49beda9 reference defaults and unmerged alternatives.
- [x] Reproduce the signed 19/22 candidate differences on the latest composition and classify each actual layer before editing.
- [ ] Implement coherent intrinsic execution/state gas, CPSB, access/write pricing, reservoir charges/refills, frame merge and halt/revert behavior.
- [ ] Wire authorization/pre-execution, creation/destruction/refund/floor, receipt/accounting and per-dimension block/build/basefee rules.
- [ ] Wire pool, estimate/call/access-list/trace consumers to their selected method contracts, preserving canonical Frame behavior.
- [ ] Qualify independent signed multi-transaction/application/recovery consumers with exact unsupported boundaries and sensitive mutations; run scoped/full/tagged/race/vet/build and publish a separate PR.

The merged LIFO rules are the explicit versioned development baseline. Final release selection remains open; no unmerged refill/allowance alternative is combined with it.

