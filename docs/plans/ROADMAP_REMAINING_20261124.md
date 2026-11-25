# Roadmap remaining work - November 24, 2026

## Reviewed state

Current reviewed master baseline is b14dbb15, with PRs 155–158 merged; EIP-7495 functional source is 36b3e62d in PR159. Glamsterdam scope follows official EIP-7773 at b055feec: twenty scheduled core EIPs, with twenty current Complete implementation labels after EIP-7495 qualification in PR159. Networking/informational references are separate. The earlier 33/33 frozen task ledger and 5/5 follow-up record retain their original source-specific meaning; they are not the official-core denominator. This page correction is not a new runtime qualification.

48h/larger-scale admission and upstream Lodestar approval remain excluded from development progress at the user's direction. Original 04 attribution remains unconfirmed. Public activation is a separate network rollout decision, using existing schedule/genesis/override support. No public fork parameters or new protocol scope are selected here.

## Current specification repair delivery

Five newly confirmed Glamsterdam problem groups are implemented in independent [EL 150](https://github.com/alt-research/eth2030/pull/150), [EL 151](https://github.com/alt-research/eth2030/pull/151) and [Lighthouse 16](https://github.com/jiayaoqijia/lighthouse/pull/16) reviews. EL targets master; the companion CL targets eth2030. The roadmap now links these reviews and the [current specification audit](https://github.com/alt-research/eth2030/pull/152), separately from the earlier merged baseline. Current CI/merge outcomes are authoritative on those PRs; this does not relabel earlier binaries or claim a new long-duration result.

## Remaining workstreams

This view retains seven later-scope workstreams; the EIP-7495 official-core conformance follow-up is qualified in PR159. They do not reopen the earlier frozen task ledger. Integration can be partly implemented while production-consumer qualification is still open. The static page expands each workstream to show evidence and completion criteria.

### Completed official-core follow-up: EIP-7495

Immutable schemas, canonical Container encoding, exact progressive roots and stable field proofs are implemented and independently qualified in [PR159](../reviews/pr-159/STATUS.md). The shared list-proof gindex defect is repaired. Actual 384-validator CL state bytes/root/branches and existing execution SSZ reference vectors agree; all seven executed rejection mutations fail as required. The deprecated mutable optional-field helper is not used as a normative type declaration.

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

### 4. Later optional execution proofs and canonical Ethereum STF guest (research)

Merged EL #156 supplies real pinned RV32IM execution receipts, complete machine-state/statement binding and native asynchronous proof RPC. EIP-8025 and the executable Ethereum STF guest are later proposals outside official Glamsterdam scope; no official future fork assignment is implied. The fixed-output STF adapter fixture is not an Ethereum STF implementation.

**Remaining:** Implement an executable canonical Ethereum STF guest with actual block/prestate witnesses; qualify independent state/receipt roots and payload-bound optional proof production, verification and recovery under the adopted draft. Mandatory proof adoption is separate later protocol work.

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

Current progress uses twenty Complete implementation labels out of twenty officially scheduled Glamsterdam core EIPs, including independently qualified EIP-7495. Future project phases keep their mapped planning entries. A separate later-proof section retains EIP-8025 and the canonical Ethereum STF guest without counting them against Glamsterdam or assigning them an official activation date.

Archived 16/17 and 33-task accounting records are not rewritten. They describe earlier project-specific selections and are superseded for the current page denominator.

Four planned-only entries list no implementation or tests: tapered issuance burn, decentralized state, ethp2p unification and blob-transaction removal. These are identified as planned scope, not as already implemented features. Every other historical non-complete entry remains visible in the page's per-phase expandable inventory; a fresh source/consumer audit is required before promoting or recasting those labels.

## Review depth and limits

The earlier review inspected fast-confirmation constructor/input consumers, canonical.VerifyExecution versus CanonicalExecutor.VerifyGuestProof, and native-rollup transition/withdrawal hash simulations. A retained [historical executable observation](../reviews/roadmap-remaining-20261124/CANONICAL_OBSERVATION.json) confirms that the standalone VerifyExecution helper accepts 256 all-zero proof bytes with Success=true; the source is retained as canonical-observation.go and can be run from pkg. The structural legacy helper is not conflated with the separate executor verification method. No new failure of the qualified native Glamsterdam import/signature path is established by these research findings.

Dora sibling revision 6f702f24e8445d9e4b764d5bf141c604b1c99d47 was inspected read-only: writedb.go still has Gloas/Heze EL-fallback handling. That corroborates a migration/consumer-qualification boundary; it does not claim no decoder or no indexing exists. Frame/paymaster receipts retain their November 18 synchronized-EL and source-specific boundaries. No new independent Frame or production service acceptance was executed here.

## Recommended sequence

1. Adopt a compatible independent Frame/Bogotá target and close independent-client/alternate-peer/restart qualification.
2. Complete Dora consumer/schema/rebuild migration and workload/paymaster service cutover in separate branches/PRs.
3. EL #155/#156 and Lighthouse #17 integration are merged. EIP-7495 qualification is delivered in PR159. Build a real Ethereum STF guest as a separate major feature; select independent binary-state, PQ and rollup/finality contracts before public consumers. Do not promote structural/simulated verification to protocol authority.
4. Revisit planned-only scope only when a concrete contract is adopted; retain the two deferred acceptance items outside the development denominator.

[Roadmap](../ROADMAP.md) · [Current frozen accounting](../reviews/development-scope-20261124/CURRENT_STATE.json) · [Review checklist](../reviews/roadmap-remaining-20261124/CHECK.md).

## Extension delivery update

The November 24 extension review supersedes the earlier Fast confirmation and canonical verifier observations: EL #155 implements vote isolation and actual signed CL/native consumers; Lighthouse #17 repairs durable FCR publication; EL #156 supplies real pinned execution receipts and native proof RPC. The remaining broad canonical quest is an executable Ethereum STF guest and adopted public proof consumers, not the old hash/length verifier defect. Current page scope is the official twenty-core-EIP list (20 Complete implementation labels); later proof proposals are excluded; see [the extension record](../reviews/glamsterdam-extensions-20261124/STATUS.md) for source-specific validation and integration state.
