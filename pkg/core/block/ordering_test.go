package block

import (
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestFrameCBuilderOrdersSenderHeadsAtBlobFee(t *testing.T) {
	sender := types.Address{19: 42}
	makeTx := func(nonce uint64, price int64) *types.Transaction {
		tx := types.NewTransaction(&types.LegacyTx{Nonce: nonce, GasPrice: big.NewInt(price), Gas: 21000})
		tx.SetSender(sender)
		return tx
	}
	low, high := makeTx(0, 1), makeTx(1, 100)
	other := makeTx(0, 50)
	other.SetSender(types.Address{19: 43})
	ordered := orderTransactionsAt([]*types.Transaction{high, low, other}, big.NewInt(1), big.NewInt(1))
	if len(ordered) != 3 || ordered[0] != other || ordered[1] != low || ordered[2] != high {
		t.Fatal("higher-priced nonce bypassed its sender head")
	}
	frame := &types.FrameTx8141{Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(100)}, BlobVersionedHashes: []types.Hash{{0: 1}}}
	tx := types.NewTransaction(frame)
	if effectiveGasPriceAt(tx, big.NewInt(1), big.NewInt(1)).Cmp(effectiveGasPriceAt(tx, big.NewInt(1), big.NewInt(5))) <= 0 {
		t.Fatal("blob fees did not reduce total-budget priority")
	}
}
