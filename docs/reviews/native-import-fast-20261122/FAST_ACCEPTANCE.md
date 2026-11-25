# Fast Glamsterdam development acceptance

Use an accelerated event gate for each relevant PR and reserve continuous uptime observation for a frozen final candidate. Never restart a48-hour acceptance clock for every edit. This profile already found a genuine native import defect in under one minute after node startup.

## Measured feedback and development gates

| Gate | Checks and retained evidence | Measured duration or budget |
| --- | --- | --- |
| Causal lock regression | A barrier pauses a real saved-prefix read. New native execution and atomic CL forkchoice must complete before it is released; cancellation and downloaded-history authority are checked. Old code fails, fixed code passes20 repetitions. | 0.881s for20 fixed repetitions; no slot sleeps |
| Small native/Engine regression |16 genuine empty blocks, full native execution, forced Engine re-execution, repeated payloads, FCU and concurrent SDK MPT authentication. |2.488s; five race repetitions12.556s on the composed fix |
| Mature incident-state replay | Private clone of the unchanged failed04 archive; real Geth RLP77623–77878,512 forced native executions plus256 repeated payload calls,256 FCUs, SDK header/proof checks and persistent restart. Same2CPU/4096000000B limits and8s import budget. |295.296s including restore/download;232.538s for node execution and restart |
| Independent signed-span replay | Fresh unchanged reference execution, active real BLS backend, complete parent-hash paths and every prior finalized anchor linked across7 captured windows, slots73639–80256. |9.277s; fresh results, no cached pass reuse |
| CL integration | Unchanged default basic and fallback simulators, real configured BLS, provider recovery,129 canonical blocks and existing95% thresholds. | About7min each from prior qualified3782 runs; rerun relevant final CL changes |
| Native live observation | Final EL/CL tuple, real peer traffic and independent MPT/BLS observations, peer loss/recovery and clean stop. |30–60min development observation target; a short observation is explicitly partial |
| Continuous stability | One immutable final tuple; original sampling, BLS ancestry, limits, deadlines and cleanup requirements. | Background48-hour observation remains a separate uptime result |

A normal change can receive substantial functional feedback in about20–25min when the two CL integration runs are needed. Compile time and host I/O add variability. The table separates measurements from targets. Event replay accelerates existing blocks, with their original timestamps, signatures and hashes; it does not accelerate garbage collection, disk aging, reconnect timing or actual elapsed uptime. It does not by itself prove48-hour stability. A full48-hour historical block-count replay is also distinct from uptime and its runtime has not been measured here.

## Reproduce the native gate

Build the test binary from the candidate including the replay test. The production history-lock fix is a separate PR and can be composed in an isolated qualification branch. Both target master independently.

```sh
cd pkg
GOTOOLCHAIN=go1.26.11 go test -c -o /absolute/qualification/node.test ./node
cd ..
python3 tools/native_import_replay.py \
  --snapshot /absolute/private-el-database.tar \
  --snapshot-sha256 VERIFIED_ARCHIVE_SHA256 \
  --genesis /absolute/genesis.json \
  --reference-rpc http://127.0.0.1:GETH_RPC_PORT \
  --first 77623 --count 256 \
  --test-binary /absolute/qualification/node.test \
  --source EXACT_BUILD_COMMIT \
  --image LOCALLY_QUALIFIED_IMAGE_DIGEST \
  --output /absolute/new-exclusive-replay-directory
```

The archive must contain `execution-data/chaindata`. The tool checks its digest, restores a new exclusive directory and never mounts the original database. It starts an owned network-isolated2CPU/4.096GB container and retains it after exit. Logs, raw reference exchanges, input digests, executable digest and result remain in that directory. Credentials and private databases are not publication artifacts. An8s wait records goroutines before cleanup; an empty or skipped test cannot qualify. The source argument records the build declaration; establish its exact binary/source mapping when building.

`TestNativeMatureImportReplay` visibly skips when no explicit external manifest is supplied; that skip is not incident-state coverage in ordinary CI. `TestNativeFastReplayWithRealEmptyBlocks` always executes in the node suite. The causal barrier test belongs to the separate production-fix PR. All ordinary header, execution, receipt, requests, BAL and MPT checks remain active. The replay first obtains a real native execution result and its committed BAL, then resets only the disposable clone's validation marker to force a second real execution through Engine. Native/Geth header identity and committed roots are independently checked; existing marker/cache hits cannot replace those executions.

The BLS fast gate consumes original captured exchanges with `reference_operating_window.py --replay --capture CAPTURE --trusted-bootstrap BOOTSTRAP --output NEW_RESULT`, first verifies the pinned reference modules/backend, and then runs `linked_window` for every consecutive result. Preserve the original failed operating result. This authenticates BLS-finalized anchors and all observed ancestry; it is not a claim that every block signature was separately re-executed through a full CL transition.

## Remaining incident attribution

The mature replay demonstrated startup historical-prefix authentication holding `insertMu`; the separate fix releases that lock before immutable prefix traversal. However, the original04 EL logged history completion at05:10:33Z, before its05:11:01Z acceptance clock and hours before its10:41 stall. That original incident therefore remains unresolved. Keep the original diagnostic running and capture live waiting stacks when it actually stalls. A passing fast replay must not close this different incident, and failed03/04 must not be relabeled successful.
