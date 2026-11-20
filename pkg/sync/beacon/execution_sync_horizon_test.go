package beacon

import (
	"context"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
)

// This is a sync-coordinator counterexample using the repository's existing
// hash-linked fixture chain. Execution validation is not reached on failure;
// this does not substitute for a real 65k-block consensus fixture campaign.
func TestFullSyncBeyondAncestryWindowMakesProgress(t *testing.T) {
	const height = 65537
	blocks := make([]*types.Block, height+1)
	blocks[0] = types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	source := map[types.Hash]*types.Block{blocks[0].Hash(): blocks[0]}
	for i := 1; i <= height; i++ {
		blocks[i] = types.NewBlock(&types.Header{Number: big.NewInt(int64(i)), ParentHash: blocks[i-1].Hash(), Time: uint64(i)}, nil)
		source[blocks[i].Hash()] = blocks[i]
	}
	chain := &executionTestChain{blocks: map[types.Hash]*types.Block{blocks[0].Hash(): blocks[0]}}
	syncer := NewExecutionSyncer(chain, executionTestFetcher(func(_ context.Context, hash types.Hash) (*types.Block, error) { return source[hash], nil }), nil)
	defer syncer.Stop()
	if err := syncer.synchronize(context.Background(), ExecutionTarget{Head: blocks[height].Hash()}); err != nil {
		t.Fatal(err)
	}
	if chain.GetBlock(blocks[height].Hash()) == nil || len(chain.blocks) != height+1 {
		t.Fatalf("no complete validated progress: %d", len(chain.blocks))
	}
	if len(syncer.ancestry.files) != 0 {
		t.Fatal("completed ancestry staging was not released")
	}
	if syncer.downloadHeaders.Len() > maxExecutionAncestry || syncer.downloadBodies.Size() > executionBodyCacheBytes {
		t.Fatal("original memory budgets exceeded")
	}
}

func TestLargeMovingHeadsResumeSpoolWithBoundedMetadata(t *testing.T) {
	const initial = 65537
	const changes = 96
	blocks := make([]*types.Block, initial+changes+1)
	blocks[0] = types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	source := map[types.Hash]*types.Block{blocks[0].Hash(): blocks[0]}
	for i := 1; i < len(blocks); i++ {
		blocks[i] = types.NewBlock(&types.Header{ParentHash: blocks[i-1].Hash(), Number: big.NewInt(int64(i)), Time: uint64(i)}, nil)
		source[blocks[i].Hash()] = blocks[i]
	}
	var updates, fetches atomic.Uint64
	chain := &movingProgressChain{executionTestChain: executionTestChain{blocks: map[types.Hash]*types.Block{blocks[0].Hash(): blocks[0]}}, updates: &updates, first: make(chan uint64, 1)}
	var syncer *ExecutionSyncer
	applied := make(chan ExecutionTarget, 1)
	syncer = NewExecutionSyncer(chain, executionTestFetcher(func(ctx context.Context, hash types.Hash) (*types.Block, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, p := range syncer.ancestry.paths {
			if len(p.segments) > retainedAncestrySegments {
				t.Error("retained segment count is unbounded")
			}
		}
		if len(syncer.ancestry.files) > 3*(retainedAncestrySegments+2) {
			t.Error("staging descriptor count is unbounded")
		}
		count := fetches.Add(1)
		if count%1024 == 0 && updates.Load() < changes {
			u := updates.Add(1)
			syncer.Request(ExecutionTarget{Head: blocks[initial+int(u)].Hash()})
		}
		return source[hash], nil
	}), func(target ExecutionTarget) error { applied <- target; return nil }, t.TempDir())
	syncer.Start()
	defer syncer.Stop()
	syncer.Request(ExecutionTarget{Head: blocks[initial].Hash()})
	select {
	case count := <-chain.first:
		if count >= changes {
			t.Fatal("all target churn had to stop before any execution progress")
		}
	case <-time.After(45 * time.Second):
		t.Fatal("large moving target never made progress")
	}
	select {
	case target := <-applied:
		if chain.GetBlock(target.Head) == nil {
			t.Fatal("target published without execution")
		}
	case <-time.After(45 * time.Second):
		t.Fatal("latest target not applied")
	}
	syncer.Stop()
	if len(syncer.ancestry.files) != 0 {
		t.Fatal("staging handles survived Stop")
	}
	if fetches.Load() > 3*uint64(len(blocks)) {
		t.Fatalf("completed ranges repeatedly transferred: %d", fetches.Load())
	}
}

func TestSpoolFailureAndCorruptionDoNotPublishOrValidate(t *testing.T) {
	genesis := types.NewBlock(&types.Header{Number: new(big.Int)}, nil)
	blocks := []*types.Block{genesis}
	source := map[types.Hash]*types.Block{}
	for i := 1; i <= 20; i++ {
		b := types.NewBlock(&types.Header{ParentHash: blocks[i-1].Hash(), Number: big.NewInt(int64(i)), Time: uint64(i)}, nil)
		blocks = append(blocks, b)
		source[b.Hash()] = b
	}
	for _, fault := range []string{"create", "corrupt-number", "corrupt-hash", "corrupt-parent"} {
		t.Run(fault, func(t *testing.T) {
			chain := &executionTestChain{blocks: map[types.Hash]*types.Block{genesis.Hash(): genesis}}
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			syncer := NewExecutionSyncer(chain, executionTestFetcher(func(_ context.Context, hash types.Hash) (*types.Block, error) {
				calls++
				if fault != "create" && calls == 5 {
					cancel()
				}
				return source[hash], nil
			}), nil, t.TempDir())
			defer syncer.Stop()
			target := ExecutionTarget{Head: blocks[20].Hash()}
			if fault == "create" {
				bad := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(bad, []byte{1}, 0600); err != nil {
					t.Fatal(err)
				}
				syncer.ancestry.dir = bad
			}
			if err := syncer.synchronize(ctx, target); err == nil {
				t.Fatal("fault unexpectedly succeeded")
			}
			if len(chain.blocks) != 1 {
				t.Fatal("failed staging validated a block")
			}
			if fault != "create" {
				if len(syncer.ancestry.paths) == 0 {
					t.Fatal("cancelled progress not retained")
				}
				seg := syncer.ancestry.paths[0].segments[0]
				if _, err := seg.file.WriteAt([]byte{0xff}, int64(seg.start*ancestryRecordBytes+map[string]uint64{"corrupt-number": 64, "corrupt-hash": 0, "corrupt-parent": 32}[fault])); err != nil {
					t.Fatal(err)
				}
				if err := syncer.synchronize(context.Background(), target); err == nil || errors.Is(err, context.Canceled) {
					t.Fatalf("corrupt metadata accepted: %v", err)
				}
				if chain.GetBlock(target.Head) != nil || len(syncer.ancestry.files) != 0 {
					t.Fatal("corrupt staged head was validated or staging survived cleanup")
				}
			}
			syncer.ancestry.dir = t.TempDir()
			if err := syncer.synchronize(context.Background(), target); err != nil {
				t.Fatal(err)
			}
			if chain.GetBlock(target.Head) == nil {
				t.Fatal("clean retry did not recover")
			}
		})
	}
}
