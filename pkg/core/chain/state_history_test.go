package chain

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

type historyReadProbe struct {
	rawdb.Database
	headerReads int
}

func TestHistoryBatchFailureDoesNotPublishRootOrValidationAuthority(t *testing.T) {
	source := newBuilderRequestTestState()
	genesis := makeGenesis(30_000_000, big.NewInt(1))
	db := &overlayFailDB{Database: rawdb.NewMemoryDB()}
	chain, err := NewBlockchain(config.TestConfig, genesis, source, db)
	if err != nil {
		t.Fatal(err)
	}
	child := makeBlockWithState(genesis, nil, source)
	db.fail = true
	if err := chain.InsertBlock(child); err == nil {
		t.Fatal("accepted failed atomic block write")
	}
	if available, err := state.HistoryStateAvailable(db, child.Root()); err != nil || available {
		t.Fatalf("failed batch published a historical root: %t %v", available, err)
	}
	if chain.HasValidatedBlock(child.Hash()) || chain.CurrentBlock().Hash() != genesis.Hash() {
		t.Fatal("failed history publication advanced validation or canonical authority")
	}
	db.fail = false
	if err := chain.InsertBlock(child); err != nil {
		t.Fatal(err)
	}
	if !chain.HasValidatedBlock(child.Hash()) {
		t.Fatal("retry lost validation marker")
	}
	if _, err := state.OpenTrieHistory(context.Background(), db, child.Root()); err != nil {
		t.Fatalf("atomic retry did not publish all required nodes: %v", err)
	}
}

func (p *historyReadProbe) Get(key []byte) ([]byte, error) {
	if len(key) > 1 && key[0] == 'h' {
		p.headerReads++
	}
	return p.Database.Get(key)
}
func newHistoryChain(t *testing.T) (*Blockchain, *historyReadProbe, []*types.Block) {
	t.Helper()
	source := newBuilderRequestTestState()
	genesis := makeGenesis(30_000_000, big.NewInt(1))
	probe := &historyReadProbe{Database: rawdb.NewMemoryDB()}
	chain, err := NewBlockchain(config.TestConfig, genesis, source, probe, BlockchainOpts{BlockCacheSize: 1, StateCacheSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	var blocks []*types.Block
	parent := genesis
	for i := 0; i < 6; i++ {
		block := makeBlockWithState(parent, nil, source)
		if err := chain.InsertBlock(block); err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, block)
		parent = block
	}
	return chain, probe, blocks
}
func TestStoredMerkleHistoryAvoidsAncestorReplayAfterEviction(t *testing.T) {
	chain, probe, blocks := newHistoryChain(t)
	target := blocks[1]
	chain.sc = newStateCache(1)
	chain.memSC = newStateCache(1)
	probe.headerReads = 0
	state, err := chain.StateAtBlockContext(context.Background(), target)
	if err != nil || state.GetRoot() != target.Root() {
		t.Fatalf("cold exact state: %v", err)
	}
	if probe.headerReads != 0 {
		t.Fatalf("historical root was re-executed instead of opened: %d ancestor reads", probe.headerReads)
	}
	before := chain.CurrentBlock().Hash()
	state.AddBalance(types.Address{19: 0x77}, big.NewInt(1))
	chain.sc = newStateCache(1)
	chain.memSC = newStateCache(1)
	again, err := chain.StateAtBlock(target)
	if err != nil || again.GetRoot() != target.Root() || chain.CurrentBlock().Hash() != before {
		t.Fatalf("historical overlay changed authoritative state: %v", err)
	}
}
func TestExclusiveHistoryBuildUpgradesLegacyIndexAndIsIdempotent(t *testing.T) {
	chain, probe, blocks := newHistoryChain(t)
	var keys [][]byte
	iterator := probe.NewIterator([]byte("mpt-history-root/"))
	for iterator.Next() {
		keys = append(keys, bytes.Clone(iterator.Key()))
	}
	iterator.Release()
	for _, key := range keys {
		if err := probe.Delete(key); err != nil {
			t.Fatal(err)
		}
	}
	before := chain.CurrentBlock().Hash()
	result, err := chain.BuildTrieHistory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Head != before || result.Number != 6 || result.Indexed == 0 || chain.CurrentBlock().Hash() != before {
		t.Fatalf("migration selected/changed head: %+v", result)
	}
	chain.sc = newStateCache(1)
	chain.memSC = newStateCache(1)
	restored, err := chain.StateAtBlock(blocks[2])
	if err != nil || restored.GetRoot() != blocks[2].Root() {
		t.Fatalf("migrated history unavailable: %v", err)
	}
	result, err = chain.BuildTrieHistory(context.Background())
	if err != nil || result.Indexed != 0 || result.Existing != 6 {
		t.Fatalf("history build not idempotent: %+v %v", result, err)
	}
}
func TestHistoryPreparationQueueCancelsWithoutBlockingCachedState(t *testing.T) {
	chain, _, blocks := newHistoryChain(t)
	head := blocks[len(blocks)-1]
	chain.queryStateSlots <- struct{}{}
	chain.queryStateSlots <- struct{}{}
	defer func() { <-chain.queryStateSlots; <-chain.queryStateSlots }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := chain.StateAtBlockContext(ctx, head); err != nil {
		t.Fatalf("cached state was queued behind heavy preparation: %v", err)
	}
	chain.sc = newStateCache(1)
	chain.memSC = newStateCache(1)
	if _, err := chain.StateAtBlockContext(ctx, blocks[0]); err != context.DeadlineExceeded {
		t.Fatalf("queued historical preparation did not respect deadline: %v", err)
	}
}
