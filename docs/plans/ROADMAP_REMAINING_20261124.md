# Roadmap remaining work - November 24, 2026

## Reviewed state

PR 148 is merged into master b4b88ed0e410e5b735a9c6bf0f033180c89dc8c0. The frozen LIFO / signed Gloas node-development scope remains complete: 33/33 original tasks and 5/5 confirmed follow-up repairs. Its pkg tree remains equal to the measured cumulative code; this page refresh is not a new runtime qualification. The old Glamsterdam card incorrectly said integrated CL CI was open; that wording is corrected.

48h/larger-scale admission and upstream Lodestar approval remain excluded from development progress at the user's direction. Original 04 attribution remains unconfirmed. Public activation is a separate network rollout decision, using existing schedule/genesis/override support. No public fork parameters or new protocol scope are selected here.

## Current specification repair delivery

Five newly confirmed Glamsterdam problem groups are implemented in independent [EL 150](https://github.com/alt-research/eth2030/pull/150), [EL 151](https://github.com/alt-research/eth2030/pull/151) and [Lighthouse 16](https://github.com/jiayaoqijia/lighthouse/pull/16) reviews. EL targets master; the companion CL targets eth2030. The roadmap now links these reviews and the [current specification audit](https://github.com/alt-research/eth2030/pull/152), separately from the earlier merged baseline. Current CI/merge outcomes are authoritative on those PRs; this does not relabel earlier binaries or claim a new long-duration result.

## Remaining workstreams

These seven workstreams are grouped next-scope work, not seven newly discovered frozen-target bugs. Integration can be partly implemented while production-consumer qualification is still open. The static page expands each workstream to show evidence and completion criteria.

### 1. Independent Frame / FOCIL agreement (qualification)

Native paired signed Frame and inclusion-list controls are delivered. The fresh CL used a synchronized EL.

**Remaining:** Qualify an independent client using the same seven-field Frame/Bogota contract, plus alternate-peer recovery and redundant BN / remote-signer paths.

**Done when:** Independent state and receipt roots, signed verdicts, late join and restart agree under the adopted contract.

Sources: [docs/verification/frame-8141-20261118-paymaster-history-final.json](../../docs/verification/frame-8141-20261118-paymaster-history-final.json), [docs/next-stage-work-plan.md](../../docs/next-stage-work-plan.md).

### 2. Dora production consumers (integration)

Decoder and inclusion-list display work do not establish the full production consumer migration.

**Remaining:** Finish and qualify epoch, receipt and statistics consumers, schema migration, missed-event recovery, pruning and persisted rebuild.

**Done when:** Real consumer reads survive competing roots, Req/Resp recovery and restart before retained-view adapters are retired.

Sources: [docs/next-stage-work-plan.md](../../docs/next-stage-work-plan.md).

### 3. Workload and paymaster service cutover (integration)

The pinned local paymaster ABI and paired sponsored/blob controls are delivered.

**Remaining:** Qualify ordinary, delegated and sponsored services against the selected protocol and actual deployment consumers.

**Done when:** Real service settlement, fees, receipts, retry and restart agree at declared source pins.

Sources: [docs/verification/frame-8141-20261118-paymaster-history-final.json](../../docs/verification/frame-8141-20261118-paymaster-history-final.json), [docs/next-stage-work-plan.md](../../docs/next-stage-work-plan.md).

### 4. Canonical Ethereum STF guest and public proof consumers (research)

EL #156 supplies real pinned RV32IM execution receipts, complete machine-state/statement binding and native asynchronous proof RPC. The fixed-output STF adapter fixture is not an Ethereum STF implementation.

**Remaining:** Implement an executable canonical Ethereum STF guest with actual block/prestate witnesses; qualify independent state/receipt roots and mandatory public deciding consumers under an adopted protocol.

**Done when:** Real Ethereum block transitions execute inside the proved guest, and independent positive and invalid witness/root/program/receipt controls agree at the deciding block consumer.

Sources: [the real-proof review](https://github.com/alt-research/eth2030/pull/156), [extension validation](../reviews/glamsterdam-extensions-20261124/STATUS.md).

### 5. Public binary-state contract and recovery (research)

Private binary overlay/conversion controls do not define a public compatible layout.

**Remaining:** Agree layout, addressing and witness rules; qualify authenticated recovery and resource bounds.

**Done when:** Independent clients agree on roots, witnesses and persisted recovery under the chosen public contract.

Sources: [docs/ROADMAP.md](../../docs/ROADMAP.md), [docs/next-stage-work-plan.md](../../docs/next-stage-work-plan.md).

### 6. PQ validator keys, transactions and sampling (research)

PQ primitives and private provider/transaction controls are narrower than a public protocol.

**Remaining:** Define validator-key lifecycle, public signing/sampling contracts and independent implementation vectors.

**Done when:** Key registration, rotation, revocation, replay rejection and live consumers agree across independent implementations.

Sources: [docs/ROADMAP.md](../../docs/ROADMAP.md), [pkg/crypto/pqc/pq_tx_signer.go](../../pkg/crypto/pqc/pq_tx_signer.go).

### 7. Proof, native-rollup and finality consumers (research)

Native rollup transition/withdrawal helpers use simulated hash conditions; these are not real rollup proof acceptance.

**Remaining:** Adopt compatible execution, proof and consensus contracts; connect independent verifiers and actual deciding consumers.

**Done when:** State transitions, withdrawals, fees and finality pass independent end-to-end positive and rejection controls.

Sources: [pkg/rollup/native_rollup.go](../../pkg/rollup/native_rollup.go), [docs/ROADMAP.md](../../docs/ROADMAP.md).

## Historical label inventory

Across the six mapped cards there are 51 entries: 20 labelled Complete and 31 non-complete. The current distribution is Glamsterdam 1, Hegotá 7, I* 7, J* 7, K* 2 and L* 7. These counts are unrelated to the 33 completed frozen-development task IDs; no whole-client completion percentage is derived from either.

Glamsterdam is 16/17 within stated development scopes. The canonical quest remains Substantial because an executable Ethereum STF guest is unfinished. ePBS completion covers the adopted signed Gloas contract, and FCR uses the normative signed CL path and standard native Engine consumer. Separate PR integration and finite runtime receipts do not imply public activation or unrestricted acceptance.

Four planned-only entries list no implementation or tests: tapered issuance burn, decentralized state, ethp2p unification and blob-transaction removal. These are identified as planned scope, not as already implemented features. Every other historical non-complete entry remains visible in the page's per-phase expandable inventory; a fresh source/consumer audit is required before promoting or recasting those labels.

## Review depth and limits

The earlier review inspected fast-confirmation constructor/input consumers, canonical.VerifyExecution versus CanonicalExecutor.VerifyGuestProof, and native-rollup transition/withdrawal hash simulations. A retained [historical executable observation](../reviews/roadmap-remaining-20261124/CANONICAL_OBSERVATION.json) confirms that the standalone VerifyExecution helper accepts 256 all-zero proof bytes with Success=true; the source is retained as canonical-observation.go and can be run from pkg. The structural legacy helper is not conflated with the separate executor verification method. No new failure of the qualified native Glamsterdam import/signature path is established by these research findings.

Dora sibling revision 6f702f24e8445d9e4b764d5bf141c604b1c99d47 was inspected read-only: writedb.go still has Gloas/Heze EL-fallback handling. That corroborates a migration/consumer-qualification boundary; it does not claim no decoder or no indexing exists. Frame/paymaster receipts retain their November 18 synchronized-EL and source-specific boundaries. No new independent Frame or production service acceptance was executed here.

## Recommended sequence

1. Adopt a compatible independent Frame/Bogotá target and close independent-client/alternate-peer/restart qualification.
2. Complete Dora consumer/schema/rebuild migration and workload/paymaster service cutover in separate branches/PRs.
3. Review EL #155/#156 master integration and the authorized Lighthouse #17 eth2030 integration. Build a real Ethereum STF guest as a separate major feature; select independent binary-state, PQ and rollup/finality contracts before public consumers. Do not promote structural/simulated verification to protocol authority.
4. Revisit planned-only scope only when a concrete contract is adopted; retain the two deferred acceptance items outside the development denominator.

[Roadmap](../ROADMAP.md) · [Current frozen accounting](../reviews/development-scope-20261124/CURRENT_STATE.json) · [Review checklist](../reviews/roadmap-remaining-20261124/CHECK.md).

## Extension delivery update

The November 24 extension review supersedes the earlier Fast confirmation and canonical verifier observations: EL #155 implements vote isolation and actual signed CL/native consumers; Lighthouse #17 repairs durable FCR publication; EL #156 supplies real pinned execution receipts and native proof RPC. The remaining broad canonical quest is an executable Ethereum STF guest and adopted public proof consumers, not the old hash/length verifier defect. Current mapped progress is 16/17; see [the extension record](../reviews/glamsterdam-extensions-20261124/STATUS.md) for source-specific validation and integration state.
