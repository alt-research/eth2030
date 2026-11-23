# Glamsterdam actual progress and next steps — 2026-11-23

## Current decision

The selected merged LIFO / signed Gloas development scope and its original 33 task IDs are delivered. EL PRs 131–140 are merged at master84200d5322398ebe473ff26c6f54de2665cd2586. Lighthouse PR4–12 are integrated into eth203034cecfc4fc3bd743c14c38a742167064a95dbc25;1–3 were already contained and their redundant reviews are closed. A source merge is not full protocol/public activation or a new runtime measurement.

The completed bounded run covers6h45m5s from startup and6h42m of ready observations. Its post-5.5h tail has144 healthy samples and 72 full stacks;14 unchanged-reference BLS/finality windows cover 8068 observed slots with 494–512 participants.73 early missing slots,686 EL memory-limit events and three recovered local reveal advisories remain retained. The actual stopped CL/native slot 97868 is authenticated through the captured BLS parent path, full persisted beacon SSZ, independently decoded execution header and MPT proofs.160 live account proofs and two stopped-head proofs pass. This is not a 48h pass or retrospective proof of original 04's exact waiter.

Fresh pre-merge combined checks pass219 Go suites, five related race packages, full vet/native build and 65 Python controls. Merged Lighthouse format/all-target compilation and 500 scoped library/API/snapshot tests pass. The actual integrated hosted run45134918940 is **not green**: check-code/audit-CI rejects locked url 2.5.8 under RUSTSEC-2026-0592; its recorded patched range is >=2.5.9. Two additional jobs now fail: http-api-tests (the two optimistic-head builder tests, PayloadIdUnavailable;109 passed, 2 failed and 105 unrun due to fail-fast in the affected Fulu invocation), and basic-simulator-ubuntu (slot 114 sync aggregate 15/32). The beacon-chain-tests job also fails:18 payload-invalidation cases with PayloadIdUnavailable and two Heze deferred-verdict cases with NoViablePayloadBid (492 passed,20 failed,2 skipped in the affected Fulu invocation). Their causes are not established; minimal EF remained running at the dated inspection. The previous libz patch does not close this newer dependency failure. This is a concrete CI/dependency todo, not evidence of a hash/BLS validation bypass or a new EL stall.

[Detailed new hosted failure inventory](../../reviews/progress-20261123/CI_FINDINGS.md).

## Active remaining work

