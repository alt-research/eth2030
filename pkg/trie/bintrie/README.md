# trie/bintrie — EIP-7864 binary Merkle trie

Experimental binary state trie with SHA-256 and EIP-7864 draft BLAKE3 hashing. This implementation is not activated in the node state lifecycle.

[← trie](../README.md)

## Overview

The binary trie organises state under 32-byte keys. The first 31 bytes form a **stem** that routes through `InternalNode` branch nodes; the final byte selects one of 256 leaves in a `StemNode`. Hashing uses SHA-256 by default and optionally BLAKE3 (`NewWithHashFunc(HashFunctionBlake3)`).

The legacy `GetBinaryTreeKey` helpers use SHA-256 and are not the pinned EIP-7864 key derivation. The BLAKE3 state APIs use `GetTreeKeyForBasicData`, `GetTreeKeyForCodeHash`, `GetTreeKeyForStorageSlot`, and `GetTreeKeyForCodeChunk`, which use the draft BLAKE3 and little-endian tree-index rules.

`BinaryTrie` exposes high-level account and storage APIs (`GetAccount`, `UpdateAccount`, `GetStorage`, `UpdateStorage`, `UpdateContractCode`) that encode/decode the `BasicDataLeaf` format (version, code size, nonce, balance) and manage code chunking via `ChunkifyCode` (which returns `[][32]byte` per spec).

Structured proofs in `proof_verifier.go` support SHA-256 and BLAKE3; select `NewBlake3ProofVerifier` for BLAKE3. `proof.go` is a separate legacy SHA-256-only proof API. Epoch metadata for state expiry is written via `UpdateLeafMetadata`. An `EpochUpdater` in `epoch_updater.go` performs batch expiry updates.

## Functionality

### Types

| Type | Purpose |
|------|---------|
| `BinaryTrie` | Top-level trie: Get/Put/Delete, account and storage helpers, Hash/Copy |
| `InternalNode` | Branch node routing on key bits |
| `StemNode` | Leaf array node holding 256 values under a 31-byte stem |
| `Empty` | Sentinel for an empty subtree |

### Persistence

`Persist` writes immutable, content-addressed nodes to a caller-provided key-value store and returns a root only after all node writes succeed. The caller must publish that root separately and provide the desired durability guarantees. `OpenPersisted` eagerly loads and authenticates nodes against that root, rejecting missing, corrupt, or misplaced nodes. The storage format is local to this package, not an EIP wire format. Eager loading and whole-tree traversal are not designed for production-scale state; this is not wired into node startup or fork activation.

### Key Functions

- `New()` / `NewWithHashFunc(hashFunc)` / `Get(key)` / `Put(key, value)` / `Delete(key)` / `Hash()` / `Persist(store)` / `OpenPersisted(store, hashFunc, root)`
- `GetAccount(addr)` / `UpdateAccount(addr, acc, codeLen)` / `GetStorage(addr, key)` / `UpdateStorage(addr, key, value)` / `DeleteStorage(addr, key)`
- `UpdateContractCode(addr, code)` / `UpdateStem(key, values)` / `UpdateLeafMetadata(stem, subindex, epoch)`
- `GetBinaryTreeKey(addr, key)` / `GetBinaryTreeKeyBasicData(addr)` / `GetBinaryTreeKeyCodeHash(addr)` / `GetBinaryTreeKeyStorageSlot(addr, key)` / `StorageIndex(storageKey)`
- `ChunkifyCode(code)` (returns `[][32]byte`) / `PackBasicDataLeaf` / `UnpackBasicDataLeaf`

## Usage

```go
t := bintrie.New()
t.UpdateAccount(addr, &types.Account{Nonce: 1, Balance: big.NewInt(1e18)}, 0)
t.UpdateStorage(addr, storageKey, value)
root := t.Hash()

acc, _ := t.GetAccount(addr)
```
