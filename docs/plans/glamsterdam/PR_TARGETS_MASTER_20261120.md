# Direct-master PR delivery audit — 2026-11-20

This document supersedes the target-branch instructions in [the earlier merge handoff](PR_MERGE_HANDOFF_20261120.md). The user requested a separate master-targeted follow-up for PR 91, an audit/correction of all other PR targets, then a stop. No merge into EL master, force push, branch deletion or additional feature work is performed.

## Outcome

- [PR 115](https://github.com/alt-research/eth2030/pull/115) independently delivers original PR 91 to **master**, retaining its original ancestry. Only the same three CLZ source/test files and their two review records differ from the current master; all other outstanding features remain outside this follow-up.
- All **22 currently open PRs target master directly**. Twenty existing open reviews were retargeted; PR 92 already targeted master, and PR 115 is the new follow-up. Every original head remains in the branch ancestry and existing Draft status is preserved. Review heads advance only through ordinary prerequisite/master merge commits, as described below. PRs 100/101/113 remain Draft; PR 92 was already non-draft when this task began.
- All 22 heads merge cleanly against the fetched current master. A cumulative simulation in the dependency order below also has **zero conflicts**. These are frozen-head merge checks, not approval of every review or completion of release acceptance.
- [PR 99](https://github.com/alt-research/eth2030/pull/99) was redundant: its complete original head is already an ancestor of master through PR 90. It is closed with an explicit delivery explanation; no code is discarded or newly merged. As a closed historical review, its old feature-base label is retained rather than reopening an already-delivered PR merely to relabel it.

The audit baseline is master `209607b9ba1ce81c63b6bb1d1e2a45a6144f14c2`. Before correction, the repository had 22 open reviews, 21 targeting feature branches. Recent merged heads 88/89/90/96 are already in master; original PR 91 is the only recently merged head still outside it. Its corresponding pending delivery is now explicit in PR 115. A merged label on a feature-base PR does not establish delivery to master.

## Safe merge sequence

Use **Create a merge commit** and review in this order:

```
115 → 92 → 93 → 94 → 95 → 97 → 98 → 100 → 102 → 103 → 104 → 105 → 101 → 106 → 107 → 108 → 109 → 110 → 111 → 112 → 114 → 113
```

Every PR now has master as its actual merge destination. There is no longer a dependency on remembering to retarget a child after its parent is merged. Dependencies remain code dependencies: merge prerequisites first, then refresh the child's diff and checks. Until then, a child's master-relative diff legitimately includes unmerged prerequisites. Updated PR descriptions retain a link to each original feature delta for focused review. Existing local feature checkouts are preserved; their remote review branches advance by non-forced merges, so local checkouts may remain behind their origin refs until the owner updates them.

Do not independently squash/rebase the overlapping stack: that discards shared ancestry and can reintroduce duplicate changes and conflicts. The earlier executed squash control remains evidence of that risk. Keep Draft/review/CI requirements separate from mergeability; this routing correction does not promote unfinished acceptance reviews.

## GitHub merge-base reconciliation

Changing the target alone was insufficient: the [first GitHub snapshot](../../reviews/pr-target-audit/RETARGET_ONLY_GITHUB_FLAGS.json) reports 20 dirty reviews even though local merge-tree succeeds. The [diagnostic](../../reviews/pr-target-audit/MERGE_BASE_DIAGNOSTIC.json) records three common ancestors for PR 93. This is an observed local/server discrepancy; it does not establish which internal algorithm GitHub uses.

Normally merge current master and updated prerequisite review heads into the dependent feature heads. This preserves every original commit, makes master and required earlier heads explicit ancestors, and prevents the same ambiguity from recurring merely because a prerequisite is merged first. No rebase, force push or merge into master is used. Each resulting node/source tree is verified against the original head's predicted clean master merge. Where a required current parent includes later PR 101 operator/consumer tools, the only additional paths must match the already reviewed f57b582e blobs exactly; those tool tests and native/tool builds are repeated. The six early reviews pick up only the three already-canonical PR 96/99 timeout/sparse-test files and are checked with affected backend/eth tests and native builds. The other node implementation and protocol tests remain unchanged. Some descendants inherit the nine existing PR 101 tool/script files from their required parent; these are retained existing consumers, not new implementation. Exact merge, path/blob guards and validation records are retained in [HEAD_ALIGNMENT.json](../../reviews/pr-target-audit/HEAD_ALIGNMENT.json).

## Current targets and frozen source heads

| PR | Previous target | Current target | Head | Review state |
| --- | --- | --- | --- | --- |
| [#92](https://github.com/alt-research/eth2030/pull/92) | `master` | `master` | `4d75b6829928` | Ready for review |
| [#93](https://github.com/alt-research/eth2030/pull/93) | `integration/glamsterdam-conformance-foundation` | `master` | `00eac65313de` | Ready for review |
| [#94](https://github.com/alt-research/eth2030/pull/94) | `fix/native-vm-metering-boundaries` | `master` | `222ec7fc710c` | Ready for review |
| [#95](https://github.com/alt-research/eth2030/pull/95) | `fix/native-transaction-validation` | `master` | `36a943c4d07d` | Ready for review |
| [#97](https://github.com/alt-research/eth2030/pull/97) | `test/glamsterdam-native-fixtures` | `master` | `05378fe26fd4` | Ready for review |
| [#98](https://github.com/alt-research/eth2030/pull/98) | `fix/ordinary-transaction-context` | `master` | `af7e96c7a5b9` | Ready for review |
| [#100](https://github.com/alt-research/eth2030/pull/100) | `test/glamsterdam-application-compatibility` | `master` | `cf787c2ba6d3` | Draft |
| [#101](https://github.com/alt-research/eth2030/pull/101) | `integration/glamsterdam-runtime-candidate` | `master` | `27d7f0099e9c` | Draft |
| [#102](https://github.com/alt-research/eth2030/pull/102) | `integration/glamsterdam-runtime-candidate` | `master` | `041cb8dc1da0` | Ready for review |
| [#103](https://github.com/alt-research/eth2030/pull/103) | `integration/glamsterdam-runtime-candidate` | `master` | `5a4e15afa8e5` | Ready for review |
| [#104](https://github.com/alt-research/eth2030/pull/104) | `integration/glamsterdam-runtime-candidate` | `master` | `119ce54c47b1` | Ready for review |
| [#105](https://github.com/alt-research/eth2030/pull/105) | `integration/glamsterdam-runtime-candidate` | `master` | `5eeffb84f46a` | Ready for review |
| [#106](https://github.com/alt-research/eth2030/pull/106) | `test/glamsterdam-multiclient-gate` | `master` | `58b988325008` | Ready for review |
| [#107](https://github.com/alt-research/eth2030/pull/107) | `fix/rpc-request-execution-cancellation` | `master` | `780da97aeabb` | Ready for review |
| [#108](https://github.com/alt-research/eth2030/pull/108) | `fix/websocket-execution-disconnect` | `master` | `0c7a5a2a5371` | Ready for review |
| [#109](https://github.com/alt-research/eth2030/pull/109) | `fix/full-sync-target-progress` | `master` | `c6f95c546c3e` | Ready for review |
| [#110](https://github.com/alt-research/eth2030/pull/110) | `fix/merkle-state-history` | `master` | `a6a7b55183ed` | Ready for review |
| [#111](https://github.com/alt-research/eth2030/pull/111) | `fix/filedb-wal-recovery` | `master` | `a6ed926ac2dd` | Ready for review |
| [#112](https://github.com/alt-research/eth2030/pull/112) | `fix/verified-history-restart` | `master` | `f8bed945bd88` | Ready for review |
| [#113](https://github.com/alt-research/eth2030/pull/113) | `fix/snap-pivot-peer-recovery` | `master` | `e896af6e0ad6` | Draft |
| [#114](https://github.com/alt-research/eth2030/pull/114) | `fix/snap-pivot-peer-recovery` | `master` | `8dc8cab45b35` | Ready for review |
| [#115](https://github.com/alt-research/eth2030/pull/115) | `new follow-up` | `master` | `5dd685790da6` | Ready for review |

PR 99 is deliberately absent from the open-PR table because all its commits are already delivered. Original PR 91 remains historically closed; PR 115 is the new direct-master delivery review.

## Verification and evidence

[Machine-readable results](../../reviews/pr-target-audit/RESULTS.json) record each actual head, dependencies, both sets of merge-tree results, the original merged-head reachability checks, the already-delivered PR 99 and exact PR 115 CI links. [Original targets](../../reviews/pr-target-audit/TARGETS_BEFORE.json), [target changes](../../reviews/pr-target-audit/TARGET_CHANGES.json) and [GitHub target/mergeability snapshot](../../reviews/pr-target-audit/GITHUB_TARGETS_AFTER.json) retain the transition without rewriting historical evidence.

The embedded snapshot records the aligned PR 113 head `e896af6e`, immediately before publishing the task checklist/report commits. All 22 GitHub `mergeable` flags are true at that snapshot. All seven PR 115 CI jobs pass at 5dd68579; synchronization CI on the 21 aligned review heads is separate and remains in progress. Subsequent PR 113 changes are task documentation only, and the final published head is rechecked after push. Its exact final audit is recorded in that PR description rather than embedding a commit's own hash in itself.

At the new master/CLZ composition, uncached core/vm, core/execution and node tests pass; the full core/vm race suite, scoped vet and a local native executable build pass. The original three source/test files match original PR 91 byte for byte. The existing EIP-7939 test exercises real native and unchanged Geth EVMs for eight boundary inputs, exact gas, Osaka/Amsterdam activation and Prague rejection. No new implementation or weakened expected outcome is introduced. All seven new-PR CI jobs pass and their links are retained in the machine-readable snapshot; older full-corpus results are not relabelled as a new campaign.

Only merge preparation is completed. The broader Glamsterdam lifecycle, peer, storage-fault, steady/pressure, final-source-tuple and release acceptance gates remain as documented in the prior handoff and draft PR 113. The original dirty checkout, running original devnet and all retained data are untouched by this task. Work stops after the final published-ref verification.
