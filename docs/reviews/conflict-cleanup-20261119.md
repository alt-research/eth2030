# Shared tracking conflict cleanup validation

Target master: `98716284b049933888bcd68d64c1957004d759b2` (PRs 64 and 65 merged). Scope: existing Glamsterdam EL reviews 66–87. No upstream PR is merged by this cleanup.

## Changes and retained records

All branches share identical root CHECK/STATUS content and the same tracking policy ancestry. Feature tracking is relocated to `pr-<number>/CHECK.md` and `STATUS.md`; the exact original root diff is retained in `legacy-tracking.patch`. Existing detailed verification evidence is preserved. PR 83 remains Draft. Branches are updated without rebasing, force-pushing or importing another unmerged feature into an independent review.

Master imports PR 65 into reviews 66–81. Reviews 82–87 already contain its functional source, so their source trees remain unchanged. All source results match Git automatic three-way merging, except the reviewed adjacent conflict regions in 71 (retain both Amsterdam ETH logs and EIP-7954 activation) and 72 (initcode validity before BAL tracking, then original seeding/reconciliation). Review 82 retains the newer dated PR 81 evidence from master; it does not gain a new source repair by importing that document.

## Validation

Every review passes `go test -count=1 ./core/config ./core/vm ./core/execution ./txpool` and `go build ./...` from `pkg/`. Reviews 71 and 72 also pass race and vet for those four packages. Exact original heads, validated source trees, commands, exit codes and local log paths are in the [machine-readable report](conflict-cleanup-20261119.json) and each review directory. All original tracking diffs match their archived bytes; relocated evidence links resolve.

## Merge boundaries

All 22 branches merge cleanly against the target master after this cleanup. Shared root trackers are byte-identical, so merging one cannot introduce another competing feature entry into those files. This does not guarantee that later functional merges are automatic.

A full dependency-ordered virtual merge dry run passed reviews 66–71, then found a genuine conflict in `pkg/core/vm/instructions.go` at review 72 after review 71. The shared trackers were clean. The dry run stops there without discarding either implementation or changing a branch ref. That ETH-log/SELFDESTRUCT overlap needs explicit functional review when integrating those features; later merge steps are not claimed tested. Keep the existing [dependency review](../plans/glamsterdam/MERGE_REVIEW.md), including Draft 83, and recheck each actual target change.

Publication uses one non-forced atomic push to the existing fork branches. Verify the exact live heads and GitHub conflict state afterward. CI is rerun on the updated heads; pending/failed checks and approval remain separate from conflict resolution. No CI success or release qualification is inferred here.

| Review | Original head | Validated merge head before final shared documentation | Source conflict resolution |
| --- | --- | --- | --- |
| [#66](https://github.com/alt-research/eth2030/pull/66) | `632853ed1307` | `426260ee675a` | Automatic source merge |
| [#67](https://github.com/alt-research/eth2030/pull/67) | `8bac2058c11f` | `31b7436fd21b` | Automatic source merge |
| [#68](https://github.com/alt-research/eth2030/pull/68) | `323bf8fda9bf` | `90d613344196` | Automatic source merge |
| [#69](https://github.com/alt-research/eth2030/pull/69) | `5ab6ce83f58f` | `cf752df1ab1f` | Automatic source merge |
| [#70](https://github.com/alt-research/eth2030/pull/70) | `86a3322133a0` | `b303319245dc` | Automatic source merge |
| [#71](https://github.com/alt-research/eth2030/pull/71) | `4d82484fe510` | `ade917c1295c` | pkg/core/config/chain_config.go |
| [#72](https://github.com/alt-research/eth2030/pull/72) | `8cf3f9afde17` | `53e3bd21ecac` | pkg/core/execution/processor.go |
| [#73](https://github.com/alt-research/eth2030/pull/73) | `3ec63b64dd2b` | `44020ad0866b` | Automatic source merge |
| [#74](https://github.com/alt-research/eth2030/pull/74) | `2505e9947e2d` | `a481b8cf4cd2` | Automatic source merge |
| [#75](https://github.com/alt-research/eth2030/pull/75) | `3ee4dae928a8` | `3cf6558589ab` | Automatic source merge |
| [#76](https://github.com/alt-research/eth2030/pull/76) | `bc2f58506b63` | `57f49a9696ae` | Automatic source merge |
| [#77](https://github.com/alt-research/eth2030/pull/77) | `6c86693815e1` | `910a91514a4a` | Automatic source merge |
| [#78](https://github.com/alt-research/eth2030/pull/78) | `9ad2d9d61bd0` | `3b287c7cf8be` | Automatic source merge |
| [#79](https://github.com/alt-research/eth2030/pull/79) | `901ec06331bd` | `09120deba942` | Automatic source merge |
| [#80](https://github.com/alt-research/eth2030/pull/80) | `12e22aac2f78` | `74e105d47921` | Automatic source merge |
| [#81](https://github.com/alt-research/eth2030/pull/81) | `6ce3eceeef53` | `357536d565e2` | Automatic source merge |
| [#82](https://github.com/alt-research/eth2030/pull/82) | `08534b6dd5dd` | `7f315a133b2f` | Automatic source merge |
| [#83](https://github.com/alt-research/eth2030/pull/83) | `bd0f33577c0d` | `975e9c6bfcda` | Automatic source merge |
| [#84](https://github.com/alt-research/eth2030/pull/84) | `6ed6f2562cb1` | `1d080908460e` | Automatic source merge |
| [#85](https://github.com/alt-research/eth2030/pull/85) | `eb3ae4f10d9f` | `2e8c48aab9f3` | Automatic source merge |
| [#86](https://github.com/alt-research/eth2030/pull/86) | `7d79bbc5cd61` | `51333b43b3d0` | Automatic source merge |
| [#87](https://github.com/alt-research/eth2030/pull/87) | `0a1aa9374c06` | `0380be9d9cfb` | Automatic source merge |
