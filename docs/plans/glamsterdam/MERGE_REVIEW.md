# Glamsterdam PR merge review

Snapshot: 2026-11-19. Fetched `origin/master` is still `38ea9259901c7873f483752ea390962e26563648`. This document prepares a review decision; no upstream PR is merged by this task. [Exact heads, states and dependencies](merge-review-snapshot-20261119.json).

## Current state

All EL reviews 64–87 are open. PR 83 is Draft; the rest are not. [Tracing PR 87](https://github.com/alt-research/eth2030/pull/87) is published with its own evidence. Lighthouse reviews 1–3 are open against `unstable` in the independent CL repository.

The stable EL development composition through reviews 65–82 and 84 is published at `b1a696d1`. Gas 83, RPC 85, simulator 86 and tracing are separately qualified development branches. Their full local suites do not mean that upstream master has received those changes or that the full release matrix is complete.

Queries of the **fork head** status/check APIs for reviews 83–86 return `pending` with zero status contexts and zero check runs. This means no positive CI evidence was observed there, not that an actual CI job is running or has failed. Recheck required target-repository PR checks and branch protection immediately before merging. Local test results remain separate evidence.

## Dependency order

The history is stacked. In particular, the later features already contain many earlier code commits. Merging only a top PR would implicitly import its dependency changes, including Draft 83. Review-only foundation compare links in each PR separate the actual feature delta from that ancestry.

| Stage | Reviews | Dependency and review condition |
| --- | --- | --- |
| Documentation | 64 | Updated plan, source decisions and evidence inventory; keep CHECK/STATUS conflicts scoped to documentation. |
| Foundational EL changes | 65–79 | Review their own feature deltas. They supply limits, retention, factory, access discovery, networking, logs/lifecycle/opcodes, slot/schema/configuration and blob/request validation. |
| Blob schedule | 80 | Includes functional ancestors 76 and 79. |
| Snap state catch-up | 81 | Includes functional ancestors 69 and 77. |
| Sparse relay aggregate | 82 | Already includes the earlier feature foundation plus 80. Inspect its review-only delta instead of assuming it is independent. |
| Canonical system transitions | 84 | Based on the composition including 81 and 82. |
| Current merged LIFO | 83 | Depends on 84. The user selected this development model with final design review retained; review the coherent gas change and explicitly resolve its Draft status before a dependent merge. |
| Method consumers | 85 | Depends on 83; actual call/estimate/access-list and canonical opcode replay. |
| Stateful simulation | 86 | Depends on 85; includes systems/withdrawals/requests/BAL, isolated state and bounded simulation. |
| Nested/dimensional tracing | 87 | Depends on 86. Review its actual hooks, two EVM frame-boundary fixes, proposed wire fields and validation evidence before the final stack merge. |

For an auditable bottom-up merge, preserve merge ancestry rather than squash/rebase each overlapping feature independently. After every target change, recompute the remaining PR diffs and confirm the intended delta. A single consolidated candidate is an alternative review strategy, but choosing/creating/merging that replacement is a separate decision; do not accidentally combine both strategies.

The top candidate contains all recorded functional sources for reviews 65–86. The initial snapshot had four latest **review heads** outside the candidate. After simulator publication composition, three (73, 74 and 84) remain outside the final candidate because their functional commits are already included while later review/publication documentation commits differ. This is not missing runtime implementation. Reconcile those documentation heads and any remaining PR auto-close state explicitly; do not duplicate or discard code based on a head-only ancestry test.

## Gates before a merge decision

1. Choose the merge strategy and exact target heads; confirm Draft 83 is intentionally included. No final mainnet activation time is invented.
2. Review the execution changes in 83 as one coherent LIFO model. Keep unmerged 12581/12582 alternatives excluded. Final gas/application review remains separate from this explicit development choice.
3. Review tracing's EIP-211 return-buffer and EIP-214 read-only precheck repairs. Existing raw EOF-interpreter tests are preserved; the actual call-boundary regressions, independent SDK traces and affected/full tests qualify the intended behavior.
4. Keep the API boundaries explicit: execution-apis 852/855 remain open; 976's alternative nested state field is excluded. The simulator's conflicting prose extraData default is recorded, not silently called settled.
5. Run required target-repository CI on the exact final merge candidate. Retain full Go/tagged/race/vet/build and meaningful SDK/mutation/actual-consumer evidence; resolve genuine failures rather than relying on PR counts or stale green states.
6. Reconcile Lighthouse 1–3 separately against its target `unstable` and pair its exact resulting revision with the EL candidate. An EL merge cannot close CL authentication, optimistic import, payment, SSZ or partial-network tasks.

## Work still required for release acceptance

The 33-task development plan is still incomplete. Official/native fixture and deployed-application coverage, complete historical/API fixture compatibility, full CL lifecycle, independent multiclient/builder devnet, cold sync/restart/reorg/retention campaigns and measured latency/memory/disk/queue operating budgets remain open. Development code may be reviewed for merging without representing those unrun release gates as passed. Complete Glamsterdam or mainnet readiness is not inferred from these reviews.

The original EL checkout remains `feat/track2-hegota` at `6f60df69`, with its pre-existing submodule/generated dirt preserved. Review and integration work use isolated branches/worktrees and exact owned paths.

Final tracing candidate: `0a1aa9374c061aacfb429fbfd1942d17c5faf1e5`. Its functional source is `9700ed6b`, composition `217e0164`; full 212/tagged 27/race 48/vet/build, nine SDK classic trees, seventeen mutations and the five-package composition repeat pass. The candidate descends from the fetched master; that ancestry check is not a replacement for review or required target CI. The user requested stopping after this task; no further implementation or upstream merge proceeds.
