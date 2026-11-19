package chain

import (
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

// CanIgnoreStaleForkchoice implements the Engine Paris ancestor-skip option for
// recovering CLs whose canonical notification predates durable EL finality.
// It only reads a serialized snapshot; it never lowers a checkpoint or head.
func (bc *Blockchain) CanIgnoreStaleForkchoice(headHash, safeHash, finalHash types.Hash) (bool, error) {
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	head, current, prior := bc.GetBlock(headHash), bc.CurrentBlock(), bc.CurrentFinalBlock()
	if head == nil || current == nil || prior == nil || !bc.HasValidatedBlock(headHash) || head.NumberU64() >= current.NumberU64() {
		return false, nil
	}
	canonical, err := rawdb.ReadCanonicalHash(bc.db, head.NumberU64())
	if err != nil || canonical != headHash {
		return false, nil
	}
	var final, safe *types.Block
	if finalHash != (types.Hash{}) {
		final = bc.GetBlock(finalHash)
	}
	stale := head.NumberU64() < prior.NumberU64() || final != nil && final.NumberU64() < prior.NumberU64()
	if !stale {
		return false, nil
	}
	for _, checkpoint := range []struct {
		name string
		hash types.Hash
	}{{"safe", safeHash}, {"finalized", finalHash}} {
		if checkpoint.hash == (types.Hash{}) {
			continue
		}
		block := bc.GetBlock(checkpoint.hash)
		if block == nil || !bc.HasValidatedBlock(checkpoint.hash) {
			return false, fmt.Errorf("forkchoice: unknown %s block", checkpoint.name)
		}
		canonical, err := rawdb.ReadCanonicalHash(bc.db, block.NumberU64())
		if err != nil || canonical != checkpoint.hash || block.NumberU64() > head.NumberU64() {
			return false, fmt.Errorf("forkchoice: %s checkpoint is not a nominated head ancestor", checkpoint.name)
		}
		if checkpoint.name == "safe" {
			safe = block
		}
	}
	if safe != nil && final != nil && safe.NumberU64() < final.NumberU64() {
		return false, fmt.Errorf("forkchoice: safe block precedes finalized block")
	}
	return true, nil
}
