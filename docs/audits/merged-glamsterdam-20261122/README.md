# Merged Glamsterdam progress and remaining gaps

**Follow-up:** The findings below are the original merged9a094830 observation. [Current corrective delivery and remaining acceptance](../../plans/glamsterdam/REMAINING_CODE_CLOSURE_20261122.md) records the user-approved retirement of legacy extensions, transactional startup and bounded FileDB residency in separate PRs. Preserve the original reproduction against its original revision; those retired Go helpers no longer exist on corrected source.

Reviewed November 22, 2026 against master `9a094830` after PRs [123](https://github.com/alt-research/eth2030/pull/123), [124](https://github.com/alt-research/eth2030/pull/124) and [125](https://github.com/alt-research/eth2030/pull/125) merged. `pkg` and the code-CI workflow are byte-identical to qualified composition `d9201b0`; merge commit metadata does not relabel the earlier running binary. Exact observations are in [MERGED_EVIDENCE.json](MERGED_EVIDENCE.json).

## What is delivered

The original [33 task IDs and R01–R13](../../reviews/glamsterdam-final-campaign/TASK_RECONCILIATION.json) are complete for the selected merged LIFO / signed Gloas development target. All 18 scheduled core EIPs, five supporting networking protocols and SSZ dependencies are accounted for. The original nine archives and the final 245-file post-merge archive remain source-specific and unchanged.

[Final results](../../reviews/glamsterdam-postmerge-20261121/RESULTS.json) cover 219 fresh packages, 107 official RPC files / 108 requests, six affected race suites, full vet/build, a byte-identical second binary, four original-datadir recoveries within 300 seconds, three automatic provider recoveries, 400/400 steady and 60/60 pressure slots, independent finality and 270 pressure proofs. The final external-builder two-row blob passes 544 KZG cell proofs. Historical state/factory/BAL readback, independent Geth cold full/snap sync, moving-pivot/restart controls and original failures retain their exact earlier sources.

This pass verifies merged source and inspects remaining consumers; it does not claim a new live campaign or re-run every older native fixture on the merge commit. Existing measured results are not extended to arbitrary hardware, public-network activation, all future drafts or later forks.

## Confirmed code gap: legacy Builder Engine extensions

**GAP-01 · P1 · Code change required before advertising these extensions as authenticated builder interfaces.**

The actual exported call chain is `EngineAPI.RegisterBuilderV1` / `SubmitBuilderBidV1` → `BuilderRegistry.RegisterBuilder` / `SubmitBid`. It is dispatched by the authenticated Engine server in `pkg/engine/handler.go` and constructed on the ordinary node in `pkg/node/node.go`. Registration ignores the supplied signature and supplies `MinBuilderStake` locally; the local registry does not prove an on-chain deposit. Bid insertion checks local registration/status/value/hashes, with no BLS signature verification. `GetPayloadHeaderV1` creates another bid object without preserving its signature or resolving its registered builder public key. Separately, the generic `epbs.ValidateBuilderBid` helper permits zero-key/signature inputs.

A [minimal executable observation](legacy-builder-observation.go), using the real exported methods with no execution backend or live endpoint, confirms:

- Zero-signature registration is accepted and receives a local balance of `1000000000000000000` wei.
- A zero-signature bid and a nonzero invalid-signature bid are accepted into that local registry.
- The header conversion returns a zero signature even for the stored nonzero-signature input.

[Original output](LEGACY_BUILDER_OBSERVATION.json) is retained. Reproduce from `pkg` with Go 1.26.11:

```sh
GOTOOLCHAIN=go1.26.11 go run ../docs/audits/merged-glamsterdam-20261122/legacy-builder-observation.go
```

These are legacy EL extensions, **not evidence that the qualified CL accepts an unsigned Gloas consensus envelope**. The signed CL route has separate actual retired/current signature rejection and independent BLS evidence. Engine JWT authentication also remains a separate caller-authentication boundary. Neither turns an unchecked bid signature into verified builder authorization.

**Next development:** classify each legacy method against the selected Engine/CL contract. Remove or clearly retire unsupported prototype methods, or give retained methods explicit signing-domain, registered-key and finalized-deposit authority. Preserve signatures/identities through conversions. Require real BLS positives and zero/mutated/wrong-key/wrong-domain/retired-key rejections, with no registry/auction mutation on rejection. Publish this as a separate functional PR; this progress update does not silently redesign those interfaces.

## Remaining qualification and release gates

| ID | Priority / type | Evidence and completion condition |
| --- | --- | --- |
| GAP-02 | P1 · Upstream integration | Lodestar [10693](https://github.com/ChainSafe/lodestar/pull/10693) remains open against `unstable` at `4e224659`. The final dependency patch has fifteen local regressions, full normal local checks and actual process acceptance. The unchanged external-PR lockfile guard and full hosted unit/spec/e2e/browser approval still require upstream maintainers. A locally qualified override is not an upstream merge. |
| GAP-03 | P1 · Capacity/soak qualification | `FileDB.loadIndex` (`pkg/core/rawdb/filedb.go`) reads every data file/value into its index. `restoreIndexedHead` (`pkg/core/chain/history_restart.go`) still audits the complete head trie and walks the saved canonical prefix; it retains a header slice proportional to that prefix. The final million-file development datadirs meet the measured budgets, but these paths still grow with retained data. Measure larger account/storage/history cardinalities, retained bytes/RSS/I/O and multi-day churn/restart. Design bounded indexing/checkpoint/retention only with the original root/marker/finality guarantees preserved. Current 20-minute/3-minute acceptance is not a multi-day production result. |
| GAP-04 | Reserved decision · Public protocol/release | Final public gas/model and signed-envelope protocol review remains reserved by the user. Frozen EIPs/Engine/consensus pins and public fork activation must be reviewed as a compatible tuple. EIPs12581/12582 were closed unmerged; that does not adopt a replacement gas model. Public activation is not authorized by source merge or private devnet results. |
| GAP-06 | P1 · Startup readiness | `Node.Start` logs metrics binding errors and still reaches `n.running = true`; RPC/Engine/WS also bind in background goroutines. The original collision log records a successful node start despite the metrics failure. This review reserves the successful metrics listener synchronously but preserves existing error policy. Define required versus optional interfaces, return required bind failures, roll back already-started subsystems, and assert no leaked listeners/goroutines plus correct retry/health behavior. |
| GAP-05 | P2 · Coverage boundary | Official native and RPC corpora are finite, pinned formats/forks. Their exact counts are retained and are not all-Ethereum, all-client or all-draft conformance. Extend coverage when a concrete adopted spec, unsupported format or public deployment requirement changes; keep native and Geth-backed totals distinct. |

GAP-01 and GAP-06 are confirmed remaining code issues; GAP-03 contains confirmed scaling mechanisms and **unmeasured larger-scale acceptance**, not a newly reproduced failure at the qualified size. GAP-02/04 are external/review boundaries. No new divergence of the qualified signed CL consensus route was established in this scoped review; that is not a proof that the whole node has no bugs.

## Later-fork and service work remains separate

Independent seven-field Frame/Bogotá agreement, Dora production epoch/receipt/schema migration, paymaster service cutover, broader VC/remote-signer/redundant-BN reliability, public binary/PQ/proof adoption and native rollups remain separate roadmap work. The paired Frame record at slots 66/165 and fresh CL join 256 used a synchronized EL; it is not converted into independent Frame EL acceptance by the separate Glamsterdam Geth sync campaign. These do not reopen the completed frozen Glamsterdam task list.

## Progress-data corrections in this review

- Replace the November18 homepage/roadmap snapshot with merged November22 Glamsterdam evidence and source pins.
- Add seven previously omitted accepted EIPs:7688/8045/8061/8246/8282/8189/8136. Companion-CL-only implementations link their qualification instead of pretending to be missing local EL code. No new EF drawing position is inferred for these entries.
- Refresh eth/70 and eth/71 labels and remove obsolete 7702/PeerDAS gaps and speculative old percentages. Keep broader ePBS `Substantial` because the legacy extensions above remain unauthenticated even though the frozen signed CL route is qualified.
- Correct the stale EIP-7778 title: user refunds remain applied; only block accounting excludes them.
- Align homepage and roadmap through the shared snapshot; verify rendered values against retained records and detect deliberately altered milestones.
- Correct the tracker verifier's matrix parser: it previously treated the Spec column as Implementation. The new companion-CL empty-local-implementation rows reproduce that false error before the parser correction.

[Current roadmap](../../ROADMAP.md) · [Development progress](../../PROGRESS.md) · [Next development plan](../../next-stage-work-plan.md) · [Qualified recovery instructions](../../operations/glamsterdam-recovery.md)

## Validation and publication

[VALIDATION.json](VALIDATION.json) records the157-page production build,149-entry/92-row tracker verification,398 present package paths, retained-claim export checks and two deliberate drift rejections. Playwright checks five routes at1440px and375px with no page errors or horizontal overflow. The new long CL evidence path originally overflowed at375px (413px scroll width); word breaking fixes it and the actual browser retry passes. A read-only PR site workflow now runs the tracker/build/evidence checks before merge, in addition to the existing master-only Pages deployment. A submitted branch does not change the deployed site until merged.

## CI-discovered metrics listener collision (fixed in this review)

The final pre-fix head538408cb passed nine jobs, but full Go tests failed at `TestMetricsServer_StartsAndResponds`. [Original CI job](https://github.com/alt-research/eth2030/actions/runs/44932319511/job/136931496711) and [exact failure excerpt](metrics-ci-collision.log) show that P2P's ephemeral listener took43369, the same deterministic port derived from the metrics test name. Metrics binding failed and the HTTP test reached P2P, returning EOF. A longer sleep or an unchanged rerun does not remove that port collision.

The node now binds the configured metrics address synchronously, records the actual port-zero listener address, then serves on that already-owned listener. Ordinary configured addresses and existing bind-error reporting policy remain unchanged. The test uses port0, verifies readiness after Start without a sleep, decodes the real expvar JSON and confirms Stop releases the listener. The [new deterministic assertion fails on the original source](metrics-bound-listener-red.log). Repeated normal/race metrics controls, the node/full suites, vet and native build qualify this operational correction separately.

This is the only new production `pkg` change in the progress review: metrics listener startup, not execution/consensus behavior. The source-equivalence statements above apply to the merged baseline9a094830 and frozen runtime evidence. That earlier running executable is not relabelled as this review head, and no fresh devnet campaign is claimed.
