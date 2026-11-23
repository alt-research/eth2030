# ETH2030 development roadmap

Updated November24,2026 against master `cfd39f3f` with PR142-147 merged and Lighthouse eth2030 `fa16f70a`. Homepage and roadmap use one development-counting policy: the adopted frozen node implementation and confirmed follow-ups are complete.48h/larger-scale acceptance and upstream Lodestar approval are deferred outside that progress. Historical mapped EIP percentages retain their separate denominators.

## Glamsterdam: adopted node development complete

The frozen merged LIFO / signed Gloas scope, 33 original task IDs and R01–R13 are delivered. This does not complete every broader draft/framework or public activation requirement. [Actual progress](PROGRESS.md), [reviewed state](reviews/development-scope-20261124/CURRENT_STATE.json), and [actionable remaining plan](plans/glamsterdam/NEXT_STEPS_20261124.md) retain exact source/evidence boundaries.

| Area | Delivered result / boundary |
| --- | --- |
| Execution and APIs |219 combined Go suites, related race checks, vet/native build. Earlier official107 RPC files/108 requests retain their own source. Native and Geth-backed corpus scope stays finite. |
| Builder/CL | Signed external bids, payments, withheld/late/reorg/registry controls and progressive SSZ. Unsupported local Builder extensions are retired; authoritative state/signatures use the standard CL/EIP-8282 route. The repaired baseline completed27 hosted jobs; final simulator/bootstrap qualification is separate. |
| Data/networking | eth/70 receipts, eth/71 BALs, eth/72 cells, snap/2 and CL partial columns; earlier external-builder two-row blob 544 KZG proof record retained. |
| Sync/durability | Independent Geth full/snap late join, restart/retention controls, original-datadir recovery; queued history publication and actual captured FCU fsync/read-lock liveness defects repaired. |
| Bounded operation | Completed 6h45m05s; 144 healthy samples and 72 stacks after ready 5.5h. 14 unchanged-reference BLS/finality windows and actual stopped slot 97868 authenticated, 160 live + 2 stopped proofs. No 48h verdict. |

Original rounds 03/04 remain failed. The later captured FCU convoy is a confirmed defect with a merged repair; the original 04 precise holder/physical I/O cause is unconfirmed. 73 early missing slots, 686 memory-limit events and three local reveal advisories remain in the successful bounded record. No recurrence is not a root-cause proof.

## Current completion and next scope

| Item | Development treatment |
| --- | --- |
| Original tasks and confirmed fixes | Complete:33/33 original task IDs and5/5 follow-ups, all merged. No active confirmed adopted-scope implementation TODO. |
| Integrated CL CI | Complete for the selected profile:27 suite successes and local-testnet9success/1conditional skip; merged tree equals qualified source. |
|48h/larger-scale acceptance; Lodestar upstream approval | Deferred by the user, excluded from progress; actual acceptance/approval status remains separately recorded. |
| Original04 attribution | Historical unknown; not declared resolved and not an unimplemented feature. Future actual recurrence capture remains available. |
| Public activation | Separate chosen-network rollout decision using existing schedule/genesis/override support. No public preset is changed here. |
| Additional corpus/research/later forks | Separate explicitly adopted scope, not implicit work inside the completed ledger. |

[Exact policy, activation explanation and source audit](plans/glamsterdam/NEXT_STEPS_20261124.md). Known defects remain actionable even when found during deferred acceptance. No whole-public-fork or unrestricted production-readiness score is asserted.

## Other roadmap phases

| Strawmap phase | Retained evidence | Next work |
| --- | --- | --- |
| Hegotá | Paired native seven-field Frame/FOCIL, sponsored slot 66, blob slot 165, fresh CL join 256 and receiver restart. | Compatible independent Frame/Bogotá contract, Dora production schema/epoch/receipt migration, service cutover and broader CL reliability. |
| I* | Private binary overlay/conversion and PQ transaction/provider controls. | Public layout/address/witness agreement, authenticated state recovery and resource bounds. |
| J* | PQ attestation and sampling research. | Authenticated validator-key lifecycle and independent vectors/clients. |
| K* | Execution-proof and state research. | Agreed proof statements, independent verifiers and actual deciding consumers. |
| L* | Native rollup and finality research. | Compatible execution/consensus contracts and independent end-to-end acceptance. |

The earlier Frame fresh CL used a synchronized EL. Independent Glamsterdam Geth sync is a different fork/profile and does not close independent Frame acceptance. I*–L* remain placeholder names from the retained EF drawing, without a new schedule assertion.

[Next-stage plan](next-stage-work-plan.md) · [CL roadmap](cl-iteration-roadmap.md) · [Review checklist](reviews/development-scope-20261124/CHECK.md)
