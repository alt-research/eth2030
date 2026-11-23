# ETH2030 development roadmap

Updated November 23, 2026 against EL master `84200d53` and Lighthouse `eth2030` `34cecfc4`. EL PRs 131–140 and CL PRs 4–12 are integrated. The homepage and `/roadmap` share the same dated progress and open gates. Per-EIP mapped labels retain the November2 EF Strawmap placement baseline; those percentages are not whole-fork readiness.

## Glamsterdam: selected development delivered; readiness gates open

The frozen merged LIFO / signed Gloas scope, 33 original task IDs and R01–R13 are delivered. This does not complete every broader draft/framework or public activation requirement. [Actual progress](PROGRESS.md), [reviewed state](reviews/progress-20261123/CURRENT_STATE.json), and [actionable remaining plan](plans/glamsterdam/NEXT_STEPS_20261123.md) retain exact source/evidence boundaries.

| Area | Delivered result / boundary |
| --- | --- |
| Execution and APIs |219 combined Go suites, related race checks, vet/native build. Earlier official107 RPC files/108 requests retain their own source. Native and Geth-backed corpus scope stays finite. |
| Builder/CL | Signed external bids, payments, withheld/late/reorg/registry controls and progressive SSZ. Unsupported local Builder extensions are retired; authoritative state/signatures use the standard CL/EIP-8282 route. Merged CL 500 local tests are narrower than full hosted CI. |
| Data/networking | eth/70 receipts, eth/71 BALs, eth/72 cells, snap/2 and CL partial columns; earlier external-builder two-row blob 544 KZG proof record retained. |
| Sync/durability | Independent Geth full/snap late join, restart/retention controls, original-datadir recovery; queued history publication and actual captured FCU fsync/read-lock liveness defects repaired. |
| Bounded operation | Completed 6h45m05s; 144 healthy samples and 72 stacks after ready 5.5h. 14 unchanged-reference BLS/finality windows and actual stopped slot 97868 authenticated, 160 live + 2 stopped proofs. No 48h verdict. |

Original rounds 03/04 remain failed. The later captured FCU convoy is a confirmed defect with a merged repair; the original 04 precise holder/physical I/O cause is unconfirmed. 73 early missing slots, 686 memory-limit events and three local reveal advisories remain in the successful bounded record. No recurrence is not a root-cause proof.

## Next acceptance sequence

| Order | Gate | Completion condition |
| --- | --- | --- |
| 1 · P0 | G-01 integrated CL CI | Repair the actual url dependency audit, optimistic-head API beacon-chain invalidation/deferred-verdict and simulator failures in workflow 45134918940; inspect all remaining jobs and rerun unchanged complete profiles. Local scoped passes do not substitute. |
| 2 · P1 | G-02 incident evidence; G-03 duration/capacity | Preserve unknown historical attribution; retain actual correlated waits on recurrence. Declare growing-state/history/resource/restart workload; actually complete strict 48h if retained, or explicitly qualify a separate bounded profile. |
| Parallel · P1 | G-04 upstream Lodestar | PR 10693 maintainer lock review, full upstream hosted qualification and integration. |
| Before public activation · P1 | G-05 final protocol/model | Select compatible gas/Engine/consensus/CL/fixture target and activation, then requalify affected paths. |
| Conditional · P2 | G-06 corpus expansion | Execute specifically adopted formats/handlers/consumers, with native versus wrapper scope and negative controls. |

Broader historical fast-confirmation / EVM-ASM framework labels, optional EL sampler/beam and unused helper stubs need separately scoped implementation/consumer acceptance before promotion. They are not uncompleted obligations silently inside the selected 33 tasks. [Source TODO triage and exact criteria](plans/glamsterdam/NEXT_STEPS_20261123.md).

## Other roadmap phases

| Strawmap phase | Retained evidence | Next work |
| --- | --- | --- |
| Hegotá | Paired native seven-field Frame/FOCIL, sponsored slot 66, blob slot 165, fresh CL join 256 and receiver restart. | Compatible independent Frame/Bogotá contract, Dora production schema/epoch/receipt migration, service cutover and broader CL reliability. |
| I* | Private binary overlay/conversion and PQ transaction/provider controls. | Public layout/address/witness agreement, authenticated state recovery and resource bounds. |
| J* | PQ attestation and sampling research. | Authenticated validator-key lifecycle and independent vectors/clients. |
| K* | Execution-proof and state research. | Agreed proof statements, independent verifiers and actual deciding consumers. |
| L* | Native rollup and finality research. | Compatible execution/consensus contracts and independent end-to-end acceptance. |

The earlier Frame fresh CL used a synchronized EL. Independent Glamsterdam Geth sync is a different fork/profile and does not close independent Frame acceptance. I*–L* remain placeholder names from the retained EF drawing, without a new schedule assertion.

[Next-stage plan](next-stage-work-plan.md) · [CL roadmap](cl-iteration-roadmap.md) · [Review checklist](reviews/progress-20261123/CHECK.md)
