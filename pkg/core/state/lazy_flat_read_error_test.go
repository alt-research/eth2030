package state

import (
	"encoding/hex"
	"errors"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestColdFlatStateReadErrorsRemainErrorsAcrossRollbackAndCopy(t *testing.T) {
	for _, kind := range []string{"account", "slot", "code", "code-hash"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			db, err := rawdb.NewFileDB(dir)
			if err != nil {
				t.Fatal(err)
			}
			st := NewTrieStateDB(db)
			addr := types.Address{1}
			slot := types.Hash{2}
			st.AddBalance(addr, big.NewInt(9))
			st.SetState(addr, slot, types.Hash{3})
			st.SetCode(addr, []byte{0x60, 0x00})
			codeHash := st.GetCodeHash(addr)
			if _, err := st.Commit(); err != nil {
				t.Fatal(err)
			}
			db.Close()
			db, err = rawdb.NewFileDB(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			cold := NewTrieStateDB(db)
			snap := cold.Snapshot()
			key := accountDBKey(addr)
			if kind == "slot" {
				cold.GetBalance(addr)
				key = storageDBKey(addr, slot)
			}
			if kind == "code" || kind == "code-hash" {
				key = codeDBKey(codeHash[:])
			}
			path := filepath.Join(dir, "data", hex.EncodeToString(key))
			if kind == "code-hash" {
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				body[0] ^= 1
				if err := os.WriteFile(path, body, 0600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "account":
				cold.GetBalance(addr)
			case "slot":
				cold.GetState(addr, slot)
			case "code", "code-hash":
				cold.GetCode(addr)
			}
			if ReadError(cold) == nil {
				t.Fatal("cold read failure became an empty account/slot/code")
			}
			cold.RevertToSnapshot(snap)
			if ReadError(cold.Dup()) == nil {
				t.Fatal("copy/revert erased persistent read failure")
			}
			if _, err := cold.Commit(); err == nil {
				t.Fatal("failed read published a state root")
			}
		})
	}
}

type failedSnapshotDB struct {
	rawdb.Database
	failure error
}
type failedSnapshotIterator struct {
	rawdb.Iterator
	failure error
}

func (it *failedSnapshotIterator) Next() bool   { return false }
func (it *failedSnapshotIterator) Error() error { return it.failure }
func (db *failedSnapshotDB) NewIterator(prefix []byte) rawdb.Iterator {
	return &failedSnapshotIterator{db.Database.NewIterator(prefix), db.failure}
}
func TestEmptySnapshotAndMigrationCannotHideIteratorErrors(t *testing.T) {
	cause := errors.New("actual iterator read failure")
	base := rawdb.NewMemoryDB()
	root := types.EmptyRootHash
	if err := StoreStateSnapshot(base, root, NewStateSnapshot()); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadStateSnapshot(base, root); err != nil {
		t.Fatalf("positive empty snapshot failed: %v", err)
	}
	db := &failedSnapshotDB{base, cause}
	if _, err := LoadStateSnapshot(db, root); !errors.Is(err, cause) {
		t.Fatal("empty-root snapshot ignored read failure")
	}
	destination := rawdb.NewMemoryDB()
	if _, err := MigrateCommittedBinaryBatch(db, root, destination, 16); !errors.Is(err, cause) {
		t.Fatalf("migration hid iterator failure: %v", err)
	}
	if destination.Len() != 0 {
		t.Fatal("failed source scan published migration metadata")
	}
	st := NewTrieStateDB(db)
	st.AddBalance(types.Address{1}, big.NewInt(1))
	if _, err := st.Commit(); !errors.Is(err, cause) {
		t.Fatal("commit ignored iterator failure")
	}
}
