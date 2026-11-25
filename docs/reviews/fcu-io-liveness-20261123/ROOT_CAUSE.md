# Spontaneous FCU durable-I/O wait

## Actual observation and limits

The old-source native diagnostic started at 2026-11-22 12:22:19 UTC. The first Engine timeouts were observed at 17:16:11 UTC, approximately 4h54m after startup. It stopped after the stall-capture grace period at 17:23:14 UTC. **It did not reach 5.5h or pass a 48h admission.** Exit zero and subsequent recovery do not reclassify that failed observation.

The first complete native snapshot, requested before any CL debugger, identifies goroutine 58577 in `ApplyForkchoice -> fileBatch.Write -> FileDB.applyOpsLocked -> writeDataFile -> File.Sync`. The source retains the chain publication mutex and database writer mutex across that synchronous I/O. Goroutine 58585, an actual newPayload, waits on the same FileDB mutex in `CheckStorage -> ReadHeadBlockHash -> Get`; syncing requests wait in `CurrentBlock.RLock` on the same Blockchain. The selected complete stacks are included in ACTUAL_WAIT_STACKS.txt; EVIDENCE.json pins the original full snapshot.

An independent rolling capture observes goroutine 58897 on the same FCU path in WAL fsync across 9.799 seconds. The first pressure capture reports I/O some/full avg10 46.90%, avg60 59.81%, avg300 32.01%; CPU and memory pressure are zero, cgroup throttling totals only 1255 microseconds, and neither container had OOM. This establishes an actual storage wait amplified into a global read/health lock convoy. It does not determine the physical cause of the underlying device latency. Later debugger timeouts are post-fault interference, not the initiating cause.

Original run 04 lacked a contemporaneous stack. This spontaneous recurrence demonstrates a real defect in its unchanged native storage/FCU source; it cannot independently prove the exact holder of that earlier unprofiled incident. The old controlled response/commit pauses and the earlier 30-minute pass remain separate evidence.

## Change and durability boundary

FileDB retains its last committed bounded cache while performing WAL and per-key durable I/O under the storage writer lock. Only after successful persistence does it take the chain publication lock, publish the database cache/index, and update head, state and checkpoints. Cold reads can still wait for storage; chain getters release the chain read lock before those fallbacks, preventing a reader from blocking publication while waiting for that same writer. Legacy canonical mutations retain serialization with insertMu. A version check keeps multi-key canonical transaction lookups coherent across reorganization.

The committed-cache path checks the failure/close fence under its own short mutex. Persistence errors never publish the in-memory selection, and subsequently cached reads report the fence. Execution sync progress uses an immutable atomic telemetry snapshot while application/target ordering remains serialized. No head, checkpoint or VALID result is returned early.

FCU still validates checkpoint ancestry, finalized monotonicity, validated execution and state roots on repeated calls. Its batch rewrites only changed durable mappings, transaction locations and pointers. Shared transactions deleted by an old-branch batch are always restored on the selected branch. An empty batch publishes memory-only safe changes without appending/syncing an empty WAL record. WAL and per-key fsync are retained for actual changes.

This repairs the demonstrated read-lock coupling and redundant I/O. A genuine changed durable write can still exceed the Engine deadline if the physical storage stalls; no synchronous durable database can promise otherwise. Further post-fix live observation must test that boundary and preserve any recurrence. A rapid replay or a controlled gate passing alone does not close the original long-run incident.

## Specification basis

Read-only frozen references: EIP-3675 (`refs/EIPs/EIPS/eip-3675.md`, execution validity and atomic head/marker updates) and Engine Paris (`refs/execution-apis/src/engine/paris.md`, newPayload/forkchoice validity, atomicity and 8-second response requirements). These changes concern storage scheduling and publication; they do not change consensus rules or relax BLS, hashes, state-root checks, resource limits or response deadlines.

## Causal verification

The committed-reader regression fails on the old chain code with `durable FCU I/O retained the chain read lock`; the progress regression fails with `progress read waits for durable apply callback`. Their setup compiled and reached the intended assertion. The fixed tests use the actual FileDB and chain cache/cold-read paths, verify the old committed view before publication, ensure cold I/O cannot retain the chain lock, and check the final head/state/checkpoints. Additional tests cover real materialization failure fencing, closed-cache fencing, zero-WAL empty publication, unchanged/checkpoint-only FCU writes and same-height reorg transaction-index preservation.

Runtime, full-suite and PR validation receipts are recorded in STATUS.md as they complete. No pending qualification is reported as passed.
