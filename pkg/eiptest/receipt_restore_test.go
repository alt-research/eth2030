package eiptest_test

import (
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/eiptest"
	"testing"
)

func TestReceivedReceiptsRevalidateAndRebuildPersistentMetadata(t *testing.T) {
	s := eiptest.New(t)
	result := s.STF().Transfer(s.Alice(), s.Bob().Address, eiptest.Wei(3))
	_ = result
	block := s.CurrentBlock()
	original := s.Chain().GetReceipts(block.Hash())
	if len(original) != 1 {
		t.Fatal("missing native receipt")
	}
	bad := *original[0]
	bad.CumulativeGasUsed++
	if err := s.Chain().StoreReceivedReceipts(block.Hash(), []*types.Receipt{&bad}); err == nil {
		t.Fatal("stored a receipt with the wrong root")
	}
	if err := s.Chain().StoreReceivedReceipts(block.Hash(), nil); err == nil {
		t.Fatal("stored the wrong receipt count")
	}
	if err := s.Chain().StoreReceivedReceipts(block.Hash(), original); err != nil {
		t.Fatal(err)
	}
	restored := s.Chain().GetReceipts(block.Hash())
	if len(restored) != 1 || restored[0] == original[0] || restored[0].TxHash != block.Transactions()[0].Hash() || restored[0].BlockHash != block.Hash() || restored[0].GasUsed != original[0].GasUsed {
		t.Fatal("restored receipts did not rebuild persistent metadata")
	}
}
