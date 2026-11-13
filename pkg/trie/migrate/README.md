# trie/migrate — MPT-to-binary-trie migration

Experimental migration utilities for a legacy `mpt.BinaryTrie`. These utilities do not produce an EIP-7864 state tree or a production migration checkpoint.

[← trie](../README.md)

## Overview

`MigrateFromMPT` is the simple one-shot converter: it iterates all key-value pairs from the source MPT, rehashes each key with Keccak256 (matching binary trie key derivation), and inserts them into a new `mpt.BinaryTrie`.

`MPTToBinaryTrieMigrator` is an experimental migrator for that legacy tree. It collects all MPT pairs once, then exposes `MigrateBatch()` which processes one batch per call, charges gas via a `GasAccountant`, and saves a `MigrationCheckpoint` after each batch for crash recovery. The migrator is composed of:

- `BatchConverter` — inserts up to `batchSize` pairs per call into the destination trie.
- `AddressSpaceSplitter` — divides the 256-bit key space into N equal ranges for parallel migration workers.
- `StateProofGenerator` — caches MPT inclusion proofs generated during migration for cross-verification.
- `MigrationCheckpointer` — stores a stack of `MigrationCheckpoint` values with `Latest()` for resume.
- `GasAccountant` — tracks read/write/proof costs against a gas budget.

`OverlayMigrator` (`overlay.go`) and `MigrationPlanner` (`migration_planner.go`) provide experimental overlay and planning primitives.

For the project's committed `TrieStateDB` flat tables, `state.MigrateCommittedToBinary` separately builds a draft EIP-7864 BLAKE3 tree using account addresses and storage-slot preimages. It verifies the trusted account root, each storage root and code hash before persisting binary nodes. It requires a quiescent source, does not publish or activate the resulting root, and does not supply a scalable, checkpointed fork migration. A generic MPT with only hashed keys cannot supply the required address and slot preimages.

## Functionality

### Types

| Type | Purpose |
|------|---------|
| `MPTToBinaryTrieMigrator` | Full-featured orchestrator (batch, gas, checkpoint, proofs) |
| `BatchConverter` | Converts fixed-size key-value batches into the destination trie |
| `AddressSpaceSplitter` | Splits 256-bit space into N `AddressRange` slices |
| `MigrationCheckpointer` | Stores/retrieves `MigrationCheckpoint` for crash recovery |
| `GasAccountant` | Tracks per-op gas against a budget |
| `StateProofGenerator` | Generates and caches MPT proofs during migration |

### Key Functions

- `MigrateFromMPT(source)` — simple one-shot migration
- `NewMPTToBinaryTrieMigrator(source, batchSize, gasBudget)` / `MigrateBatch()` / `Destination()` / `KeysMigrated()`
- `NewBatchConverter(batchSize)` / `ConvertBatch(pairs, dest)`
- `NewAddressSpaceSplitter(n)` / `Ranges()` / `InRange(key, r)`
- `NewMigrationCheckpointer()` / `Save(cp)` / `Latest()`
- `NewGasAccountant(budget, perRead, perWrite, perProof)` / `ChargeRead()` / `ChargeWrite()` / `ChargeProof()`

## Usage

```go
migrator, _ := migrate.NewMPTToBinaryTrieMigrator(sourceTrie, 1000, 1_000_000)
for {
    count, done, err := migrator.MigrateBatch()
    if err != nil || done { break }
    _ = count
}
binaryTrie := migrator.Destination()
```
