# Pre-merge decision record

Captured 2026-11-23T03:02:28.940177+00:00. No target branch was merged.

## Decision

The ETH2030 runtime queue is technically ready for the user's merge review at the exact heads below: all nine existing PRs target master, their configured checks pass, and their combined source passes fresh local regressions/builds. This is experimental development-source readiness, not original-incident closure,48h admission, public activation or unrestricted production capacity.

The Lighthouse runtime queue is mechanically compatible and its combined production/test inputs equal the already qualified source, but **its standalone hosted CI is still red**. Do not describe every repository as all-green or approve the entire CL batch on compiler checks/non-recurrence alone. Stage the complete dependency set and obtain its required combined test-suite evidence before target integration. Three older unstable-target CL PRs are already ancestors of eth2030 and are not missing runtime changes; they still conflict with one another upstream.

## ETH2030: each current PR

| PR | Target | Exact head | Configured checks | Review scope |
| --- | --- | --- | --- | --- |
| [#131](https://github.com/alt-research/eth2030/pull/131) | `master` | `720566e8cc1bf71f2f6fb774b727c8106b7d8d79` | 1/1 success | Progress/site evidence; stale merged-state labels corrected in this audit |
| [#132](https://github.com/alt-research/eth2030/pull/132) | `master` | `dc59280c8b7b92ea056b5ea6f2a312373c28a926` | 9/9 success | Strict historical48h evidence/termination gate; pins do not relabel a new tuple |
| [#133](https://github.com/alt-research/eth2030/pull/133) | `master` | `958e407d6fe55e487a8a0bc65ee732ebbcaba70f` | 9/9 success | Explicit goroutine diagnostics on the metrics mux |
| [#134](https://github.com/alt-research/eth2030/pull/134) | `master` | `2558a1924d28e507f97ea38b5bd8005e5a0ae81c` | 9/9 success | Private-clone mature native/Engine/MPT/restart replay, separate from elapsed time |
| [#135](https://github.com/alt-research/eth2030/pull/135) | `master` | `3076dabbcf598832fbe529e9b318047f0a91e3a6` | 9/9 success | Immutable snap-history authentication outside insertMu, no authority bypass |
| [#136](https://github.com/alt-research/eth2030/pull/136) | `master` | `ed4e9705db81a104453a06969a8cf550778c80da` | 9/9 success | Early quiet-pipe Engine-timeout evidence capture |
| [#137](https://github.com/alt-research/eth2030/pull/137) | `master` | `1c68ece80a7bc13930d632024205ec535f433fed` | 9/9 success | Real checked-batch pause fixture; production constructor keeps FileDB |
| [#138](https://github.com/alt-research/eth2030/pull/138) | `master` | `d6d63b4174e99a0873b2646b60f2e2f112df2b1a` | 9/9 success | Durable-before-publication cache/chain selection, fencing and read liveness |
| [#139](https://github.com/alt-research/eth2030/pull/139) | `master` | `cf72af70d85c38ed2968e5a72f3f55dc1cb762ff` | 9/9 success | Operational/BLS review and finite closing wait, including corrupt-report error path |

PR131 intentionally has the site/evidence workflow rather than the package CI. Review-thread and review-submission queries show no outstanding submitted reviews; this audit is not a maintainer approval. Recheck heads/base/checks if anything changes.

## Lighthouse: each current PR

| PR | Target | Exact head | Success / failed / cancelled / skipped | Runtime disposition |
| --- | --- | --- | --- | --- |
| [#1](https://github.com/jiayaoqijia/lighthouse/pull/1) | `unstable` | `9b27d866c2c747abb28b760a8c52c0602c8d0d1f` | 30 / 4 / 0 / 3 | Already included in eth2030; separate upstream review |
| [#2](https://github.com/jiayaoqijia/lighthouse/pull/2) | `unstable` | `b57d54d45ec45b47d940a6689a2f2f8fb39b8f8a` | 31 / 4 / 0 / 2 | Already included in eth2030; separate upstream review |
| [#3](https://github.com/jiayaoqijia/lighthouse/pull/3) | `unstable` | `cb72ff1530715c2323aae17b03ceca2710a4cf66` | 33 / 2 / 0 / 2 | Already included in eth2030; separate upstream review |
| [#4](https://github.com/jiayaoqijia/lighthouse/pull/4) | `eth2030` | `1da38247c0a673c956639052a249b052ef8cbb04` | 22 / 9 / 3 / 3 | Hold standalone green claim; composed dependency qualification required |
| [#5](https://github.com/jiayaoqijia/lighthouse/pull/5) | `eth2030` | `7210741372e113bf0745420d381fa5db2dbe552a` | 32 / 3 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#6](https://github.com/jiayaoqijia/lighthouse/pull/6) | `eth2030` | `40e5d609233a8b4b0a869484e287db1f59b2acd2` | 30 / 5 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#7](https://github.com/jiayaoqijia/lighthouse/pull/7) | `eth2030` | `57d2c2c87a829268f35703df3073483213b4c9e1` | 31 / 4 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#8](https://github.com/jiayaoqijia/lighthouse/pull/8) | `eth2030` | `54e1884460caf56889e35f8e887ba3ce8f74da5d` | 30 / 5 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#9](https://github.com/jiayaoqijia/lighthouse/pull/9) | `eth2030` | `476789481f02d69800db0fa8258d2d6f03a5961d` | 30 / 5 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#10](https://github.com/jiayaoqijia/lighthouse/pull/10) | `eth2030` | `c07689f4fd81e5e0c5645f19b6bef270e7c3ef9c` | 30 / 5 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#11](https://github.com/jiayaoqijia/lighthouse/pull/11) | `eth2030` | `a46dbc2961ff5a9088b7e2adc64648548fe18fca` | 31 / 4 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |
| [#12](https://github.com/jiayaoqijia/lighthouse/pull/12) | `eth2030` | `c8f1f3860367965550c0541ff364180c0438b77c` | 30 / 5 / 0 / 2 | Hold standalone green claim; composed dependency qualification required |

Every CL4–12 head includes actual fetched eth2030 base42f6fc26e5c48c4c1ec16368a0a7df705bcc3e69; CL1–3 heads are already ancestors of that target. Keep unstable/upstream review separate; do not retarget or merge those old changes again merely to make a queue look complete.

Dora and Spamoor have no open PRs. The Lodestar fork has no PR to itself, but related upstream [ChainSafe/lodestar10693](https://github.com/ChainSafe/lodestar/pull/10693) remains open to unstable. Its current target97ba67c17ea8323b8e69d94fbaba76412611217a merges cleanly with the PR head, while only title checks are present and commit status is pending. Full hosted/maintainer lock review is not established.

## Compatibility and source identity

Each runtime group (EL131–139 against current master55f7fd84, CL4–12 against current eth2030) passes72 ordered pairs and2304 representative subset transitions with equal resulting subset trees, plus ascending/descending full sequences. This is confluence testing with representative ancestry, not enumeration of every possible history. CL1–3 individually apply to unstable but every pair conflicts in CHECK.md; CL1/2 also conflict in consensus/state_processing/src/upgrade/gloas.rs. See exact conflict stages in CL_UPSTREAM_PAIR_CONFLICTS.json.

Source comparison confirms the composed CL production/test/configuration tree equals qualified3782df1 when only docs/shared trackers are excluded. Native EL production package files equal the actually executed db4781de build when test files and diagnostic scripts are excluded. Local staged/synthetic commits are simulations, never actual master/eth2030 merges. API base SHAs may be cached at earlier PR updates (e.g. PR131); current fetched target refs are authoritative for the merge tests.

## New findings repaired during this audit

1. PR139's outer unreadable-summary exception path could bypass the3600s finished-runtime closing bound and retry forever. The old-source directed regression fails at the intended beyond-deadline retry assertion. The fix consults the independent valid runtime endpoint, retains failure, writes a failed review and exits nonzero. No node/clock/hash/BLS rule changes.
2. PR131 still described merged GAP-01/GAP-06 fixes as awaiting merge and labelled a pinned review source as unmerged. Correct merged-source copy and plan sequencing without promoting operating/48h readiness; Next static build/export assertions pass.

## Functional review and fresh verification

Reviewed the pending FileDB cache/fence/write-publication paths, canonical/checkpoint/state publication and reorg transaction lookup, cold-reader lock ordering, finalized snap-history frontier, progress snapshot and target ordering, default constructor versus test-only injection, explicit profile route, capture EOF/timeout handling, private replay boundaries, strict evidence/cleanup and final BLS report ordering. No additional production EL blocker was found in those reviewed call chains. This is not a proof that the whole client is bug-free.

Fresh combined checks:219 Go suites; five relevant race packages; full vet/native build;65 Python operational controls and tools controls. PR131 Next build/export checks pass. CL locked workspace/all-target compilation passes on the qualified source. Full standalone CL CI and every EF handler are not represented as passed; see EXTERNAL_CI_BLOCKERS.md.

The retained6h45m run,14 fresh original BLS replays, actual stopped-head SSZ/header/MPT binding and post-5.5h data remain supporting qualification. Their source/measurement boundaries and original failures remain intact. The exact unconfirmed original-fault questions are separated in UNCONFIRMED_ORIGINAL_FAILURE.md.

## Concrete merge procedure

1. Freeze exact heads below and current target tips. Use ordinary ancestry-preserving merges; do not squash away the source identity assumed by retained merge evidence.
2. EL review order:135 →138 →133 →134 →137 →136 →132 →139 →131. All tested orders are compatible; the suggested order groups production liveness before fixtures/tools/evidence. Treat the audit document PR separately and merge it last.
3. Before interpreting new merged master as the qualified runtime, compare its production tree and fixture/CL tuple to the pinned qualification. A merge SHA alone does not create a new measurement.
4. For CL, first stage the complete4–12 source set on an integration branch and run the actual required combined workflow with the pinned Gloas input. Existing per-feature red checks are not waived. Keep legacy1–3/unstable and Lodestar maintainer work separate.
5. Do not infer original04 closure or48h/public-release approval from merge count, zero-event counters or a successful bounded profile. Stop here for the user's merge decision; no target was merged.
