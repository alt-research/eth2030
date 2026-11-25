package eips_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func TestBuilderRequestCanonicalConstructors(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		init, runtime, independent []byte
		inhibited                  bool
	}{
		{"deposit", eips.BuilderDepositInitCode(), eips.BuilderDepositRuntime(), params.BuilderDepositCode, false},
		{"exit", eips.BuilderExitInitCode(), eips.BuilderExitRuntime(), params.BuilderExitCode, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Equal(tc.runtime, tc.independent) {
				t.Fatal("pinned sys-asm and independent client runtime differ")
			}
			db := state.NewMemoryStateDB()
			sender := types.HexToAddress("0x123456")
			db.AddBalance(sender, big.NewInt(10000000))
			evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, db)
			result, address, _, err := evm.Create(sender, tc.init, 2000000, new(big.Int))
			if err != nil || !bytes.Equal(result, tc.runtime) || !bytes.Equal(db.GetCode(address), tc.runtime) || db.GetNonce(address) != 1 {
				t.Fatalf("constructor result mismatch: %v", err)
			}
			value := db.GetState(address, types.Hash{})
			expected := types.Hash{}
			if tc.inhibited {
				for i := range expected {
					expected[i] = 255
				}
			}
			if value != expected {
				t.Fatal("constructor inhibitor state differs")
			}
		})
	}
}
