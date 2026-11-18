package eiptest_test

import (
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/eiptest"
	"math/big"
	"testing"
)

// EIP-7918 changes excess accumulation, not the current exponential price.
// Actual signed HTTP/state/receipt consumers are covered by the node regression.
func TestEIP7918_ReserveChangesExcessWithoutDirectPriceFloor(t *testing.T) {
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkOsaka, 0))
	zero, used := uint64(0), uint64(7*gas.GasPerBlob)
	parent := &types.Header{Number: big.NewInt(1), Time: 12, BaseFee: big.NewInt(1_000_000_000), BlobGasUsed: &used, ExcessBlobGas: &zero}
	if price := gas.BlobBaseFeeForHeader(s.Config(), parent); price.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("direct reserve-price clamp: %s", price)
	}
	excess, err := gas.CalcExcessBlobGasForHeader(parent, s.Config(), 24)
	want := used * 3 / 9 // Prague/BPO-inherited target six, max nine.
	if err != nil || excess != want {
		t.Fatalf("reserve excess = %d want %d: %v", excess, want, err)
	}
}
