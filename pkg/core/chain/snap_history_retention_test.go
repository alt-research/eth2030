package chain

import (
	"context"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func retainedSnapHistory(t *testing.T, finalDelta int64, finalized bool) (*Blockchain, *Blockchain, *types.Block, rawdb.Database) {
	t.Helper()
	full, _ := testChain(t)
	builder := block.NewBlockBuilder(config.TestConfig, full, overlayTestPool(nil))
	build := func(slot uint64) *types.Block {
		parent := full.CurrentBlock()
		b, _, err := builder.BuildBlockWithResult(parent.Header(), &block.BuildBlockAttributes{Timestamp: slot * 12, GasLimit: parent.GasLimit(), SlotNumber: &slot})
		if err != nil {
			t.Fatal(err)
		}
		if err := full.InsertBlock(b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	var blocks []*types.Block
	for slot := uint64(10); slot <= 13; slot++ {
		blocks = append(blocks, build(slot))
	}
	pivot, parent := blocks[3], blocks[2]
	source, err := full.StateAtBlock(pivot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := state.ExportStateSnapshot(source)
	if err != nil {
		t.Fatal(err)
	}
	db := rawdb.NewMemoryDB()
	if err := state.StoreStateSnapshot(db, pivot.Root(), snapshot); err != nil {
		t.Fatal(err)
	}
	target, err := NewBlockchain(config.TestConfig, full.genesis, full.execGenesisState.Copy(), db)
	if err != nil {
		t.Fatal(err)
	}
	headers := []*types.Header{parent.Header(), blocks[1].Header(), blocks[0].Header(), full.genesis.Header()}
	if err := target.ImportSnapPivot(pivot, headers); err != nil {
		t.Fatal(err)
	}
	restore := func(b *types.Block) {
		data, err := full.BlockAccessList(b.Hash())
		if err != nil || data == nil {
			t.Fatalf("original BAL: %v", err)
		}
		if err := target.StoreBlockAccessListRLP(b.Hash(), data); err != nil {
			t.Fatal(err)
		}
		if err := target.CompleteSnapHistoryBlock(b.Hash()); err != nil {
			t.Fatal(err)
		}
	}
	restore(pivot)
	if err := target.StageSnapHistoryParent(pivot.Hash(), parent); err != nil {
		t.Fatal(err)
	}
	restore(parent)
	// A real executed descendant advances elapsed slots while keeping block count small.
	latest := build(uint64(int64(*parent.Header().SlotNumber+rawdb.BALRetentionSlots) + finalDelta))
	if err := target.InsertBlock(latest); err != nil {
		t.Fatal(err)
	}
	if finalized {
		if err := target.ApplyForkchoice(latest.Hash(), latest.Hash(), latest.Hash()); err != nil {
			t.Fatal(err)
		}
	}
	return target, full, parent, db
}

func TestSnapHistoryResumeBALRetentionBoundary(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		delta                    int64
		finalized, corrupt, want bool
	}{
		{"expired", 1, true, false, true},
		{"exact-boundary", 0, true, false, false},
		{"within-window", -1, true, false, false},
		{"unfinalized-head-does-not-expire-history", 1, false, false, false},
		{"expired-corrupt-data-still-rejected", 1, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, _, old, db := retainedSnapHistory(t, tc.delta, tc.finalized)
			head, final, root := target.CurrentBlock().Hash(), target.CurrentFinalBlock().Hash(), target.State().GetRoot()
			var err error
			if tc.corrupt {
				err = db.Put(blockAccessListKey(old.Hash()), []byte{0xff})
			} else {
				err = db.Delete(blockAccessListKey(old.Hash()))
			}
			if err != nil {
				t.Fatal(err)
			}
			cursor, err := target.SnapHistoryCursor(context.Background())
			if tc.want {
				if err != nil || cursor == nil || cursor.Hash() != old.Hash() {
					t.Fatalf("legally pruned prefix cannot resume: %v", err)
				}
			} else if err == nil {
				t.Fatal("accepted missing retained BAL, unfinalized expiry or corrupt bytes")
			}
			if target.CurrentBlock().Hash() != head || target.CurrentFinalBlock().Hash() != final || target.State().GetRoot() != root || target.HasValidatedBlock(old.Hash()) {
				t.Fatal("resume changed execution/state/head/finality authority")
			}
		})
	}
}

func TestSnapHistoryPrunedPrefixSurvivesFileDBReopen(t *testing.T) {
	target, full, old, memory := retainedSnapHistory(t, 1, true)
	if err := memory.Delete(blockAccessListKey(old.Hash())); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	db, err := rawdb.NewFileDB(directory)
	if err != nil {
		t.Fatal(err)
	}
	batch := db.NewBatch()
	it := memory.NewIterator(nil)
	for it.Next() {
		if err := batch.Put(it.Key(), it.Value()); err != nil {
			t.Fatal(err)
		}
	}
	it.Release()
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = rawdb.NewFileDB(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	reopened, err := NewBlockchain(config.TestConfig, full.genesis, full.execGenesisState.Copy(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	cursor, err := reopened.SnapHistoryCursor(context.Background())
	if err != nil || cursor == nil || cursor.Hash() != old.Hash() {
		t.Fatalf("expired BAL prevented durable cursor recovery: %v", err)
	}
	if reopened.CurrentBlock().Hash() != target.CurrentBlock().Hash() || reopened.CurrentFinalBlock().Hash() != target.CurrentFinalBlock().Hash() || reopened.State().GetRoot() != target.State().GetRoot() || reopened.HasValidatedBlock(old.Hash()) {
		t.Fatal("reopen changed original authenticated authority")
	}
	if data, err := reopened.BlockAccessList(old.Hash()); err != nil || data != nil {
		t.Fatalf("pruned BAL was fabricated: %x %v", data, err)
	}
}

func TestSnapHistoryUnknownSlotsKeepBALRequired(t *testing.T) {
	target, full, db, blocks := snapHistoryFixture(t)
	pivot, parent := blocks[4], blocks[3]
	if err := target.CompleteSnapHistoryBlock(pivot.Hash()); err != nil {
		t.Fatal(err)
	}
	if err := target.StageSnapHistoryParent(pivot.Hash(), parent); err != nil {
		t.Fatal(err)
	}
	data, err := full.BlockAccessList(parent.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := target.StoreBlockAccessListRLP(parent.Hash(), data); err != nil {
		t.Fatal(err)
	}
	if err := target.CompleteSnapHistoryBlock(parent.Hash()); err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(blockAccessListKey(parent.Hash())); err != nil {
		t.Fatal(err)
	}
	if _, err := target.SnapHistoryCursor(context.Background()); err == nil {
		t.Fatal("unknown beacon slots waived a required BAL")
	}
}
