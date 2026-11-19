package state

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
)

func historyFixture() (*MemoryStateDB, types.Address, types.Hash) {
	s := NewMemoryStateDB()
	a := types.Address{19: 0x41}
	k := types.Hash{31: 1}
	s.AddBalance(a, big.NewInt(100))
	s.SetNonce(a, 1)
	s.SetCode(a, []byte{0x60, 1, 0x50, 0})
	s.SetState(a, k, types.Hash{31: 11})
	s.SetState(a, types.Hash{31: 2}, types.Hash{31: 12})
	return s, a, k
}
func TestMerkleHistoryPublishesAtomicallyAndRetainsExactRootsAfterRestart(t *testing.T) {
	dir := t.TempDir()
	db, err := rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, a, k := historyFixture()
	first := s.GetRoot()
	batch := db.NewBatch()
	if err := StoreTrieHistory(db, batch, first, s); err != nil {
		t.Fatal(err)
	}
	if available, _ := HistoryStateAvailable(db, first); available {
		t.Fatal("root visible before block batch commits")
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	secondState := s.Copy()
	secondState.SetState(a, k, types.Hash{31: 22})
	secondState.SetNonce(a, 2)
	second := secondState.GetRoot()
	batch = db.NewBatch()
	if err := StoreTrieHistory(db, batch, second, secondState); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, test := range []struct {
		root  types.Hash
		value byte
		nonce uint64
	}{{first, 11, 1}, {second, 22, 2}} {
		restored, err := OpenTrieHistory(context.Background(), db, test.root)
		if err != nil {
			t.Fatal(err)
		}
		if restored.GetRoot() != test.root || restored.GetBalance(a).Cmp(big.NewInt(100)) != 0 || restored.GetState(a, k) != (types.Hash{31: test.value}) || restored.GetNonce(a) != test.nonce || !bytes.Equal(restored.GetCode(a), s.GetCode(a)) {
			t.Fatalf("wrong point-in-time state at %s", test.root)
		}
		snapshot, err := ExportStateSnapshot(restored)
		if err != nil {
			t.Fatal(err)
		}
		proof, err := snapshot.AccountProof(crypto.Keccak256Hash(a[:]))
		if err != nil {
			t.Fatal(err)
		}
		proofDB := memorydb.New()
		for _, node := range proof {
			if err := proofDB.Put(gethcrypto.Keccak256(node), node); err != nil {
				t.Fatal(err)
			}
		}
		encoded, err := gethtrie.VerifyProof(gethcommon.Hash(test.root), gethcrypto.Keccak256(a[:]), proofDB)
		if err != nil {
			t.Fatal(err)
		}
		var account gethtypes.StateAccount
		if err := gethrlp.DecodeBytes(encoded, &account); err != nil {
			t.Fatal(err)
		}
		if account.Nonce != test.nonce || account.Balance.ToBig().Cmp(big.NewInt(100)) != 0 || !bytes.Equal(account.CodeHash, gethcrypto.Keccak256(s.GetCode(a))) {
			t.Fatal("independent historical account proof contradicts the original commitment")
		}
		proof, err = snapshot.StorageProof(crypto.Keccak256Hash(a[:]), crypto.Keccak256Hash(k[:]))
		if err != nil {
			t.Fatal(err)
		}
		proofDB = memorydb.New()
		for _, node := range proof {
			if err := proofDB.Put(gethcrypto.Keccak256(node), node); err != nil {
				t.Fatal(err)
			}
		}
		encoded, err = gethtrie.VerifyProof(account.Root, gethcrypto.Keccak256(k[:]), proofDB)
		if err != nil {
			t.Fatal(err)
		}
		var value []byte
		if err := gethrlp.DecodeBytes(encoded, &value); err != nil || new(big.Int).SetBytes(value).Uint64() != uint64(test.value) {
			t.Fatalf("independent historical storage proof/value: %v", err)
		}
		clone := restored.Dup()
		clone.SetState(a, k, types.Hash{31: 99})
		if restored.GetState(a, k) != (types.Hash{31: test.value}) || restored.GetRoot() != test.root {
			t.Fatal("request overlay changed immutable history")
		}
	}
}
func TestMerkleHistoryRejectsCorruptNodesCodeAndMarkers(t *testing.T) {
	for _, fault := range []string{"account-node", "storage-node", "missing-storage", "code", "marker"} {
		t.Run(fault, func(t *testing.T) {
			db := rawdb.NewMemoryDB()
			s, a, _ := historyFixture()
			root := s.GetRoot()
			batch := db.NewBatch()
			if err := StoreTrieHistory(db, batch, root, s); err != nil {
				t.Fatal(err)
			}
			if err := batch.Write(); err != nil {
				t.Fatal(err)
			}
			snapshot, _ := ExportStateSnapshot(s)
			accountHash := crypto.Keccak256Hash(a[:])
			var account rlpAccount
			if err := rlp.DecodeBytes(snapshot.Accounts[accountHash], &account); err != nil {
				t.Fatal(err)
			}
			key := historyNodeKey(root)
			switch fault {
			case "storage-node", "missing-storage":
				key = historyNodeKey(types.BytesToHash(account.Root))
			case "code":
				key = append([]byte("snap-code/"), account.CodeHash...)
			case "marker":
				key = historyRootKey(root)
			}
			if fault == "missing-storage" {
				if err := db.Delete(key); err != nil {
					t.Fatal(err)
				}
			} else {
				value, err := db.Get(key)
				if err != nil {
					t.Fatal(err)
				}
				value[len(value)-1] ^= 1
				if err := db.Put(key, value); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := OpenTrieHistory(context.Background(), db, root); err == nil || errors.Is(err, rawdb.ErrNotFound) {
				t.Fatalf("corruption was accepted or hidden as absent history: %v", err)
			}
		})
	}
}
func TestMerkleHistoryRejectsConflictingImmutableDataBeforeNewRootPublication(t *testing.T) {
	db := rawdb.NewMemoryDB()
	s, a, k := historyFixture()
	root := s.GetRoot()
	batch := db.NewBatch()
	if err := StoreTrieHistory(db, batch, root, s); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := ExportStateSnapshot(s)
	var account rlpAccount
	_ = rlp.DecodeBytes(snapshot.Accounts[crypto.Keccak256Hash(a[:])], &account)
	key := historyNodeKey(types.BytesToHash(account.Root))
	value, _ := db.Get(key)
	value[len(value)-1] ^= 1
	_ = db.Put(key, value)
	changed := s.Copy()
	changed.AddBalance(a, big.NewInt(1))
	next := changed.GetRoot()
	batch = db.NewBatch()
	if err := StoreTrieHistory(db, batch, next, changed); err == nil {
		t.Fatal("conflicting content-addressed node was overwritten")
	}
	if available, _ := HistoryStateAvailable(db, next); available {
		t.Fatal("failed state root became available")
	}
	if _, err := OpenTrieHistory(context.Background(), db, types.Hash{31: 99}); !errors.Is(err, rawdb.ErrNotFound) {
		t.Fatalf("absent root classification: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenTrieHistory(ctx, db, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled read did not stop: %v", err)
	}
	if s.GetState(a, k) != (types.Hash{31: 11}) {
		t.Fatal("history persistence changed source state")
	}
}
