# Glamsterdam final post-merge closure (2026-11-22)

The selected merged LIFO and signed Gloas development target is implemented and requalified. This record supersedes current-progress statements in the [2026-11-20 handoff](COMPLETION_HANDOFF_20261120.md), preserving its reconciliation of all33 original tasks and R01–R13 and its nine unchanged source-specific archives. It does not adopt an unmerged gas/envelope alternative or approve a public-network fork.

## Delivery and defects

| Review | Delivered behavior | Target |
| --- | --- | --- |
| [123](https://github.com/alt-research/eth2030/pull/123) | Resume downloaded snap history after legal BAL expiry using actual finalized/block slots; missing retained bytes and present corruption still reject. | master |
| [124](https://github.com/alt-research/eth2030/pull/124) | Restore authenticated indexed snap descendants without redundant replay; preserve IO/malformed-finality errors instead of fabricating zero. | master |
| [125](https://github.com/alt-research/eth2030/pull/125) | Original failures, actual normal launch profile, final source/runtime/protocol archives, independent replay and merge-order checks. | master |
| [Lodestar10693](https://github.com/ChainSafe/lodestar/pull/10693) | Correct delayed cleanup, pending identify and shared GossipSub protocol aliases using bounded logical-topology generations. Fifteen regressions and actual recovery pass. | unstable |

Original EL reviews120/121/122 are merged in masterc27a5bb09507461bb6e9f2ca72c2456d04072f80. Its package/workflow bytes equal qualified1221d968. Final source d9201b0cdb8524a28c81ae6b09017efa8a39b6a6 composes that master and independent123/124; no aggregate feature PR hides those reviews. Only seven package files differ. Native Go1.26.11 executableSHA256 is8d56e60988a014622c3fe8882eb74af9ead972785ea6318e5aa5eee660f64bec; a fresh detached rebuild is byte-identical.

The actual compatible CL tuple retains Lighthouse d547 and Lodestar applicationcf705 plus explicitly hashed libp2p patchb1b03520dcf17504a10641301d16abca4e8434c02b1c9e81dd5e7baf9fc8e156 from functional sourcecad22db. Runtime/module/inode/image/configuration/resource identities are recorded independently; an old CLI label or base image tag is not substituted for the actual executable/module identity. Original Geth and Buildoor producer remain unchanged.

## Completed acceptance

| Obligation | Final evidence |
| --- | --- |
| Native execution and consumers |219 fresh packages;107 unchanged official files/108 requests; six affected race suites; full vet/native build. Original baseline also passes seven affected race suites and26 Python controls. |
| Ordinary original-datadir recovery | Docker-start conservative189.038/212.551/219.661/176.953seconds for main2/main3/full/snap, all under unchanged300s; frozen finalized commitments and eight independent proofs. |
| Network lifecycle | Fifteen normal-root dependency regressions; three automatic6s provider outages recover in0.155/5.493/1.168s under90s; normal native mainnet/minimal/gnosis and all workspace checks pass. |
| Steady/pressure | Unchanged1200s/180s windows,400/400 and60/60 slots, no normal failures/gaps/identity changes; RPC/Engine latency budgets pass. Pressure-invalid hashes are correctly INVALID, not counted as normal failures. |
| Independent consumers |512-participant BLS finality and original/offline replay for each window;270 SDK account proofs under load. Historical getter42/four proofs and original361-byte BAL hash/range readback match independent Geth; unknown body null and pre-pivot state unavailable. |
| Final-source partial/blob | Actual external-builder slot59163 with two ordered rows;544 independent KZG cell proofs; signed block/envelope, parent SSZ builder-key authentication and mutation rejection;228 useful target partial cells/114 completions with getBlobs disabled, observer32/16. Parent/blob span independently finalizes and replays. |
| Ordinary lifecycle final state | All three exact SSZ roots at independently finalizedslot56096 prove builder1 balance0 after epoch903. Existing two complete sweep/reuse transition replays retain their own evidence; no third transition replay is claimed. |
| Provenance/reproduction |245 selected public files, complete byte manifest, exact native/dependency patches, genesis/bootstrap, original failures, reference/source/license metadata, and fresh public-byte independent replay. Prior nine archives remain unchanged. |

Main2/main3 previously ran a completed one-time migration tool before every daemon launch. The final normal profile directly executes the verified daemon, preserving the old script/migration executable/datadir archive hashes. Its redundant scan caused an original over-budget run that remains failed. Backup preparation has its own declared600s limit; the daemon deadline remains300s. Directory-copy timeout, earlier snap replay timeout, launcher-path/import errors and original blob API404 remain recorded separately. None is relabelled as a final pass. No consensus/hash validation, resources, clock, gas model or peer scoring was weakened.

See the [complete result/source/recovery records](../../reviews/glamsterdam-postmerge-20261121/STATUS.md), [public archive and replay instructions](../../reviews/glamsterdam-postmerge-20261121/artifacts/README.md), and [operator runbook](../../operations/glamsterdam-recovery.md).

## Remaining decisions and merge sequence

1. Review and merge the independent master-targeted EL fixes123/124 and this evidence review. Each major function stays separately reviewable. The final remote-head checks and all six cumulative merge orders are recorded separately; no EL merge is performed by this task. After merge, verify the resulting package tree equals the qualified composition before assigning a release identity.
2. Lodestar maintainers must review the required patch-lock update and run/approve full hosted unit/spec/e2e/browser CI. The unchanged external-PR lockfile guard rejects the change; its failure is retained, not bypassed or described as a green upstream pipeline. Local normal checks and actual-process acceptance are already complete.
3. The user retains the final public protocol/model decision. EIPs12581/12582 closed unmerged on2026-11-21; closure alone adopts no replacement. Reopen implementation only when a concrete adopted change supersedes the frozen merged LIFO/signed-envelope target; requalify the affected gas/state/Engine/CL tuple then.

There is no known unfinished implementation item in the selected frozen development scope after these corrections. The three actions above remain explicit review/approval boundaries, so this document does not claim all PRs merged or a public-network release approved. Later-fork FOCIL/frame/SSZ execution/PQ work remains outside this Glamsterdam program as originally scoped.
