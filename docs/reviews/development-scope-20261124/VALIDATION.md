# Development-accounting validation - November 24, 2026

## Fresh checks

All commands ran in the isolated development-completion worktree; the original checkout was preserved.

- `GOTOOLCHAIN=go1.26.11 go build ./cmd/eth2030 ./cmd/eth2030-geth`: passed. The native and Geth-wrapper entrypoints remain distinct.
- `GOTOOLCHAIN=go1.26.11 go test ./core/config ./node -run 'TestCanonicalForkScheduleDependencies|TestConfigCompatibilityIsSymmetricAtEitherActivation|TestApplyForkOverrides_|TestNodeForkOverridesDoNotMutateNetworkDefaults|TestActualNodeRestartRejectsHistoricalConfigChanges|TestEveryMutableScheduleFieldHasCompatibilityAuthority' -count=1`: both packages passed. These cover schedule dependencies, either-side activation compatibility, genesis overrides, isolation of network defaults and actual persisted-node restart rejection.
- `npm ci --no-audit --no-fund`: passed.
- `node --experimental-strip-types scripts/verify-eips.ts --files`: 149 entries, 398 source paths, zero missing paths and no blocking discrepancy. One pre-existing EIP-7885 inventory note remains non-blocking.
- `npm run build`: passed, all 157 static pages exported, including homepage and roadmap.
- `node scripts/check-export.mjs`: passed. The homepage and detail page share identical development accounting, exclusions, historical failures and source-pinned evidence links.
- Twelve executed record/render mutations were rejected by assertion failures; exact inputs were restored and the baseline export check passed again. [Machine-readable controls](NEGATIVE_CONTROLS.json).
- `git diff --check`: passed before submission.

The first new export check caught nested-section interference with the existing homepage-card extraction. The new labelled subregions use div elements; the final export includes all fork cards and passes the existing guards. No guard was removed to conceal that failure.

## Integration and evidence boundaries

The current branch, merged master cfd39f3f and measured cumulative source 50514bf3 have identical pkg tree 48afbc345e9418d562be901c13e2e54eaae500b9. No Go implementation changed here, so the previously retained 219-package and seven-package race receipts are not claimed as newly executed tests. The prior master Pages deployment run [45220592334](https://github.com/alt-research/eth2030/actions/runs/45220592334) succeeded; that deploys the previous page, not this unmerged update.

The final website points to evidence commit 1eca90d7b9f4dfdd4354156b7f660dab910520c2. Dated November 23 receipts retain their historical failed, unconfirmed and unmerged statuses. Current merged PR records and user-directed exclusions are in CURRENT_STATE.json. No fresh 48h/larger-scale test, upstream approval, public activation or original-04 root-cause resolution is asserted.

PR and exact-head hosted CI are checked after submission; those results remain attached to the PR rather than being confused with local export checks or a new public deployment.