| ID | Priority / kind | Current evidence | Concrete next action and completion criterion |
| --- | --- | --- | --- |
| G-01 | P0 · integrated CI | [Merged CL workflow](https://github.com/jiayaoqijia/lighthouse/actions/runs/45134918940), [actual blocker](https://github.com/jiayaoqijia/lighthouse/actions/runs/45134918940/job/137558805418). Local500 checks are narrower than the hosted suite. | Resolve the compatible url dependency graph without weakening audit. Reproduce/trace both optimistic-head API and18 beacon-chain payload-invalidation failures, plus both Heze deferred-verdict NoViablePayloadBid failures. Inspect the simulator artifact 14800183126 before attributing its partial aggregate; preserve the remaining unrun tests and rerun the full affected profiles. rerun locked build/lint/relevant regressions and the required integrated workflow. Inspect every remaining job result; do not convert skipped/inactive cases or old standalone failures to passes. |
| G-02 | P1 · incident evidence | A later old-source recurrence captured a real FCU fsync/read-lock convoy and the merged fix addresses that defect. Original 04 has no contemporaneous waiting stack. | Keep historical attribution explicitly unconfirmed. For any actual recurrence, retain request-correlated rolling stacks plus fd/path/fsync/cgroup/block-I/O timing before shutdown or debugger interference. No new test can manufacture missing historical evidence. [Separate unknowns](../../reviews/premerge-20261123/UNCONFIRMED_ORIGINAL_FAILURE.md). |
| G-03 | P1 · duration/capacity | A bounded 6h45m run and finite restart fixtures pass; no 172800s run exists. Value/WAL residency is bounded, key/header/canonical metadata and complete authentication still grow. | Declare a final runtime/CL/fixture and state/history/transaction workload; measure I/O, latency, RSS, startup, retention, pruning and restart against independent roots/proofs. If strict48h admission is retained, actually complete it; a separately approved bounded profile must remain separately named. Do not extend old timestamps or disable durability/validation. |
| G-04 | P1 · upstream dependency | [Lodestar 10693](https://github.com/ChainSafe/lodestar/pull/10693) remains open; local patch/process controls do not establish full upstream hosted qualification. | Finish maintainer lock review, full upstream checks and integration; retain the exact current dependency patch and its real execution traces meanwhile. |
| G-05 | P1 · public target decision | Frozen development target is delivered; public gas/model, compatible fork/Engine/reference tuple and activation are reserved to the user. | Explicitly select any new target, compare normative specs, then requalify affected consumers. Do not infer official fork inclusion/finalization or public readiness from historical tracker labels. |
| G-06 | P2 · conditional coverage | Native/Geth corpora and selected CL real-EF execution retain finite counts.24 BLS skips and43 inactive handlers are not executed passes. | Extend only specifically adopted formats/handlers/consumer cases; retain real execution counts, signatures/hashes, negative controls and restart/readback. Current complete-suite CI verdict is part of G-01. |

No new mandatory production EL implementation gap was established in the reviewed selected Glams call chains. That is a scoped review result, not proof the whole client is bug-free. The active qualification gates above remain real work.

## Source TODO/stub triage

Search covered bal,epbs,engine,core chain/execution/VM,eth,p2p,sync and node non-test Go source. A TODO word is not by itself proof of a missing live consumer.

| Source hit | Actual route / boundary | Todo disposition |
| --- | --- | --- |
| engine/lighthouse_compat.go: progressive inline IL parsing | The helper does not decode non-null snake_case progressive inclusion-list data. Native Bogota validation currently requires the adopted camelCase IL contract and rejects incompatible attributes before parsing; the qualified alternate IL route is separate. | Later-fork interoperability work: choose the compatible new contract, decode/reject explicitly, and verify actual builder/enforcement consumers plus negative controls. Do not call this an unresolved frozen Glams BAL/ePBS implementation requirement or claim every upstream shape supported. |
| engine/lighthouse_compat.go: custody TODO | Actual FCU path uses parseCellMask → EngineBackend.SetCustodyColumns → eth.Handler.SetCustodyColumns and the relay custody generation. The helper's other use is optional getBlobsV3 indicator validation with a sparse response. | Stale broad comment/documentation cleanup; not a missing FCU custody consumer. Any additional serving-policy change needs its own adopted contract and tests. |
| node.stubCustodyManager / stubBeamFetcher | Scaffolds are constructed; the DAS manager is started, but no live sampler request consumer was found in this Node graph. The qualified cell route is the CL/eth sparse relay; beamSync has no non-constructor production consumer here. | Optional research integration or scaffold cleanup, not an implemented independent EL sampler/beam mode. Do not advertise those capabilities as production-complete. |
| engine/blobsbundle helper placeholder Merkle paths | No non-test caller of PrepareSidecars/GetSidecar was found beyond exported aliases. Actual Node/Engine publication calls payload.BuildBlobsBundleV1/V2, which use real blob/cell proofs and fail incomplete data. | Retire/isolate or implement before promoting that helper. Do not misreport the active standard getPayload path as using placeholder KZG. |
| eth.BroadcastNonce placeholder signature | This old helper has no non-test caller in the searched graph; selected nonce wire handling is separate. | Research/helper cleanup or real signer integration if adopted; not evidence that the qualified standard wire uses this placeholder. |
| shielded/RVCREATE/portal/mixnet/Kohaku stubs | Research/later-fork/protocol-specific paths, outside the selected frozen Glamsterdam acceptance. | Keep separately scoped. Signature/proof frameworks, precise RV cycle accounting, portal validation and real transport formats need their own contracts, production wiring and independent tests before promotion. |

The historical mapped Glamsterdam tracker also retains ePBS as Substantial and fast confirmation / EVM-ASM canonical quest as Functional framework labels. These broader scopes are not the selected 33-task acceptance ledger; promoting them requires their own complete contracts and independent consumers. Do not manufacture a 100% whole-fork score.

## Separate Hegota/service work

Independent Frame/Bogota client agreement, Dora epoch/receipt/schema consumers, paymaster/service cutover and public binary/PQ/proof/native-rollup work retain their later-fork plans. Earlier synchronized-EL CL join is not independent Frame EL sync evidence. Do not inflate Glamsterdam's todo count with those different targets or silently mark them complete.

## Order and reporting rules

1. Close G-01 on the merged source; preserve the exact CI conclusion/job/source, including any new failures.
2. Preserve G-02 observability and qualify G-03 against a declared operational envelope; use rapid native replays as causal/consumer checks, never as substituted elapsed time.
3. Progress G-04 in parallel. Select G-05 before public activation or adopting a changed gas/CL model; perform G-06 when a concrete new scope is adopted.
4. Every large implementation remains a separate branch/PR. This progress update changes documentation/site claims, not protocol behavior. Merge/source equivalence, measured runtime identity, hosted CI and deployed website revision remain distinct.

[Machine-readable reviewed state](../../reviews/progress-20261123/CURRENT_STATE.json) · [merged progress](../../PROGRESS.md) · [roadmap](../../ROADMAP.md).
