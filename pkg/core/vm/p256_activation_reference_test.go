package vm

import (
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
)

func TestCanonicalP256ActivationMatchesIndependentForkSets(t *testing.T) {
	address := types.Address{18: 1}
	for _, test := range []struct {
		name        string
		rules       ForkRules
		independent map[gethcommon.Address]gethvm.PrecompiledContract
	}{
		{"Cancun", ForkRules{IsCancun: true}, gethvm.PrecompiledContractsCancun},
		{"Prague", ForkRules{IsCancun: true, IsPrague: true}, gethvm.PrecompiledContractsPrague},
		{"Osaka", ForkRules{IsCancun: true, IsPrague: true, IsOsaka: true}, gethvm.PrecompiledContractsOsaka},
	} {
		t.Run(test.name, func(t *testing.T) {
			native, present := SelectPrecompiles(test.rules)[address]
			reference, expected := test.independent[gethcommon.Address(address)]
			if present != expected {
				t.Fatalf("P256 activation native=%t independent=%t", present, expected)
			}
			if present && native.RequiredGas(nil) != reference.RequiredGas(nil) {
				t.Fatal("canonical P256 price differs")
			}
		})
	}
}
