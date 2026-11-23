# Actual original waiting-chain investigation

The original04 native/CL source diagnostic is still running without restarting either daemon. It has not yet reached the original roughly5.5-hour failure time. No actual original Engine timeout/waiting holder has been captured and the root remains unresolved. This report does not reinterpret the controlled recovery passes as original-fault closure.

## Original stopped-database evidence

Read-only metadata from the untouched original stopped database shows the final WAL write at10:43:41.604Z, the first block77623 header file write at10:43:41.630Z, its validation marker at10:43:41.649Z and its history-root publication at10:43:41.782Z. The main thread logged SIGTERM at10:43:41.649Z and accepted the payload at10:43:41.793Z.

WAL writing and header materialization therefore preceded the logged shutdown. The timing does not establish that an explicit Node.Stop cancellation released the original dependency. It also does not identify whether the earlier delay was in execution, waiting for a database mutex, a write syscall or another stage. Do not patch an assumed holder from these timestamps alone. The original files were inspected without opening/replaying FileDB or modifying their contents.

## Background workload and passive capture

The original operation transcript also records preparation of another private snapshot: a separate approximately1.64-million-entry EL archive was restored with idle-class ionice, completing at10:41:17.655Z before the failing lookup. That is an operational correlation, not evidence that its I/O caused the native wait. A repeat of that exact restoration into a new unrelated directory completed in60.171s; no sustained native request or Engine timeout was observed in the surrounding actual samples so far. This does not rule out all resource/timing combinations.

The unchanged diagnostic now has an additional read-only native profile sampler. It retains a bounded128-profile ring, records actual backend goroutines observed continuously for at least3s, and preserves the first/current/past-deadline stacks. That3s value is an evidence trigger, not a changed Engine deadline or automatic deadlock claim. No pause, clock manipulation, fake payload, daemon restart or validation bypass is introduced. Sampling has taken about1–2ms per read on the observed healthy node. The existing early8s timeout collector still captures both native and CL evidence independently.

Docker retained-event history for the original period was empty and may have expired; that absence is not used to prove no interference. The available operation transcript shows snapshot preparation and an independent image build, not a proven native pause or actual waiting holder. Raw private logs and the original failed03/04 outcomes remain retained. Actual waiting-chain capture is still required before claiming an original root fix.
