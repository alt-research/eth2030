package state

import (
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestEmptyFinalizationTouchRollbackAndDurability(t *testing.T) {
	for _, kind := range []string{"memory", "trie", "snap", "stateless"} {
		t.Run(kind, func(t *testing.T) {
			empty, untouched, reverted := types.Address{19: 1}, types.Address{19: 2}, types.Address{19: 3}
			mem := NewMemoryStateDB()
			for _, addr := range []types.Address{empty, untouched, reverted} {
				mem.CreateAccount(addr)
			}
			if _, err := mem.Commit(); err != nil {
				t.Fatal(err)
			}
			store := rawdb.NewMemoryDB()
			var db StateDB = mem
			switch kind {
			case "trie":
				db = NewTrieStateDBFromMemory(store, mem.Copy())
				if _, err := db.Commit(); err != nil {
					t.Fatal(err)
				}
			case "snap":
				snapshot, err := ExportStateSnapshot(mem)
				if err != nil {
					t.Fatal(err)
				}
				if err := StoreStateSnapshot(store, mem.GetRoot(), snapshot); err != nil {
					t.Fatal(err)
				}
				db, err = OpenSnapState(store, mem.GetRoot())
				if err != nil {
					t.Fatal(err)
				}
			case "stateless":
				w := NewStatelessWitness(mem.GetRoot())
				for _, addr := range []types.Address{empty, untouched, reverted} {
					w.AddAccount(addr, &WitnessAccount{Exists: true, Balance: new(big.Int), CodeHash: types.EmptyCodeHash})
				}
				db = NewStatelessStateDB(w)
			}
			db.(interface{ FinalizePreState() }).FinalizePreState()
			// Merely reading an empty account never qualifies it for clearing.
			_ = db.GetBalance(untouched)
			before := db.Snapshot()
			db.AddBalance(empty, new(big.Int))
			inner := db.Snapshot()
			db.AddBalance(reverted, new(big.Int))
			db.RevertToSnapshot(inner)
			db.Finalise(false)
			if !db.Exist(empty) {
				t.Fatal("pre-EIP-161 finalization deleted empty account")
			}
			db.Finalise(true)
			if db.Exist(empty) || !db.Exist(untouched) || !db.Exist(reverted) {
				t.Fatal("finalization did not distinguish touched, read-only and reverted accounts")
			}
			db.RevertToSnapshot(before)
			if !db.Exist(empty) {
				t.Fatal("finalized empty-account deletion did not revert")
			}
			db.AddBalance(empty, new(big.Int))
			db.Finalise(true)
			if kind == "trie" || kind == "snap" {
				root, err := db.Commit()
				if err != nil {
					t.Fatal(err)
				}
				if kind == "trie" {
					db = NewTrieStateDB(store)
				} else {
					db, err = OpenSnapState(store, root)
					if err != nil {
						t.Fatal(err)
					}
				}
				if db.Exist(empty) || !db.Exist(untouched) {
					t.Fatal("empty deletion or untouched account changed after reload")
				}
			}
			db.AddBalance(empty, big.NewInt(5))
			db.Finalise(true)
			if !db.Exist(empty) || db.GetBalance(empty).Cmp(big.NewInt(5)) != 0 || db.GetNonce(empty) != 0 || len(db.GetCode(empty)) != 0 {
				t.Fatal("finalized deletion could not be recreated in a later transaction")
			}

		})
	}
}
