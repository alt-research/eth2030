package state

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/trie/bintrie"
)

func TestBinaryOverlayRevertClearAndConvert(t *testing.T) {
	mem := NewMemoryStateDB()
	addr := types.Address{19: 1}
	cleared, reverted := types.Hash{31: 1}, types.Hash{31: 2}
	old := types.Hash{31: 5}
	code := bytes.Repeat([]byte{0x5b}, 68)
	mem.SetCode(addr, code)
	mem.SetState(addr, cleared, old)
	mem.SetState(addr, reverted, old)
	s, err := NewBinaryOverlayState(mem)
	if err != nil {
		t.Fatal(err)
	}

	s.SetState(addr, cleared, types.Hash{})
	snap := s.Snapshot()
	s.SetState(addr, reverted, types.Hash{31: 9})
	s.SetCode(addr, []byte{0x00})
	s.RevertToSnapshot(snap)
	root, err := s.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if s.GetCommittedState(addr, cleared) != (types.Hash{}) || s.GetCommittedState(addr, reverted) != old || !bytes.Equal(s.GetCode(addr), code) {
		t.Fatal("logical state disagrees with surviving writes")
	}
	want := bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3)
	clearKey := bintrie.GetTreeKeyForStorageSlot(addr, big.NewInt(1))
	if err := want.Put(clearKey[:], make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if root != want.Hash() {
		t.Fatalf("only the explicit zero leaf should be committed, got %s", root)
	}

	branch := s.Dup().(*BinaryOverlayState)
	if err := branch.Convert(3); err != nil {
		t.Fatal(err)
	}
	if !branch.converted || s.converted || s.GetRoot() != root || branch.GetRoot() == root {
		t.Fatal("conversion did not advance independently on the branch")
	}
	zero, err := branch.tree.GetStorage(addr, cleared[:])
	if err != nil || !bytes.Equal(zero, make([]byte, 32)) {
		t.Fatalf("conversion replaced zero override: %x, %v", zero, err)
	}
	got, err := branch.tree.GetStorage(addr, reverted[:])
	if err != nil || !bytes.Equal(got, old[:]) {
		t.Fatalf("conversion copied a reverted write: %x, %v", got, err)
	}

	branch.SetCode(addr, []byte{0x00})
	if _, err := branch.Commit(); err != nil {
		t.Fatal(err)
	}
	lastChunk := bintrie.GetTreeKeyForCodeChunk(addr, 2)
	if got, err := branch.tree.Get(lastChunk[:]); err != nil || got != nil {
		t.Fatalf("stale code chunk: %x, %v", got, err)
	}
	branch.SelfDestruct(addr)
	if _, err := branch.Commit(); err != nil {
		t.Fatal(err)
	}
	if account, err := branch.tree.GetAccount(addr); err != nil || account != nil || branch.Exist(addr) {
		t.Fatalf("deleted account remains visible: %+v, %v", account, err)
	}
	got, err = branch.tree.GetStorage(addr, reverted[:])
	if err != nil || !bytes.Equal(got, make([]byte, 32)) {
		t.Fatalf("deleted account has a visible slot: %x, %v", got, err)
	}
}

func TestBinaryOverlayFailedCommitPreservesRoot(t *testing.T) {
	s, err := NewBinaryOverlayState(NewMemoryStateDB())
	if err != nil {
		t.Fatal(err)
	}
	addr := types.Address{19: 1}
	root := s.GetRoot()
	snap := s.Snapshot()
	s.AddBalance(addr, new(big.Int).Lsh(big.NewInt(1), 128))
	if _, err := s.Commit(); !errors.Is(err, bintrie.ErrInvalidValueLength) {
		t.Fatalf("expected overflow to fail, got %v", err)
	}
	if s.GetRoot() != root {
		t.Fatal("failed commit changed the root")
	}
	s.RevertToSnapshot(snap)
	if got, err := s.Commit(); err != nil || got != root || s.Exist(addr) {
		t.Fatalf("failed commit did not preserve the journal: %s, %v", got, err)
	}
}

