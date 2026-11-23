package chain

import (
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

// InsertSyncedBlock validates and stores a downloaded block without publishing
// it as canonical. Only the completed CL target is selected by ApplyForkchoice.
func (bc *Blockchain) InsertSyncedBlock(block *types.Block) error {
	_, err := bc.InsertSyncedBlockWithResult(block)
	return err
}

func (bc *Blockchain) InsertSyncedBlockWithResult(block *types.Block) (*execution.ProcessResult, error) {
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	var result *execution.ProcessResult
	err := bc.insertBlock(block, &result, true)
	return result, err
}

// ApplyForkchoice atomically publishes a fully validated CL-nominated head,
// safe block and finalized block (EIP-3675; Engine Paris forkchoice rules).
func (bc *Blockchain) ApplyForkchoice(headHash, safeHash, finalHash types.Hash) error {
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	if err := bc.CheckStorage(); err != nil {
		return err
	}
	head := bc.GetBlock(headHash)
	if head == nil || !bc.HasValidatedBlock(headHash) {
		return fmt.Errorf("%w: forkchoice head not validated", ErrStateNotFound)
	}
	var safe, final *types.Block
	if safeHash != (types.Hash{}) {
		safe = bc.GetBlock(safeHash)
		if safe == nil || !bc.HasValidatedBlock(safeHash) {
			return fmt.Errorf("forkchoice: unknown safe block")
		}
	}
	if finalHash != (types.Hash{}) {
		final = bc.GetBlock(finalHash)
		if final == nil || !bc.HasValidatedBlock(finalHash) {
			return fmt.Errorf("forkchoice: unknown finalized block")
		}
	}
	prior := bc.CurrentFinalBlock()
	if prior != nil && final != nil && final.NumberU64() < prior.NumberU64() {
		return fmt.Errorf("forkchoice: finalized block regresses")
	}
	if safe != nil && final != nil && safe.NumberU64() < final.NumberU64() {
		return fmt.Errorf("forkchoice: safe block precedes finalized block")
	}
	ancestors := make(map[types.Hash]*types.Block)
	canonicalAlready := make(map[types.Hash]bool)
	oldHead := bc.CurrentBlock()
	minimum := head.NumberU64()
	if oldHead.NumberU64() < minimum {
		minimum = oldHead.NumberU64()
	}
	for _, required := range []*types.Block{safe, final, prior} {
		if required != nil && required.NumberU64() < minimum {
			minimum = required.NumberU64()
		}
	}
	for block := head; block != nil; block = bc.GetBlock(block.ParentHash()) {
		ancestors[block.Hash()] = block
		canonical, err := rawdb.ReadCanonicalHash(bc.db, block.NumberU64())
		if err != nil && !errors.Is(err, rawdb.ErrNotFound) {
			return storageFailure(err)
		}
		canonicalAlready[block.Hash()] = err == nil && canonical == block.Hash()
		if block.NumberU64() <= minimum && err == nil && canonical == block.Hash() {
			break
		}
		if block.NumberU64() == 0 {
			break
		}
	}
	for _, required := range []*types.Block{safe, final, prior} {
		if required != nil && ancestors[required.Hash()] == nil {
			return fmt.Errorf("forkchoice: checkpoint is not a head ancestor")
		}
	}
	var removed []*types.Block
	old := oldHead
	for old != nil && ancestors[old.Hash()] == nil {
		removed = append(removed, old)
		if old.NumberU64() == 0 {
			return fmt.Errorf("forkchoice: no common ancestor")
		}
		old = bc.GetBlock(old.ParentHash())
	}
	if old == nil {
		return fmt.Errorf("forkchoice: missing canonical ancestor")
	}
	st, err := bc.StateAtBlock(head)
	if err != nil {
		return err
	}
	if st.GetRoot() != head.Header().Root {
		bc.sc.remove(headHash)
		bc.memSC.remove(headHash)
		return fmt.Errorf("%w: forkchoice head state root mismatch", ErrStateNotFound)
	}
	if oldHead.Hash() != headHash {
		if bc.config != nil && bc.config.BinaryOverlayTime != nil {
			if err := bc.verifyOverlayCheckpoint(head, st); err != nil {
				return storageFailure(err)
			}
		}
		root, err := st.Commit()
		if err != nil {
			return storageFailure(err)
		}
		if root != head.Header().Root {
			return fmt.Errorf("forkchoice: committed root mismatch")
		}
	}
	batch := bc.db.NewBatch()
	deletedLookups := make(map[types.Hash]bool)
	for _, block := range removed {
		for _, tx := range block.Transactions() {
			deletedLookups[tx.Hash()] = true
			if err := rawdb.DeleteTxLookup(batch, tx.Hash()); err != nil {
				return storageFailure(err)
			}
		}
		if block.NumberU64() > head.NumberU64() {
			if err := rawdb.DeleteCanonicalHash(batch, block.NumberU64()); err != nil {
				return storageFailure(err)
			}
		}
	}
	for _, block := range ancestors {
		if !canonicalAlready[block.Hash()] {
			if err := rawdb.WriteCanonicalHash(batch, block.NumberU64(), block.Hash()); err != nil {
				return storageFailure(err)
			}
		}
		for _, tx := range block.Transactions() {
			oldNumber, err := rawdb.ReadTxLookup(bc.db, tx.Hash())
			if err != nil && !errors.Is(err, rawdb.ErrNotFound) {
				return storageFailure(err)
			}
			if deletedLookups[tx.Hash()] || err != nil || oldNumber != block.NumberU64() {
				if err := rawdb.WriteTxLookup(batch, tx.Hash(), block.NumberU64()); err != nil {
					return storageFailure(err)
				}
			}
		}
	}
	for _, pointer := range []struct {
		target types.Hash
		read   func(rawdb.KeyValueReader) ([32]byte, error)
		write  func(rawdb.KeyValueWriter, [32]byte) error
	}{
		{headHash, rawdb.ReadHeadBlockHash, rawdb.WriteHeadBlockHash},
		{headHash, rawdb.ReadHeadHeaderHash, rawdb.WriteHeadHeaderHash},
	} {
		old, err := pointer.read(bc.db)
		if err != nil && !errors.Is(err, rawdb.ErrNotFound) {
			return storageFailure(err)
		}
		if err != nil || old != pointer.target {
			if err := pointer.write(batch, pointer.target); err != nil {
				return storageFailure(err)
			}
		}
	}
	if final != nil {
		old, err := rawdb.ReadFinalizedBlockHashChecked(bc.db)
		if err != nil && !errors.Is(err, rawdb.ErrNotFound) {
			return storageFailure(err)
		}
		if err != nil || old != finalHash {
			if err := rawdb.WriteFinalizedBlockHash(batch, finalHash); err != nil {
				return storageFailure(err)
			}
		}
	}
	err = rawdb.WriteBatchAndPublish(batch, &bc.mu, func() {
		bc.canonicalVersion++
		for _, block := range removed {
			for _, tx := range block.Transactions() {
				delete(bc.txLookup, tx.Hash())
			}
		}
		for _, block := range ancestors {
			bc.canonCache[block.NumberU64()] = block.Hash()
			for index, tx := range block.Transactions() {
				bc.txLookup[tx.Hash()] = TxLookupEntry{BlockHash: block.Hash(), BlockNumber: block.NumberU64(), TxIndex: uint64(index)}
			}
		}
		for number := range bc.canonCache {
			if number > head.NumberU64() {
				delete(bc.canonCache, number)
			}
		}
		bc.currentBlock, bc.currentState, bc.currentSafeBlock = head, st, safe
		if final != nil {
			bc.currentFinalBlock = final
		}
		// Replace a pre-commit cache entry with the independent committed state.
		bc.sc.remove(headHash)
		bc.memSC.remove(headHash)
		bc.sc.put(headHash, head.NumberU64(), st)
		bc.sc.protect(headHash)
		bc.memSC.put(headHash, head.NumberU64(), st)
		bc.memSC.protect(headHash)
		bc.hc.mu.Lock()
		for number := range bc.hc.headers {
			if number > head.NumberU64() {
				delete(bc.hc.headers, number)
			}
		}
		for _, block := range ancestors {
			bc.hc.headers[block.NumberU64()] = block.Header()
			bc.hc.headersByHash[block.Hash()] = block.Header()
		}
		bc.hc.currentHeader = head.Header()
		bc.hc.mu.Unlock()
	})
	return storageFailure(err)
}
