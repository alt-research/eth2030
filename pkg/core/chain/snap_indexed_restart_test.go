package chain

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
)

func indexedSnapRestart(t *testing.T, fault string) (*Blockchain, *Blockchain, *restartWriteGuard) {
	t.Helper()
	target, full, memory, blocks := snapHistoryFixture(t)
	pivot := blocks[4]
	if err := target.CompleteSnapHistoryBlock(pivot.Hash()); err != nil {
		t.Fatal(err)
	}
	parent := blocks[3]
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
	if fault != "pivot-head" {
		if err := target.InsertBlock(blocks[5]); err != nil {
			t.Fatal(err)
		}
		if err := target.ApplyForkchoice(blocks[5].Hash(), blocks[5].Hash(), blocks[5].Hash()); err != nil {
			t.Fatal(err)
		}
	}
	switch fault {
	case "missing-marker":
		err = memory.Delete(validatedKey(blocks[5].Hash()))
	case "missing-index":
		err = memory.Delete(append([]byte("mpt-history-root/"), blocks[5].Root().Bytes()...))
	case "bad-marker":
		err = memory.Put(validatedKey(blocks[5].Hash()), []byte{2})
	case "head-state":
		err = memory.Put(append([]byte("mpt-history-node/"), blocks[5].Root().Bytes()...), []byte{0xc0})
	case "canonical-pivot":
		err = rawdb.WriteCanonicalHash(memory, pivot.NumberU64(), blocks[3].Hash())
	case "finality-before-pivot":
		err = rawdb.WriteFinalizedBlockHash(memory, blocks[3].Hash())
	case "missing-finality":
		err = rawdb.WriteFinalizedBlockHash(memory, [32]byte{})
	}
	if err != nil {
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
	guard := &restartWriteGuard{Database: db, rejectReexecution: true}
	reopened, err := NewBlockchain(config.TestConfig, full.genesis, full.execGenesisState.Copy(), guard, BlockchainOpts{BlockCacheSize: 2, StateCacheSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	return reopened, target, guard
}

func TestIndexedSnapRestartRetainsOriginalCheckpointAuthorityWithoutReexecution(t *testing.T) {
	reopened, original, guard := indexedSnapRestart(t, "")
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if reopened.CurrentBlock().Hash() != original.CurrentBlock().Hash() || reopened.State().GetRoot() != original.State().GetRoot() || reopened.CurrentFinalBlock().Hash() != original.CurrentFinalBlock().Hash() {
		t.Fatal("recovery changed original head/state/finality")
	}
	cursor, err := reopened.SnapHistoryCursor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.HasValidatedBlock(cursor.Hash()) {
		t.Fatal("downloaded pre-pivot body acquired execution authority")
	}
	if _, err := reopened.StateAtBlock(cursor); !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("pre-pivot state invented: %v", err)
	}
	if len(reopened.blockCache) > 2 || reopened.blockCache[reopened.CurrentBlock().Hash()] == nil {
		t.Fatal("bounded cache lost current head")
	}
	guard.rejectReexecution = false
	child := makeBlockWithState(reopened.CurrentBlock(), nil, reopened.State())
	if err := reopened.InsertBlock(child); err != nil {
		t.Fatal(err)
	}
	if !reopened.HasValidatedBlock(child.Hash()) || reopened.State().GetRoot() != child.Root() {
		t.Fatal("ordinary subsequent execution failed")
	}
}

func TestIndexedSnapRestartRejectsCorruptStateAndFinality(t *testing.T) {
	for _, fault := range []string{"bad-marker", "head-state", "canonical-pivot", "finality-before-pivot", "missing-finality"} {
		t.Run(fault, func(t *testing.T) {
			reopened, _, _ := indexedSnapRestart(t, fault)
			if err := reopened.RestoreSnapPivot(); err == nil {
				t.Fatal("corrupt saved recovery was accepted")
			}
			if reopened.CurrentBlock().Hash() != reopened.genesis.Hash() {
				t.Fatal("unverified recovery advanced in-memory head")
			}
		})
	}
}

func TestIndexedSnapMissingValidationReexecutesLegacyDescendants(t *testing.T) {
	reopened, original, guard := indexedSnapRestart(t, "missing-marker")
	if err := reopened.RestoreSnapPivot(); err == nil {
		t.Fatal("missing execution authority was replaced by indexed state")
	}
	guard.rejectReexecution = false
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if reopened.CurrentBlock().Hash() != original.CurrentBlock().Hash() || !reopened.HasValidatedBlock(original.CurrentBlock().Hash()) {
		t.Fatal("legacy execution failed to recover original validated head")
	}
}

func TestIndexedSnapRestartPreservesMissingAuxiliaryBAL(t *testing.T) {
	reopened, original, _ := indexedSnapRestart(t, "")
	if err := reopened.db.Delete(blockAccessListKey(original.CurrentBlock().Hash())); err != nil {
		t.Fatal(err)
	}
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	data, err := reopened.BlockAccessList(original.CurrentBlock().Hash())
	if err != nil || data != nil {
		t.Fatalf("missing auxiliary bytes fabricated: %x %v", data, err)
	}
	if !bytes.Equal(reopened.State().GetRoot().Bytes(), original.State().GetRoot().Bytes()) {
		t.Fatal("auxiliary availability changed authenticated state")
	}
}

func TestIndexedSnapCheckpointOnlyRestart(t *testing.T) {
	reopened, original, _ := indexedSnapRestart(t, "pivot-head")
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if reopened.CurrentBlock().Hash() != original.CurrentBlock().Hash() || reopened.CurrentFinalBlock().Hash() != original.CurrentFinalBlock().Hash() {
		t.Fatal("checkpoint-only authority changed")
	}
}

func TestIndexedSnapMissingIndexUsesLegacyExecution(t *testing.T) {
	reopened, original, guard := indexedSnapRestart(t, "missing-index")
	if err := reopened.RestoreSnapPivot(); err == nil {
		t.Fatal("absent indexed state skipped legacy execution")
	}
	guard.rejectReexecution = false
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if reopened.CurrentBlock().Hash() != original.CurrentBlock().Hash() || reopened.State().GetRoot() != original.State().GetRoot() {
		t.Fatal("legacy recovery lost original validated root")
	}
}
