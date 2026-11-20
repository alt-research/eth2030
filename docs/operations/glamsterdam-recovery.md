# Glamsterdam qualified development network operations

## Supported target and identity

This runbook applies to the explicitly pinned private-chain implementation, not a public Ethereum/mainnet release. Resolve the final campaign manifest before deployment: source commits, executable SHA-256, image IDs, genesis, fork schedule, resource limits, cryptographic backends and independent clients must match. The selected merged LIFO gas model and signed payload-envelope format remain in force. Open alternatives are not implicitly compatible.

Use the native `cmd/eth2030` executable. `cmd/eth2030-geth` is a different backend and cannot replace native execution acceptance. Build from a clean recorded composition with Go 1.26.11, `-trimpath`, and the actual source revision in `main.commit`. Preserve the build command and resulting executable digest. An image tag or CLI version alone cannot establish the running inode; the campaign capture tool hashes `/proc/<pid>/exe` inside the exact container.

## Safe rolling EL upgrade

1. Discover the exact owned service and live published ports from Docker/Kurtosis. Require the expected chain ID and exact genesis hash. Capture common finalized hashes, full commitments, original application receipts/getters/proofs and the runtime/resource inventory.
2. Upgrade one original pair at a time. Stop the EL and verify its process ended and its database lock is available. Preserve the stopped complete datadir, original genesis/configuration and exact previous executable. Do not copy validator keys or run duplicate validator clients.
3. Build missing immutable authenticated history with the maintenance executable from the same recorded source:

```sh
eth2030-state-history --datadir=/path/to/original/execution-data --genesis=/path/to/original/genesis.json
```

The command requires exclusive access and never starts network/RPC/validators. It follows only the original locally validated canonical path and checks reconstructed roots. SIGTERM cancels the operation and retains completed atomic index batches. A completed index is not itself proof that the normal node starts or catches up within its separate deadline.

4. Start the new native executable with the original datadir/genesis and recorded fork schedule. Capture startup duration separately from RPC polling and catch-up. Require nonoptimistic CL, expected finalized ancestry, independent full header/state/receipt commitments and exact retained historical consumers. Only then proceed to the next pair.
5. Keep the original restart/full/snap budgets (300/3600/600 seconds for the recorded roles). A missed deadline remains a failed attempt. Investigate the cause before declaring a new attempt with a new source or workload tuple.

## Storage failure and crash recovery

The database's durable WAL commit is the recovery authority. A materialization error may occur after commit, so a failed RPC cannot by itself establish that the operation was rolled back. FileDB fences further reads/writes after a failed write; Engine must preserve a local storage error rather than return fabricated state, consensus INVALID or peer SYNCING. Stop the process and repair the actual filesystem problem before reopening.

Preserve the complete failed datadir and WAL. Never delete, truncate, synthesize or selectively copy its WAL/head/finality files. Reopen through normal FileDB recovery: complete committed intent is materialized, and an incomplete suffix is discarded. Require exact recovered head, safe/finalized ancestry, authenticated state and normal next-block execution. Restore an older executable only with its matching stopped database/genesis backup unless cross-version recovery has been explicitly qualified. Retain post-upgrade data separately; do not overwrite it with the rollback backup.

The process qualification uses three actual OS faults on isolated data: a temporary head-file directory collision (EISDIR), a process file-size limit permitting only one additional WAL byte (short write/EFBIG), and a dedicated limited tmpfs filled to ENOSPC. SIGKILL and restart happen after the fault. The expected head is derived from actual complete WAL commits, never from a permissive choice of either old or new head. These tests do not authorize fault injection into an unrelated or production database.

## Partial data and availability

The qualified peer target uses partial-message group version 0x01 and strict bitmap/row/index/KZG checks. To observe actual peer acquisition, the receiving CL uses its existing `--disable-get-blobs` option; ordinary full gossip fallback and all validation remain enabled. Capture useful verified partial-cell counters and actual imported commitments so an HTTP response or fallback alone cannot be counted as partial acquisition. Unknown extension versions are ignored according to the pinned spec; known-invalid messages retain rejection/scoring behavior.

## Historical retention and pruning

Record retention policy and wall-clock/epoch schedule before running pruning. EIP-7928 requires exact BAL availability for at least 14,299 epochs; block count cannot substitute for elapsed epochs when slots are missed. Distinguish unavailable history, explicitly pruned state, malformed/corrupt committed data and temporary storage failure. Never treat missing/corrupt committed state as zero or silently replace the root. Full startup/maintenance audits remain explicit even though normal point reads and persistence use authenticated changed paths.

## Qualification and release handoff

Run the final steady and valid/invalid pressure profiles only after source, actual binaries and resource limits are frozen. Retain complete failed RPC/Engine samples, original missing slots and independently authenticated finality. Changing resources, peers, binaries or workload during a window invalidates that window's unchanged-tuple claim. The original 129/400 missing-slot failure remains separate. Publish compatible artifacts, license/source inventory, supported boundaries, exact reproducible commands and all outstanding gates before a release decision.
