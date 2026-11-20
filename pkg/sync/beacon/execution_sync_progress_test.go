package beacon

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/lru"
)

type movingProgressChain struct {
	executionTestChain
	updates *atomic.Uint64
	first   chan uint64
	once    sync.Once
}

func (c *movingProgressChain) InsertBlock(block *types.Block) error {
	if err := c.executionTestChain.InsertBlock(block); err != nil {
		return err
	}
	c.once.Do(func() { c.first <- c.updates.Load() })
	return nil
}

// The source changes the nominated head after every two completed downloads.
// This models an ancestry walk longer than a slot without relying on scheduler
// sleeps: replacement remains immediate, but completed hash-bound data must help
// the next worker rather than being fetched again from scratch.
func TestMovingTargetsRetainCompletedDownloadsBeforeHeadChurnEnds(t *testing.T) {
	const initial = 128
	const changes = 384
	blocks := make([]*types.Block, initial+changes+1)
	blocks[0] = types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	byHash := map[types.Hash]*types.Block{blocks[0].Hash(): blocks[0]}
	for i := 1; i < len(blocks); i++ {
		blocks[i] = types.NewBlock(&types.Header{Number: big.NewInt(int64(i)), ParentHash: blocks[i-1].Hash(), Time: uint64(i)}, nil)
		byHash[blocks[i].Hash()] = blocks[i]
	}
	var updates, fetches atomic.Uint64
	chain := &movingProgressChain{executionTestChain: executionTestChain{blocks: map[types.Hash]*types.Block{blocks[0].Hash(): blocks[0]}}, updates: &updates, first: make(chan uint64, 1)}
	var syncer *ExecutionSyncer
	fetcher := executionTestFetcher(func(ctx context.Context, hash types.Hash) (*types.Block, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		block := byHash[hash]
		if block == nil {
			t.Errorf("unknown requested hash %s", hash)
			return nil, nil
		}
		count := fetches.Add(1)
		if count%2 == 0 && updates.Load() < changes {
			update := updates.Add(1)
			syncer.Request(ExecutionTarget{Head: blocks[initial+int(update)].Hash()})
		}
		return block, nil
	})
	applied := make(chan ExecutionTarget, 1)
	syncer = NewExecutionSyncer(chain, fetcher, func(target ExecutionTarget) error {
		// The syncer invokes publication under its target lock.
		if target != syncer.target {
			t.Error("stale target published")
		}
		applied <- target
		return nil
	})
	syncer.Start()
	defer syncer.Stop()
	syncer.Request(ExecutionTarget{Head: blocks[initial].Hash()})
	select {
	case count := <-chain.first:
		if count >= changes {
			t.Fatalf("no validated progress until all %d head updates stopped; downloads=%d", count, fetches.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("continuous target updates prevented all execution progress")
	}
	select {
	case <-applied:
	case <-time.After(5 * time.Second):
		t.Fatal("latest nominated target was not published")
	}
	if fetches.Load() > 2*(initial+changes) {
		t.Fatalf("completed ancestry was repeatedly downloaded: %d", fetches.Load())
	}
}

func TestExecutionDownloadBoundsAndUnvalidatedDataStaySeparate(t *testing.T) {
	genesis := types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	source := make(map[types.Hash]*types.Block)
	parent := genesis
	for i := 1; i <= 20; i++ {
		block := types.NewBlock(&types.Header{ParentHash: parent.Hash(), Number: big.NewInt(int64(i)), Time: uint64(i)}, nil)
		source[block.Hash()] = block
		parent = block
	}
	chain := &executionTestChain{blocks: map[types.Hash]*types.Block{genesis.Hash(): genesis}}
	fetcher := executionTestFetcher(func(_ context.Context, hash types.Hash) (*types.Block, error) {
		if block := source[hash]; block != nil {
			return block, nil
		}
		return nil, fmt.Errorf("missing test block")
	})
	syncer := NewExecutionSyncer(chain, fetcher, nil)
	syncer.downloadHeaders = lru.NewCache[types.Hash, downloadedHeader](4)
	syncer.downloadBodyLimit = 1024
	syncer.downloadBodies = lru.NewSizeConstrainedCacheWithKeySize[types.Hash, []byte](syncer.downloadBodyLimit, func(types.Hash) uint64 { return 32 })
	for hash := range source {
		if _, err := syncer.downloadedHeader(context.Background(), hash); err != nil {
			t.Fatal(err)
		}
		if syncer.downloadHeaders.Len() > 4 || syncer.downloadBodies.Size() > syncer.downloadBodyLimit {
			t.Fatal("download resource bound exceeded")
		}
		if syncer.validated(hash) || chain.GetBlock(hash) != nil {
			t.Fatal("downloaded data acquired execution-validation authority")
		}
	}
	// Even a single oversized body cannot exceed the upstream blob-cache bound.
	large := types.NewBlock(&types.Header{ParentHash: genesis.Hash(), Number: big.NewInt(1), Time: 100}, &types.Body{Transactions: []*types.Transaction{types.NewTransaction(&types.LegacyTx{Gas: 21000, GasPrice: new(big.Int), Value: new(big.Int), Data: make([]byte, 2048), V: new(big.Int), R: new(big.Int), S: new(big.Int)})}})
	source[large.Hash()] = large
	if _, err := syncer.downloadedBlock(context.Background(), large.Hash()); err != nil {
		t.Fatal(err)
	}
	if syncer.downloadBodies.Size() > syncer.downloadBodyLimit {
		t.Fatal("oversized single-body exception escaped the strict budget")
	}
	if _, cached := syncer.downloadBodies.Get(large.Hash()); cached {
		t.Fatal("oversized block body retained")
	}
}

func TestEvictedExecutionBodiesRefetchWithoutLosingHeaderProgress(t *testing.T) {
	genesis := types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	first := types.NewBlock(&types.Header{Number: big.NewInt(1), ParentHash: genesis.Hash()}, nil)
	second := types.NewBlock(&types.Header{Number: big.NewInt(2), ParentHash: first.Hash()}, nil)
	chain := &executionTestChain{blocks: map[types.Hash]*types.Block{genesis.Hash(): genesis}}
	var calls atomic.Uint64
	source := map[types.Hash]*types.Block{first.Hash(): first, second.Hash(): second}
	syncer := NewExecutionSyncer(chain, executionTestFetcher(func(_ context.Context, hash types.Hash) (*types.Block, error) { calls.Add(1); return source[hash], nil }), nil)
	encoded, _ := first.EncodeRLP()
	syncer.downloadBodyLimit = uint64(len(encoded)) + 32
	syncer.downloadBodies = lru.NewSizeConstrainedCacheWithKeySize[types.Hash, []byte](syncer.downloadBodyLimit, func(types.Hash) uint64 { return 32 })
	if _, err := syncer.downloadedHeader(context.Background(), first.Hash()); err != nil {
		t.Fatal(err)
	}
	if _, err := syncer.downloadedHeader(context.Background(), second.Hash()); err != nil {
		t.Fatal(err)
	}
	before := calls.Load()
	if _, err := syncer.downloadedHeader(context.Background(), first.Hash()); err != nil || calls.Load() != before {
		t.Fatalf("header cursor lost with evicted body: %v", err)
	}
	if err := syncer.synchronize(context.Background(), ExecutionTarget{Head: second.Hash()}); err != nil {
		t.Fatal(err)
	}
	if chain.GetBlock(second.Hash()) == nil || calls.Load() <= before {
		t.Fatal("forward execution did not refetch evicted bodies and validate the chain")
	}
}

func TestCompletedExecutionBodyRemainsUsableWhenSourceDisappears(t *testing.T) {
	genesis := types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	block := types.NewBlock(&types.Header{ParentHash: genesis.Hash(), Number: big.NewInt(1)}, nil)
	chain := &executionTestChain{blocks: map[types.Hash]*types.Block{genesis.Hash(): genesis}}
	var calls atomic.Int32
	syncer := NewExecutionSyncer(chain, executionTestFetcher(func(_ context.Context, hash types.Hash) (*types.Block, error) {
		if calls.Add(1) > 1 {
			return nil, fmt.Errorf("source is no longer available")
		}
		return block, nil
	}), nil)
	first, err := syncer.downloadedBlock(context.Background(), block.Hash())
	if err != nil {
		t.Fatal(err)
	}
	second, err := syncer.downloadedBlock(context.Background(), block.Hash())
	if err != nil {
		t.Fatal(err)
	}
	firstRLP, _ := first.EncodeRLP()
	secondRLP, _ := second.EncodeRLP()
	if string(firstRLP) != string(secondRLP) || second.Hash() != block.Hash() || calls.Load() != 1 {
		t.Fatal("completed body was not reused with its exact original commitment")
	}
	if syncer.validated(block.Hash()) {
		t.Fatal("body availability was confused with execution validity")
	}
}
