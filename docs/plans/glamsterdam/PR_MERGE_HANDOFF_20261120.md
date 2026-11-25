# PR merge and user-stop handoff — 2026-11-20

> Target-branch and merge-order instructions below are historical. The subsequent user-requested [direct-master delivery correction](PR_TARGETS_MASTER_20261120.md) opens PR 115, routes every open PR to master and supersedes the earlier stacked-base procedure. Original evidence and unfinished release gates remain retained.

The current official RPC work is submitted as [PR 114](https://github.com/alt-research/eth2030/pull/114). All **27 open EL PRs** are clean against their fetched current declared targets, and the complete dependency-ordered **merge-commit** simulation has **zero conflicts**. Work stops here as requested; no EL PR has been merged into master.

This is a merge-preparation result, not complete Glamsterdam release acceptance. [PR 113](https://github.com/alt-research/eth2030/pull/113) remains draft and carries the explicit unfinished gates below. Draft reviews 92, 100 and 101 also retain their status.

## Exact scope and verification

Master is `5fbc7c133b0e57352ca2a3d6fdd9a63726c42502`. The [frozen audit](../../reviews/pr-113/MERGE_AUDIT.json) records every head/base SHA, dependency order and both sets of 27 merge-tree results. The [GitHub snapshot](../../reviews/pr-113/GITHUB_MERGEABILITY.json) reports `mergeable=true` for every PR. Some master-targeted PRs report `mergeable_state=blocked`; that does not denote a content conflict and does not waive separate CI/review requirements.

The embedded snapshot uses PR 113 source `60ec1be23fb9269ac6e989f8655a1db656924f5a`, immediately before this handoff-only documentation commit. Its production and test tree is byte-identical to fully qualified composed source `247d2f87`; only review documents differ. The final published documentation head is checked again after push, and its exact result is recorded in the PR description. This avoids embedding a commit's own hash in that commit.

Checks use actual fetched base branch refs. GitHub's cached PR base SHA for reviews 93 and 106 lags behind those branch refs, so cached base metadata alone is insufficient. No temporary audit commit is published, no branch is reset, and no merge into master is performed. Repository settings permit merge commits; settings were not changed.

## Merge procedure and conflict prevention

Use **Create a merge commit**, retaining the original feature ancestry. The verified technical order is:

```
88 → 89 → 91 → 92 → 93 → 94 → 95 → 96 → 97 → 98 → 99 → 90
→ 100 → 102 → 103 → 104 → 105 → 101 → 106 → 107 → 108
→ 109 → 110 → 111 → 112 → 114 → 113
```

1. Review the individual functional deltas and Draft status first. This sequence does not itself approve a Draft PR or complete a release gate.
2. Once a dependency is merged, retarget its remaining children to the surviving integration target (normally master) before merging them. Verify that the original parent head is an ancestor of that target. Retain base branches until dependents are retargeted; do not lose the recorded ancestry through automatic branch deletion.
3. Recheck the current PR diff and mergeability after each target change. The simulation accumulates original heads into master in the order above using merge commits with both parents. Arbitrary feature order or future head changes require a fresh check.
4. Preserve shared trackers through the integration owner; keep new feature tracking in `docs/reviews/pr-N/`. Do not independently rewrite root CHECK.md/STATUS.md on every feature branch or repeatedly merge unrelated tracker-only commits.

**Do not repeatedly squash/rebase this existing overlapping stack.** The [executed squash control](../../reviews/pr-113/SQUASH_CONFLICT_CONTROL.json) first conflicts at PR 93 in `STATUS.md`, `docs/plans/glamsterdam/COMPLETION_PROGRESS_20261119.md` and `pkg/core/vm/interpreter.go`. This is loss of shared ancestry, rather than an unresolved conflict in the current heads. There is no need for a broad force-push/rebase campaign: both current-base and normal cumulative merge checks are clean.

The conflict result is bounded by these frozen heads, merge method and order. It is not a guarantee for arbitrary later commits, squash merges or acceptance of unreviewed Draft work.

## Current RPC work and validation

[PR 114](https://github.com/alt-research/eth2030/pull/114), head `ae6e0eb5121f0e3dca036ac0a4c19fec559e6a1a`, is based directly on PR 112, independently of unfinished PR 113. Production/test source is `ad38817f153072421d7b3a023dfbce9f44088048`; its later commit changes review documentation only.

The unchanged official execution-apis corpus at `a9e9095c453d7b3c95b89abfcf46afb2eff4e1b7` passes **107 files / 108 actual native HTTP/RPC/EVM requests**: call 6, estimateGas 7, createAccessList 4, simulateV1 91. Corrections cover historical fee/fork context, ABI revert output, typed errors, verified gas estimates, fork-correct P256, small EIP-1559 base fees, journalled complete storage overrides and historical simulation response behavior. [Exact results and original wire replies](../../reviews/pr-114/RESULTS.json) are retained. Expected upstream outputs were not weakened.

Authenticated fixture pre-states are independently reconstructed and rehashed to their original roots. This finite RPC qualification does not claim native PoW import of the fixture chain. Genuine native import/admission/process evidence is recorded separately.

The independent feature branch passes **217 full package suites**. Original composed source `247d2f87` passes **218 full suites, 24 affected race suites, 49 tagged integration suites and full vet**; its extra package is the unrelated offline builder-ledger tool. Four executed/restored behavior mutations detect the old fee floor, early P256, lost revert bytes and missing storage journal. All nine GitHub CI jobs pass at both production/test source `ad38817f` and latest documentation head `ae6e0eb5`; the [latest check snapshot](../../reviews/pr-113/RPC_CI.json) retains the individual job links. The final composed native binary also builds locally at `60ec1be2`.

## Runtime left at the stop boundary

Only the owned enclave `eth2030-glamsterdam-completion-20261119` was maintained. Unrelated enclaves, checkouts and data are preserved.

- Original pair 2 completes exclusive indexing of 10,857 canonical historical roots, resumes the normal native executable at source `14c5289c`, then resumes Lodestar `cf705b46` and its validator. It uses the declared corrected Lodestar memory budget; the original failed budget and OOM observations remain retained.
- Final [read-only original-pair health](../../reviews/pr-113/USER_STOP_HEALTH.json), captured at 2026-11-20 02:42:29 UTC, shows all three CLs at slot 13078, `is_syncing=false`, `is_optimistic=false`, `el_offline=false`, distance zero. All three ELs report head 11690 and agree with independent Geth on finalized block 11603 (`0x2d53`), hash `0xe7af8ad9b281e834a2a33a1272f7a355905c88a0b1a28a51b62b36376b17ff28` and state root `0xc496f4cc748dc90713e6a2aec8448a6197ef3204cc49bb0ca617cadba222daaf`.
- Original pair 3 remains at native `1bba8a12`; it was deliberately not upgraded after the stop instruction. Original Lighthouse daemon source remains `6666199`; later source/test integrations are not relabelled as deployed binaries.
- Completed auxiliary cold/full/finalized-snap roles are [stopped with their data retained](../../reviews/pr-113/USER_STOP_AUXILIARY_ROLES.json). The original three validator-bearing pairs remain running. No new workload, steady-window, pressure or storage-fault campaign is started.

Lighthouse source/test work is directly pushed to the authorized `eth2030` branch through `42f6fc26e`. The qualified auxiliary production binary remains `d547ba9`; later actual-SSZ negative controls are test/documentation changes. The independent transport fix remains [rust-libp2p PR 579](https://github.com/sigp/rust-libp2p/pull/579), and the independent-client fork-boundary correction remains [Lodestar PR 10678](https://github.com/ChainSafe/lodestar/pull/10678). Neither external PR is represented as merged upstream.

## What remains unfinished after stopping

| Area | Completed evidence retained | Remaining acceptance / deferred work |
| --- | --- | --- |
| R01–R04 EL execution/API | Authorization, VM and admission corrections; 31,573 modern plus 1,223 historical native state cases; actual 31-transaction/17-query application sequence; finite 108-request official RPC corpus; historical, trace and cancellation consumers. | Reconcile the complete declared API/version/schema matrix and source tuple before release. Do not infer unexecuted blockchain-fixture formats or ecosystem-wide compatibility. |
| R05–R06 CL/lifecycle | Explicit fork tests, complete-state-before-observe guards, actual SSZ/BLS/KZG consumers and authentic negatives. | Finish the declared process-level FULL/EMPTY/PENDING, late payload, reorg, invalidation and restart matrix; freeze deployed sources. |
| R07 settlement | Independent actual signed state transition 9600→9601, exact 0.101-ETH debit/credit with three-EL execution proofs, registry lifecycle and signature negatives. | Remaining full lifecycle/process combinations and final tuple reconciliation. One authenticated transfer is not whole-chain lifetime payment or total-supply accounting. |
| R08 proof migration | Actual 384-validator Rust/reference parity, signed bootstrap and period-one 512-participant finality, 462 ancestry links authenticating the observed payment; replay tools and original bytes published in PR 101. | Reconcile the final deployed-source tuple and complete recovery/release packaging. |
| R09 peer/custody | Actual two-row blob across three CLs; native encrypted TCP partial-peer delivery/reconnect; actual-state bitmap/row negatives; independent KZG. | Observe the remaining full-process partial-column acquisition with getBlobs disabled and complete version/group/custody/fault composition. Component TCP/SSZ success is not that process observation. |
| R10 multiclient | Actual three-pair common finalized chain, winning/revealing external builder, applications, blobs and independent signed payment/finality consumers. | Complete remaining lifecycle/peer combinations and final canonical runtime/resource tuple. |
| R11 sync/durability | Fresh independent-peer snap 273.973s under 600s; fresh full sync and original application-history readback; local restart 28.459s under 300s; WAL recovery and verified MPT history; original pair 2 restored. | Pair 3 final upgrade plus retained BAL/history, orphan/nonfinality/pruning and process-level storage-failure matrix. Original failed snap and restart controls remain failed observations. |
| R12 operating budgets | Original steady/latency/memory observations and cancellation corrections retained. | New final-tuple steady, valid/adversarial pressure and capacity campaigns. Original declared 20-minute window failed: 129/400 missing slots (32.25% > 10%); manual pubsub recovery does not turn it into a pass. |
| R13 release | Versioned evidence, independent replay tools, separate feature PRs and this merge handoff. | Complete all prerequisite gates, select/freeze a compatible tuple, finalize upgrade/recovery runbooks and release decision. |

The selected development gas model remains the user's current merged LIFO specification. EIPs 12581/12582 were rechecked open/unmerged; final model/compatible-target review remains open. This stop does not silently adopt an alternative or waive an obligation.

No remaining item above is started automatically. Further implementation waits for the user's next instruction after PR review.
