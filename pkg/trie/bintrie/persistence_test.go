package bintrie

import (
	"bytes"
	"errors"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

func TestPersistReopenAndVerify(t *testing.T) {
	dir := t.TempDir()
	db, err := rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	trie := NewWithHashFunc(HashFunctionBlake3)
	keys := [][32]byte{makeKey(0x10, 2), makeKey(0x10, 9), makeKey(0x80, 3)}
	value := makeVal(0x42)
	for _, key := range keys {
		if err := trie.Put(key[:], value[:]); err != nil {
			t.Fatal(err)
		}
	}
	root, err := trie.Persist(db)
	if err != nil {
		t.Fatal(err)
	}
	if root != trie.Hash() {
		t.Fatal("persisted root mismatch")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restored, err := OpenPersisted(db, HashFunctionBlake3, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		got, err := restored.Get(key[:])
		if err != nil || !bytes.Equal(got, value[:]) {
			t.Fatalf("readback %x: %x %v", key, got, err)
		}
		proof, err := BuildInclusionProof(restored, key[:])
		if err != nil || !NewBlake3ProofVerifier(root[:]).VerifyInclusion(proof) {
			t.Fatalf("proof after reopen: %v", err)
		}
	}
	if err := restored.Put(keys[0][:], makeValBytes(0x77)); err != nil {
		t.Fatal(err)
	}
	changed, err := restored.Persist(db)
	if err != nil || changed == root {
		t.Fatalf("updated root: %x %v", changed, err)
	}
	historical, err := OpenPersisted(db, HashFunctionBlake3, root)
	if err != nil {
		t.Fatal(err)
	}
	old, err := historical.Get(keys[0][:])
	if err != nil || !bytes.Equal(old, value[:]) {
		t.Fatalf("old root read: %x %v", old, err)
	}
	// A changed serialized root must never be accepted as an authenticated tree.
	data, err := db.Get(storedNodeKey(HashFunctionBlake3, root))
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 1
	if err := db.Put(storedNodeKey(HashFunctionBlake3, root), data); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenPersisted(db, HashFunctionBlake3, root); err == nil {
		t.Fatal("accepted corrupted node")
	}
}

func makeValBytes(b byte) []byte { value := makeVal(b); return value[:] }

type failingNodeStore struct{}

func (failingNodeStore) Get([]byte) ([]byte, error) { return nil, errors.New("read failed") }
func (failingNodeStore) Put([]byte, []byte) error   { return errors.New("write failed") }

func TestPersistFailureDoesNotReturnRoot(t *testing.T) {
	trie := NewWithHashFunc(HashFunctionBlake3)
	key, value := makeKey(1, 2), makeVal(3)
	if err := trie.Put(key[:], value[:]); err != nil {
		t.Fatal(err)
	}
	root, err := trie.Persist(failingNodeStore{})
	if err == nil || root != (types.Hash{}) {
		t.Fatalf("failed write returned root %x, %v", root, err)
	}
}

func TestPersistLegacySHAAndMissingNode(t *testing.T) {
	db := rawdb.NewMemoryDB()
	trie := New()
	a, b := makeKey(0, 1), makeKey(0x80, 2)
	value := makeVal(4)
	for _, key := range [][32]byte{a, b} {
		if err := trie.Put(key[:], value[:]); err != nil {
			t.Fatal(err)
		}
	}
	root, err := trie.Persist(db)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := OpenPersisted(db, HashFunctionSHA256, root)
	if err != nil || restored.Hash() != root {
		t.Fatalf("SHA restore: %v", err)
	}
	blob, err := db.Get(storedNodeKey(HashFunctionSHA256, root))
	if err != nil {
		t.Fatal(err)
	}
	child := types.BytesToHash(blob[1:33])
	if err := db.Delete(storedNodeKey(HashFunctionSHA256, child)); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenPersisted(db, HashFunctionSHA256, root); err == nil {
		t.Fatal("accepted missing child")
	}
	if _, err := OpenPersisted(db, HashFunctionBlake3, root); err == nil {
		t.Fatal("accepted root under wrong hash function")
	}
}

func TestPersistIncrementalWritesOnlyChangedPath(t *testing.T) {
	tree := NewWithHashFunc(HashFunctionBlake3)
	db := rawdb.NewMemoryDB()
	left, right := makeKey(0, 1), makeKey(0x80, 1)
	value := makeVal(7)
	for _, key := range [][32]byte{left, right} {
		if err := tree.Put(key[:], value[:]); err != nil {
			t.Fatal(err)
		}
	}
	first, err := tree.PersistIncremental(db, db.Has)
	if err != nil || db.Len() != 3 {
		t.Fatalf("first write: %d nodes, err %v", db.Len(), err)
	}
	fork := tree.Copy()
	value[31]++
	if err := fork.Put(right[:], value[:]); err != nil {
		t.Fatal(err)
	}
	second, err := fork.PersistIncremental(db, db.Has)
	if err != nil || first == second || db.Len() != 5 {
		t.Fatalf("fork write: %d nodes, err %v", db.Len(), err)
	}
	for _, root := range []types.Hash{first, second} {
		loaded, err := OpenPersisted(db, HashFunctionBlake3, root)
		if err != nil || loaded.Hash() != root {
			t.Fatalf("reopen root %s: %v", root, err)
		}
	}
	if _, err := fork.PersistIncremental(failingNodeStore{}, db.Has); err != nil {
		t.Fatalf("republishing an existing root tried to write a node: %v", err)
	}
}
