# ETH2030 next-stage work plan

Updated November 18, 2026 after [PR 61](https://github.com/alt-research/eth2030/pull/61)
merged at `911afb0c`. The pinned native Frame implementation and paired history
control are delivered. The next milestone is independent client and service
acceptance against explicit protocol/schema pins.

## Verified starting point

| Component | Delivered result | Scope |
| --- | --- | --- |
| ETH2030 master | Native Bogotá, seven-field Frame execution/admission, split gas, paymaster v1 and historical body/BAL persistence. | [Execution record](frame-8141-execution.md) |
| Live native EL | `e0451092` finalized coded sponsorship and blob execution, then reopened original historical state/receipts. | [Final control](verification/frame-8141-20261118-paymaster-history-final.json) |
| Lighthouse fork | `eth2030` includes qualified runtime `a4ceb86e` and tracking commit `29d66c238`. Strict live IL handling and trusted-checkpoint import pass. | [CL branch](https://github.com/jiayaoqijia/lighthouse/tree/eth2030) |
| Fresh CL | New database joined at slot 256 and returned the original finalized sponsored slot-66 and blob slot-165 envelopes. | Uses the synchronized second EL; independent EL late join remains open. |
| Services | Released Go client `v0.1.11` and upstream-shaped compatibility types replay real finalized Heze state and blocks against committed roots. | [Replay candidate](../tools/consensus-replay/README.md); production consumer cutover is pending. |
| Resource lifecycle | Bounded control completed and removed its enclave, labeled containers/volumes, engine, owned runtime images and temporary binaries. | Retain this lifecycle for every next live control. |

The [upstream implementation plan](devnet-upstream-implementation-plan.md) and
[legacy branch audit](lighthouse-legacy-branch-merge-plan.md) retain earlier
source comparisons and integration decisions. They predate the completed Frame
milestone; use the gates below for new work.

## P0 · Independent Frame and Bogotá contract

The [base Frame draft](https://github.com/ethereum/EIPs/blob/dc5d5fa688b7fe87b95a907fcc128b4fde2f4d2f/EIPS/eip-8141.md)
and [Bogotá Engine draft](https://github.com/ethereum/execution-apis/blob/981aa66d79364be8322a9c358aaf0a571cca0873/src/engine/bogota.md)
remain the private control's wire pins. Current Geth Bogotá development adds
keyed nonces and other protocol changes; it is not a matching seven-field oracle.

1. Select an independent source with the same envelope, gas, receipt and Engine
   contract, or create a separately versioned EIP-8250 experiment. Preserve the
   qualified scalar-nonce control while changing protocol versions.
2. Freeze cross-client fixtures for approvals, creation, fee settlement,
   atomic/refill rollback, blob ordering, canonical receipts and nonce bounds.
3. Compare actual V6 execution validity and inclusion satisfaction, deferred
   ACCEPTED-to-FCU results, competing views and restart. Keep execution validity
   and fork-choice eligibility independent.
4. Agree the paymaster runtime/witness ABI. ETH2030 v1 is an experimental local
   candidate; exact-code recognition and delayed-withdrawal accounting are tested.

**Acceptance:** The independent client reproduces wire responses, canonical
state/receipt roots, payer/proposer deltas, ordered blob commitments and restart
readback. A same-implementation pair or codec-only oracle does not close this gate.

## P1 · Service and CL reliability

| Order | Work | Required acceptance |
| --- | --- | --- |
| 1 | Dora upstream/schema migration | Integrate the qualified upstream-shaped codec at the fork decoder; process real SSZ through epoch and receipt consumers. Match validator balances/statistics and Frame display with Beacon/EL APIs. |
| 2 | Dora retention/recovery | Competing proposal roots, late/equivocated lists, event loss, Req/Resp recovery, CL pruning, fresh database rebuild and persisted restart. Retire the retained-view adapter only after equivalent readback. |
| 3 | Upstream workload qualification | Included EOA/token/blob/7702 traffic, funding/receipt failover across EL endpoints, delegated and sponsored Frame vectors. Retire old modifications only for the covered contract. |
| 4 | CL reliability | VC restart, remote signing, redundant BN publication, alternate peers, partial responses and competing dependent roots; include skipped-slot checkpoints through actual envelope/custody import. |
| 5 | Gloas builder/availability | Actual external signed bid/payment, late or withheld envelope, PTC outcomes, bad BAL rejection and blob-column delivery/custody. Self-build evidence remains a separate control. |
| 6 | Package/genesis promotion | Trial the newer package/generator pin independently; verify generated SSZ and simultaneous Heze/Bogotá activation before moving combined profiles. Direct Heze genesis remains unqualified on the older generator. |

The [service audit](devnet-fork-upstream-audit.md) explains the 92-byte legacy
Heze state difference and the additional gaps in the March local codec. Use
released upstream behavior where it covers the active profile; keep experimental
compatibility at the decoder boundary, with the upstream field/Merkle organization.

## P1 · Independent node sync and recovery

Run independent full then snap late join against Geth/Lighthouse sources. Capture
P2P and authenticated Engine traffic, moving finality, adopted roots, receipts,
restart and a transaction executed after the pivot. Continue authenticated reuse
and durable account/storage cursors across A → B → C pivots; measure requests and
bytes through interruption. Reject stale proofs, missing leaves and conflicting
finalized targets. Measure FileDB indexing, execution/serving copies and reopen
before promoting the storage path.
[Track 3 acceptance](track3-remaining-acceptance.md) · [Node boundaries](track3-node.md)

## P2 · Public Track 4 and longer-horizon protocols

Keep the merged private PQ/binary overlay profile as a regression control.
For public state adoption, agree EIP-7864 layout/addresses, authenticated frozen
state/checkpoint reads and EIP-4762 witness charging; reuse Track 3 scale tests.
For PQ consensus, authenticate committee identity and keys, then qualify key
transitions and restart. Define independent statements/vectors for STARK,
lattice and aggregation providers before using them in deciding consumers.
Standard blob verification remains KZG. Quick Slots, optional proof relay and
Lean consensus experiments retain separate protocol pins and acceptance.

## Execution and review discipline

Use isolated branches/worktrees and commit each repository separately. Build
and identify the actual source/image under test. Retain complete inputs, terminal
outcomes and negative controls. Run one capped enclave at a time: at most 2 GiB
per EL/BN, 1 GiB per VC, and a separately bounded budget for added services.
Remove owned resources after success, failure or timeout and verify Docker
ownership labels without reviving a removed Kurtosis engine.

[Current roadmap](ROADMAP.md) · [Progress](PROGRESS.md) ·
[CL iteration](cl-iteration-roadmap.md) · [Task checklist](../CHECK.md)
