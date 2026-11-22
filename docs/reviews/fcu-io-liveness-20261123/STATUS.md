# Confirmed spontaneous FCU storage-wait amplification

The unchanged old-source diagnostic captured a spontaneous failure at approximately4h54m. Its first complete pre-debugger native snapshot shows goroutine58577 in FileDB.writeDataFile -> File.Sync during ApplyForkchoice, with the FileDB writer and Blockchain publication locks retained. NewPayload goroutine58585 waits in CheckStorage -> FileDB.Get; syncing requests wait in Blockchain.CurrentBlock.RLock. A later real rolling capture shows the same FCU path waiting in WAL fsync. Native cgroup I/O pressure was high while CPU throttling andOOM were absent.

This is an actual wait chain, not the injected commit-pause fixture. The fix must keep committed reads and progress responsive during durable I/O, retain atomic publication and error fencing, and avoid redundant forkchoice writes. No VALID/head/checkpoint may be published before persistence succeeds. The original03/04 evidence remains failed and separate; exact original04 identity cannot be proven without its missing stack.

## Implementation and local verification

Implemented durable-before-publication FileDB batches, independently readable committed cache with persistent-error/close fencing, cold-reader lock release and canonical transaction fallback versioning, immutable sync-progress telemetry, and changed-key-only FCU writes. Validity, state roots, finalized ancestry and durability remain required. The same-height shared-transaction reorg case is explicitly covered.

Go 1.26.11: `go test ./... -count=1 -timeout=10m` passed; `go vet ./...` passed; the native `./cmd/eth2030` build passed. Targeted race passed for core/rawdb (1.823s), core/chain (6.257s), sync/beacon (62.684s). The compiled old-source causal head/progress failures are retained separately. Native mature-state replay and post-fix elapsed-time/BLS qualification are pending; no long-run closure is claimed.
