# ETH2030 development progress

Updated November24,2026 against master `cfd39f3f4ad05479470466fc14d430c216724347` (PRs142-147 merged) and Lighthouse eth2030 `fa16f70a`. Adopted frozen LIFO/signed Gloas development is complete:33/33 original task IDs and5/5 confirmed software follow-ups; no active confirmed implementation TODO remains. The merged Go package tree equals the actually qualified cumulative code. Per the user's current instruction,48h/larger-scale acceptance and upstream Lodestar approval are excluded from development progress and retained separately.

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

[Current tuple](reviews/remaining-20261123/CURRENT_TUPLE.json) records the separately declared30-minute ready window, whole-window BLS/offline replay, independent proofs and actual stopped endpoint. It does not extend the historical6h45m or establish48h. Eight finite synthetic KV/WAL reopen/RSS profiles pass declared limits; growth of key/header/canonical metadata and unrestricted capacity qualification are deferred outside development progress. EL142 improves actual recurrence capture;144 repairs the canonical-height gauge observed stuck at0 despite RPC progress;145 bounds large durable batch materialization;146 repairs the confirmed ext4 flat-directory capacity limit with compatible sharded values and protected old-reader rollback. The cumulative recovery passes91s RPC/212s catchup/301s ready,102signed slots and actual stopped113651. The initial300s catchup failed and remains failed. [Post-repair receipt](reviews/remaining-20261123/POST_REPAIR_RECOVERY.json). All five implementation follow-ups are now merged into master.

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

## Completion accounting and separate records

The [current plan](plans/glamsterdam/NEXT_STEPS_20261124.md) and [machine accounting](reviews/development-scope-20261124/CURRENT_STATE.json) distinguish completed code from acceptance and deployment decisions.

- Development:33/33 original tasks,5/5 confirmed follow-ups and integrated CL CI are complete/merged.
- Deferred by the user:48h/larger-scale admission and Lodestar maintainer approval do not lower development progress. Neither is recast as passed/merged acceptance.
- Historical record: original04 exact waiter remains unconfirmed; confirmed independent mechanisms are repaired. No-recurrence cannot reconstruct its missing stack.
- Public activation: a separate network/epoch/timestamp/client-tuple rollout decision. Timestamp/genesis/override and restart-compatibility support already exists; no new public network parameters are selected here.
- Extra corpus, research and later-fork consumers: not newly adopted into the current denominator. Skipped/inactive cases retain their actual counts.

The [new source audit](reviews/development-scope-20261124/AUDIT.md) follows current TODO consumers. No additional confirmed missing adopted-scope production consumer was established. This is a scoped review, not a whole-client bug-free or whole-public-fork readiness claim.

Later-fork independent Frame/Bogotá agreement, Dora production consumers, paymaster services and public binary/PQ/proof/native-rollup work retain separate gates. The paired Frame 66/165 and CL join 256 record is preserved [at its original source](verification/frame-8141-20261118-paymaster-history-final.json); its synchronized-EL boundary is unchanged.

[Roadmap](ROADMAP.md) · [Next-stage plan](next-stage-work-plan.md) · [Immutable original audit](audits/merged-glamsterdam-20261122/README.md)
