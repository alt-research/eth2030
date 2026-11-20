package state

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	"github.com/eth2030/eth2030/trie/mpt"
)

func historyRootKey(root types.Hash) []byte { return append([]byte("mpt-history-root/"), root[:]...) }
func historyNodeKey(hash types.Hash) []byte { return append([]byte("mpt-history-node/"), hash[:]...) }

// HistoryStateAvailable distinguishes an absent old index from corrupt metadata.
func HistoryStateAvailable(db rawdb.KeyValueReader, root types.Hash) (bool, error) {
	marker, err := db.Get(historyRootKey(root))
	if errors.Is(err, rawdb.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !bytes.Equal(marker, []byte{1}) {
		return false, fmt.Errorf("state history: invalid root marker")
	}
	return true, nil
}

type historyNodeWriter struct {
	db    rawdb.KeyValueReader
	batch rawdb.Batch
}

func immutableHistoryPut(db rawdb.KeyValueReader, batch rawdb.Batch, key, value []byte) error {
	old, err := db.Get(key)
	if err == nil {
		if !bytes.Equal(old, value) {
			return fmt.Errorf("state history: conflicting immutable data")
		}
		return nil
	}
	if !errors.Is(err, rawdb.ErrNotFound) {
		return err
	}
	return batch.Put(key, value)
}
func (w historyNodeWriter) Put(hash types.Hash, data []byte) error {
	if crypto.Keccak256Hash(data) != hash {
		return fmt.Errorf("state history: encoded node commitment mismatch")
	}
	return immutableHistoryPut(w.db, w.batch, historyNodeKey(hash), data)
}

// StoreTrieHistory adds immutable account/storage trie nodes and code to the
// caller's block batch. The root marker is not visible before that batch commits.
func StoreTrieHistory(db rawdb.KeyValueReader, batch rawdb.Batch, root types.Hash, source StateDB) error {
	if available, err := HistoryStateAvailable(db, root); err != nil || available {
		return err
	}
	snapshot, err := ExportStateSnapshot(source)
	if err != nil {
		return err
	}
	if err := snapshot.Validate(root); err != nil {
		return err
	}
	nodes := mpt.NewNodeDatabase(nil)
	for account, slots := range snapshot.Storage {
		var value rlpAccount
		if err := rlp.DecodeBytes(snapshot.Accounts[account], &value); err != nil {
			return err
		}
		computed, err := mpt.CommitTrie(snapshotTrie(slots), nodes)
		if err != nil {
			return err
		}
		if computed != types.BytesToHash(value.Root) {
			return fmt.Errorf("state history: storage commitment mismatch")
		}
	}
	computed, err := mpt.CommitTrie(snapshotTrie(snapshot.Accounts), nodes)
	if err != nil {
		return err
	}
	if computed != root {
		return fmt.Errorf("state history: account commitment mismatch")
	}
	if err := nodes.Commit(historyNodeWriter{db: db, batch: batch}); err != nil {
		return err
	}
	for hash, code := range snapshot.Codes {
		if crypto.Keccak256Hash(code) != hash {
			return fmt.Errorf("state history: code commitment mismatch")
		}
		if err := immutableHistoryPut(db, batch, append([]byte("snap-code/"), hash[:]...), code); err != nil {
			return err
		}
	}
	return batch.Put(historyRootKey(root), []byte{1})
}

type historyNodeReader struct {
	ctx context.Context
	db  rawdb.KeyValueReader
}

func (r historyNodeReader) Node(hash types.Hash) ([]byte, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	data, err := r.db.Get(historyNodeKey(hash))
	if err != nil {
		return nil, fmt.Errorf("state history: missing/unreadable node %s: %v", hash, err)
	}
	if crypto.Keccak256Hash(data) != hash {
		return nil, fmt.Errorf("state history: corrupt node %s", hash)
	}
	return data, nil
}
func historyLeaves(ctx context.Context, nodes *mpt.NodeDatabase, root types.Hash) (map[types.Hash][]byte, error) {
	tree, err := mpt.NewResolvableTrie(root, nodes)
	if err != nil {
		return nil, err
	}
	leaves := make(map[types.Hash][]byte)
	iterator := mpt.NewResolvableIterator(tree)
	for iterator.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(iterator.Key) != 32 {
			return nil, fmt.Errorf("state history: invalid secure trie key")
		}
		leaves[types.BytesToHash(iterator.Key)] = bytes.Clone(iterator.Value)
	}
	if err := iterator.Err(); err != nil {
		return nil, err
	}
	return leaves, nil
}

// OpenTrieHistory materializes a verified native secure snapshot at exactly the
// original root. It never consults mutable current-state account/storage keys.
func OpenTrieHistory(ctx context.Context, db rawdb.Database, root types.Hash) (*SnapStateDB, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	available, err := HistoryStateAvailable(db, root)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, rawdb.ErrNotFound
	}
	nodes := mpt.NewNodeDatabase(historyNodeReader{ctx: ctx, db: db})
	snapshot := NewStateSnapshot()
	snapshot.Accounts, err = historyLeaves(ctx, nodes, root)
	if err != nil {
		return nil, err
	}
	for account, body := range snapshot.Accounts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var value rlpAccount
		if err := rlp.DecodeBytes(body, &value); err != nil {
			return nil, err
		}
		if len(value.Root) != 32 || len(value.CodeHash) != 32 {
			return nil, fmt.Errorf("state history: invalid account encoding")
		}
		storage, err := historyLeaves(ctx, nodes, types.BytesToHash(value.Root))
		if err != nil {
			return nil, err
		}
		if len(storage) > 0 {
			snapshot.Storage[account] = storage
		}
		codeHash := types.BytesToHash(value.CodeHash)
		if codeHash != types.EmptyCodeHash {
			code, err := db.Get(append([]byte("snap-code/"), codeHash[:]...))
			if err != nil {
				return nil, fmt.Errorf("state history: missing/unreadable code: %v", err)
			}
			if crypto.Keccak256Hash(code) != codeHash {
				return nil, fmt.Errorf("state history: corrupt code")
			}
			snapshot.Codes[codeHash] = code
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := snapshot.Validate(root); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Execution overlays stay in memory. Publishing a newly validated root is
	// owned by StoreTrieHistory and its caller's atomic block batch, rather than
	// writing a duplicate full flat snapshot for each intermediate replay root.
	return &SnapStateDB{memory: NewMemoryStateDB(), base: snapshot}, nil
}
