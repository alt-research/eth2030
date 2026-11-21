package chain

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func snapHistoryFixture(t *testing.T) (*Blockchain, *Blockchain, *overlayFailDB, []*types.Block) {
	t.Helper()
	full, _, blocks := newHistoryChain(t)
	pivot := blocks[4]
	st, err := full.StateAtBlock(pivot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := state.ExportStateSnapshot(st)
	if err != nil {
		t.Fatal(err)
	}
	db := &overlayFailDB{Database: rawdb.NewMemoryDB()}
	if err := state.StoreStateSnapshot(db, pivot.Root(), snapshot); err != nil {
		t.Fatal(err)
	}
	target, err := NewBlockchain(config.TestConfig, full.genesis, full.execGenesisState.Copy(), db)
	if err != nil {
		t.Fatal(err)
	}
	parents := []*types.Header{blocks[3].Header(), blocks[2].Header(), blocks[1].Header(), blocks[0].Header(), full.genesis.Header()}
	if err := target.ImportSnapPivot(pivot, parents); err != nil {
		t.Fatal(err)
	}
	bal, err := full.BlockAccessList(pivot.Hash())
	if err != nil || bal == nil {
		t.Fatalf("fixture lacks actual pivot BAL: %v", err)
	}
	if err := target.StoreBlockAccessListRLP(pivot.Hash(), bal); err != nil {
		t.Fatal(err)
	}
	return target, full, db, blocks
}

func TestSnapHistoryRestoresOriginalBytesWithoutExecutionOrStateAuthority(t *testing.T) {
	target, full, db, blocks := snapHistoryFixture(t)
	pivot := blocks[4]
	parent := blocks[3]
	before, root := target.CurrentBlock().Hash(), target.State().GetRoot()
	if err := target.CompleteSnapHistoryBlock(pivot.Hash()); err != nil {
		t.Fatal(err)
	}
	if err := target.StageSnapHistoryParent(pivot.Hash(), parent); err != nil {
		t.Fatal(err)
	}
	if err := target.CompleteSnapHistoryBlock(parent.Hash()); err == nil {
		t.Fatal("frontier advanced before required BAL became available")
	}
	want, err := full.BlockAccessList(parent.Hash())
	if err != nil || want == nil {
		t.Fatal("missing independent source BAL")
	}
	if err := target.StoreBlockAccessListRLP(parent.Hash(), append(bytes.Clone(want), 0)); err == nil {
		t.Fatal("accepted changed original BAL bytes")
	}
	if err := target.StoreBlockAccessListRLP(parent.Hash(), want); err != nil {
		t.Fatal(err)
	}
	if err := target.CompleteSnapHistoryBlock(parent.Hash()); err != nil {
		t.Fatal(err)
	}
	got, err := target.BlockAccessList(parent.Hash())
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("original bytes lost: %v", err)
	}
	if target.HasValidatedBlock(parent.Hash()) || target.CurrentBlock().Hash() != before || target.State().GetRoot() != root {
		t.Fatal("historical body acquisition changed execution/state/head authority")
	}
	if _, err := target.StateAtBlock(parent); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("historical download manufactured pre-pivot state: %v", err)
	}
	cursor, err := target.SnapHistoryCursor(context.Background())
	if err != nil || cursor.Hash() != parent.Hash() {
		t.Fatalf("resume frontier: %v", err)
	}
	if err := db.Put(snapHistoryCursorKey, append(pivot.Hash().Bytes(), blocks[0].Hash().Bytes()...)); err != nil {
		t.Fatal(err)
	}
	if _, err := target.SnapHistoryCursor(context.Background()); err == nil {
		t.Fatal("accepted frontier beyond the saved authenticated prefix")
	}
}

func TestSnapHistoryRejectsDisconnectedBodiesFailedBatchAndCancellation(t *testing.T) {
	target, _, db, blocks := snapHistoryFixture(t)
	pivot, parent := blocks[4], blocks[3]
	if err := target.StageSnapHistoryParent(blocks[2].Hash(), parent); err == nil {
		t.Fatal("accepted a different cursor")
	}
	wrong := types.NewBlock(parent.Header(), &types.Body{Withdrawals: []*types.Withdrawal{{Amount: 1}}})
	if err := target.StageSnapHistoryParent(pivot.Hash(), wrong); err == nil {
		t.Fatal("accepted a body that does not match its original header")
	}
	db.fail = true
	if err := target.StageSnapHistoryParent(pivot.Hash(), parent); err == nil {
		t.Fatal("failed batch was accepted")
	}
	db.fail = false
	if ready, err := target.snapHistoryBlock(parent.Hash()); err != nil || ready {
		t.Fatalf("failed batch published an authenticated marker: %t %v", ready, err)
	}
	if err := target.StageSnapHistoryParent(pivot.Hash(), parent); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := target.SnapHistoryCursor(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled history read: %v", err)
	}
}
