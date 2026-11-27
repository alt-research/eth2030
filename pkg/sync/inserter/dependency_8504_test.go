package inserter

import (
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

type dependencyExecutor8504 struct{ calls int }

func (e *dependencyExecutor8504) ExecuteBlock(b *types.Block) (types.Hash, []*types.Receipt, error) {
	e.calls++
	return b.Root(), nil, nil
}

// A delegated state-root executor cannot stand in for the independent witness
// predicate. Verify it before invoking either the executor or chain writer.
func TestDependency8504SyncPipelinesCheckWitnessesBeforeDelegation(t *testing.T) {
	for _, path := range []string{"processor", "chain", "batch"} {
		for _, kind := range []string{"valid", "missing", "tampered", "extra", "bare", "active-empty", "gas-bound"} {
			genesis := bpTestBlock(0, types.Hash{})
			block := proof8504.Block()
			h, body := block.Header(), block.Body()
			h.ParentHash = genesis.Hash()
			switch kind {
			case "missing":
				body.DependencyWitnesses = nil
			case "tampered":
				body.DependencyWitnesses[0][0] ^= 1
				hash := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
				h.DependencyWitnessesHash = &hash
			case "extra":
				body.DependencyWitnesses = append(body.DependencyWitnesses, body.DependencyWitnesses[0])
				hash := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
				h.DependencyWitnessesHash = &hash
			case "bare":
				h.DependencyWitnessesHash, body.DependencyWitnesses = nil, nil
			case "active-empty":
				body.Transactions, body.DependencyWitnesses = nil, [][]byte{}
				hash := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
				h.DependencyWitnessesHash = &hash
			case "gas-bound":
				h.GasLimit = 1
			}
			block = types.NewBlock(h, body)
			writer := newTestBlockInserter(genesis)
			executor := new(dependencyExecutor8504)
			var count int
			var err error
			if path == "processor" {
				p := NewBlockProcessor(DefaultBlockProcessorConfig(), writer)
				p.SetNextExpected(1)
				p.SetExecutor(executor)
				if err := p.Enqueue(block, nil); err != nil {
					t.Fatal(err)
				}
				count, err = p.ProcessReady()
			} else {
				p := NewChainInserter(DefaultChainInserterConfig(), writer)
				p.SetExecutor(executor)
				if path == "chain" {
					count, err = p.InsertBlocks([]*types.Block{block})
				} else {
					count, err = p.InsertBatch([]*types.Block{block})
				}
			}
			valid := kind == "valid" || kind == "active-empty"
			if valid {
				if err != nil || count != 1 || executor.calls != 1 || len(writer.blocks) != 2 {
					t.Fatal(path, kind, "valid declared witness predicate rejected", count, executor.calls, err)
				}
			} else if err == nil || count != 0 || executor.calls != 0 || len(writer.blocks) != 1 {
				t.Fatal(path, kind, "invalid dependencies reached delegated execution or storage", count, executor.calls, err)
			}
		}
	}
}
