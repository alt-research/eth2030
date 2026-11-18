# ETH2030 development roadmap

Updated November 18, 2026 against merged mainline `911afb0c` ([PR 61](https://github.com/alt-research/eth2030/pull/61)).
Fork placement retains the site's November 2, 2026 Strawmap baseline. Ethereum
proposal placement, local implementation and independent acceptance are separate
signals. The [site tracker](../site/src/data/eips.ts) supplies the homepage and
`/roadmap` labels; the [next-stage plan](next-stage-work-plan.md) defines active work.

## Merged and qualified milestone

The pinned seven-field EIP-8141 implementation now executes through the native
EL, pool, Engine, receipt store and historical body APIs. The qualified pair was
EL `e0451092` with Lighthouse `a4ceb86e`; the fork's `eth2030` branch includes
that code and its verification record at `29d66c238`.

| Acceptance | Retained result |
| --- | --- |
| Coded sponsorship through FOCIL | Signed list at slot 65 delivered the Frame to the isolated proposing EL at slot 66; both pairs finalized it. |
| Blob Frame | Slot 165 finalized on both pairs; ordered commitments, receipts and execution/blob fee settlement matched. |
| Fresh CL history | A validator-free CL joined at slot 256, checkpoint-synced and returned both original finalized execution envelopes. |
| Receiver restart | Both workloads retained their original receipts, historical application storage and sponsor balances. |
| Runtime identity and cleanup | Binary/image version matched `a4ceb86`; client caps were enforced; enclave, labeled containers/volumes and engine were removed. Owned runtime images and temporary binaries were also removed. |

The new CL used the already-synchronized second EL. This qualifies a paired
native control. Independent EL late join, cross-client ABI agreement and the
production explorer migration remain open.
[Run record](verification/frame-8141-20261118-paymaster-history-final.json) ·
[Implementation and test boundaries](frame-8141-execution.md)

## Roadmap placement and current evidence

| Strawmap phase | ETH2030 development evidence | Next acceptance |
| --- | --- | --- |
| Glamsterdam | Native BAL commitment/validation and persisted RLP historical bodies; paired Gloas self-build and scheduled transition evidence. | External builder payment, late/withheld envelopes, PTC and independent-client BAL/availability tests. |
| Hegotá | Heze signed inclusion-list production/recovery, native Bogotá verdict handling, pinned Frame execution, coded sponsorship, blob history and receiver restart. | Compatible independent EL verdicts, service consumers, skipped-slot checkpoints and VC/peer recovery variants. |
| I* | Private binary overlay/conversion and PQ transaction/provider paths have retained Track 4 regression evidence. | Agree public key layout/addresses, witness charging, authenticated state recovery and resource bounds. |
| J* | PQ attestation and sampling research paths remain separately tracked. | Authenticated validator-key lifecycle and independent protocol vectors/clients. |
| K* | Execution-proof and state research remains protocol-specific. | Agreed proof statements, independent verification and deciding-consumer acceptance. |
| L* | Native rollup and finality research remains in its own experimental scope. | Matched execution/consensus contracts and end-to-end independent acceptance. |

I*–L* are placeholder phases in the retained Strawmap drawing. Historical
tracker percentages count entries labelled Complete; they do not measure whole
fork completion, current-spec conformance or network activation. Older aggregate
claims such as “65/65 complete” are not the current development acceptance model.

## Next implementation sequence

1. Freeze a compatible independent Frame/Bogotá client contract. Current Geth
   experiments include keyed nonces and are not a seven-field execution oracle.
2. Integrate the upstream-shaped Heze codec into Dora's actual epoch/receipt
   consumer and qualify pruning, competing roots and restart. Qualify ordinary
   upstream workload services before retiring maintained modifications.
3. Extend CL reliability: skipped-slot checkpoints, VC restart, remote signing,
   redundant BN publication and alternate-peer/partial-response recovery.
4. Exercise external Gloas builders, PTC outcomes and blob availability, then
   qualify the newer genesis/package pin on the same combined fork profile.
5. Run independent full/snap late join; continue authenticated moving-pivot
   recovery and storage/memory scale tests.
6. Advance public Track 4 state and PQ/proof protocols with independent vectors.

Each live case uses one bounded enclave and retains source/image identities,
actual requests/results, failure outcomes and cleanup. Keep ordinary KZG blob
verification distinct from optional PQ commitment research.

[Active plan](next-stage-work-plan.md) · [CL iteration](cl-iteration-roadmap.md) ·
[Track 3 acceptance](track3-remaining-acceptance.md) ·
[Upstream/service audit](devnet-fork-upstream-audit.md) · [Task checklist](../CHECK.md)
