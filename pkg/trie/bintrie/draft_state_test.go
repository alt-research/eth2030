package bintrie

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

func TestDraftDeleteCollapsesAndReinserts(t *testing.T) {
	trie := NewWithHashFunc(HashFunctionBlake3)
	a, b, c := makeKey(0x10, 1), makeKey(0x11, 2), makeKey(0x80, 3)
	value := makeVal(7)
	for _, key := range [][32]byte{a, b, c} {
		if err := trie.Put(key[:], value[:]); err != nil {
			t.Fatal(err)
		}
	}
	if err := trie.Delete(b[:]); err != nil {
		t.Fatal(err)
	}
	expected := NewWithHashFunc(HashFunctionBlake3)
	for _, key := range [][32]byte{a, c} {
		if err := expected.Put(key[:], value[:]); err != nil {
			t.Fatal(err)
		}
	}
	if trie.Hash() != expected.Hash() {
		t.Fatal("deletion left a noncanonical branch")
	}
	if got, err := trie.Get(b[:]); err != nil || got != nil {
		t.Fatalf("deleted key: %x, %v", got, err)
	}
	if err := trie.Put(b[:], value[:]); err != nil {
		t.Fatal(err)
	}
	if err := expected.Put(b[:], value[:]); err != nil {
		t.Fatal(err)
	}
	if trie.Hash() != expected.Hash() {
		t.Fatal("reinsertion after collapse has incorrect root")
	}
	for _, key := range [][32]byte{a, b, c} {
		if err := trie.Delete(key[:]); err != nil {
			t.Fatal(err)
		}
	}
	if trie.Hash() != (types.Hash{}) {
		t.Fatal("deleting all leaves did not produce the empty root")
	}
	if err := trie.Delete(a[:]); err != nil || trie.Hash() != (types.Hash{}) {
		t.Fatal("deleting a missing key changed the empty tree")
	}
}

func TestDraftStorageZeroDeletesLeaf(t *testing.T) {
	trie := NewWithHashFunc(HashFunctionBlake3)
	address, slot := types.Address{1}, types.Hash{31: 2}
	if err := trie.UpdateStorage(address, slot[:], []byte{9}); err != nil {
		t.Fatal(err)
	}
	if err := trie.UpdateStorage(address, slot[:], []byte{0}); err != nil {
		t.Fatal(err)
	}
	if got, err := trie.GetStorage(address, slot[:]); err != nil || got != nil {
		t.Fatalf("zero storage: %x %v", got, err)
	}
	if trie.Hash() != (types.Hash{}) {
		t.Fatal("zero storage retained a leaf")
	}
	if err := trie.UpdateStorage(address, slot[:], make([]byte, 33)); err != ErrInvalidValueLength {
		t.Fatalf("oversize: %v", err)
	}
}

func TestDraftReplaceCodeAndPersist(t *testing.T) {
	trie := NewWithHashFunc(HashFunctionBlake3)
	address := types.Address{1}
	account := &types.Account{Balance: big.NewInt(42), Nonce: 7, CodeHash: types.EmptyCodeHash.Bytes()}
	if err := trie.UpdateAccount(address, account, 0); err != nil {
		t.Fatal(err)
	}
	original := bytes.Repeat([]byte{0x60, 0x01, 0x50}, 1500)
	if err := trie.ReplaceContractCode(address, original); err != nil {
		t.Fatal(err)
	}
	got, err := trie.GetContractCode(address)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("code read: %v", err)
	}
	short := []byte{0x60, 0x02, 0x00}
	if err := trie.ReplaceContractCode(address, short); err != nil {
		t.Fatal(err)
	}
	got, err = trie.GetContractCode(address)
	if err != nil || !bytes.Equal(got, short) {
		t.Fatalf("short code read: %v", err)
	}
	oldKey := GetTreeKeyForCodeChunk(address, 128)
	if chunk, err := trie.Get(oldKey[:]); err != nil || chunk != nil {
		t.Fatalf("obsolete chunk remains: %x %v", chunk, err)
	}
	expected := NewWithHashFunc(HashFunctionBlake3)
	if err := expected.UpdateAccount(address, account, 0); err != nil {
		t.Fatal(err)
	}
	if err := expected.ReplaceContractCode(address, short); err != nil {
		t.Fatal(err)
	}
	if trie.Hash() != expected.Hash() {
		t.Fatal("replacement retained stale code chunks")
	}
	db := rawdb.NewMemoryDB()
	root, err := trie.Persist(db)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenPersisted(db, HashFunctionBlake3, root)
	if err != nil {
		t.Fatal(err)
	}
	got, err = reopened.GetContractCode(address)
	if err != nil || !bytes.Equal(got, short) {
		t.Fatalf("reopened code: %v", err)
	}
	key := GetTreeKeyForCodeChunk(address, 0)
	chunk, _ := reopened.Get(key[:])
	chunk[0] ^= 1
	if err := reopened.Put(key[:], chunk); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.GetContractCode(address); err == nil {
		t.Fatal("accepted invalid code chunk metadata")
	}
}
