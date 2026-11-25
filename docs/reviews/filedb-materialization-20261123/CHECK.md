# Captured catch-up materialization convoy

- [x] Capture actual Go insertMu waiters and FileDB owner, fd/inode/device/syscall/wchan.
- [x] Review Engine Paris mandatory validation/atomicity; do not synthesize VALID/SYNCING to bypass pending durable work.
- [x] Replay the observed12056-entry canonical batch shape on the same filesystem.
- [x] Bound concurrent independent-key materialization; retain WAL commit, per-file sync, atomic cache/head publication and failure fencing.
- [x] Exercise duplicate-key order, failure, restart, concurrency and existing durable publication controls.
- [x] Measure before/after and publish a separate master PR with explicit source/runtime limits.
