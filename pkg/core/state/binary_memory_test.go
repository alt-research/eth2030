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

func TestBuildBinaryTreeFromMemoryMatchesCommittedMigration(t *testing.T) {
	source := NewMemoryStateDB()
	address, other := types.Address{1}, types.Address{2}
	inline, remote := types.Hash{31: 2}, types.Hash{30: 2, 31: 4}
	source.AddBalance(address, big.NewInt(123))
	source.SetNonce(address, 4)
	code := []byte{0x60, 0x01, 0x50}
	source.SetCode(address, code)
	source.SetState(address, inline, types.Hash{31: 5})
	source.SetState(address, remote, types.Hash{31: 6})
	source.AddBalance(other, big.NewInt(9))
	before, err := BuildBinaryTreeFromMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := source.Snapshot()
	source.SetState(address, inline, types.Hash{})
	source.SetCode(address, []byte{0x00})
	source.SelfDestruct(other)
	changed, err := BuildBinaryTreeFromMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Hash() == before.Hash() {
		t.Fatal("state changes did not change root")
	}
	if account, err := changed.GetAccount(other); err != nil || account != nil {
		t.Fatalf("self-destructed account: %+v %v", account, err)
	}
	if value, err := changed.GetStorage(address, inline[:]); err != nil || value != nil {
		t.Fatalf("zeroed storage: %x %v", value, err)
	}
	source.RevertToSnapshot(snapshot)
	reverted, err := BuildBinaryTreeFromMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	if reverted.Hash() != before.Hash() {
		t.Fatal("snapshot revert did not restore binary root")
	}
	store := rawdb.NewMemoryDB()
	committed := NewTrieStateDBFromMemory(store, source.Copy())
	sourceRoot, err := committed.Commit()
	if err != nil {
		t.Fatal(err)
	}
	binaryStore := rawdb.NewMemoryDB()
	migratedRoot, err := MigrateCommittedToBinary(store, sourceRoot, binaryStore)
	if err != nil {
		t.Fatal(err)
	}
	if reverted.Hash() != migratedRoot {
		t.Fatalf("memory and committed migration roots differ: %x vs %x", reverted.Hash(), migratedRoot)
	}
	restored, err := bintrie.OpenPersisted(binaryStore, bintrie.HashFunctionBlake3, migratedRoot)
	if err != nil {
		t.Fatal(err)
	}
	readCode, err := restored.GetContractCode(address)
	if err != nil || !bytes.Equal(readCode, code) {
		t.Fatalf("code readback: %x %v", readCode, err)
	}
}

func TestBuildBinaryTreeFromMemoryRejectsInvalidState(t *testing.T) {
	if _, err := BuildBinaryTreeFromMemory(nil); !errors.Is(err, ErrBinaryMemoryState) {
		t.Fatalf("nil: %v", err)
	}
	s := NewMemoryStateDB()
	address := types.Address{1}
	code := []byte{1, 2, 3}
	s.SetCode(address, code)
	code[0] ^= 1
	if s.GetCode(address)[0] != 1 {
		t.Fatal("source code slice mutated state")
	}
	returned := s.GetCode(address)
	returned[0] ^= 1
	if s.GetCode(address)[0] != 1 {
		t.Fatal("returned code slice mutated state")
	}
	s.stateObjects[address].code[0] ^= 1 // simulate internal corruption.
	if _, err := BuildBinaryTreeFromMemory(s); !errors.Is(err, ErrBinaryMemoryState) {
		t.Fatalf("inconsistent code: %v", err)
	}
	s = NewMemoryStateDB()
	s.AddBalance(address, new(big.Int).Lsh(big.NewInt(1), 128))
	if _, err := BuildBinaryTreeFromMemory(s); err == nil {
		t.Fatal("accepted balance outside draft header width")
	}
}

