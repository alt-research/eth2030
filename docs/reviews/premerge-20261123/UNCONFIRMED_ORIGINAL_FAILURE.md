# Original import-stall questions that remain unconfirmed

Source merge and a completed bounded campaign do not settle every historical or physical question. Keep these items separate from already repaired defects.

## Established facts

- Original rounds03/04 failed. Original04's empty native block77623 was only logged accepted near shutdown after an approximately5.5h observed prefix. Its actual waiting goroutine/dependency was not captured. Its original data, logs and failed verdict remain retained.
- An unchanged-core old-source diagnostic later failed naturally at approximately4h54m. Its pre-debugger stack showed actual FCU per-key fsync retaining FileDB and Blockchain locks; newPayload CheckStorage and eth_syncing waited behind them. A second real rolling capture showed FCU WAL fsync across9.799s. This confirms a real storage-wait/read-lock convoy defect, independent of controlled pause fixtures.
- PR138 repairs the demonstrated read/publication coupling and redundant FCU writes. Compiled old-source causal regressions fail at the intended assertions; durable publication, error fencing, cold reader order, reorg lookup, race and actual native/mature-state tests pass. Required fsync, signatures, state roots, finalized ancestry and VALID ordering remain enforced.
- The corrected composed runtime completed6h45m5s; post-readiness5.5h was actually covered.144 later healthy samples,72 complete stack receipts and1451 completed newPayload responses cover another72min. The actual stopped slot97868, ten beyond the last health sample, is cryptographically bound through unchanged BLS ancestry/full beacon SSZ/execution header/MPT checks. This is positive bounded supporting evidence, not retrospective reconstruction.

## Still unconfirmed / not achieved

| Item | Why it remains open | What would justify a stronger claim |
| --- | --- | --- |
| Exact holder of the original04 incident | No contemporaneous stack. Later matching-source recurrence establishes a real defect but not independent identity of that earlier holder. | Preserve this irreducible historical boundary; do not claim new tests can recreate missing original evidence. New actual recurrence traces can establish a current mechanism, separately. |
| Physical cause of the actual old fsync delay | Real pressure and syscall sites were captured; device queue, filesystem/writeback and exact competing I/O are not causally separated. | Correlated actual fsync/fd/path timing and block-I/O/cgroup traces, with the measured workload/source retained. Archive-restore timing correlation alone is insufficient. |
| Every future durable write meeting8s | Serialization and durability still require actual storage completion. Cached health reads staying responsive does not bound a blocked device or make the write complete. | A declared operational envelope, real I/O fault/latency attribution and measured recovery; never premature VALID or skipped fsync. |
| Strict48h / larger-state multi-day admission | The actual completed interval is bounded6h45m; no172800s pass exists. Metadata/startup/history cardinality still grows. | A fresh declared final source/CL/fixture tuple and complete actual duration/resource/readback/BLS closure, if that acceptance is requested. Do not pad old prefixes or count post-processing as uptime. |
| Full corpus / public fork readiness | Native/Geth and CL fixture coverage is finite.152 selected real EF cases do not convert24 BLS skips or43 inactive handlers into executed passes. CL standalone hosted CI remains red. | Actual execution of specifically adopted additional cases and a complete compatible composed CI tuple, with failures/skips retained. Public protocol/model activation remains the user's reserved decision. |
| Cause of three late local envelope observations | Slots97047/97077/97567 arrived by block lookup near2.19s and envelope RPC after the boundary; EL validation was64–66ms. The configured local policy warns on locally late reveal. | Actual gossip delivery/peer/request correlation for those events. Local late receipt is not proof of global builder withholding or an EL fsync stall. |
| Upstream Lodestar integration | PR10693 is still open. Current checks only validate title; full hosted/maintainer lock review is not established. | Upstream approval/full hosted qualification and dependency integration. Local/process evidence cannot substitute for the external guard. |

## Claims explicitly rejected

- Zero sustained events in24244 successful post-fix rolling samples means only that the configured sampling did not capture a sustained event in that interval. It does not prove the historical issue is impossible.
- An artificial hold/release, delayed response or controlled recovery is not the original blocker. Those tests remain valuable causal/recovery controls within their declared scope.
- Original04 WAL/header writes precede the logged SIGTERM. Acceptance near shutdown does not by itself prove shutdown cancellation freed the operation.
- Clean exits/noOOM do not turn a timed-out campaign into a pass, nor does noOOM erase the retained686 memory-limit events.
- BLS/finality ancestry authenticity does not make missing slots disappear or grant an operating-budget pass.73 early missing slots remain reported.
- A merge SHA, compiler pass, PR count or green title check is not a new runtime measurement or complete hosted CL qualification.

The known FCU software defect has a concrete fix and bounded qualification. The statements above remain explicit unknowns/gates; no original raw failed verdict is edited or promoted.
