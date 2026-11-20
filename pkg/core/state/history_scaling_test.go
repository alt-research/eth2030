package state

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/trie/mpt"
	"math/big"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

type countHistoryDB struct {
	rawdb.Database
	gets int
}

func (d *countHistoryDB) Get(key []byte) ([]byte, error) { d.gets++; return d.Database.Get(key) }

// Records work amplification without inventing a mainnet latency threshold.
// Database Get counts are logical reads, not physical disk I/O measurements.
func TestHistoryPersistenceAndColdReadArePathBounded(t *testing.T) {
	for _, accounts := range []int{256, 2048, 8192} {
		db := &countHistoryDB{Database: rawdb.NewMemoryDB()}
		original := NewMemoryStateDB()
		var changed types.Address
		for i := 0; i < accounts; i++ {
			var address types.Address
			binary.BigEndian.PutUint64(address[12:], uint64(i+1))
			original.AddBalance(address, big.NewInt(1))
			changed = address
		}
		root := original.GetRoot()
		batch := db.NewBatch()
		if err := StoreTrieHistory(db, batch, root, original); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		original.AddBalance(changed, big.NewInt(1))
		next := original.GetRoot()
		db.gets = 0
		start := time.Now()
		batch = db.NewBatch()
		if err := StoreTrieHistory(db, batch, next, original); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		storeGets, storeTime := db.gets, time.Since(start)
		db.gets = 0
		start = time.Now()
		loaded, err := OpenTrieHistory(context.Background(), db, next)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.GetBalance(changed).Cmp(big.NewInt(2)) != 0 {
			t.Fatal("historical value changed")
		}

		if storeGets > 256 || db.gets > 128 {
			t.Fatalf("one-account work still scales with total state: accounts=%d store=%d open=%d", accounts, storeGets, db.gets)
		}
		t.Logf("accounts=%d changed_accounts=1 store_gets=%d store_elapsed=%s open_gets=%d open_elapsed=%s", accounts, storeGets, storeTime, db.gets, time.Since(start))
	}
}

func TestHistoryStorageUpdateAndProofsArePathBounded(t *testing.T) {
	for _, count := range []int{256, 8192} {
		db := &countHistoryDB{Database: rawdb.NewMemoryDB()}
		memory := NewMemoryStateDB()
		address := types.Address{19: 7}
		memory.SetNonce(address, 1)
		var last types.Hash
		for i := 1; i <= count; i++ {
			var key types.Hash
			binary.BigEndian.PutUint64(key[24:], uint64(i))
			memory.SetState(address, key, types.Hash{31: 1})
			last = key
		}
		root := memory.GetRoot()
		batch := db.NewBatch()
		if err := StoreTrieHistory(db, batch, root, memory); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		lazy, err := OpenTrieHistory(context.Background(), db, root)
		if err != nil {
			t.Fatal(err)
		}
		db.gets = 0
		lazy.SetState(address, last, types.Hash{31: 2})
		newRoot := lazy.GetRoot()
		memory.SetState(address, last, types.Hash{31: 2})
		if newRoot != memory.GetRoot() {
			t.Fatal("incremental storage root differs from independent native full trie")
		}
		batch = db.NewBatch()
		if err := StoreTrieHistory(db, batch, newRoot, lazy); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		if db.gets > 512 {
			t.Fatalf("one storage change scanned unrelated slots: %d", db.gets)
		}
		db.gets = 0
		opened, err := OpenTrieHistory(context.Background(), db, newRoot)
		if err != nil {
			t.Fatal(err)
		}
		absent := types.Hash{0: 0xff}
		proof, err := opened.HistoricalProof(address, []types.Hash{last, absent})
		if err != nil {
			t.Fatal(err)
		}
		body, err := mpt.VerifyProof(newRoot, crypto.Keccak256(address[:]), proof.AccountProof)
		if err != nil || len(body) == 0 {
			t.Fatalf("native independent account proof: %v", err)
		}
		for i, key := range []types.Hash{last, absent} {
			value, err := mpt.VerifyProof(proof.StorageHash, crypto.Keccak256(key[:]), proof.StorageProof[i].Proof)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 && proof.StorageProof[i].Value.Uint64() != 2 {
				t.Fatal("wrong decoded storage proof value")
			}
			if i == 1 && len(value) != 0 {
				t.Fatal("absence proof contains a value")
			}
		}
		if db.gets > 256 {
			t.Fatalf("point proof scanned unrelated storage: %d", db.gets)
		}
		proofGets := db.gets
		old, err := OpenTrieHistory(context.Background(), db, root)
		if err != nil || old.GetState(address, last) != (types.Hash{31: 1}) {
			t.Fatal("new root mutated old storage history")
		}
		t.Logf("slots=%d proof_gets=%d", count, proofGets)
	}
}

