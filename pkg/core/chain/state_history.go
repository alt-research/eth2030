package chain

import (
	"context"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

type StateHistoryResult struct {
	Head     types.Hash `json:"head"`
	Number   uint64     `json:"number"`
	Indexed  uint64     `json:"indexed"`
	Existing uint64     `json:"existing"`
}

// BuildTrieHistory upgrades local validated MPT history without selecting a new
// head or trusting network data. The caller must open the database exclusively
// and leave networking/RPC stopped, as the standalone maintenance command does.
func (bc *Blockchain) BuildTrieHistory(ctx context.Context) (StateHistoryResult, error) {
	if err := ctx.Err(); err != nil {
		return StateHistoryResult{}, err
	}
	if bc.config != nil && bc.config.BinaryOverlayTime != nil {
		return StateHistoryResult{}, fmt.Errorf("binary overlay history uses its existing checkpoint format")
	}
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	headHash, err := rawdb.ReadHeadBlockHash(bc.db)
	if err != nil {
		return StateHistoryResult{}, storageFailure(err)
	}
	head := bc.GetBlock(headHash)
	if head == nil || head.Hash() != headHash {
		return StateHistoryResult{}, fmt.Errorf("state history: missing durable head")
	}
	result := StateHistoryResult{Head: head.Hash(), Number: head.NumberU64()}
	var base state.StateDB
	parent := bc.genesis
	for number := uint64(1); number <= result.Number; number++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		hash, err := rawdb.ReadCanonicalHash(bc.db, number)
		if err != nil {
			return result, storageFailure(err)
		}
		block := bc.GetBlock(hash)
		if block == nil || block.Hash() != hash || block.NumberU64() != number || block.ParentHash() != parent.Hash() || !bc.HasValidatedBlock(hash) {
			return result, fmt.Errorf("state history: incomplete validated canonical path at %d", number)
		}
		available, err := state.HistoryStateAvailable(bc.db, block.Root())
		if err != nil {
			return result, storageFailure(err)
		}
		if available {
			base = nil
			result.Existing++
		} else {
			if base == nil {
				if parent.Hash() == bc.genesis.Hash() {
					base = bc.execGenesisState.Copy()
				} else {
					base, err = state.OpenTrieHistory(ctx, bc.db, parent.Root())
					if err != nil {
						return result, storageFailure(err)
					}
				}
			}
			next, _, err := bc.executeStateContext(ctx, block, base, bc.BranchHashFn(parent.Header()))
			if err != nil {
				return result, err
			}
			if next.GetRoot() != block.Root() {
				return result, fmt.Errorf("state history: replay root mismatch at %d", number)
			}
			if err := ctx.Err(); err != nil {
				return result, err
			}
			batch := bc.db.NewBatch()
			if err := state.StoreTrieHistory(bc.db, batch, block.Root(), next); err != nil {
				return result, storageFailure(err)
			}
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if err := batch.Write(); err != nil {
				return result, storageFailure(err)
			}
			base = next
			result.Indexed++
		}
		parent = block
	}
	finalHead, err := rawdb.ReadHeadBlockHash(bc.db)
	if err != nil {
		return result, storageFailure(err)
	}
	if finalHead != result.Head {
		return result, fmt.Errorf("state history: canonical head changed during exclusive maintenance")
	}

	if result.Number > 0 {
		if err := state.AuditTrieHistory(ctx, bc.db, head.Root()); err != nil {
			return result, storageFailure(err)
		}
	}
	return result, nil
}
