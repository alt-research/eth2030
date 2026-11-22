# Diagnostic review status

PR138 exact head67739c2873244a635393d8254c187d96ef264844 has9/9 successful CI checks and targets master. The new6h45m runtime began2026-11-22T18:36:54Z and first reported ready at18:39:26Z. At19:17Z it had not covered5.5h; the ready-time threshold is2026-11-23T00:09:26Z (08:09:26 Asia/Shanghai). Existing30s samples remained ready and the native1s collector had no sustained3s events. This is incomplete evidence, not incident closure.

The first bounded BLS window89791–90419 passed real reference/replay with512 participants, but records63 missing slots out of629 (10.016%). Authentication does not establish a successful operating budget. The raw response, missing slots and all prior failures remain retained.

Source inspection found that the ad-hoc runtime driver can keep an observing/no-stall label when a later sample reports syncing/offline, and does not enforce the finalized-progress watchdog. Its rolling stack ring retains only128 latest samples. Add an independent strict review and append-only periodic snapshots/checkpoint receipts alongside the existing run, without changing runtime source, restarting containers or overwriting original results.

## Implemented and qualified

The new `pkg/devnet/kurtosis/scripts/review_diagnostic.py` reads the existing runtime ledger and fails on lost readiness, finalized regression/stall, stale/missing/duplicate observations, resource/OOM failure, inconsistent source/build metadata, premature termination or disconnected reported BLS windows. It retains the existing60s observation-gap and600s finalized-progress bounds. Runtime duration and first-ready5.5h coverage are separate. Its output never asserts original-fault closure or48h acceptance; reported BLS-window coverage is explicitly distinct from signature verification and final stopped-head closure.

The passive watch mode verifies runtime starts/resources and executable digests, retains minute goroutine/pressure snapshots in exclusive directories, appends reviews, and captures the5.5h time checkpoint even if readiness has failed. It never signals/restarts a node or changes runtime source. The old128-sample ring remains unchanged. Evidence starts at watch attachment; overwritten earlier profiles cannot be reconstructed.

Nine new behavior tests cover legacy false greens, finalized stall,60s gaps/duplicates/staleness, readiness-based timing, threshold capture after failure, BLS discontinuity/zero participants, early completion, restart rejection and retained memory pressure. They run in the existing consensus CI discovery. All35 devnet Python control tests passed, and the unchanged native Go build passed with Go1.26.11.

The first BLS window was replayed anew with the unchanged frozen real-BLS verifier, original wire capture and all three negative controls. All63 missing slots independently returned404 on both Lodestar and the second Lighthouse, while terminal slot90419 returned200 on both (128 checks). This supports canonical chain absence rather than native import loss, and does not turn a10.016% missing-slot window into an operating-budget pass. See FIRST_WINDOW_REPLAY.json and MISSING_SLOTS_CROSSCHECK.json.

A watch was attached to the existing run without restarting either EL/CL. Its first short attachment used a90s review bound; only that observer was replaced to retain the pre-existing60s bound, and its receipts/handoff remain retained. The authoritative attachment is `/projects/eth2030/qualification/soak-audit-20261122/actual-fcu-io-postfix/strict-evidence-guard-02/`; its profile pins the tool checksum and attachment time. Both daemon start times/binaries remain identical. Read CURRENT_REVIEW.json as a dated snapshot, not completed5.5h evidence. The6h45m driver, original1s rolling collector and30min real-BLS monitor continue unchanged.

## Review after the threshold

At or after2026-11-23T00:09:26Z, inspect `THRESHOLD_5_5H.json`, its full stack and pressure receipt, append-only `reviews.jsonl`, actual native sustained3s/8s events, all post-threshold runtime samples and every BLS window/previous-anchor link. Check sample/collector gaps and final stopped native/CL head binding separately. A no-event counter or a successful cryptographic prefix alone cannot close the original unprofiled incident.

## PR and CI handoff

Submitted https://github.com/alt-research/eth2030/pull/139 to master. At the recorded heads, all9 pending EL PRs131–139 target master;72 ordered pairs,2304 subset extensions and18 explicitly executed complete orders are clean with a single resulting tree. This is representative-ancestry/subset confluence and explicit order testing, not enumeration of all362880 distinct histories. No master merge was performed. See MERGE_AUDIT.json; exact-head metadata will be rechecked after the documentation commit.

Review of the hosted PR138 race log showed that its old green label omitted rawdb/chain/beacon-sync. PR138 follow-up d6d63b4174e99a0873b2646b60f2e2f112df2b1a adds all three packages to the existing hosted race variant. The exact expanded command passed locally (rawdb1.613s, chain4.034s, beacon-sync57.053s); full hosted CI is rerunning. Its package source is unchanged, so the current native/CL binaries and elapsed clock remain valid.

The current review remains strictly before5.5h. The passive guard has retained successful live goroutine/pressure/identity receipts with no daemon restart. Source-level audit gaps are repaired; neither a no-event prefix nor these controlled regression tests resolve the earlier unprofiled incident.
