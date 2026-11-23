# ETH2030 development progress

Updated November23,2026 against EL tracking master`d743925e` (throughPR141; frozen qualified Go package tree unchanged) and Lighthouse`eth2030` at`fa16f70abee7d31c9817ab8d782af73bce241484` (PR4-15 integrated;1-3 already contained). The33 frozen LIFO/signed Gloas task IDs remain delivered. New EL142-146 remain independent master-target reviews; Lighthouse15 is integrated. Source integration, actual executables, hosted CI and website deployment remain distinct.

## Latest actual qualification

| Area | Retained result | Evidence / limitation |
| --- | --- | --- |
| Combined source checks |219 Go package suites, five related race packages, full vet/native build and 65 Python controls. | [Pre-merge validation](reviews/premerge-20261123/VALIDATION.json); measured composition retains its original source, not a renamed merge measurement. |
| Merged Lighthouse | Compatible dependency and healthy-producer/faulted-consumer fixture repairs; full repaired baseline27 hosted jobs pass. | [Current CI repairs](reviews/remaining-20261123/CI_REPAIRS.md); final simulator/bootstrap exact-head status is recorded separately. |
| Completed bounded runtime |6h45m05s startup duration; 6h42m ready observations. After the ready 5.5h threshold: 144 healthy samples, 72 full stack receipts; 1s collector 24244 successful observations, 0 sustained events. | [Completed run](reviews/diagnostic-threshold-20261123/COMPLETED_RUN.json). No new waiting chain is not proof of original-fault resolution or 48h acceptance. |
| Independent finality |14 unchanged-reference window replays, linked anchors, observed slots 89791–97858 inclusive (8068 slots), 494–512 participants. | [All-window replay](reviews/diagnostic-threshold-20261123/ALL_WINDOWS_REREPLAY.json).73 early missing slots remain; cryptographic authenticity does not erase operating-budget failures. |
| Actual stopped endpoint | Actual stopped CL slot 97868, 10 slots beyond the last health sample; persisted full beacon body → execution bid/header/state → independently verified account proofs. 160 live + 2 stopped proofs pass. | [Stopped binding](reviews/diagnostic-threshold-20261123/STOPPED_EXECUTION_BINDING.json). Original observation window through 97858 is unchanged. |

The bounded run retains 686 EL memory-limit events (no OOM) and three recovered local late-reveal advisories. It is not warning-free or multi-day capacity acceptance. Full runtime identities, CL/reference/fixture tuple and sources are retained in the linked records.

## New follow-up campaign

[Current tuple](reviews/remaining-20261123/CURRENT_TUPLE.json) records the separately declared30-minute ready window, whole-window BLS/offline replay, independent proofs and actual stopped endpoint. It does not extend the historical6h45m or establish48h. Eight finite synthetic KV/WAL reopen/RSS profiles pass declared limits; key/header/canonical metadata and unrestricted capacity remain open. EL142 improves actual recurrence capture;144 repairs the canonical-height gauge observed stuck at0 despite RPC progress;145 bounds large durable batch materialization;146 repairs the confirmed ext4 flat-directory capacity limit with compatible sharded values and protected old-reader rollback. The cumulative recovery passes91s RPC/212s catchup/301s ready,102signed slots and actual stopped113651. The initial300s catchup failed and remains failed. [Post-repair receipt](reviews/remaining-20261123/POST_REPAIR_RECOVERY.json). All remain independent master PRs.

## Current merged acceptance

| Area | Retained result | Evidence |
| --- | --- | --- |
| Execution/RPC |219 fresh packages, 107 official files/108 requests, six affected race suites, full vet/native build, byte-identical second Go1.26.11 executable. | [Post-merge results](reviews/glamsterdam-postmerge-20261121/RESULTS.json) |
| Snap/BAL recovery | Legal BAL expiry respects actual finalized slots; indexed snap recovery authenticates original checkpoint/head/markers/canonical ancestry; malformed/failed finalized reads remain errors. | PR 123/124; `core/chain/snap_history_retention_test.go`, `snap_indexed_restart_test.go`, `core/rawdb/finality_checked_test.go` under `pkg/` |
| Original datadirs | Main2/main3/full/snap conservative Docker-start bounds189.038/212.551/219.661/176.953 seconds, all under 300s, with frozen finalized commitments/eight independent proofs. | [Recovery record](reviews/glamsterdam-postmerge-20261121/RECOVERY.json) |
| Operating windows |400/400 steady and 60/60 pressure slots, original latency budgets, 512-participant independent finality/offline controls and270 account proofs. | [Results](reviews/glamsterdam-postmerge-20261121/RESULTS.json) |
| Final-source blob | Actual external-builder two-row blobslot59163; 544 KZG proofs, authenticated parent SSZ/signatures/ordered commitments; useful getBlobs-disabled partial cells228/114 columns and observer32/16. | Final-source archive; same original successful transaction, original API404 preserved |
| Source reproduction |245 public files, complete manifest, fresh extraction and11 independent replay checks; original nine archives/failures unchanged. | [Public replay](reviews/glamsterdam-postmerge-20261121/PUBLIC_REPRODUCTION.json) |
| Earlier application/CL scope | Signed Uniswap/Safe/7702 sequences, canonical factory, builder payments/registry, progressive SSZ, native official fixtures and independent cold sync retain exact sources. | [Task reconciliation](reviews/glamsterdam-final-campaign/TASK_RECONCILIATION.json) |

