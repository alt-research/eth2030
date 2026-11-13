package main

import (
	"bytes"
	"encoding/json"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/trie/bintrie"
)

func TestBinaryMigrateFileDBResume(t *testing.T) {
	dir := t.TempDir()
	source, dest := filepath.Join(dir, "source"), filepath.Join(dir, "dest")
	db, err := rawdb.NewFileDB(source)
	if err != nil {
		t.Fatal(err)
	}
	mem := state.NewMemoryStateDB()
	addresses := []types.Address{{19: 1}, {19: 2}}
	slot := types.Hash{31: 64}
	code := []byte{0x60, 0x01, 0x50}
	for i, address := range addresses {
		mem.AddBalance(address, big.NewInt(int64(i+10)))
		mem.SetCode(address, code)
		mem.SetState(address, slot, types.Hash{31: byte(i + 1)})
	}
	committed := state.NewTrieStateDBFromMemory(db, mem.Copy())
	root, err := committed.Commit()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := state.BuildBinaryTreeFromMemory(mem)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{"-source", source, "-destination", dest, "-source-root", root.Hex(), "-batch-size", "1"}
	var stdout, stderr bytes.Buffer
	if exit := runBinaryMigration(args, &stdout, &stderr); exit != 1 || stdout.Len() != 0 {
		t.Fatalf("locked source exit = %d, stdout = %q, stderr = %q", exit, &stdout, &stderr)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	wrongRoot := append(append([]string{}, args...), "-source-root", types.EmptyRootHash.Hex())
	if exit := runBinaryMigration(wrongRoot, &stdout, &stderr); exit != 1 || stdout.Len() != 0 {
		t.Fatalf("incorrect source root exit = %d, stdout = %q, stderr = %q", exit, &stdout, &stderr)
	}
	stdout.Reset()
	stderr.Reset()
	alias := append(append([]string{}, args...), "-destination", source)
	if exit := runBinaryMigration(alias, &stdout, &stderr); exit != 1 || stdout.Len() != 0 {
		t.Fatalf("source used as destination exit = %d, stdout = %q, stderr = %q", exit, &stdout, &stderr)
	}

	invoke := func(extra ...string) (binaryRoot types.Hash, accounts uint64, complete bool) {
		t.Helper()
		stdout.Reset()
		stderr.Reset()
		if exit := runBinaryMigration(append(append([]string{}, args...), extra...), &stdout, &stderr); exit != 0 {
			t.Fatalf("migration exit = %d: %s", exit, &stderr)
		}
		var result struct {
			SourceRoot string `json:"source_root"`
			BinaryRoot string `json:"binary_root"`
			Accounts   uint64 `json:"accounts"`
			Complete   bool   `json:"complete"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.SourceRoot != root.Hex() {
			t.Fatalf("invalid migration result: %s, error %v", &stdout, err)
		}
		return types.HexToHash(result.BinaryRoot), result.Accounts, result.Complete
	}

	partial, count, complete := invoke("-max-batches", "1")
	if complete || count != 1 || partial == expected.Hash() {
		t.Fatalf("first checkpoint = %s, accounts %d, complete %v", partial, count, complete)
	}
	final, count, complete := invoke()
	if !complete || count != 2 || final != expected.Hash() {
		t.Fatalf("final checkpoint = %s, accounts %d, complete %v", final, count, complete)
	}
	out, err := rawdb.NewFileDB(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	tree, err := bintrie.OpenPersisted(out, bintrie.HashFunctionBlake3, final)
	if err != nil {
		t.Fatal(err)
	}
	for i, address := range addresses {
		balance, err := tree.GetAccount(address)
		if err != nil || balance == nil || balance.Balance.Cmp(big.NewInt(int64(i+10))) != 0 {
			t.Fatalf("account %d: %+v, error %v", i, balance, err)
		}
		gotCode, err := tree.GetContractCode(address)
		if err != nil || !bytes.Equal(gotCode, code) {
			t.Fatalf("code %d: %x, error %v", i, gotCode, err)
		}
		storage, err := tree.GetStorage(address, slot[:])
		if err != nil || types.BytesToHash(storage) != (types.Hash{31: byte(i + 1)}) {
			t.Fatalf("storage %d: %x, error %v", i, storage, err)
		}
	}
}

func TestBinaryMigrateRejectsInvalidInput(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	exit := runBinaryMigration([]string{"-source", dir, "-destination", filepath.Join(dir, "dest"), "-source-root", "00"}, &stdout, &stderr)
	if exit != 2 || stdout.Len() != 0 {
		t.Fatalf("invalid root exit = %d, stdout = %q", exit, &stdout)
	}
	exit = runBinaryMigration([]string{"-source", dir, "-destination", filepath.Join(dir, "dest"), "-source-root", types.EmptyRootHash.Hex()}, &stdout, &stderr)
	if exit != 1 || stdout.Len() != 0 {
		t.Fatalf("missing source exit = %d, stdout = %q", exit, &stdout)
	}
}
