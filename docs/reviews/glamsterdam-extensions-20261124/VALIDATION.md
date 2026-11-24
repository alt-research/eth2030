# Source-specific extension validation

## Runtime and signed window

The final private profile used two native ELs, two Lighthouse CLs, 256 real-BLS validators and 6-second slots, with Gloas active from genesis. The actual 12-minute window covered slots 0–115, CL SIGKILL/start at slot 72, real local proof generation and independent verification on the follower. See RUNTIME_WINDOW.json and LIVE_PROOF.json.

The collector saves signed blocks, states and execution envelopes before CL pruning. BLS_WINDOW.json independently verifies every captured signed block at unchanged consensus-specs d2021d2719235f0df4726a1d330502cb15aa8ea3. Runtime network fork versions/epochs and SLOT_DURATION_MS are applied as configuration; no algorithm or BLS implementation is patched. Full state-root and genesis-validator-roster bindings are checked. The retained fork graph contains 113 canonical blocks and one authenticated orphan; it is not relabelled as a no-reorg window. Six altered-signature categories are rejected. Self-build bids have no separate builder signature and are not counted as one. Unexpected slashings, exits, credential changes and deposits fail rather than being silently ignored.

The final canonical-tip header has a separately timestamped follow-up capture. It is not misrepresented as an original contemporaneous snapshot. Earlier failed/aborted launches, transport capture, configuration, pruning and linear-ancestry checks remain in FAILED_ATTEMPTS.md and the local directories. The final raw-byte inventory is WINDOW_FILES.json.

Replay the retained window with the pinned generated reference environment:

```sh
/projects/eth2030/toolchains/consensus-reference-d202-venv/bin/python \
  pkg/devnet/kurtosis/scripts/reference_fast_confirmation_window.py \
  --capture /projects/eth2030/qualification/fcr-proof-20261124-attempt4/window \
  --output /tmp/fcr-reference-replay-new-output
```

Run from the Fast confirmation feature checkout. The output directory must not already exist. Replay uses retained bytes and performs no node HTTP reads.

## Source and proof boundaries

The measured native bundle is clean cumulative b7a0ec69a189186c4d110670de92cfefc47b8cc2; Lighthouse is qualified 274100659f310772e68c11c684eed2ce4b407a62. CL_MERGE.json proves the authorized eth2030 merge has the identical source tree. RUNTIME_SOURCE_COMPARISON.json separates subsequent collection and legacy-counter changes from unchanged native Engine/RPC/FCR and proof-verification algorithms; the subsequent CPU proving budget is recorded explicitly. The legacy helper's additional NaN/minimum-vote/negative/empty-set fixes have their own executed red/green, complete counter race, native HTTP race and vet coverage; the earlier binary is not relabelled as the later whole package tree.

Real execution receipts use the pinned 48b22fafa4f7f1759ab61bed7d1386ea61d61fdc7c186b6162be09838a67447a RV32IM interpreter image. Actual program/input/output, configuration, cycles, terminal PC/register/RAM and journal verification are bound. The runtime-only bundle proves with networking disabled and no SDK install. Native async proof RPC remains responsive and exported artifacts verify after node restart. Full Go/Rust state vectors and altered receipt/program/input/output/state controls are separate from synthetic fixtures.

This is not an executable Ethereum STF guest, a mandatory public block-proof fork, 48h/scale acceptance or historical original-04 root-cause attribution. The broad canonical quest remains Substantial, giving scoped mapped Glamsterdam progress 16/17.

## Site and review checks

Production export builds 157 pages. All 149 entries and 400 listed source/test paths reconcile. Export guards bind current and historical receipts separately. Desktop/mobile checks cover ten route/view combinations and preserve both bottom panels default collapsed. Exact final PR heads, hosted CI and six cumulative merge orders are recorded at review delivery, outside self-referential source-commit assertions.

## Hosted proving capacity correction

Initial actual-proof hosted run 45336500029 failed all producers at the inherited 300-second deadline; its Node waiter was only 240 seconds. HOSTED_INITIAL_TIMEOUT.json preserves every failed case. The final-image full local run passed all eleven required cases without skipping (FINAL_IMAGE_LOCAL_ACCEPTANCE.json). A separately affinity-limited four-core actual proof took 385.808 seconds and completed successfully, exceeding the original budget; FOUR_CORE_PROVING.json retains the measurement.

The shared CPU proving deadline is now bounded at fifteen minutes, with the Node waiter using that same budget. Short verifier deadlines, asynchronous submission latency, cancellation, process-group cleanup and worker limits are retained. Independent hosted families and a final aggregate gate still require all eleven real positive cases, zero required skips, identical source and the pinned guest image. An earlier runtime binary is not relabelled as this later deadline configuration. Final hosted results are authoritative on the proof review and final delivery record.

The new-budget source 40a78da6 also passed an actual four-core-limited native Node TCP test in 170.01s, including real proving, ordinary RPC responsiveness, changed input/state rejection, artifact verification after restart and rejection of the old node-lifetime job ID. See NEW_BUDGET_NODE.json.