These results are retained source-specific measurements, not a new live campaign on the merge metadata. Native EF results remain distinct from Geth-backed runners. The initial failed preparations, over-budget runs and corrected harness inputs remain visible.

## Delivered follow-up repairs

PR 127 retired unsupported local Builder Engine extensions, synthetic stake and the signature-dropping header route; 128 made required startup failure/rollback transactional; 129 bounded FileDB value/WAL residency; 130 fixed restored snap ancestry eligibility. Later PR 135 corrected queued snap-history publication. PR 138 repaired the actual later-diagnostic FCU per-key/WAL fsync read-lock convoy and redundant publication writes while preserving durability and validation. All are merged. Local test/recovery controls and a bounded successful retest are distinct from retrospective attribution.

Rounds 03/04 remain failed. Original 04 had no contemporaneous waiting stack. Its exact holder and physical storage latency cause remain unconfirmed; shutdown timing does not establish cancellation as the cause. See [separate unknowns](reviews/premerge-20261123/UNCONFIRMED_ORIGINAL_FAILURE.md) and [captured later chain](reviews/fcu-io-liveness-20261123/ROOT_CAUSE.md).

## Remaining acceptance and source TODOs

The [dated actionable plan](plans/glamsterdam/NEXT_STEPS_20261123.md) and [machine-readable reviewed state](reviews/remaining-20261123/CURRENT_STATE.json) separate:

1. **G-01 / integrated CL CI:** Lighthouse #13-15 are merged into eth2030. Exact CI head `215525de` passed all 27 test-suite jobs and local-testnet (9 passed, one conditional checkpoint skip); merged `fa16f70a` has the same tree. G-01 is closed for this private profile. The primary simulator uses explicit 6s timing; the original 3s failure remains retained. See [repair evidence](reviews/remaining-20261123/CI_REPAIRS.md) and [current campaign](reviews/remaining-20261123/CURRENT_STATE.json).
2. **G-02 / P1 — original incident evidence:** maintain request-correlated rolling stacks/storage timing. The precise unprofiled historical waiter remains unknown.
3. **G-03 / P1 — duration/capacity:** no actual 48h pass; key/header/canonical metadata and history still grow. Declare workload/resource/restart budgets and qualify the final tuple. A rapid or bounded profile remains separately named.
4. **G-04 / P1 — upstream Lodestar10693/10725:** recovery and empty-coverage repairs await maintainer workflow approval/review/integration. Local5547passed/9skipped and821TS coverage records remain separate from upstream hosted qualification.
5. **G-05 / P1 — public target:** final gas/model, compatible fork/Engine/CL/reference tuple and activation remain reserved. Complete development does not select a new draft or imply public readiness.
6. **G-06 / P2 — conditional coverage:** selected native/Geth corpora and real-EF tests retain finite coverage. Skipped/inactive cases are not passes.

The source-hit triage in the plan records progressive IL compatibility, unused sidecar/nonce helpers, Node sampler/beam scaffolds and research proof/transport TODOs. Actual FCU custody and standard getPayload blob proof consumers are wired; no additional missing frozen-target wire consumer was established in those reviewed routes. Concrete metrics and persistence defects found in the live follow-up are tracked in their separate repair PRs. This is a scoped finding, not a whole-client bug-free claim. The broader historical fast-confirmation/EVM-ASM framework labels and ePBS draft scope remain distinct from the 33-task ledger.

Later-fork independent Frame/Bogotá agreement, Dora production consumers, paymaster services and public binary/PQ/proof/native-rollup work retain separate gates. The paired Frame 66/165 and CL join 256 record is preserved [at its original source](verification/frame-8141-20261118-paymaster-history-final.json); its synchronized-EL boundary is unchanged.

[Roadmap](ROADMAP.md) · [Next-stage plan](next-stage-work-plan.md) · [Immutable original audit](audits/merged-glamsterdam-20261122/README.md)
