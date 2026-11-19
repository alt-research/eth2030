package chain

import (
	"fmt"

	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/types"
)

// StoreReceivedReceipts restores a known validated block's canonical receipt
// data. The next read rebuilds derived transaction/log fields from that block.
func (bc *Blockchain) StoreReceivedReceipts(hash types.Hash, receipts []*types.Receipt) error {
	block := bc.GetBlock(hash)
	if block == nil || !bc.HasValidatedBlock(hash) {
		return fmt.Errorf("receipt target is not execution-validated")
	}
	if len(receipts) != len(block.Transactions()) {
		return fmt.Errorf("receipt count mismatch")
	}
	for _, receipt := range receipts {
		if receipt == nil {
			return fmt.Errorf("nil receipt")
		}
	}
	if coreblock.ComputeReceiptsRoot(receipts) != block.Header().ReceiptHash {
		return fmt.Errorf("receipt root mismatch")
	}
	if err := bc.persistValidatedBlock(block, receipts); err != nil {
		return storageFailure(err)
	}
	bc.rcMu.Lock()
	delete(bc.receiptCache, hash)
	bc.rcMu.Unlock()
	return nil
}
