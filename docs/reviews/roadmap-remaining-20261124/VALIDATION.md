# Validation - roadmap remaining-work refresh

- `npm ci --no-audit --no-fund`: passed in the isolated worktree.
- `npm run build`: passed, all 157 static pages exported with TypeScript checks.
- `node --experimental-strip-types scripts/verify-eips.ts --files`: 149 entries, 398 source paths, zero missing and no blocking discrepancy; pre-existing EIP-7885 matrix note retained.
- `node --experimental-strip-types scripts/check-export.mjs`: passed. The guards compare all rendered non-complete IDs and planned-only entries with actual EIPS source, check evidence paths and completion criteria, preserve frozen-scope accounting and reject stale integrated-CI wording.
- [Eight executed mutations](NEGATIVE_CONTROLS.json) rejected missing completion criteria, missing planned scope, nonexistent evidence, contradictory frozen TODO count, acceptance recounting, omitted historical inventory entries, stale open-CI text and Pages runtime drift. Exact inputs were restored and baseline passed again.
- [Ten actual Playwright route checks](BROWSER_CHECKS.json): homepage, roadmap, ePBS, fast-confirmation and canonical-guest detail at 1440px and 375px. No uncaught page errors or horizontal overflow, including all expanded workstreams and inventories. Every details disclosure opens, completion criteria become visible and the homepage remaining-work anchor navigates correctly. Both screenshots were visually inspected. The first automation attempt checked client navigation before the URL settled; the corrected check waits for the actual URL and visible destination.
- `GOTOOLCHAIN=go1.26.11 go run ../docs/reviews/roadmap-remaining-20261124/canonical-observation.go` from pkg: compiled and confirmed standalone structural acceptance of 256 zero bytes; [receipt](CANONICAL_OBSERVATION.json). This is a research-helper observation, not a new qualified devnet test or a failure of the production signed Gloas route.
- `git diff --check`: passed before submission.

The new inventory guard loads actual TypeScript EIP data. Pages previously ran Node 20 while PR checks used 22; both now explicitly use Node 22 with type stripping, avoiding a deployment-only script incompatibility introduced by source-level verification. The workflow guards cover both configurations and new review/plan inputs.

No Go implementation or normative fork schedule changes. Retained runtime/BLS/failure evidence and deferred acceptance stay at their original sources. No new 48h, larger-scale, independent Frame, production Dora/service or upstream approval result is claimed.

[PR 149](https://github.com/alt-research/eth2030/pull/149) targets master. Initial head bc358eb5e1088d7ddd0bd96aeac0fe6aa2631912 passed [hosted site run 45240017399](https://github.com/alt-research/eth2030/actions/runs/45240017399); GitHub and local merge-tree checks reported no conflicts. The subsequent research-detail/handoff commit is verified again on the PR and is not attributed to this earlier run.

The current-specification repair table adds independently targeted reviews (EL 150/151 to master, Lighthouse 16 to eth2030) and audit PR 152. The 157-page build, tracker/export guards and all ten desktop/mobile browser routes were rerun after this addition, including three visible repair cards and the CL target label. Two further executed mutations reject a wrong CL target and a wrong audit URL; exact source was restored and the baseline passed. SPEC_REVIEW_LINK_CONTROLS.json retains those controls. No CI/merge state is inferred from a static label; each row links the actual review.