func TestHistoryLazyErrorsCancellationAndFailedPublication(t *testing.T) {
	db := rawdb.NewMemoryDB()
	memory, address, slot := historyFixture()
	root := memory.GetRoot()
	batch := db.NewBatch()
	if err := StoreTrieHistory(db, batch, root, memory); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	lazy, err := OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := lazy.Snapshot()
	lazy.SetState(address, slot, types.Hash{31: 99})
	lazy.SetCode(address, []byte{0x00})
	lazy.RevertToSnapshot(snapshot)
	if lazy.GetRoot() != root || !bytes.Equal(lazy.GetCode(address), memory.GetCode(address)) {
		t.Fatal("reverted incremental state changed its commitment/code")
	}
	lazy.AddBalance(address, big.NewInt(1))
	next := lazy.GetRoot()
	batch = db.NewBatch()
	if err := StoreTrieHistory(db, batch, next, lazy); err != nil {
		t.Fatal(err)
	}
	if present, _ := HistoryStateAvailable(db, next); present {
		t.Fatal("uncommitted batch published history")
	}
	// Discard the prepared batch as a write-failure control, then retry from
	// the same private state. Pending immutable nodes must not be lost.
	batch = db.NewBatch()
	if err := StoreTrieHistory(db, batch, next, lazy); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenTrieHistory(context.Background(), db, next)
	if err != nil || reopened.GetBalance(address).Cmp(big.NewInt(101)) != 0 {
		t.Fatal("failed-publication retry lost nodes")
	}
	cancelled, err := OpenTrieHistory(context.Background(), db, next)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancelled.SetReadContext(ctx)
	cancel()
	_ = cancelled.GetState(address, slot)
	if !errors.Is(ReadError(cancelled), context.Canceled) {
		t.Fatalf("lazy cancellation hidden as zero: %v", ReadError(cancelled))
	}
	codeHash := memory.GetCodeHash(address)
	if err := db.Put(append([]byte("snap-code/"), codeHash[:]...), []byte{0xff}); err != nil {
		t.Fatal(err)
	}
	damaged, err := OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	_ = damaged.GetCode(address)
	if ReadError(damaged) == nil {
		t.Fatal("corrupt lazy code returned successful empty code")
	}
	if _, err := damaged.Commit(); err == nil {
		t.Fatal("corrupt accessed state committed")
	}
	if err := AuditTrieHistory(context.Background(), db, root); err == nil {
		t.Fatal("complete state audit missed corrupt code")
	}
}

func TestHistoryFullExportOwnsPreparedUnpublishedCode(t *testing.T) {
	db := rawdb.NewMemoryDB()
	memory, address, _ := historyFixture()
	root := memory.GetRoot()
	batch := db.NewBatch()
	if err := StoreTrieHistory(db, batch, root, memory); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	lazy, err := OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	code := []byte{0x60, 0x02, 0x50, 0x00}
	lazy.SetCode(address, code)
	prepared, err := lazy.Commit()
	if err != nil {
		t.Fatal(err)
	}
	sibling := lazy.Dup()
	exported, err := ExportStateSnapshot(lazy)
	if err != nil {
		t.Fatal(err)
	}
	hash := crypto.Keccak256Hash(code)
	if !bytes.Equal(exported.Codes[hash], code) {
		t.Fatal("missing prepared code")
	}
	exported.Codes[hash][0] ^= 0xff
	if !bytes.Equal(lazy.GetCode(address), code) || !bytes.Equal(sibling.GetCode(address), code) || ReadError(lazy) != nil || ReadError(sibling) != nil {
		t.Fatal("full export mutated an immutable prepared source view")
	}
	if lazy.GetRoot() != prepared || sibling.GetRoot() != prepared {
		t.Fatal("caller export changed source commitment")
	}
	if present, _ := HistoryStateAvailable(db, prepared); present {
		t.Fatal("private commit/export published authority")
	}
}

func TestHistoryRevertedDeletionAndRecreationMatchFullNativeRoots(t *testing.T) {
	db := rawdb.NewMemoryDB()
	memory, address, slot := historyFixture()
	root := memory.GetRoot()
	batch := db.NewBatch()
	if err := StoreTrieHistory(db, batch, root, memory); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	lazy, err := OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, referenceSnapshot := lazy.Snapshot(), memory.Snapshot()
	lazy.SelfDestruct(address)
	lazy.Finalise(true)
	memory.SelfDestruct(address)
	memory.Finalise(true)
	if lazy.GetRoot() != memory.GetRoot() {
		t.Fatal("deletion root differs from full native trie")
	}
	lazy.RevertToSnapshot(snapshot)
	memory.RevertToSnapshot(referenceSnapshot)
	if lazy.GetRoot() != root || lazy.GetRoot() != memory.GetRoot() {
		t.Fatal("reverted deletion lost original account")
	}
	lazy.SelfDestruct(address)
	lazy.Finalise(true)
	memory.SelfDestruct(address)
	memory.Finalise(true)
	deleted, err := lazy.Commit()
	if err != nil {
		t.Fatal(err)
	}
	batch = db.NewBatch()
	if err := StoreTrieHistory(db, batch, deleted, lazy); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenTrieHistory(context.Background(), db, deleted)
	if err != nil {
		t.Fatal(err)
	}
	reopened.CreateAccount(address)
	reopened.SetNonce(address, 1)
	memory.CreateAccount(address)
	memory.SetNonce(address, 1)
	if reopened.GetState(address, slot) != (types.Hash{}) {
		t.Fatal("recreated account inherited deleted storage")
	}
	other := types.Hash{31: 99}
	reopened.SetState(address, other, types.Hash{31: 3})
	memory.SetState(address, other, types.Hash{31: 3})
	if reopened.GetRoot() != memory.GetRoot() {
		t.Fatal("recreation root differs from native full state")
	}
	old, err := OpenTrieHistory(context.Background(), db, root)
	if err != nil || old.GetState(address, slot) != (types.Hash{31: 11}) {
		t.Fatal("deletion/recreation altered old immutable root")
	}
}
