package state

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/trie/bintrie"
)

func TestBinaryOverlayConversionOrderBudgetAndOverrides(t *testing.T) {
	frozen := NewMemoryStateDB()
	first, second := types.Address{19: 1}, types.Address{19: 2}
	h1, h2 := crypto.Keccak256Hash(first[:]), crypto.Keccak256Hash(second[:])
	if bytes.Compare(h1[:], h2[:]) < 0 {
		first, second = second, first
	}
	// The account with the larger raw address hash is deliberately inserted first.
	frozen.SetNonce(first, 8)
	frozen.SetCode(first, []byte{0x60, 1, 0x50})
	frozen.SetNonce(second, 7)
	frozen.AddBalance(second, big.NewInt(9))
	frozen.SetCode(second, []byte{0x60, 2, 0x50})
	slotA, slotB := types.Hash{31: 1}, types.Hash{31: 2}
	h1, h2 = crypto.Keccak256Hash(slotA[:]), crypto.Keccak256Hash(slotB[:])
	if bytes.Compare(h1[:], h2[:]) > 0 {
		slotA, slotB = slotB, slotA
	}
	frozen.SetState(second, slotB, types.Hash{31: 22})
	frozen.SetState(second, slotA, types.Hash{31: 11})
	skipped := types.Address{19: 3}
	frozen.SetState(skipped, slotA, types.Hash{31: 99})

	conversion, err := NewBinaryOverlayConversion(frozen)
	if err != nil {
		t.Fatal(err)
	}
	// The conversion owns a snapshot. Later source changes cannot alter it.
	frozen.SetState(second, slotB, types.Hash{31: 99})
	overlay := bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3)
	zeroKey := bintrie.GetTreeKeyForStorageSlot(second, new(big.Int).SetBytes(slotA[:]))
	zero := types.Hash{}
	if err := overlay.Put(zeroKey[:], zero[:]); err != nil {
		t.Fatal(err)
	}
	start := overlay.Hash()
	branch := conversion.Copy()
	checkSlot := func(tree *bintrie.BinaryTrie, address types.Address, slot types.Hash, want []byte) {
		t.Helper()
		got, err := tree.GetStorage(address, slot[:])
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("slot %s/%s = %x, want %x, error %v", address, slot, got, want, err)
		}
	}
	step := func(tree *bintrie.BinaryTrie, budget, wantUsed int, wantComplete bool) *bintrie.BinaryTrie {
		t.Helper()
		result, used, complete, err := conversion.Step(tree, budget)
		if err != nil || used != wantUsed || complete != wantComplete {
			t.Fatalf("step used %d, complete %v, error %v", used, complete, err)
		}
		return result
	}
	if _, _, _, err := conversion.Step(overlay, 0); err != ErrBinaryConversionStride {
		t.Fatalf("zero budget: %v", err)
	}
	one := step(overlay, 1, 1, false)
	if one.Hash() != start || overlay.Hash() != start {
		t.Fatal("conversion replaced an explicit zero or changed the source overlay")
	}
	checkSlot(one, second, slotA, zero[:])
	checkSlot(one, second, slotB, nil)
	two := step(one, 1, 1, false)
	frozenValue := types.Hash{31: 22}
	checkSlot(two, second, slotB, frozenValue[:])
	account, err := two.GetAccount(second)
	if err != nil || account != nil {
		t.Fatalf("converted account before its storage completed: %v, %v", account, err)
	}
	three := step(two, 1, 1, false)
	account, err = three.GetAccount(second)
	if err != nil || account == nil || account.Nonce != 7 || account.Balance.Cmp(big.NewInt(9)) != 0 {
		t.Fatalf("frozen account data: %v, %v", account, err)
	}
	code, err := three.GetContractCode(second)
	if err != nil || !bytes.Equal(code, []byte{0x60, 2, 0x50}) {
		t.Fatalf("frozen account code: %x, %v", code, err)
	}
	finish := step(three, 10, 1, true)
	checkSlot(finish, skipped, slotA, nil)

	alternate := overlay.Copy()
	if err := alternate.UpdateStorage(second, slotB[:], []byte{77}); err != nil {
		t.Fatal(err)
	}
	if err := alternate.UpdateAccount(first, &types.Account{Nonce: 20, Balance: big.NewInt(1), CodeHash: types.EmptyCodeHash.Bytes()}, 0); err != nil {
		t.Fatal(err)
	}
	alternate, used, complete, err := branch.Step(alternate, 10)
	if err != nil || used != 4 || !complete {
		t.Fatalf("independent branch = %d, %v, %v", used, complete, err)
	}
	newValue := types.Hash{31: 77}
	checkSlot(alternate, second, slotB, newValue[:])
	account, err = alternate.GetAccount(first)
	if err != nil || account == nil || account.Nonce != 20 {
		t.Fatalf("overwrote live account: %v, %v", account, err)
	}
	db := rawdb.NewMemoryDB()
	root, err := finish.Persist(db)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := bintrie.OpenPersisted(db, bintrie.HashFunctionBlake3, root)
	if err != nil {
		t.Fatal(err)
	}
	checkSlot(reopened, second, slotA, zero[:])
	checkSlot(reopened, second, slotB, frozenValue[:])
}

func TestBinaryOverlayConversionResumesWithinAccount(t *testing.T) {
	frozen := NewMemoryStateDB()
	address := types.Address{19: 0x19}
	frozen.SetNonce(address, 3)
	frozen.SetCode(address, []byte{0x00})
	for i := byte(0); i < 3; i++ {
		frozen.SetState(address, types.Hash{31: i}, types.Hash{31: i + 1})
	}
	conversion, err := NewBinaryOverlayConversion(frozen)
	if err != nil {
		t.Fatal(err)
	}
	tree, used, done, err := conversion.Step(bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3), 2)
	if err != nil || used != 2 || done {
		t.Fatalf("first step: %d, %v, %v", used, done, err)
	}
	path := t.TempDir()
	db, err := rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := conversion.Persist(tree, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if checkpoint.Account != 0 || checkpoint.Slot != 2 {
		t.Fatalf("checkpoint cursor %+v", checkpoint)
	}
	resumed, reopened, err := ResumeBinaryOverlayConversion(frozen, db, checkpoint)
	if err != nil || reopened.Hash() != tree.Hash() {
		t.Fatalf("resume: %v", err)
	}
	final, used, done, err := resumed.Step(reopened, 2)
	expected, expectedErr := BuildBinaryTreeFromMemory(frozen)
	if err != nil || expectedErr != nil || !done || used != 2 || final.Hash() != expected.Hash() {
		t.Fatalf("resumed completion: %d, %v, %v, materialization %v", used, done, err, expectedErr)
	}

	wrongFrozen := frozen.Copy()
	wrongFrozen.SetNonce(address, 10)
	if _, _, err := ResumeBinaryOverlayConversion(wrongFrozen, db, checkpoint); err != ErrBinaryConversionCheckpoint {
		t.Fatalf("resumed with wrong frozen MPT: %v", err)
	}
	badCursor := checkpoint
	badCursor.Slot = 4
	if _, _, err := ResumeBinaryOverlayConversion(frozen, db, badCursor); err != ErrBinaryConversionCheckpoint {
		t.Fatalf("resumed with an out-of-bounds cursor: %v", err)
	}
	badRoot := checkpoint
	badRoot.OverlayRoot[0] ^= 1
	if _, _, err := ResumeBinaryOverlayConversion(frozen, db, badRoot); err == nil {
		t.Fatal("resumed with a missing overlay root")
	}
}
