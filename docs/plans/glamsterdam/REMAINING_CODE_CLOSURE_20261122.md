# Remaining-code corrections and review handoff (2026-11-22)

The user's selected merged LIFO / signed Gloas development target is unchanged. The user explicitly chose to retire unsupported Builder Engine extensions and retain the existing standard CL/EIP-8282 route. This record updates the [original merged audit](../../audits/merged-glamsterdam-20261122/README.md); it preserves that audit and all earlier source-specific failures and measurements. Reviews126–130 are now merged in master55f7fd84; this does not claim48-hour qualification complete.

## Separately reviewable delivery

| Review | Corrected behavior | Source / evidence |
| --- | --- | --- |
| [126](https://github.com/alt-research/eth2030/pull/126) | Merged-progress audit/site and synchronous metrics reservation. |43002a5d; original merged baseline9a094830. |
| [127](https://github.com/alt-research/eth2030/pull/127) | Remove unsupported Builder Engine dispatch, advertised methods, Go APIs, local registry/synthetic stake and signature-dropping header conversion. JWT-authenticated calls now return method-not-found. |76427149; [migration and exact controls](../../reviews/builder-authority/MIGRATION.md). |
| [128](https://github.com/alt-research/eth2030/pull/128) | Reserve required RPC/Engine/enabled WS/metrics listeners before workers; propagate failures and unwind initialized owners. Stopped/failed instances are terminal; freshNew on the released datadir/ports is the retry contract. Join lifecycle work and cancel DNS. |6c8c8b12; [startup controls](../../reviews/startup-rollback/STATUS.md). |
| [129](https://github.com/alt-research/eth2030/pull/129) | Keep a16MiB accounted LRU instead of permanent values and stream WAL replay. Preserve known-key IO, write fencing, atomic markers and on-disk formats. Propagate iterator/deferred-state/code-hash errors through snapshots, trie roots, migration and RPC consumers. |20293b2a; [design, profiles and detected mutants](../../reviews/filedb-bounded-residency/DESIGN.md). |
| [130](https://github.com/alt-research/eth2030/pull/130) | Restore full hash-linked execution progress over non-genesis snap state when CL targets advance; keep genesis finalized-pivot authentication. The live preflight exposed the separate retained-ancestry eligibility mismatch. |2c0df12f; [red regression, genesis positive and qualification](../../reviews/restored-snap-moving-target/STATUS.md). |

At the historical pre-merge frozen remote heads, all five reviews have10/10 successful checks, mergeable=true and no unresolved inline review threads. Protected-branch status remainsblocked; this is not a conflict and does not supply human approval. All120 ordinary source merge orders are conflict-free with the same final tree149ee79570ac76da4727bd65ab1b7736c2bed9ab. [Exact heads/CI](../../reviews/remaining-code-closure/SOURCE_PRS.json) and [all orders](../../reviews/remaining-code-closure/MERGE_ORDERS.json) are retained.

All targetmaster. The inherited common tracking ancestor prevents independent edits to rootCHECK/STATUS. Each feature owns nested tracking. Those source merges are already delivered; this documentation review remains independently targeted. Review the last two against their prerequisite composition; their base remainsmaster so delivery reachesmaster. Preserve ancestry rather than flattening stacked prerequisites with squash/rebase. No EL master merge is performed here.

The source-only compositionbcf9e13e and final functional source2c0df12ff4c89d6407c5f27c8de8ea56517d7a4e each pass219 fresh full-package suites,107 official RPC input files/108 requests and full vet/native build. The former passes11 affected race suites; the final sync change additionally passes beacon/node race and20 moving-target/genesis controls. The original individual source reviews retain their own exact-head checks and package counts. A removed legacy package and added profiling command explain differences; counts are not silently added together.

Snapshot executableSHA256:bc680041045f9124e83c6493f0cee62cb15212f70c7611a48162f20a654981cb. Final observation executableSHA256:4e0f7a54765ce0992d0477d65565fcb41c27a8bac6525bb6f2497721939a9043. Both useGo1.26.11. Original d920 runtime evidence remains attached to its original binary.

## Builder authority and complete signed message

No local EL registration is promoted to on-chain stake. The selected Gloas CL state supplies the builder public key, finalized active-registration status, balance and available collateral; its signed SSZ message includes execution_requests_root and ordered commitments. The independent unchanged2ea9d96a reference authenticates the original parent state/root and real signed bid, active/funded builder0 and exact signed SSZ roundtrip. Thirteen mutations reject altered signed fields, signatures, commitment order, registry key, fork/genesis domains, unfinalized deposit, exited builder and insufficient collateral. The separate512-participant signed finality replay authenticates the parent59162/blob59163 span through seven links.

Six retired names return-32601 over real JWT HTTP; the frozen Engine68d76a5 capabilities and actual captured standard methods stay usable. Engine capability negotiation correctly does not advertise engine_exchangeCapabilities itself. The earlier guard mistake remains a failed harness observation. Generic epbs research helpers are not made into a CL oracle and are no longer reachable through the retired Engine/node auction route. Broader experimental APIs/types retain their own scope.

## Capacity and original-state recovery

| Declared input | Original observation | Corrected observation | Limit / interpretation |
| --- | --- | --- | --- |
|10,000 values x65,536 bytes;655,360,000 logical bytes |738,905,568 heap bytes after open;939,364KiB peakRSS; open0.471s/read-all0.577s. |1,974,992 heap bytes;52,440KiB peakRSS; open0.019s/read-all0.751s. |Both pass the same1GiB/300s fixture budgets. Cold-read cost increases. Synthetic data is not real chain acceptance. |
|100 keys x4096 bytes x1000 committed WAL generations;412,201,000 WAL bytes |898,580KiB peakRSS; open0.692s. |12,784KiB peakRSS; open0.355s. |Same formats/commit/recovery guards, resources and fixture budget. Both pass. |
|Private1,234,207-file original snap archive;873,877,615 raw file bytes; original575,809,314-byte WAL restored before first open |Original archive preserved withSHA256047482f8a1b35fc43625fde2de96cc80240fba855647e255fdcc35693717ae49. |Conservative Docker-start62.504s then normal reopen10.719s. ProcessHWM985,190,400/797,982,720 bytes; cgroup peaks3,093,041,152/763,346,944 bytes; noOOM. |Same300s deadline,4,096,000,000-byte Docker/cgroup cap and2CPUs. Networknone; OS cache not flushed. Four independently verified MPT proofs and the original Geth hash/state root agree. |

The first archive-clone attempt failed a strict cgroup-peak assertion. Its raw peak value was not captured; it remains unknown and the run remains failed. Diagnostics distinguish anonymous process memory and filesystem cache. A new acceptance contract was declared before a new run: unchanged physical cgroup cap, noOOM/kill and processHWM below the cap, retaining cgroup counters separately. The kernel documents temporary memory.max overrun during reclaim ([cgroupv2 documentation](https://docs.kernel.org/6.14/admin-guide/cgroup-v2.html)); the later accepted run also happens to report peaks below the cap. No resource or daemon deadline was raised and the earlier failure is not converted to a pass.

Value residency and WAL working buffers are bounded; total memory is not constant. Startup key/filename metadata isO(keys), replay metadataO(touched keys), working buffers depend on the largest value, and snapshot outputs depend on output size. Known indexed files that disappear remain hard errors. Header/canonical maps, complete head-trie authentication and canonical-prefix walking still grow. No validation is bypassed to obtain faster startup.

## Actual operating duration remains open

**Authoritative follow-up:** rounds03/04 failed, most recently after658 samples /19836.586s. Six signed windows plus a freshly replayed tail authenticate the entire observed73639–80256 span, without qualifying its failed duration. Both old containers are stopped, databases privately archived, and native empty-block import blocking remains under diagnosis. No accepted48-hour run is currently active. [Current results PR132](https://github.com/alt-research/eth2030/pull/132) and [diagnostic PR133](https://github.com/alt-research/eth2030/pull/133) supersede the historical attempt snapshots below.

The final source runs as a new validator-free observation pair against the unchanged original producer/validator network. The first deployment omitted Lighthouse's explicit entrypoint and never qualified. After correcting that harness, the original bcf9 preflight remained optimistic and was interrupted for the separate progress correction. The first2c0df12f pair became non-optimistic in about60s, matched original Geth finalized commitments and passed independent MPT consumers; it then lost everyCLpeer and the controller failed/stopped it. Logs show two validator BNs sendingToo many peers, followed by backfill rate limiting and a ban from the last provider. A short successful catchup is not a completed duration test.

A new predeclared profile adds the existing full/snap observerENRs, including a provider measured with two peers. Only the new CL's seeds change: original node configuration, peer scoring, hash/signature checks, historical sync, resource limits and acceptance duration are preserved. The same owned databases and identities are retained; no ban/identity reset hides the failed run. Its duration starts from zero after catchup. All failures and the new profile remain separately recorded.

The owned controller requires172,800 real seconds,30s samples, non-optimistic/synced CL, independent Geth finalized hash/root agreement with at most128-block lag, finalized progress within600s, repeated independent MPT consumers, stable actual binaries/limits, noOOM and a20GiB shared-disk stop margin. Logs are capped at100MB x3 per role. It stops only its labeled observer containers on failure/completion; no original validator keys are mounted. Whole-span independent BLS finality is a separate required follow-up; the old reference tool's4096-link bound must be respected with authenticated contiguous chunks rather than silently widened.

The corrected seed profile initially advances with four peers. Its finite64059–64079 prefix independently authenticates21slots through512 BLS participants and38 parent links, with no missing slots; public offline replay retains the original bytes. The companion collector now captures contiguous windows hourly and replays each under the unchanged4096-link bound. Both final operating duration and complete contiguous finality coverage remain required.

Live records are under/projects/eth2030/qualification/remaining-code-20261122/live-soak-03/run; the public archive is an explicitly dated snapshot of the attempts, not a mutable live result or a claim that future48h work completed. If the revised profile fails, preserve it and diagnose the cause; do not restart unchanged just to search for a pass.

## Remaining gates and next actions

| Item | Status / concrete next action |
| --- | --- |
|GAP-01/GAP-06 |Merged in master55f7fd84 with package bytes equal to functional2c0df12f; No replacement legacy protocol is invented. |
|GAP-03 |Value/WAL residency and private million-file recovery measured; actual48h and independent final-span BLS remain pending. Inspect cardinality/history growth before broader capacity claims. |
|GAP-02 |Lodestar10693 upstream lockfile/full hosted CI/merge remains maintainer-owned; local/fork controls do not make that pipeline green. |
|GAP-04 |Public gas/model/compatible fork tuple and activation approval remain reserved by the user. Closed-unmerged EIP alternatives adopt no replacement by themselves. |
|GAP-05 |Pinned native/RPC corpora are finite; extend for concrete adopted formats or uncovered consumers. Do not conflate native and Geth-backed totals. |
|Later forks/services |Independent Frame/Bogotá agreement, Dora consumers, paymaster cutover, public binary/PQ/proof/native-rollup work retain their separate roadmap gates. |

The [public evidence and replay instructions](../../reviews/remaining-code-closure/artifacts/README.md) distinguish verifiable public captures from private database input. Reproduction of the full database timing requires the private original archive; public proof/header/CL replay alone does not reproduce all storage files. Earlier nine archives and the245-file post-merge archive remain unchanged.
