package chain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

type restartWriteGuard struct {
	rawdb.Database
	rejectReexecution bool
}
type restartGuardBatch struct {
	rawdb.Batch
	rejectReexecution bool
}

func (d *restartWriteGuard) NewBatch() rawdb.Batch {
	return restartGuardBatch{d.Database.NewBatch(), d.rejectReexecution}
}
func (b restartGuardBatch) Put(key, value []byte) error {
	if b.rejectReexecution && bytes.HasPrefix(key, []byte("validated-block/")) {
		return errors.New("re-executed already validated block during indexed recovery")
	}
	return b.Batch.Put(key, value)
}

func restartFixture(t *testing.T, mutate func(rawdb.Database, []*types.Block)) (*Blockchain, *restartWriteGuard, []*types.Block) {
	t.Helper()
	original, probe, blocks := newHistoryChain(t)
	directory := t.TempDir()
	db, err := rawdb.NewFileDB(directory)
	if err != nil {
		t.Fatal(err)
	}
	batch := db.NewBatch()
	iterator := probe.NewIterator(nil)
	for iterator.Next() {
		if err := batch.Put(iterator.Key(), iterator.Value()); err != nil {
			t.Fatal(err)
		}
	}
	iterator.Release()
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteFinalizedBlockHash(db, blocks[1].Hash()); err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(db, blocks)
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
	chain, err := NewBlockchain(config.TestConfig, original.genesis, original.execGenesisState.Copy(), guard, BlockchainOpts{BlockCacheSize: 2, StateCacheSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	return chain, guard, blocks
}

func TestIndexedFileDBRestartAuthenticatesOriginalStateWithoutReexecution(t *testing.T) {
	chain, guard, blocks := restartFixture(t, nil)
	if err := chain.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	head := blocks[len(blocks)-1]
	if chain.CurrentBlock().Hash() != head.Hash() || chain.State().GetRoot() != head.Root() {
		t.Fatal("recovery did not retain original committed head/state")
	}
	if chain.CurrentFinalBlock().Hash() != blocks[1].Hash() || chain.CurrentSafeBlock().Hash() != blocks[1].Hash() {
		t.Fatal("recovery changed original finality")
	}
	if len(chain.blockCache) > 2 || len(chain.blockCacheOrder) != len(chain.blockCache) || chain.blockCache[head.Hash()] == nil {
		t.Fatal("backwards recovery lost bounded chronological body cache")
	}
	for _, blk := range blocks {
		historical, err := chain.StateAtBlockContext(context.Background(), blk)
		if err != nil || historical.GetRoot() != blk.Root() {
			t.Fatalf("exact historical state changed at %d: %v", blk.NumberU64(), err)
		}
	}
	// New blocks still execute and publish through the ordinary validation path.
	guard.rejectReexecution = false
	child := makeBlockWithState(head, nil, chain.State())
	if err := chain.InsertBlock(child); err != nil {
		t.Fatalf("ordinary post-restart execution: %v", err)
	}
	if chain.State().GetRoot() != child.Root() || !chain.HasValidatedBlock(child.Hash()) {
		t.Fatal("post-restart block lost execution validation")
	}
	if _, err := state.OpenTrieHistory(context.Background(), guard, child.Root()); err != nil {
		t.Fatalf("new original root not published: %v", err)
	}
}

func TestIndexedRestartRejectsCorruptPublishedStateAndLineage(t *testing.T) {
	for _, fault := range []string{"state-node", "body", "canonical-ancestor", "validation-marker", "BAL"} {
		t.Run(fault, func(t *testing.T) {
			chain, _, _ := restartFixture(t, func(db rawdb.Database, blocks []*types.Block) {
				var err error
				switch fault {
				case "state-node":
					err = db.Put(append([]byte("mpt-history-node/"), blocks[5].Root().Bytes()...), []byte{0xc0})
				case "body":
					err = rawdb.WriteBody(db, blocks[3].NumberU64(), blocks[3].Hash(), []byte{0xc0})
				case "canonical-ancestor":
					err = rawdb.WriteCanonicalHash(db, blocks[2].NumberU64(), blocks[1].Hash())
				case "validation-marker":
					err = db.Put(validatedKey(blocks[3].Hash()), []byte{2})
				case "BAL":
					err = db.Put(blockAccessListKey(blocks[3].Hash()), []byte{0xff})
				}
				if err != nil {
					t.Fatal(err)
				}
			})
			if err := chain.RestoreSnapPivot(); err == nil {
				t.Fatalf("accepted corrupt %s", fault)
			}
			if chain.CurrentBlock().Hash() != chain.genesis.Hash() {
				t.Fatal("corrupt recovery advanced the in-memory head")
			}
		})
	}
}

func TestMissingValidationAuthorityUsesAuthenticatedLegacyReplay(t *testing.T) {
	chain, guard, blocks := restartFixture(t, func(db rawdb.Database, blocks []*types.Block) {
		if err := db.Delete(validatedKey(blocks[2].Hash())); err != nil {
			t.Fatal(err)
		}
	})
	// The guarded run must refuse fast adoption and attempt normal execution.
	if err := chain.RestoreSnapPivot(); err == nil {
		t.Fatal("indexed state alone replaced missing execution validation")
	}
	guard.rejectReexecution = false
	if err := chain.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if !chain.HasValidatedBlock(blocks[2].Hash()) || chain.CurrentBlock().Hash() != blocks[5].Hash() {
		t.Fatal("legacy execution did not restore original validation/head")
	}
}

func TestPrunedHistoricalBALDoesNotInvalidateVerifiedStateRecovery(t *testing.T) {
	chain, _, blocks := restartFixture(t, func(db rawdb.Database, blocks []*types.Block) {
		if err := db.Delete(blockAccessListKey(blocks[2].Hash())); err != nil {
			t.Fatal(err)
		}
	})
	if err := chain.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if bal, err := chain.BlockAccessList(blocks[2].Hash()); err != nil || bal != nil {
		t.Fatalf("pruned BAL was invented: %x %v", bal, err)
	}
	if chain.CurrentBlock().Hash() != blocks[5].Hash() {
		t.Fatal(fmt.Errorf("pruned auxiliary data changed selected head"))
	}
}
