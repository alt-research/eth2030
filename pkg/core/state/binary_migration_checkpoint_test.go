package state

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/trie/bintrie"
)

type failCheckpointDB struct{ rawdb.Database }

func (db failCheckpointDB) Put(key, value []byte) error {
	if bytes.HasPrefix(key, []byte("eth2030:binary-migration:v1:")) {
		return errors.New("injected checkpoint failure")
	}
	return db.Database.Put(key, value)
}

func TestBinaryMigrationCheckpointRestartAndFailedPublish(t *testing.T) {
	source := rawdb.NewMemoryDB()
	src := NewTrieStateDB(source)
	accounts := []types.Address{{}, {1}, {2}}
	for i, address := range accounts {
		src.AddBalance(address, big.NewInt(int64(10+i)))
		src.SetCode(address, []byte{0x60, byte(i), 0x50})
		src.SetState(address, types.Hash{31: byte(i)}, types.Hash{31: byte(i + 1)})
	}
	sourceRoot, err := src.Commit()
	if err != nil {
		t.Fatal(err)
	}
	expectedStore := rawdb.NewMemoryDB()
	expectedRoot, err := MigrateCommittedToBinary(source, sourceRoot, expectedStore)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	dest, err := rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := MigrateCommittedBinaryBatch(source, sourceRoot, dest, 1)
	if err != nil || first.Complete || first.Accounts != 1 || !first.hasCursor || first.cursor != (types.Address{}) {
		t.Fatalf("first batch: %+v %v", first, err)
	}
	if _, err := MigrateCommittedBinaryBatch(source, sourceRoot, failCheckpointDB{dest}, 1); err == nil {
		t.Fatal("expected checkpoint write failure")
	}
	if err := dest.Close(); err != nil {
		t.Fatal(err)
	}
	dest, err = rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MigrateCommittedBinaryBatch(source, sourceRoot, dest, 1)
	if err != nil || second.Complete || second.Accounts != 2 {
		t.Fatalf("second batch: %+v %v", second, err)
	}
	if err := dest.Close(); err != nil {
		t.Fatal(err)
	}
	dest, err = rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer dest.Close()
	final, err := MigrateCommittedBinaryBatch(source, sourceRoot, dest, 1)
	if err != nil || !final.Complete || final.Accounts != 3 || final.Root != expectedRoot {
		t.Fatalf("final batch: %+v %v, want %x", final, err, expectedRoot)
	}
	tree, err := bintrie.OpenPersisted(dest, bintrie.HashFunctionBlake3, final.Root)
	if err != nil {
		t.Fatal(err)
	}
	for i, address := range accounts {
		got, err := tree.GetAccount(address)
		if err != nil || got == nil || got.Balance.Cmp(big.NewInt(int64(10+i))) != 0 {
			t.Fatalf("account %d: %+v %v", i, got, err)
		}
		code, err := tree.GetContractCode(address)
		if err != nil || !bytes.Equal(code, []byte{0x60, byte(i), 0x50}) {
			t.Fatalf("code %d: %x %v", i, code, err)
		}
	}
	again, err := MigrateCommittedBinaryBatch(source, sourceRoot, dest, 1)
	if err != nil || again != final {
		t.Fatalf("repeated completed batch: %+v %v", again, err)
	}
}

func TestBinaryMigrationCheckpointRejectsChangedSourceAndCorruption(t *testing.T) {
	source, dest := rawdb.NewMemoryDB(), rawdb.NewMemoryDB()
	src := NewTrieStateDB(source)
	a, b := types.Address{1}, types.Address{2}
	src.AddBalance(a, big.NewInt(1))
	src.AddBalance(b, big.NewInt(2))
	root, err := src.Commit()
	if err != nil {
		t.Fatal(err)
	}
	first, err := MigrateCommittedBinaryBatch(source, root, dest, 1)
	if err != nil {
		t.Fatal(err)
	}
	src.AddBalance(a, big.NewInt(1))
	if _, err := src.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateCommittedBinaryBatch(source, root, dest, 1); !errors.Is(err, ErrBinaryMigrationSource) {
		t.Fatalf("changed source: %v", err)
	}
	stored, err := dest.Get(binaryMigrationCheckpointKey(root))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, encodeBinaryMigrationProgress(first)) {
		t.Fatal("changed source advanced checkpoint")
	}
	stored[0] = 99
	if err := dest.Put(binaryMigrationCheckpointKey(root), stored); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateCommittedBinaryBatch(source, root, dest, 1); !errors.Is(err, ErrBinaryMigrationCheckpoint) {
		t.Fatalf("invalid checkpoint: %v", err)
	}
}

func TestBinaryMigrationCheckpointEmptyAndMissingNodes(t *testing.T) {
	t.Run("empty source", func(t *testing.T) {
		source, dest := rawdb.NewMemoryDB(), rawdb.NewMemoryDB()
		root := NewTrieStateDB(source).GetRoot()
		progress, err := MigrateCommittedBinaryBatch(source, root, dest, 1)
		if err != nil || !progress.Complete || progress.Accounts != 0 || progress.Root != (types.Hash{}) {
			t.Fatalf("empty migration: %+v %v", progress, err)
		}
		restored, err := MigrateCommittedBinaryBatch(source, root, dest, 1)
		if err != nil || restored != progress {
			t.Fatalf("empty resume: %+v %v", restored, err)
		}
	})
	t.Run("missing persisted node", func(t *testing.T) {
		source, dest := rawdb.NewMemoryDB(), rawdb.NewMemoryDB()
		src := NewTrieStateDB(source)
		src.AddBalance(types.Address{1}, big.NewInt(1))
		src.AddBalance(types.Address{2}, big.NewInt(2))
		root, err := src.Commit()
		if err != nil {
			t.Fatal(err)
		}
		progress, err := MigrateCommittedBinaryBatch(source, root, dest, 1)
		if err != nil {
			t.Fatal(err)
		}
		// Remove all binary nodes while retaining the checkpoint.
		iter := dest.NewIterator([]byte("eth2030:binary-trie:v1:"))
		var keys [][]byte
		for iter.Next() {
			keys = append(keys, bytes.Clone(iter.Key()))
		}
		iter.Release()
		for _, key := range keys {
			if err := dest.Delete(key); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := MigrateCommittedBinaryBatch(source, root, dest, 1); err == nil {
			t.Fatal("accepted checkpoint with missing root node")
		}
		encoded, err := dest.Get(binaryMigrationCheckpointKey(root))
		if err != nil || !bytes.Equal(encoded, encodeBinaryMigrationProgress(progress)) {
			t.Fatal("failed recovery advanced checkpoint")
		}
	})
}
