package execution

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math/big"
	"testing"
)

func TestCanonicalOsakaPrecompileActivationBoundary(t *testing.T) {
	cfg := *config.TestConfig
	fork := uint64(10)
	cfg.OsakaTime = &fork
	cfg.AmsterdamTime = nil
	cfg.GlamsterdanTime = nil
	cfg.HogotaTime = nil
	cfg.IPlusTime = nil
	for _, tc := range []struct{ time, gas uint64 }{{9, 200}, {10, 500}, {11, 500}} {
		header := &types.Header{Number: big.NewInt(1), Time: tc.time, GasLimit: 30000000}
		evm := vm.NewEVMWithState(vm.BlockContext{BlockNumber: header.Number, Time: tc.time}, vm.TxContext{}, vm.Config{}, state.NewMemoryStateDB())
		precompiles, err := ConfigureBlockEVM(&cfg, header, evm)
		if err != nil {
			t.Fatal(err)
		}
		if got := precompiles[types.Address{19: 5}].RequiredGas(nil); got != tc.gas {
			t.Fatalf("timestamp %d native ModExp gas=%d want %d", tc.time, got, tc.gas)
		}
	}
}
