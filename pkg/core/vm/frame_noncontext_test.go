package vm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func TestFrameInstructionsRejectNonFrameContext(t *testing.T) {
	for _, op := range []byte{0xaa, 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5} {
		t.Run(fmt.Sprintf("opcode_%02x", op), func(t *testing.T) {
			db := state.NewMemoryStateDB()
			target, caller := types.Address{19: 0x90}, types.Address{19: 0x91}
			db.SetCode(target, []byte{0x5f, 0x5f, 0x5f, 0x5f, op, 0})
			rules := ForkRules{IsMerge: true, IsLondon: true, IsBerlin: true, IsShanghai: true, IsCancun: true, IsPrague: true, IsOsaka: true, IsAmsterdam: true}
			evm := NewEVMWithState(BlockContext{BlockNumber: big.NewInt(1)}, TxContext{}, Config{}, db)
			evm.SetForkRules(rules)
			evm.SetJumpTable(SelectJumpTable(rules))
			evm.SetPrecompiles(SelectPrecompiles(rules))
			before := db.GetRoot()
			_, left, err := evm.Call(caller, target, nil, 100000, new(big.Int))
			if err == nil || left != 0 || db.GetRoot() != before {
				t.Fatalf("non-frame context: gas=%d err=%v", left, err)
			}
		})
	}
}
