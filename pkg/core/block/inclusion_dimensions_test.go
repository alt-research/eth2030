package block

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

func TestGroupEBuilderExactRLPIndependentGasPools(t *testing.T) {
	header := &types.Header{Number: big.NewInt(1), GasLimit: 30000000, BaseFee: big.NewInt(1)}
	budget, err := newBuilderRLPBudget(config.TestConfigHogota, header, &BuildBlockAttributes{})
	if err != nil {
		t.Fatal(err)
	}
	gp := new(gaspool.GasPool).AddGas(header.GasLimit)
	// A durable storage charge can dominate the header while execution remains
	// below the next RLP integer width. These are independent block dimensions.
	if err := gp.ReserveFrame(65535, 97920); err != nil {
		t.Fatal(err)
	}
	to := types.Address{19: 42}
	transaction := func(n int) *types.Transaction {
		return types.NewTransaction(&types.LegacyTx{To: &to, Gas: 30000000, GasPrice: big.NewInt(1), Value: new(big.Int), Data: make([]byte, n)})
	}
	length := MaxRLPBlockSize - 1000
	for i := 0; i < 3; i++ {
		h := budget.header.Copy()
		h.GasUsed = 97920
		wire, err := types.NewBlock(h, &types.Body{Transactions: []*types.Transaction{transaction(length)}, Withdrawals: []*types.Withdrawal{}}).EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		length += MaxRLPBlockSize - len(wire)
	}
	if fits, err := budget.appendWithPools(nil, transaction(length+1), gp, header.GasLimit, 0); err != nil || fits {
		t.Fatal("state-dominated header admitted oversized block", fits, err)
	}
	if fits, err := budget.appendWithPools(nil, transaction(length), gp, header.GasLimit, 0); err != nil || !fits {
		t.Fatal("exact state-dominated RLP size rejected", fits, err)
	}
	if gp.Gas() != header.GasLimit-65535 || gp.StateGas() != header.GasLimit-97920 {
		t.Fatal("sizing consumed capacity")
	}
}