func TestBinaryOverlayStorageOriginalIsTransactionScoped(t *testing.T) {
	mem := NewMemoryStateDB()
	address, key := types.Address{19: 1}, types.Hash{31: 9}
	first, second := types.Hash{31: 1}, types.Hash{31: 2}
	mem.SetState(address, key, first)
	s, err := NewBinaryOverlayState(mem)
	if err != nil {
		t.Fatal(err)
	}

	s.SetTxContext(types.Hash{31: 1}, 0)
	s.SetState(address, key, second)
	if s.GetCommittedState(address, key) != first {
		t.Fatal("first transaction lost its original slot")
	}
	s.SetTxContext(types.Hash{31: 2}, 1)
	snap := s.Snapshot()
	s.SetState(address, key, types.Hash{})
	if s.GetCommittedState(address, key) != second {
		t.Fatal("second transaction used the value from before the block")
	}
	s.RevertToSnapshot(snap)
	if _, err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	if s.GetCommittedState(address, key) != second {
		t.Fatal("reverted transaction changed the committed slot")
	}
}

func TestBinaryOverlayDupPreservesPendingWritesAndSnapshots(t *testing.T) {
	s, err := NewBinaryOverlayState(NewMemoryStateDB())
	if err != nil {
		t.Fatal(err)
	}
	addr, key := types.Address{19: 1}, types.Hash{31: 9}
	first, second := types.Hash{31: 1}, types.Hash{31: 2}
	s.SetTxContext(types.Hash{31: 7}, 0)
	s.SetState(addr, key, first)
	s.AddLog(&types.Log{Address: addr, Topics: []types.Hash{key}, Data: []byte{1}})
	snap := s.Snapshot()
	s.CreateAccount(addr)
	s.SetState(addr, key, second)

	branch := s.Dup().(*BinaryOverlayState)
	if branch.GetCommittedState(addr, key) != (types.Hash{}) {
		t.Fatal("duplicate lost the transaction's original storage value")
	}
	branch.GetLogs(types.Hash{31: 7})[0].Data[0] = 9
	if s.GetLogs(types.Hash{31: 7})[0].Data[0] != 1 {
		t.Fatal("duplicate shares mutable logs")
	}
	branch.RevertToSnapshot(snap)
	if branch.GetState(addr, key) != first || s.GetState(addr, key) != second {
		t.Fatal("duplicated snapshot failed to restore only its own state")
	}
	want := bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3)
	account := &types.Account{Balance: new(big.Int), CodeHash: types.EmptyCodeHash.Bytes()}
	if err := want.UpdateAccount(addr, account, 0); err != nil {
		t.Fatal(err)
	}
	if err := want.UpdateStorage(addr, key[:], first[:]); err != nil {
		t.Fatal(err)
	}
	if got, err := branch.Commit(); err != nil || got != want.Hash() {
		t.Fatalf("duplicate root = %s, want %s, err %v", got, want.Hash(), err)
	}
	if got, err := s.Commit(); err != nil || got == branch.GetRoot() {
		t.Fatalf("source root = %s, err %v", got, err)
	}
}

func TestBinaryOverlayRevertedRecreationPreservesOriginalSlot(t *testing.T) {
	mem := NewMemoryStateDB()
	addr, slot := types.Address{19: 1}, types.Hash{31: 2}
	old, replacement := types.Hash{31: 3}, types.Hash{31: 4}
	mem.SetState(addr, slot, old)
	s, err := NewBinaryOverlayState(mem)
	if err != nil {
		t.Fatal(err)
	}
	s.SetTxContext(types.Hash{31: 5}, 0)
	snap := s.Snapshot()
	s.CreateAccount(addr)
	s.SetState(addr, slot, replacement)
	s.RevertToSnapshot(snap)
	if got := s.GetCommittedState(addr, slot); got != old {
		t.Fatalf("original slot after reverting account recreation = %s, want %s", got, old)
	}
}
