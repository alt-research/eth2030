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

func TestMigrateCommittedToBinaryReopen(t *testing.T) {
	sourceDir, destDir := t.TempDir(), t.TempDir()
	source, err := rawdb.NewFileDB(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	dest, err := rawdb.NewFileDB(destDir)
	if err != nil {
		t.Fatal(err)
	}
	state := NewTrieStateDB(source)
	addr, other := types.Address{1}, types.Address{2}
	state.AddBalance(addr, big.NewInt(999))
	state.SetNonce(addr, 7)
	code := bytes.Repeat([]byte{0x60, 0x2a, 0x50}, 1400)
	state.SetCode(addr, code)
	inline, remote := types.Hash{31: 3}, types.Hash{30: 1, 31: 17}
	a, b := types.Hash{31: 42}, types.Hash{0: 7, 31: 99}
	state.SetState(addr, inline, a)
	state.SetState(addr, remote, b)
	state.AddBalance(other, big.NewInt(55))
	sourceRoot, err := state.Commit()
	if err != nil {
		t.Fatal(err)
	}
	root, err := MigrateCommittedToBinary(source, sourceRoot, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dest.Close(); err != nil {
		t.Fatal(err)
	}
	dest, err = rawdb.NewFileDB(destDir)
	if err != nil {
		t.Fatal(err)
	}
	defer dest.Close()
	restored, err := bintrie.OpenPersisted(dest, bintrie.HashFunctionBlake3, root)
	if err != nil {
		t.Fatal(err)
	}
	account, err := restored.GetAccount(addr)
	if err != nil || account == nil || account.Nonce != 7 || account.Balance.Cmp(big.NewInt(999)) != 0 {
		t.Fatalf("account: %+v %v", account, err)
	}
	for _, tc := range []struct{ slot, value types.Hash }{{inline, a}, {remote, b}} {
		value, err := restored.GetStorage(addr, tc.slot[:])
		if err != nil || !bytes.Equal(value, tc.value[:]) {
			t.Fatalf("storage %x: %x, %v", tc.slot, value, err)
		}
	}
	loadedCode, err := restored.GetContractCode(addr)
	if err != nil || !bytes.Equal(loadedCode, code) {
		t.Fatalf("contract code readback: %v", err)
	}
	chunks := bintrie.ChunkifyCode(code)
	for _, i := range []int{0, 127, 128, len(chunks) - 1} {
		key := bintrie.GetTreeKeyForCodeChunk(addr, uint64(i))
		got, err := restored.Get(key[:])
		if err != nil || !bytes.Equal(got, chunks[i][:]) {
			t.Fatalf("chunk %d: %x, %v", i, got, err)
		}
		proof, err := bintrie.BuildInclusionProof(restored, key[:])
		if err != nil || !bintrie.NewBlake3ProofVerifier(root[:]).VerifyInclusion(proof) {
			t.Fatalf("proof %d: %v", i, err)
		}
	}
	second, err := restored.GetAccount(other)
	if err != nil || second == nil || second.Balance.Cmp(big.NewInt(55)) != 0 {
		t.Fatalf("second account: %+v %v", second, err)
	}
}

func TestMigrateCommittedToBinaryRejectsInconsistentSource(t *testing.T) {
	for _, kind := range []string{"root", "storage", "code"} {
		t.Run(kind, func(t *testing.T) {
			source, dest := rawdb.NewMemoryDB(), rawdb.NewMemoryDB()
			state := NewTrieStateDB(source)
			addr := types.Address{1}
			slot := types.Hash{31: 3}
			state.AddBalance(addr, big.NewInt(1))
			state.SetCode(addr, []byte{1, 2, 3})
			state.SetState(addr, slot, types.Hash{31: 9})
			expected, err := state.Commit()
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "root":
				expected[0] ^= 1
			case "storage":
				if err := source.Put(storageDBKey(addr, slot), make([]byte, 32)); err != nil {
					t.Fatal(err)
				}
			case "code":
				if err := source.Put(codeDBKey(state.GetCodeHash(addr).Bytes()), []byte("corrupt")); err != nil {
					t.Fatal(err)
				}
			}
			root, err := MigrateCommittedToBinary(source, expected, dest)
			if !errors.Is(err, ErrBinaryMigrationSource) || root != (types.Hash{}) {
				t.Fatalf("accepted inconsistent source: %x, %v", root, err)
			}
			it := dest.NewIterator([]byte("eth2030:binary-trie:v1:"))
			defer it.Release()
			if it.Next() {
				t.Fatal("persisted nodes before source verification")
			}
		})
	}
}
