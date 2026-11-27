package block

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
)

func TestGroupEBuilderExactRLPBudgetBoundaryAndRejectedAppendRollback(t *testing.T) {
	header := &types.Header{Number: big.NewInt(1), GasLimit: 30000000, BaseFee: big.NewInt(1)}
	attrs := &BuildBlockAttributes{}
	budget, err := newBuilderRLPBudget(config.TestConfigHogota, header, attrs)
	if err != nil {
		t.Fatal(err)
	}
	to := types.Address{19: 42}
	transaction := func(length int) *types.Transaction {
		return types.NewTransaction(&types.LegacyTx{To: &to, Gas: 30000000, GasPrice: big.NewInt(1), Value: new(big.Int), Data: make([]byte, length)})
	}
	// Find the independently serialized transaction whose complete block lands
	// exactly on the protocol limit, including both list prefixes and header.
	length := MaxRLPBlockSize - 1000
	for i := 0; i < 3; i++ {
		tx := transaction(length)
		h := *budget.header
		h.GasUsed = 128
		wire, e := types.NewBlock(&h, &types.Body{Transactions: []*types.Transaction{tx}, Withdrawals: []*types.Withdrawal{}}).EncodeRLP()
		if e != nil {
			t.Fatal(e)
		}
		length += MaxRLPBlockSize - len(wire)
	}
	tooBig := transaction(length + 1)
	if fits, err := budget.append(nil, tooBig, 128, 0); err != nil || fits {
		t.Fatal("oversize append admitted", fits, err)
	}
	tx := transaction(length)
	if fits, err := budget.append(nil, tx, 128, 0); err != nil || !fits {
		t.Fatal("exact limit rejected or failed append poisoned sizing", fits, err)
	}
	if fits, err := budget.append([]*types.Transaction{tx}, transaction(0), 256, 0); err != nil || fits {
		t.Fatal("full block accepted another transaction", fits, err)
	}
}
