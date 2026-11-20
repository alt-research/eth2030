package state

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

func TestSecureSnapshotStorageReplacementIsPrivateAndRevertible(t *testing.T) {
	source, address, key := historyFixture()
	root := source.GetRoot()
	snapshot, err := ExportStateSnapshot(source)
	if err != nil {
		t.Fatal(err)
	}
	db := rawdb.NewMemoryDB()
	if err := StoreStateSnapshot(db, root, snapshot); err != nil {
		t.Fatal(err)
	}
	current, err := OpenSnapState(db, root)
	if err != nil {
		t.Fatal(err)
	}
	original := current.Dup()
	before := current.Snapshot()
	replacement := types.Hash{31: 0x71}
	value := types.Hash{31: 0x72}
	current.SetStorage(address, map[types.Hash]types.Hash{replacement: value})
	if current.GetState(address, key) != (types.Hash{}) || current.GetState(address, replacement) != value || current.GetBalance(address).Cmp(big.NewInt(100)) != 0 || current.GetNonce(address) != 1 {
		t.Fatal("full storage override lost replacement or unrelated account fields")
	}
	if original.GetRoot() != root {
		t.Fatal("private override changed authenticated base")
	}
	current.RevertToSnapshot(before)
	if current.GetRoot() != root || current.GetState(address, key) != source.GetState(address, key) {
		t.Fatal("reverting replacement did not restore original secure storage")
	}
}
