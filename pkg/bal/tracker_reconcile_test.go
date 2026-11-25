package bal

import (
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestTrackerFinalStateAndReset(t *testing.T) {
	tracker := NewTracker()
	db := state.NewMemoryStateDB()
	address := types.HexToAddress("0x123456")
	slot := types.Hash{31: 1}
	old := types.Hash{31: 2}
	db.SetState(address, slot, old)
	db.AddBalance(address, big.NewInt(10))
	db.SetNonce(address, 1)
	tracker.SeedAccount(address, big.NewInt(20), 0)
	tracker.RecordStorageChange(address, slot, old, types.Hash{31: 3})
	tracker.RecordStorageChange(address, slot, types.Hash{31: 3}, old)
	tracker.RecordBalanceChange(address, big.NewInt(15), big.NewInt(12))
	tracker.RecordBalanceChange(address, big.NewInt(12), big.NewInt(10))
	tracker.RecordNonceChange(address, 1, 2)
	tracker.Reconcile(db)
	entry := tracker.Build(1).Entries[0]
	if len(entry.StorageChanges) != 0 || len(entry.StorageReads) != 1 || entry.BalanceChange.OldValue.Int64() != 20 || entry.BalanceChange.NewValue.Int64() != 10 || entry.NonceChange.OldValue != 0 || entry.NonceChange.NewValue != 1 {
		t.Fatalf("bad finalized changes: %+v", entry)
	}
	tracker.Reset()
	tracker.RecordBalanceChange(address, big.NewInt(10), big.NewInt(9))
	tracker.RecordNonceChange(address, 1, 2)
	entry = tracker.Build(2).Entries[0]
	if entry.BalanceChange.OldValue.Int64() != 10 || entry.NonceChange.OldValue != 1 {
		t.Fatal("reset retained previous transaction baseline")
	}
}