func TestSnapshotRevertRemovesImplicitAccounts(t *testing.T) {
	for _, op := range []struct {
		name  string
		apply func(*MemoryStateDB, types.Address)
	}{
		{"balance", func(s *MemoryStateDB, a types.Address) { s.AddBalance(a, big.NewInt(5)) }},
		{"nonce", func(s *MemoryStateDB, a types.Address) { s.SetNonce(a, 1) }},
		{"code", func(s *MemoryStateDB, a types.Address) { s.SetCode(a, []byte{1}) }},
		{"storage", func(s *MemoryStateDB, a types.Address) { s.SetState(a, types.Hash{1}, types.Hash{2}) }},
	} {
		t.Run(op.name, func(t *testing.T) {
			s := NewMemoryStateDB()
			existing := types.Address{9}
			s.AddBalance(existing, big.NewInt(10))
			originalMPT := s.GetRoot()
			originalBinary, err := BuildBinaryTreeFromMemory(s)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := s.Snapshot()
			created := types.Address{1}
			op.apply(s, created)
			if !s.Exist(created) {
				t.Fatal("operation did not create account")
			}
			s.RevertToSnapshot(snapshot)
			if s.Exist(created) || s.GetRoot() != originalMPT {
				t.Fatal("revert retained implicitly created account")
			}
			binary, err := BuildBinaryTreeFromMemory(s)
			if err != nil || binary.Hash() != originalBinary.Hash() {
				t.Fatalf("binary root after revert: %v", err)
			}
		})
	}
}

func TestTrieStateCommitInvalidatesCachedRootAfterMutation(t *testing.T) {
	operations := []struct {
		name  string
		apply func(*TrieStateDB, types.Address)
	}{
		{"balance_add", func(s *TrieStateDB, a types.Address) { s.AddBalance(a, big.NewInt(2)) }},
		{"balance_sub", func(s *TrieStateDB, a types.Address) { s.SubBalance(a, big.NewInt(2)) }},
		{"nonce", func(s *TrieStateDB, a types.Address) { s.SetNonce(a, 3) }},
		{"code", func(s *TrieStateDB, a types.Address) { s.SetCode(a, []byte{1, 2}) }},
		{"storage", func(s *TrieStateDB, a types.Address) { s.SetState(a, types.Hash{1}, types.Hash{2}) }},
		{"destroy", func(s *TrieStateDB, a types.Address) { s.SelfDestruct(a) }},
		{"create", func(s *TrieStateDB, a types.Address) { s.CreateAccount(types.Address{9}) }},
	}
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			store := rawdb.NewMemoryDB()
			s := NewTrieStateDB(store)
			address := types.Address{1}
			s.AddBalance(address, big.NewInt(10))
			if _, err := s.Commit(); err != nil {
				t.Fatal(err)
			}
			s.AddBalance(address, big.NewInt(1))
			_ = s.GetRoot()
			op.apply(s, address)
			want := s.Dup().GetRoot()
			got, err := s.Commit()
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("committed stale root: got %x want %x", got, want)
			}
			reopened := NewTrieStateDB(store)
			if reopened.GetRoot() != got {
				t.Fatal("committed root differs after reopening")
			}
			if _, err := MigrateCommittedToBinary(store, got, rawdb.NewMemoryDB()); err != nil {
				t.Fatalf("committed state failed migration validation: %v", err)
			}
		})
	}
}

func TestTrieStateCommitInvalidatesCachedRootAfterRevert(t *testing.T) {
	s := NewTrieStateDB(rawdb.NewMemoryDB())
	address := types.Address{1}
	s.AddBalance(address, big.NewInt(10))
	before := s.GetRoot()
	snapshot := s.Snapshot()
	s.AddBalance(address, big.NewInt(1))
	_ = s.GetRoot()
	s.RevertToSnapshot(snapshot)
	got, err := s.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if got != before {
		t.Fatalf("reverted state committed stale root: %x want %x", got, before)
	}
}

func TestTrieStateRevertedRecreationRetainsStorage(t *testing.T) {
	store := rawdb.NewMemoryDB()
	s := NewTrieStateDB(store)
	address := types.Address{1}
	slot, value := types.Hash{1}, types.Hash{2}
	s.AddBalance(address, big.NewInt(10))
	s.SetState(address, slot, value)
	if _, err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	snapshot := s.Snapshot()
	s.CreateAccount(address)
	s.RevertToSnapshot(snapshot)
	if got := s.GetState(address, slot); got != value {
		t.Fatalf("storage before commit = %x", got)
	}
	root, err := s.Commit()
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewTrieStateDB(store)
	if got := reopened.GetState(address, slot); got != value {
		t.Fatalf("reverted recreation erased committed storage: %x", got)
	}
	target := rawdb.NewMemoryDB()
	if _, err := MigrateCommittedToBinary(store, root, target); err != nil {
		t.Fatalf("committed root/storage mismatch: %v", err)
	}
}

