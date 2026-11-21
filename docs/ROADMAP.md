# ETH2030 development roadmap

Updated November 22, 2026 against merged master `9a094830` (PRs 123/124/125). The homepage and `/roadmap` share the same dated source-specific progress; the per-EIP tracker retains the November2 EF Strawmap placement baseline. Tracker-label percentages are not public-fork readiness scores.

## Glamsterdam: frozen development target delivered

The selected merged LIFO / signed Gloas implementation and its original 33 task IDs / R01–R13 are merged and qualified. `pkg` and code-CI bytes equal the final tested `d9201b0` composition. See [merged progress](PROGRESS.md), [final source-specific qualification](plans/glamsterdam/POST_MERGE_CLOSURE_20261122.md) and the [current remaining-gap audit](audits/merged-glamsterdam-20261122/README.md).

| Area | Delivered result |
| --- | --- |
| Native execution and APIs |219 packages;107 unchanged official RPC files/108 requests; six affected race suites; full vet/build and a byte-identical independent native rebuild. Native EF corpora retain their own exact source/fork/format counts. |
| Builder/CL lifecycle | Actual signed external builders, payments, FULL/EMPTY/withheld/late/reorg controls, registry sweep/reuse, progressive SSZ and independent signed finality. |
| Data/networking | eth/70 receipts, eth/71 BALs, eth/72 cells, snap/2 and CL partial columns; actual two-row external-builder blob with544 KZG proofs and useful partials. |
| Sync/durability | Independent Geth full/snap late join, moving-pivot/restart controls, historical state/BAL readback and four original-datadir recoveries under 300s. |
| Operating evidence |400/400 steady slots and60/60 pressure slots; unchanged budgets,270 independent pressure proofs, finality and offline replay. |

Follow-up source reviews [127](https://github.com/alt-research/eth2030/pull/127), [128](https://github.com/alt-research/eth2030/pull/128) and [129](https://github.com/alt-research/eth2030/pull/129) retire unsupported legacy Builder Engine extensions, make startup failure transactional and bound FileDB value/WAL residency. [130](https://github.com/alt-research/eth2030/pull/130) corrects restored snap progress under moving CL targets. They require merge review; the merged baseline above is unchanged. Remaining work includes actual multi-day capacity/soak qualification, metadata/history growth, upstream Lodestar hosted integration, conditional corpus expansion and the user's final public protocol/model decision. [Exact delivery and evidence](plans/glamsterdam/REMAINING_CODE_CLOSURE_20261122.md). Completed private development does not imply public-network activation or unlimited production capacity.

## Other roadmap phases

| Strawmap phase | Retained evidence | Next work |
| --- | --- | --- |
| Hegotá | Paired native seven-field Frame/FOCIL, sponsored slot 66, blob slot 165, fresh CL join 256 and receiver restart. | Compatible independent Frame/Bogotá contract, Dora production schema/epoch/receipt migration, service cutover and broader CL reliability. |
| I* | Private binary overlay/conversion and PQ transaction/provider controls. | Public layout/address/witness agreement, authenticated state recovery and resource bounds. |
| J* | PQ attestation and sampling research. | Authenticated validator-key lifecycle and independent vectors/clients. |
| K* | Execution-proof and state research. | Agreed proof statements, independent verifiers and actual deciding consumers. |
| L* | Native rollup and finality research. | Compatible execution/consensus contracts and independent end-to-end acceptance. |

The earlier Frame fresh CL used a synchronized EL. Independent Glamsterdam Geth sync is a different fork/profile and does not close independent Frame acceptance. I*–L* remain placeholder names from the retained EF drawing, without a new schedule assertion.

## Next development sequence

1. Review and merge the separately qualified Builder-retirement, startup-rollback and FileDB reviews using ancestry-preserving merges; verify the resulting package tree before assigning a release identity.
2. Pursue upstream Lodestar lockfile review/hosted CI while preserving the qualified dependency patch and exact runtime identity.
3. Complete the declared48-hour operating profile; retain bounded-value/WAL measurements and original million-file recovery controls. Key/header/canonical metadata and complete startup authentication remain cardinality-dependent.
4. Complete the reserved public protocol/model and release review against a compatible source/fixture/CL tuple.
5. Continue Hegotá service/independent-client work and later public state/PQ/proof research in separate scope.

[Detailed gap evidence and acceptance criteria](audits/merged-glamsterdam-20261122/README.md) · [Next-stage plan](next-stage-work-plan.md) · [CL roadmap](cl-iteration-roadmap.md) · [Task checklist](../CHECK.md)
