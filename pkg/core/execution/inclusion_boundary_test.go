package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/types"
)

func TestGroupETransactionEndIncludesPreSystemsAndExcludesPostSystems(t *testing.T) {
	db := newBuilderRequestTestState()
	// Custom consensus allocations are executed exactly as code specifies.
	// The pre-phase writes the supplied parent hash; the post-phase writes 1.
	pre, post := eips.HistoryStorageAddress, types.WithdrawalRequestAddress
	db.SetCode(pre, []byte{0x5f, 0x35, 0x60, 7, 0x55, 0})
	db.SetCode(post, []byte{0x60, 1, 0x60, 9, 0x55, 0})
	receiver := types.Address{19: 0x70}
	parent := types.Hash{31: 0x44}
	h := &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 30000000, BaseFee: big.NewInt(1), ParentHash: parent}
	block := types.NewBlock(h, &types.Body{Withdrawals: []*types.Withdrawal{{Address: receiver, Amount: 3}}})
	cfg := *config.TestConfigHogota
	processor := NewStateProcessor(&cfg)
	prefix, whole := db.Dup(), db.Dup()
	result, err := processor.ProcessTransactionsWithBAL(block, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if prefix.GetState(pre, types.Hash{31: 7}) != parent || prefix.GetState(post, types.Hash{31: 9}) != (types.Hash{}) || prefix.GetBalance(receiver).Sign() != 0 || len(result.Requests) != 0 {
		t.Fatal("incorrect EIP-7805 cut point")
	}
	for _, entry := range result.BlockAccessList.Entries {
		if entry.AccessIndex != 0 {
			t.Fatal("post phase leaked into prefix BAL", entry)
		}
	}
	if _, err := processor.ProcessWithBAL(block, whole); err != nil {
		t.Fatal(err)
	}
	if whole.GetState(pre, types.Hash{31: 7}) != parent || whole.GetState(post, types.Hash{31: 9}) != (types.Hash{31: 1}) || whole.GetBalance(receiver).Cmp(big.NewInt(3000000000)) != 0 {
		t.Fatal("whole-block processing changed")
	}
	if db.GetState(pre, types.Hash{31: 7}) != (types.Hash{}) || db.GetBalance(receiver).Sign() != 0 {
		t.Fatal("reconstruction mutated parent")
	}
}
