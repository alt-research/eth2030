# Group E completion re-audit

> Historical pre-rebase audit. Merged EIP-7805 PR #12470 supersedes candidate
> execution, type-3 enforcement, size exemptions and cache-version assumptions
> below. Use [LATEST_7805.md](LATEST_7805.md), the current criterion plan and
> source-bound qualification for the active rules. Original findings/results
> remain preserved with their recorded source identities.

Date: November 27, 2026 (Asia/Shanghai). Baseline: merged PR #171 at
`cc31afffaa12919bbc08dc854c03af1e1928dc3c`. This audit starts from published
PR #172 head `4663af699c4c9bbc48f59976dc26e00d805a78d6` in an independent
worktree. Unrelated Lean work in the original worktree is preserved.

## Current specification review

The upstream heads remain identical to the pinned verbatim Group E sources:

| Repository | Current immutable revision |
|---|---|
| ethereum/EIPs | `68db81b0a75edb0e16d8f181241f994875eba52d` |
| ethereum/consensus-specs | `36888ee87b0faa6ad7804b017a0184e8e6e6a1c6` |
| ethereum/execution-apis | `9e85b16c7fb54a838c9a612842c138c2bdf9bc8d` |

The [7805 discussion](https://ethereum-magicians.org/t/21578) has 17 posts,
last posted November 21; the [8369 discussion](https://ethereum-magicians.org/t/29298)
has four, last posted September 9. The committee incentive debate does not
replace Heze's committee rules. The 8369 author explicitly retains claimed-index
evaluation: a fresh keyed nonce does not establish whole-transaction position
stability. [PR #12110](https://github.com/ethereum/EIPs/pull/12110), including its
inline review, still leaves final caps, code-load limits, claims and enforcing
integration to a Standards Track extension. Optional draft EIP-8504 PR #12672
remains open at `96da3c17047ad31608c40bfe75049db78dc4417d`.

## Reproduced omissions and fixes

1. **Completed verdicts were deleted at finalization.**
   Bogotá `engine_forkchoiceUpdatedV5` specification point 2.1 requires the
   correct `inclusionListSatisfied` Boolean for a VALID head; its inherited
   Paris handling permits ignoring a stale canonical ancestor without rewinding
   execution finality. Deleting the completed record turned that known head
   into SYNCING. Cleanup atomically moves completed records to a separate
   compact-verdict namespace, releasing old list bytes while retaining both
   true and false verdicts. Subsequent cleanup does not rescan historical verdicts. Unexecuted branches
   below finality are deleted. Native wire tests finalize a descendant, restart,
   query the old dissatisfied head, verify VALID/false and unchanged finality,
   and require invalid checkpoint ancestry to remain `-38002`.
2. **Live CL adapter replacement bypassed memory-cache authentication.**
   Heze gossip requires the dependent state, committee membership and BLS
   signature under the supplied CL context. The disk restore path enforced
   these checks, while an in-memory cache could cross a replacement context
   without them. Replacement now reauthenticates the live snapshot before
   atomically installing the new adapter. Missing ancestry or changed keys
   reject the switch and preserve the old adapter. Successful replacement
   preserves exact bytes, timeliness and the two-message gossip allowance.
   Initial configuration still restores disk data even though Node preallocates
   an empty store.

Both regressions were observed failing against the pre-fix implementation.
Compiled semantic-removal controls independently disable each fix and require
the unchanged native consumer to fail. No compile failure counts as detection.

## Complete task review

| Task | Reviewed implementation and boundary |
|---|---|
| HGT-040 | All 17 immutable source identities, current discussion precedence, timing, execution/satisfaction split and explicit gates remain consistent. |
| HGT-041 | Canonical SSZ, full CL authentication, exact branch-keyed bytes, timely/late/equivocation views, request indices, retention and restart; live adapter replacement now shares the authenticated restore boundary. |
| HGT-042 | Actual Bogotá dispatcher schemas/capabilities, opaque malformed IL bytes, strict DATA and 16-byte custody masks, fork rejection, VALID Boolean/non-VALID null and accepted-view FCU. |
| HGT-043 | IL/pool fixed-point nonce and funding retries, blob/PQ wrapper retrieval, rollback, immutable requests and IDs, same-ID retry after availability recovery, cancellation and deadlines. |
| HGT-044 | Profile 1 ordinary envelope/signature/nonce/fee/code/balance/intrinsic checks, declared gas and exact complete-block append RLP; blob and experimental Frame fallback remain distinct from Informational 8369. |
| HGT-045 | Real two-node Engine build/import, deferred accepted branches, restart, sync admission and finalization; completed stale-head verdicts now survive pruning and restart. |
| HGT-046 | All defined reference predicates and mandatory activation prerequisites remain implemented; Profile 2 consensus activation remains unavailable under D04. |
| HGT-047 | Authenticated native ingress and CL decision adapter, dissatisfaction without execution invalidation, optimistic transitions, branch/timely/equivocation exceptions and inclusive bid bits; external HGT-070/080 remains open. |
| HGT-048 | Canonical PQ byte capacity, real verifier and local signed SSZ/BLS/store pipeline are measured; omitted-witness composition and network tail latency remain unqualified. |

The acceptance register remains nine tasks and 36 criteria, now requiring 30
native consumers. Independent A/B/C/D expected oracles and read-only gitlinks
remain unchanged. Website progress includes the two corrected boundaries;
the existing master-only Pages workflow deploys after merge.

Implementation of all selected, defined rules does not establish complete
public activation. Informational Profile 2 enforcement, omitted PQ witness
transport and independent external CL/ePBS qualification remain explicit gates.
