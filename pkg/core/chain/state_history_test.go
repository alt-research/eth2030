package chain

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
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
	source.SetCode(eips.HistoryStorageAddress, eips.HistoryStorageRuntime())
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
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Root() == blocks[i-1].Root() {
			t.Fatal("history fixture did not execute distinct EIP-2935 storage transitions")
		}
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

type interruptedHistoryDB struct {
	rawdb.Database
	cancel context.CancelFunc
}
type interruptedHistoryBatch struct {
	rawdb.Batch
	cancel context.CancelFunc
}

func (d interruptedHistoryDB) NewBatch() rawdb.Batch {
	return interruptedHistoryBatch{Batch: d.Database.NewBatch(), cancel: d.cancel}
}
func (b interruptedHistoryBatch) Write() error {
	err := b.Batch.Write()
	if err == nil {
		b.cancel()
	}
	return err
}

func TestOfflineHistoryUsesDurableHeadWithoutStartupReplayAndResumesCancellation(t *testing.T) {
	original, probe, blocks := newHistoryChain(t)
	iterator := probe.NewIterator([]byte("mpt-history-root/"))
	var keys [][]byte
	for iterator.Next() {
		keys = append(keys, bytes.Clone(iterator.Key()))
	}
	iterator.Release()
	for _, key := range keys {
		if err := probe.Delete(key); err != nil {
			t.Fatal(err)
		}
	}
	// The maintenance chain intentionally has only genesis in memory. Replaying
	// Node startup here would do the expensive work before cancellation can act.
	offline, err := NewBlockchain(config.TestConfig, original.genesis, original.execGenesisState.Copy(), probe)
	if err != nil {
		t.Fatal(err)
	}
	if offline.CurrentBlock().Hash() != original.genesis.Hash() {
		t.Fatal("constructor unexpectedly replayed history")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	offline.db = interruptedHistoryDB{Database: probe, cancel: cancel}
	partial, err := offline.BuildTrieHistory(ctx)
	if err != context.Canceled || partial.Indexed != 1 || partial.Number != 6 {
		t.Fatalf("interrupted migration ignored durable target/cancellation: %+v %v", partial, err)
	}
	if available, err := state.HistoryStateAvailable(probe, blocks[0].Root()); err != nil || !available {
		t.Fatalf("completed prefix lost on cancellation: %t %v", available, err)
	}
	offline.db = probe
	resumed, err := offline.BuildTrieHistory(context.Background())
	if err != nil || resumed.Indexed != 5 || resumed.Existing != 1 || resumed.Head != blocks[5].Hash() {
		t.Fatalf("migration did not resume exact durable path: %+v %v", resumed, err)
	}
	storedHead, err := rawdb.ReadHeadBlockHash(probe)
	if err != nil || storedHead != original.CurrentBlock().Hash() || offline.CurrentBlock().Hash() != original.genesis.Hash() {
		t.Fatalf("maintenance selected or rewrote canonical authority: %v", err)
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
