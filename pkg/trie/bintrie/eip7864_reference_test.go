package bintrie

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

// BinaryTree.merkelize known answers from jsign/binary-tree-spec at
// ca2c719e2c85777ab4d2db640a80d90ba4015394 (test_tree.py).
func TestEIP7864ReferenceMerkleRoots(t *testing.T) {
	tree := NewWithHashFunc(HashFunctionBlake3)
	for _, entry := range []struct {
		first byte
		value byte
		root  string
	}{
		{0, 1, "694545468677064fd833cddc8455762fe6b21c6cabe2fc172529e0f573181cd5"},
		{0x80, 2, "85fc622076752a6fcda2c886c18058d639066a83473d9684704b5a29455ed2ed"},
	} {
		key := [32]byte{entry.first}
		if err := tree.Put(key[:], bytes.Repeat([]byte{entry.value}, 32)); err != nil {
			t.Fatal(err)
		}
		if got, want := tree.Hash(), types.HexToHash(entry.root); got != want {
			t.Fatalf("root = %s, want %s", got, want)
		}
	}
}

func TestEIP7864ReferenceStateEmbedding(t *testing.T) {
	// Generated with the EIP-linked Python BinaryTree and embedding.py at
	// ca2c719. Use the pinned draft's MAIN_STORAGE_OFFSET = 256**31 in
	// place of the older reference module's constant 256.
	addr := types.HexToAddress("0123456789abcdef0123456789abcdef01234567")
	code := make([]byte, 31*129)
	code[30] = 0x7f
	for i := range 32 {
		code[31+i] = byte(i + 1)
	}
	balance, _ := new(big.Int).SetString("102030405060708090a0b0c0d0e0f000", 16)
	tree := NewWithHashFunc(HashFunctionBlake3)
	if err := tree.UpdateAccount(addr, &types.Account{
		Nonce: 0x0102030405060708, Balance: balance, CodeHash: crypto.Keccak256(code),
	}, len(code)); err != nil {
		t.Fatal(err)
	}
	if err := tree.UpdateContractCode(addr, code); err != nil {
		t.Fatal(err)
	}
	for i, slot := range []string{"0", "3f", "40", "ff", "100", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"} {
		index, _ := new(big.Int).SetString(slot, 16)
		var raw [32]byte
		index.FillBytes(raw[:])
		if err := tree.UpdateStorage(addr, raw[:], []byte{byte(i + 1)}); err != nil {
			t.Fatal(err)
		}
	}
	want := types.HexToHash("809220bf192acc1d084f16403057675729919ac5affc3259543f4e2ef0a5950f")
	if got := tree.Hash(); got != want {
		t.Fatalf("state root = %s, want %s", got, want)
	}

	store := rawdb.NewMemoryDB()
	root, err := tree.Persist(store)
	if err != nil || root != want {
		t.Fatalf("persisted root = %s, want %s, error %v", root, want, err)
	}
	reopened, err := OpenPersisted(store, HashFunctionBlake3, want)
	if err != nil {
		t.Fatal(err)
	}
	chunkKey := GetTreeKeyForCodeChunk(addr, 1)
	chunk, err := reopened.Get(chunkKey[:])
	if err != nil || hex.EncodeToString(chunk) != "1f0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f" {
		t.Fatalf("PUSH32 continuation = %x, error %v", chunk, err)
	}
	proof, err := BuildInclusionProof(reopened, chunkKey[:])
	verifier := NewBlake3ProofVerifier(want[:])
	if err != nil || !verifier.VerifyInclusion(proof) {
		t.Fatalf("opening against independent root: %v", err)
	}
	proof.Value[0] ^= 1
	if verifier.VerifyInclusion(proof) {
		t.Fatal("accepted altered chunk metadata")
	}
}
