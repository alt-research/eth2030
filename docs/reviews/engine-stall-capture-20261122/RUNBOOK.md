# Capture the original multi-hour stall

## Separate incidents and results

PR135 fixes a reproduced startup critical section: authentication of a long immutable saved prefix held the execution import mutex. The original04 node logged that worker finished at05:10:33Z, before acceptance at05:11:01Z. The later10:41 failure remains a different unresolved incident. A fast replay, short operating observation or forensic BLS pass cannot close it.

| Original04 UTC time | Observed event | Evidence boundary |
| --- | --- | --- |
|05:10:33.486|Historical bodies/BALs finished backfilling|Rules out claiming an active startup history scan caused the later stall|
|10:41:22–23|Epoch boundary, finality update and block80256 imported|Block processing preceded envelope execution|
|10:41:24.164|Last CL peer sent TooManyPeers goodbye|Envelope subsequently recovered through RPC lookup|
|10:41:38.026|RPC envelope80256 submitted for execution|Real missing-envelope/retry path, not sequential import alone|
|10:41:46.028|Native Engine8s timeout and CL503 rejection|A native request did not finish within its deadline|
|10:42:17.596|CL shutdown after health timeout|Later queue saturation is an observed symptom, not a proven root|
|10:43:41.649–793|Native SIGTERM, then block77623 accepted|Requires a live waiting stack; timing alone does not establish a lock owner|

The unchanged old-source6h45m diagnostic retains the original hash/BLS checks and resources. It now has an independent raw-pipe collector, without restarting the diagnostic. A separate original-source control injects one post-readiness epoch-boundary Engine503 after8s while retaining real native execution. That control tests the CL recovery cascade; it cannot explain why the original native request first stalled.

## Reproduced evidence loss and fix

The private early collector used `select(fd)` followed by `TextIOWrapper.readline()`. A single read can prefetch `accepted block77622` and `engine_request_timeout` together. After the first line, the kernel pipe is empty even though Python already buffered the timeout. If the stalled EL writes nothing else, the timeout is never delivered. A real pipe with its writer still open reproduced this exactly; restoring this reader in an executed mutation fails the new quiet-after-burst test.

The replacement reads bytes using `os.read`, drains complete lines before another select, bounds partial and complete lines, and drains an unterminated final line on EOF. It records failed initialization/log-process startup explicitly and publishes status atomically. Eight tests include quiet-after-burst delivery without a new write/EOF, fragmented input across an idle poll, EOF, bounds, shutdown and startup failure. The ordinary26 baseline devnet control tests also pass. CI executes the new tests in its existing consensus/control job.

## Run against an owned diagnostic

```sh
python3 tools/capture_engine_stall.py \
  --record /absolute/owned-diagnostic/containers.json \
  --output /absolute/new-exclusive-capture-directory \
  --cl-debug-image eth2030-cl-debug:20261122
```

The record includes `el`, `cl` and `profile` with the two actual executable hashes/source declarations. Targets must have the owned campaign label. The existing optional metrics listener needs the goroutine route from PR133. The tool verifies both actual PID1 executables and start identities, retains a real baseline profile, then follows new native timeout events. It records four bounded native profiles, cgroup pressure, selected state and recent CL/EL logs before cleanup. A missing/truncated/error artifact remains explicit.

The optional local GDB image must already exist. After a real Engine timeout, the debugger joins the owned CL PID namespace, loads `/proc/1/exe` with its matching filesystem libraries, and obtains native thread backtraces. The helper has a20s deadline and is always force-removed, including a timed-out Docker client. Debugger interference is explicitly recorded after a fault; never count it as clean operating time. The collector never restarts or stops the EL/CL. A real PID-namespace/cleanup control against an independent sleep process passed; no healthy qualification CL was paused for that control.

A zero-capture report is not proof of node correctness. Inspect current collector status, heartbeat, runtime identities and an independent complete native timeout-log scan. Before replacement the running original diagnostic had zero actual native timeout events in that independent scan, so this finding establishes a collector defect, not evidence that an original stall already recurred.

## Remaining closure

Keep waiting for or deliberately reproduce the original native/CL waiting chain. Capture both sides before shutdown, identify the exact holder/dependency, produce a causal failing regression, then fix the owning implementation on its own branch. Re-run the real mature-state event profile and native/CL integration without weakening signature/hash/state/finality requirements. Keep original failed03/04 and every controlled fault distinct from new clean observations.
