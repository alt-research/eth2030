# Roadmap remaining work - November 24, 2026

## Reviewed state

PR 148 is merged into master b4b88ed0e410e5b735a9c6bf0f033180c89dc8c0. The frozen LIFO / signed Gloas node-development scope remains complete: 33/33 original tasks and 5/5 confirmed follow-up repairs. Its pkg tree remains equal to the measured cumulative code; this page refresh is not a new runtime qualification. The old Glamsterdam card incorrectly said integrated CL CI was open; that wording is corrected.

48h/larger-scale admission and upstream Lodestar approval remain excluded from development progress at the user's direction. Original 04 attribution remains unconfirmed. Public activation is a separate network rollout decision, using existing schedule/genesis/override support. No public fork parameters or new protocol scope are selected here.

## Remaining workstreams

These eight workstreams are grouped next-scope work, not eight newly discovered frozen-target bugs. Integration can be partly implemented while production-consumer qualification is still open. The static page expands each workstream to show evidence and completion criteria.

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

### 4. Broader fast-confirmation integration (research)

The standalone tracker counts supplied validator indices. A compatibility wrapper exists; no node/Engine integration caller was found in this review.

**Remaining:** Adopt a signed CL input and consumer contract, then connect real validator authority, conflicts and restart behavior.

**Done when:** Signed live inputs and independent deciding consumers establish the claimed confirmation semantics.

Sources: [pkg/consensus/fastconfirm/fast_confirm.go](../../pkg/consensus/fastconfirm/fast_confirm.go), [pkg/consensus/fastconfirm_compat.go](../../pkg/consensus/fastconfirm_compat.go).

### 5. Canonical guest and deciding proof verification (research)

Guest execution exists, but canonical.VerifyExecution checks success and proof length then returns true; it does not verify the claimed execution proof.

**Remaining:** Adopt the proof statement, bind program/input/output/state to a real verifier and wire an actual deciding consumer.

**Done when:** Independent vectors and mutated proof/program/input/output/state tests reject invalid claims in the selected consumer.

Sources: [pkg/zkvm/canonical.go](../../pkg/zkvm/canonical.go), [pkg/zkvm/canonical_executor.go](../../pkg/zkvm/canonical_executor.go).

### 6. Public binary-state contract and recovery (research)

Private binary overlay/conversion controls do not define a public compatible layout.

**Remaining:** Agree layout, addressing and witness rules; qualify authenticated recovery and resource bounds.

**Done when:** Independent clients agree on roots, witnesses and persisted recovery under the chosen public contract.

Sources: [docs/ROADMAP.md](../../docs/ROADMAP.md), [docs/next-stage-work-plan.md](../../docs/next-stage-work-plan.md).

### 7. PQ validator keys, transactions and sampling (research)

PQ primitives and private provider/transaction controls are narrower than a public protocol.

**Remaining:** Define validator-key lifecycle, public signing/sampling contracts and independent implementation vectors.

**Done when:** Key registration, rotation, revocation, replay rejection and live consumers agree across independent implementations.

Sources: [docs/ROADMAP.md](../../docs/ROADMAP.md), [pkg/crypto/pqc/pq_tx_signer.go](../../pkg/crypto/pqc/pq_tx_signer.go).

### 8. Proof, native-rollup and finality consumers (research)

Native rollup transition/withdrawal helpers use simulated hash conditions; these are not real rollup proof acceptance.

**Remaining:** Adopt compatible execution, proof and consensus contracts; connect independent verifiers and actual deciding consumers.

**Done when:** State transitions, withdrawals, fees and finality pass independent end-to-end positive and rejection controls.

Sources: [pkg/rollup/native_rollup.go](../../pkg/rollup/native_rollup.go), [docs/ROADMAP.md](../../docs/ROADMAP.md).

## Historical label inventory

Across the six mapped cards there are 51 entries: 18 labelled Complete and 33 with older non-complete labels. The non-complete distribution is Glamsterdam 3, Hegotá 7, I* 7, J* 7, K* 2 and L* 7. The total of 33 old labels is coincidental and unrelated to the 33 completed frozen-development task IDs. No whole-client completion percentage is derived from either count.

The three Glamsterdam labels are broad ePBS (Substantial), fast confirmation (Functional) and evm-asm canonical quest (Functional). Broad ePBS keeps its historical draft label even though the adopted signed route is delivered; it is not counted as a newly missing code feature. Both offchain research frameworks need separately adopted production input/consumer contracts. Their old phase placement does not expand the completed frozen task ledger.

Four planned-only entries list no implementation or tests: tapered issuance burn, decentralized state, ethp2p unification and blob-transaction removal. These are identified as planned scope, not as already implemented features. Every other historical non-complete entry remains visible in the page's per-phase expandable inventory; a fresh source/consumer audit is required before promoting or recasting those labels.

## Review depth and limits

The fresh source checks inspect fast-confirmation constructor/input consumers, canonical.VerifyExecution versus CanonicalExecutor.VerifyGuestProof, and native-rollup transition/withdrawal hash simulations. A [fresh executable observation](../reviews/roadmap-remaining-20261124/CANONICAL_OBSERVATION.json) confirms that the standalone VerifyExecution helper accepts 256 all-zero proof bytes with Success=true; the source is retained as canonical-observation.go and can be run from pkg. The structural legacy helper is not conflated with the separate executor verification method. No new failure of the qualified native Glamsterdam import/signature path is established by these research findings.

Dora sibling revision 6f702f24e8445d9e4b764d5bf141c604b1c99d47 was inspected read-only: writedb.go still has Gloas/Heze EL-fallback handling. That corroborates a migration/consumer-qualification boundary; it does not claim no decoder or no indexing exists. Frame/paymaster receipts retain their November 18 synchronized-EL and source-specific boundaries. No new independent Frame or production service acceptance was executed here.

## Recommended sequence

1. Adopt a compatible independent Frame/Bogotá target and close independent-client/alternate-peer/restart qualification.
2. Complete Dora consumer/schema/rebuild migration and workload/paymaster service cutover in separate branches/PRs.
3. Select individual research contracts before connecting fast-confirmation, canonical proofs, binary state, PQ and rollup/finality consumers. Do not silently enable structural/simulated verification as protocol authority.
4. Revisit planned-only scope only when a concrete contract is adopted; retain the two deferred acceptance items outside the development denominator.

[Roadmap](../ROADMAP.md) · [Current frozen accounting](../reviews/development-scope-20261124/CURRENT_STATE.json) · [Review checklist](../reviews/roadmap-remaining-20261124/CHECK.md).
