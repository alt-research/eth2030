package payload

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

func TestCanonicalFrame8141BlockValueIncludesClippedTip(t *testing.T) {
	frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: types.Address{19: 1},
		Fees:   types.FrameFeeBudget{MaxFee: new(big.Int), MaxPriorityFeePerGas: big.NewInt(10)},
		Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 100000}}}}
	budget, err := frame.GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	frame.Fees.MaxFee.Add(new(big.Int).SetUint64(budget.MaxGas), big.NewInt(1))
	used := budget.CalldataFloor
	block := types.NewBlock(&types.Header{Number: big.NewInt(1), BaseFee: big.NewInt(1)}, &types.Body{Transactions: []*types.Transaction{types.NewTransaction(frame)}})
	value := calcBlockValue(nil, block, []*types.Receipt{{GasUsed: used, EffectiveGasPrice: big.NewInt(1)}}, big.NewInt(1))
	expected := new(big.Int).Sub(frame.Fees.MaxFee, new(big.Int).SetUint64(used))
	if value.Cmp(expected) != 0 || value.Sign() == 0 {
		t.Fatalf("clipped tip was rounded away: got %s want %s", value, expected)
	}
}