func TestTrieStateRecreationResetsStorageAndRetainsNewWrites(t *testing.T) {
	store := rawdb.NewMemoryDB()
	s := NewTrieStateDB(store)
	address := types.Address{1}
	same, stale := types.Hash{1}, types.Hash{2}
	s.AddBalance(address, big.NewInt(10))
	s.SetState(address, same, types.Hash{3})
	s.SetState(address, stale, types.Hash{4})
	if _, err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	s.CreateAccount(address)
	if got := s.GetState(address, same); got != (types.Hash{}) {
		t.Fatalf("recreated account exposed old slot: %x", got)
	}
	next := types.Hash{5}
	s.SetState(address, same, next)
	expected := s.GetRoot()
	root, err := s.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if root != expected {
		t.Fatal("committed root differs from precommit root")
	}
	reopened := NewTrieStateDB(store)
	if got := reopened.GetState(address, same); got != next {
		t.Fatalf("recreated account lost new write: %x", got)
	}
	if got := reopened.GetState(address, stale); got != (types.Hash{}) {
		t.Fatalf("recreated account retained old slot: %x", got)
	}
	if _, err := MigrateCommittedToBinary(store, root, rawdb.NewMemoryDB()); err != nil {
		t.Fatalf("committed root/storage mismatch: %v", err)
	}
}

func TestTrieStateNestedSnapshotRestoresRecreationSet(t *testing.T) {
	store := rawdb.NewMemoryDB()
	s := NewTrieStateDB(store)
	a, b := types.Address{1}, types.Address{2}
	slot := types.Hash{1}
	old := types.Hash{9}
	s.SetState(a, slot, old)
	s.SetState(b, slot, old)
	if _, err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	_ = s.Snapshot()
	s.CreateAccount(a)
	inner := s.Snapshot()
	s.CreateAccount(b)
	s.RevertToSnapshot(inner)
	if got := s.GetState(a, slot); got != (types.Hash{}) {
		t.Fatal("retained old state for non-reverted recreation")
	}
	if got := s.GetState(b, slot); got != old {
		t.Fatal("lost state for reverted recreation")
	}
	root, err := s.Commit()
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewTrieStateDB(store)
	if got := reopened.GetState(a, slot); got != (types.Hash{}) {
		t.Fatal("persisted old state for recreated account")
	}
	if got := reopened.GetState(b, slot); got != old {
		t.Fatal("purged state for reverted recreation")
	}
	if _, err := MigrateCommittedToBinary(store, root, rawdb.NewMemoryDB()); err != nil {
		t.Fatalf("source root and storage disagree: %v", err)
	}
}

func TestTrieStateReopenedMutationPreservesOtherStorage(t *testing.T) {
	store := rawdb.NewMemoryDB()
	initial := NewTrieStateDB(store)
	address := types.Address{1}
	a, b := types.Hash{1}, types.Hash{2}
	initial.SetState(address, a, types.Hash{3})
	initial.SetState(address, b, types.Hash{4})
	if _, err := initial.Commit(); err != nil {
		t.Fatal(err)
	}
	reopened := NewTrieStateDB(store)
	reopened.SetState(address, a, types.Hash{5})
	preview := reopened.GetRoot()
	root, err := reopened.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if root != preview {
		t.Fatal("preview root differs from committed root")
	}
	readback := NewTrieStateDB(store)
	if got := readback.GetState(address, b); got != (types.Hash{4}) {
		t.Fatalf("lost untouched slot %x", got)
	}
	if _, err := MigrateCommittedToBinary(store, root, rawdb.NewMemoryDB()); err != nil {
		t.Fatalf("root does not commit persisted storage: %v", err)
	}
	reopened.SetState(address, a, types.Hash{6})
	root2, err := reopened.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateCommittedToBinary(store, root2, rawdb.NewMemoryDB()); err != nil {
		t.Fatalf("reopened storage index lost untouched slot: %v", err)
	}
}
