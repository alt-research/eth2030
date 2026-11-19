# Parallel feature review tracking

Root `CHECK.md` and `STATUS.md` describe shared integration state. Concurrent feature branches must not prepend competing feature records to them. Use `docs/reviews/pr-<number>/CHECK.md` and `STATUS.md` for branch progress, validation and remaining gates. Before a PR number is assigned, use a unique branch-named directory and rename it after publication. Feature verification documents remain the detailed evidence authority.

## Conflict cleanup, 2026-11-19

The batch starts from master `98716284b049933888bcd68d64c1957004d759b2`, after PRs 64 and 65 merged. Reviews 66 through 87 are still open; 83 is Draft. All 22 have root-tracker conflicts. Reviews 71 and 72 additionally have adjacent code conflicts; 82 has older snap verification records than master. The unrelated dependency update 62 is outside this batch.

All updated branches share the same integration-tracking commit and identical root tracker content. Feature-specific additions and modifications are relocated to their own review directories. Each `legacy-tracking.patch` preserves the exact original root-file diff from that review's merge base, including removed text and inherited tracking. The readable CHECK/STATUS extracts retain changed sections; these are historical records, not newly observed test results. Relative evidence links in the extracts are adjusted for their new location.

Merge current master while preserving both code changes. In 71, retain Amsterdam activation for ETH logs and the already merged EIP-7954 helper. In 72, reject oversized initcode before BAL seeding while retaining account seeding and reconciliation for valid execution. In 82, retain master's extended snap evidence as dated evidence from PR 81, without claiming that those source fixes are implemented by PR 82. Existing protocol authority, dependent feature ancestry and Draft status do not change.

## Validation and publication

For every final branch, check root equality, conflict markers, staged whitespace and the full local build. Compare the resulting source tree with Git's automatic three-way merge; only the two explicitly reviewed conflict regions may differ. Repeat core/config, core/vm, core/execution and txpool tests where master adds PR 65 to the source tree. Race and vet checks cover the source-conflict resolutions. Preserve exact original and updated heads and report GitHub's live conflict state after fast-forward publication. Local validation does not replace target CI or approval, and a conflict-free branch may still be blocked by them.

Merge dependencies from the bottom up as described in the dated [merge review](../plans/glamsterdam/MERGE_REVIEW.md). Do not solve tracker conflicts with an unconditional ours, theirs or union merge driver: those can silently lose progress or duplicate records. If master changes again, merge it and apply the same tracker discipline. This cleanup does not merge any upstream PR or qualify a Glamsterdam release.
