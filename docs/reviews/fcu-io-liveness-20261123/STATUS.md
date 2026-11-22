# Confirmed spontaneous FCU storage-wait amplification

The unchanged old-source diagnostic captured a spontaneous failure at approximately4h54m. Its first complete pre-debugger native snapshot shows goroutine58577 in FileDB.writeDataFile -> File.Sync during ApplyForkchoice, with the FileDB writer and Blockchain publication locks retained. NewPayload goroutine58585 waits in CheckStorage -> FileDB.Get; syncing requests wait in Blockchain.CurrentBlock.RLock. A later real rolling capture shows the same FCU path waiting in WAL fsync. Native cgroup I/O pressure was high while CPU throttling andOOM were absent.

This is an actual wait chain, not the injected commit-pause fixture. The fix must keep committed reads and progress responsive during durable I/O, retain atomic publication and error fencing, and avoid redundant forkchoice writes. No VALID/head/checkpoint may be published before persistence succeeds. The original03/04 evidence remains failed and separate; exact original04 identity cannot be proven without its missing stack.

## Implementation and local verification

Implemented durable-before-publication FileDB batches, independently readable committed cache with persistent-error/close fencing, cold-reader lock release and canonical transaction fallback versioning, immutable sync-progress telemetry, and changed-key-only FCU writes. Validity, state roots, finalized ancestry and durability remain required. The same-height shared-transaction reorg case is explicitly covered.

Go 1.26.11: `go test ./... -count=1 -timeout=10m` passed; `go vet ./...` passed; the native `./cmd/eth2030` build passed. Targeted race passed for core/rawdb (1.823s), core/chain (6.257s), sync/beacon (62.684s). The compiled old-source causal head/progress failures are retained separately. Native mature-state replay and post-fix elapsed-time/BLS qualification are pending; no long-run closure is claimed.

## Mature-state and composition checks

Qualification-only composition `db4781de0ce97f90c015cbeca9784b6ae90a455b` combines this production fix with pending profiling, mature replay and startup-history prerequisites. The actual native node test replayed 256 genuine reference blocks 77623–77878 from the immutable original04 snapshot, in 109.917s execution /176.926s total. Native execution, Engine repeat import, canonical selection, independent MPT/hash/root checks and durable restart all passed under2CPU/4096000000-byte limits; exit0, noOOM. See MATURE_REPLAY.json. It is not elapsed-time or BLS qualification.

The composed rawdb, chain, beacon-sync, node and backend suites also passed. All8 open EL PRs131–138 target master. At the recorded exact heads, merge simulation passed56 ordered pairs,1024 subset transitions with equal resulting trees, and16 explicitly executed complete orders placing this fix at each position in ascending and descending orders. This is not an exhaustive test of40320 distinct histories. No master merge was performed. See MERGE_AUDIT.json.

PR138: https://github.com/alt-research/eth2030/pull/138. A new6h45m diagnostic with the exact pinned native/CL binaries started at2026-11-22T18:36:54Z in an isolated copy of the stopped spontaneous-failure databases. Startup/catchup and cgroup memory-limit pressure remain recorded separately. Only the original unreadable P2P private key and three historical CL log files were excluded from the copy; database files were copied and original evidence remains untouched. A1-second rolling goroutine collector retains128 samples and pins sustained3s/8s requests. This new run is pending, not an accepted48h clock. New-span independent BLS and stopped-head closure are pending.

## Independent BLS evidence remains separate

The initial attempt to authenticate the entire failed diagnostic interval82280–88288 with the unchanged reference failed its4096-link bound; this failure is retained. A fresh bounded-tail attempt also rejected a real reference finality response at signature slot89844 with zero sync-committee bits. No BLS or ancestry assertion was relaxed and neither attempt is a pass. Later attempts/segmented ancestry, if valid, must preserve these observations and their provenance. These are audit/reference boundaries, not proof that the EL I/O defect remains or is closed.
