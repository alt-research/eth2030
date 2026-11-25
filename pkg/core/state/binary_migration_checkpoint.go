package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/trie"
	"github.com/eth2030/eth2030/trie/bintrie"
)

var ErrBinaryMigrationCheckpoint = errors.New("binary migration: invalid checkpoint")

// BinaryMigrationProgress is an experimental local checkpoint. Root is an
// intermediate tree root until Complete is true; it is not a canonical root.
type BinaryMigrationProgress struct {
	SourceRoot types.Hash
	Root       types.Hash
	Accounts   uint64
	Complete   bool
	cursor     types.Address
	hasCursor  bool
}

func binaryMigrationCheckpointKey(root types.Hash) []byte {
	return append([]byte("eth2030:binary-migration:v1:"), root[:]...)
}

func encodeBinaryMigrationProgress(p BinaryMigrationProgress) []byte {
	data := make([]byte, 63)
	data[0] = 1
	if p.hasCursor {
		data[1] = 1
	}
	if p.Complete {
		data[2] = 1
	}
	copy(data[3:23], p.cursor[:])
	copy(data[23:55], p.Root[:])
	binary.BigEndian.PutUint64(data[55:], p.Accounts)
	return data
}
func decodeBinaryMigrationProgress(source types.Hash, data []byte) (BinaryMigrationProgress, error) {
	p := BinaryMigrationProgress{SourceRoot: source}
	if len(data) != 63 || data[0] != 1 || data[1] > 1 || data[2] > 1 {
		return p, ErrBinaryMigrationCheckpoint
	}
	p.hasCursor, p.Complete = data[1] == 1, data[2] == 1
	copy(p.cursor[:], data[3:23])
	copy(p.Root[:], data[23:55])
	p.Accounts = binary.BigEndian.Uint64(data[55:])
	if !p.hasCursor && (p.cursor != (types.Address{}) || p.Accounts != 0 || p.Root != (types.Hash{})) {
		return p, ErrBinaryMigrationCheckpoint
	}
	if p.hasCursor && (p.Accounts == 0 || p.Root == (types.Hash{})) {
		return p, ErrBinaryMigrationCheckpoint
	}
	return p, nil
}

type binaryMigrationAccount struct {
	address types.Address
	body    []byte
}

// MigrateCommittedBinaryBatch copies up to limit accounts from a quiescent
// committed TrieStateDB. Each call reconstructs and checks the trusted source
// account root, verifies each copied account's storage and code, writes binary
// nodes, then stores a checkpoint. A caller must exclude concurrent writers and
// provide a durable database; this does not publish or activate a state root.
// Recovery opens and authenticates the prior checkpoint's node root.
func MigrateCommittedBinaryBatch(source rawdb.Database, expectedRoot types.Hash, dest rawdb.Database, limit int) (BinaryMigrationProgress, error) {
	progress := BinaryMigrationProgress{SourceRoot: expectedRoot}
	if source == nil || dest == nil || limit <= 0 {
		return progress, ErrBinaryMigrationCheckpoint
	}
	key := binaryMigrationCheckpointKey(expectedRoot)
	encoded, err := dest.Get(key)
	if err == nil {
		progress, err = decodeBinaryMigrationProgress(expectedRoot, encoded)
		if err != nil {
			return BinaryMigrationProgress{}, err
		}
	} else if !errors.Is(err, rawdb.ErrNotFound) {
		return BinaryMigrationProgress{}, err
	}

	// Iterate the complete account set on each call: a changed source must not
	// be resumed under a previously checked root. Keep only this batch in memory.
	mpt := trie.New()
	iter := source.NewIterator(dbPrefixAccount)
	defer iter.Release()
	pending := make([]binaryMigrationAccount, 0)
	var total, cursorIndex uint64
	var cursorFound bool
	var last types.Address
	for iter.Next() {
		key, body := iter.Key(), iter.Value()
		if len(key) != len(dbPrefixAccount)+20 {
			return BinaryMigrationProgress{}, ErrBinaryMigrationSource
		}
		var address types.Address
		copy(address[:], key[len(dbPrefixAccount):])
		if total > 0 && bytes.Compare(last[:], address[:]) >= 0 {
			return BinaryMigrationProgress{}, ErrBinaryMigrationSource
		}
		last = address
		total++
		if err := mpt.Put(crypto.Keccak256(address[:]), body); err != nil {
			return BinaryMigrationProgress{}, err
		}
		if progress.hasCursor && address == progress.cursor {
			cursorFound, cursorIndex = true, total
		}
		if (!progress.hasCursor || bytes.Compare(address[:], progress.cursor[:]) > 0) && len(pending) < limit {
			pending = append(pending, binaryMigrationAccount{address: address, body: bytes.Clone(body)})
		}
	}
	if err := rawdb.ReadError(iter); err != nil {
		return BinaryMigrationProgress{}, err
	}

	if mpt.Hash() != expectedRoot {
		return BinaryMigrationProgress{}, ErrBinaryMigrationSource
	}
	if progress.hasCursor && (!cursorFound || cursorIndex != progress.Accounts) {
		return BinaryMigrationProgress{}, ErrBinaryMigrationCheckpoint
	}
	if progress.Complete && progress.Accounts != total {
		return BinaryMigrationProgress{}, ErrBinaryMigrationCheckpoint
	}
	tree, err := bintrie.OpenPersisted(dest, bintrie.HashFunctionBlake3, progress.Root)
	if err != nil {
		return BinaryMigrationProgress{}, fmt.Errorf("binary migration: restore checkpoint: %w", err)
	}
	if progress.Complete {
		return progress, nil
	}
	for _, account := range pending {
		if err := migrateBinaryAccount(source, account.address, account.body, tree); err != nil {
			return BinaryMigrationProgress{}, err
		}
		progress.cursor, progress.hasCursor = account.address, true
		progress.Accounts++
	}
	root, err := tree.Persist(dest)
	if err != nil {
		return BinaryMigrationProgress{}, err
	}
	progress.Root, progress.Complete = root, progress.Accounts == total
	if err := dest.Put(binaryMigrationCheckpointKey(expectedRoot), encodeBinaryMigrationProgress(progress)); err != nil {
		return BinaryMigrationProgress{}, err
	}
	return progress, nil
}
