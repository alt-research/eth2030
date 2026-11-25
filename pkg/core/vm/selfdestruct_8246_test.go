package vm

import (
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestEIP8246CreationJournalRollback(t *testing.T) {
	db := state.NewMemoryStateDB()
	sender := types.HexToAddress("0x123456")
	parent := types.HexToAddress("0xc0ffee")
	db.AddBalance(sender, big.NewInt(1000000))
	// CREATE with empty initcode succeeds, then the enclosing CALL reverts.
	db.SetCode(parent, []byte{0x60, 0, 0x60, 0, 0x60, 0, 0xf0, 0x50, 0x60, 0, 0x60, 0, 0xfd})
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
	evm.SetForkRules(ForkRules{IsAmsterdam: true})
	evm.SetJumpTable(SelectJumpTable(ForkRules{IsAmsterdam: true}))
	if _, _, err := evm.Call(sender, parent, nil, 200000, new(big.Int)); err != ErrExecutionReverted {
		t.Fatalf("want revert, got %v", err)
	}
	if len(evm.createdOrder) != 0 || len(evm.createdContracts) != 0 {
		t.Fatalf("reverted creation remained transaction-created: %v", evm.createdOrder)
	}
	// Failing initcode must also remove its own creation marker.
	if _, _, _, err := evm.Create(sender, []byte{0x60, 0, 0x60, 0, 0xfd}, 200000, new(big.Int)); err != ErrExecutionReverted {
		t.Fatalf("want init revert, got %v", err)
	}
	if len(evm.createdOrder) != 0 || len(evm.createdContracts) != 0 {
		t.Fatal("failed initcode retained creation marker")
	}
}

func TestEIP8246CreationPrechecksPreserveNonce(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, overflow := range []bool{false, true} {
			db := state.NewMemoryStateDB()
			sender := types.HexToAddress("0x123456")
			value := big.NewInt(1)
			nonce := uint64(0)
			if overflow {
				db.AddBalance(sender, value)
				nonce = ^uint64(0)
				db.SetNonce(sender, nonce)
			}
			evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
			evm.SetForkRules(ForkRules{IsAmsterdam: true})
			var err error
			if create2 {
				_, _, _, err = evm.Create2(sender, nil, 200000, value, new(big.Int))
			} else {
				_, _, _, err = evm.Create(sender, nil, 200000, value)
			}
			if err == nil || db.GetNonce(sender) != nonce || len(evm.createdContracts) != 0 {
				t.Fatalf("invalid creation changed state: create2=%t overflow=%t nonce=%d error=%v", create2, overflow, db.GetNonce(sender), err)
			}
		}
	}
}
