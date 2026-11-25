## Sequential master/CI repair (2026-11-19)

- [x] Inspect exact original CI failures and existing blob/snap dependency boundaries.
- [x] Merge current master, preserving initcode validation before BAL tracking.
- [x] Reconcile repaired existing dependencies 80 and 81 without importing Draft 83 or later features.
- [x] Run full default tests, integration, relevant race/vet, formatting and build; prepare publication and require fresh exact-head CI before completion.

# PR 82 tracking cleanup

- [x] Preserve original tracking diff and changed sections from `08534b6dd5ddbb478aebf55c6ee0ab6664cb9fad`.
- [x] Synchronize shared trackers and merge current master without dropping feature behavior.
- [x] Verify merged source, local build and affected tests.
- [x] Prepare the existing PR branch for fast-forward publication; require live conflict-state verification before reporting completion.

## Preserved feature tracking

## eth version 72 sparse blob relay (2026-11-19)

- [x] Pin full EIP-8070/7594 and devp2p schemas plus Geth merged flat request-id fix.
- [x] Add actual negotiated version 72 announcements, elided pooled metadata and correlated cell exchange.
- [x] Stage signed sparse metadata/cells under resource bounds; authenticate every cell and reconstruct before ordinary pool promotion.
- [x] Wire provider/sampler selection, two-provider threshold, custody plus random extra, custody expansion and eager building.
- [x] Consume actual FCU custody updates and implement getBlobsV4 cell/proof readback.
- [x] Verify encrypted SDK/Node consumers, invalid/unrequested data, recovery and independent reconstruction; build/test/mutate and prepare a separate PR.

- [x] Publish the independent feature review as PR 82; composed qualification is tracked separately.

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

