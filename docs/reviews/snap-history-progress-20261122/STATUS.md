# Historical prefix validation blocks native imports

A deterministic mature-state replay starting at the actual failed04 empty block77623 failed the8s import budget. Its before-cleanup goroutines show InsertSyncedBlockWithResult waiting on insertMu while the startup snap-history worker traverses the entire saved prefix under that mutex. This is a real reproduced wait, not a transaction loop or a guessed lock cycle. SIGTERM cancels this worker, explaining how this mechanism can release pending imports on stop; it does not independently prove it was the sole cause of the original multi-hour incidents.

The fix must snapshot the already-finalized pivot/frontier under the mutation lock, then authenticate the immutable hash-linked prefix outside it. All existing header/body/BAL/retention/canonical checks remain required; downloaded history must never acquire execution authority. StageSnapHistoryParent and CompleteSnapHistoryBlock retain their serialized commit/check operations.

## Implemented and verified

Snapshot the validated finalized pivot/frontier under insertMu, then perform the unchanged hash-linked header/body/canonical/BAL authentication without that import mutex. Each frontier commit retains its live-cursor comparison and serialized atomic write. A final context check preserves cancellation arriving during the last historical read. No hash, state, signature or retention rule is disabled.

The deterministic barrier regression fails on old code because both native import and FCU wait for the historical read. Fixed code passes20 repetitions in0.881s; three targeted race repetitions pass4.294s. Complete core/chain, node, node/backend, engine and sync/beacon suites pass. Composed replay/test branch passes all219 package suites, affected vet and five native replay race repetitions. Native executable builds with Go1.26.11.

The same immutable mature snapshot and real Geth RLP77623–77878 qualify on the fix:256 blocks, forced execution through core and Engine, repeated payloads, FCU, concurrent independent SDK MPT checks and a durable reopen;295.296s including restore/download,232.538s node execution/restart. Resources remain2CPU/4096000000B; exit0/noOOM. The old-source controlled replay failed at block77623 after its8s gate and retained live stacks. See the independent test/driver PR and its RESULTS.json for exact inputs and executable pins.

Important counterevidence: fetching the complete original04 native log showed history completion at2026-11-22T05:10:33.486Z, before its05:11:01Z acceptance clock. The reproduced startup critical section is a separate confirmed defect; original03/04 multi-hour liveness attribution remains open. Do not use this PR to declare48-hour acceptance or the original incident resolved.
