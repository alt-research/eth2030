# External/runtime prerequisite CI that is not green

The ETH2030 EL queue has configured green checks at its recorded heads. The following is separate and prevents an unqualified “all project PRs are green” conclusion.

## Lighthouse runtime PRs4–12

All target eth2030 and merge mechanically into the current42f6fc26 target. Their combined production/test/configuration tree equals the already qualified3782df1 source; a fresh locked workspace/all-target check passes. Their individual hosted workflows still run each feature on a baseline missing other pending fixes, and remain red. The current combined37-job hosted suite has not been obtained in this audit; do not waive it or call local compilation equivalent.

Concrete failures inspected in actual hosted logs:

| Source/job | Observed failure | Relevant repair/dependency and boundary |
| --- | --- | --- |
| [PR4 check-code](https://github.com/jiayaoqijia/lighthouse/actions/runs/44991553711/job/137113679992) | Dependency audit rejects the old libz dependency. | PR6 carries the patch. A baseline failure is still a failure until the required compatible tree is tested. |
| [PR12 check-code](https://github.com/jiayaoqijia/lighthouse/actions/runs/45019793485/job/137199927615) | Clippy rejects manual assign-op in captured_progressive_proofs.rs. | Qualified source contains7bbcc3c60's correction via PR4. Optional installer402 warnings are not mistaken for the actual terminal cause. |
| [PR7 beacon-chain](https://github.com/jiayaoqijia/lighthouse/actions/runs/45017897419/job/137194098646) | Four minimum-version schema downgrade cases fail with OffsetSkipsVariableBytes(12). | PR11 preserves modern/legacy snapshot compatibility and downgrade obligations. Existing corrected selected regressions are separate from this old failing standalone run. |
| [PR5 HTTP API](https://github.com/jiayaoqijia/lighthouse/actions/runs/45008442061/job/137166078818) | envelope_api expects202 but gets400; the run stops early (88 tests unrun). | PR7/8 repair publication binding and mock validated-payload lifecycle. Unrun cases are not passes. |
| [PR12 EF minimal](https://github.com/jiayaoqijia/lighthouse/actions/runs/45019793485/job/137199953300) | fork_upgrade expected all availability bits set while the selected Gloas code clears the fork-slot bit. | PR5 aligns the pinned reference/vector source. Do not change consensus behavior to satisfy incompatible input, disable BLS or suppress a genuine failed result. A bounded retained log prefix identifies this failure; it is not an exhaustive triage of every job/case. |

These are concrete cross-PR dependencies, not permission to relabel red CI as green. Proposed gate: stage exactly4–12, preserve the pinned Gloas source and vectors, run its actual required complete workflow, and retain explicit nonexecuted/failed cases. Do not merge an arbitrary partial CL set into a deployed runtime while citing the composed6h45m result.

## Lighthouse legacy PRs1–3 to unstable

Their heads are already ancestors of eth2030; no runtime implementation is missing merely because the old upstream-facing PR remains open. All are individually mergeable into unstable, but their pairs conflict in root CHECK.md and PR1/2 also in upgrade/gloas.rs. Do not retarget/close/rewrite them automatically during this node merge review. They remain a separate upstream protocol/review queue.

## Lodestar upstream10693

It targets unstable, not EL master. Against the actual fetched97ba67c1 target it merges cleanly, but only two title checks are present and commit status remains pending. Full hosted unit/spec/e2e/browser results and maintainer lockfile review are not established. Its application/fork patch and local15-case/process qualification remain source-specific. No external-PR lock guard was weakened and no upstream merge/approval is implied.

Dora and Spamoor have no open PRs in the related forks at the recorded inventory.
