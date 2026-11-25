package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func TestSnapBootstrapExecutionPublishesOnlyAtomicMerkleHistory(t *testing.T) {
	full, _, blocks := newHistoryChain(t)
	pivot := blocks[2]
	pivotState, err := full.StateAtBlock(pivot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := state.ExportStateSnapshot(pivotState)
	if err != nil {
		t.Fatal(err)
	}
	db := rawdb.NewMemoryDB()
	if err := state.StoreStateSnapshot(db, pivot.Root(), snapshot); err != nil {
		t.Fatal(err)
	}
	chain, err := NewBlockchain(config.TestConfig, full.genesis, full.execGenesisState.Copy(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.ImportSnapPivot(pivot, []*types.Header{blocks[1].Header(), blocks[0].Header(), full.genesis.Header()}); err != nil {
		t.Fatal(err)
	}
	child := makeBlockWithState(pivot, nil, pivotState)
	if child.Root() == pivot.Root() {
		t.Fatal("fixture did not change actual EIP-2935 storage")
	}
	if err := chain.InsertBlock(child); err != nil {
		t.Fatal(err)
	}
	if chain.CurrentBlock().Hash() != child.Hash() || chain.State().GetRoot() != child.Root() {
		t.Fatal("post-pivot execution lost its exact original root")
	}
	if _, err := state.LoadStateSnapshot(db, child.Root()); !errors.Is(err, rawdb.ErrNotFound) {
		t.Fatalf("ordinary snap execution duplicated the entire flat snapshot: %v", err)
	}
	if _, err := state.OpenTrieHistory(context.Background(), db, child.Root()); err != nil {
		t.Fatalf("validated child did not retain atomic native history: %v", err)
	}
	if _, err := state.LoadStateSnapshot(db, pivot.Root()); err != nil {
		t.Fatalf("original authenticated bootstrap checkpoint was not retained: %v", err)
	}
}
