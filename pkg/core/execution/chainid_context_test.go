package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	gethtests "github.com/ethereum/go-ethereum/tests"
)

func TestCanonicalChainIDContextMatchesIndependentExecution(t *testing.T) {
	for _, chain := range []*big.Int{big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 200)} {
		cfg := *config.TestConfig
		cfg.ChainID = new(big.Int).Set(chain)
		zero := uint64(0)
		cfg.OsakaTime = &zero
		cfg.BPO1Time = &zero
		cfg.BPO2Time = &zero
		cfg.AmsterdamTime = &zero
		cfg.GlamsterdanTime = nil
		cfg.HogotaTime = nil
		cfg.IPlusTime = nil
		target, caller := types.Address{19: 0x42}, types.Address{19: 0xe0}
		code := []byte{byte(vm.CHAINID), byte(vm.PUSH0), byte(vm.MSTORE), byte(vm.PUSH1), 32, byte(vm.PUSH0), byte(vm.RETURN)}
		db := state.NewMemoryStateDB()
		db.SetCode(target, code)
		db.SetNonce(target, 1)
		evm := vm.NewEVMWithState(vm.BlockContext{BlockNumber: big.NewInt(1), Time: 12, GasLimit: 100000}, vm.TxContext{}, vm.Config{}, db)
		if _, err := ConfigureBlockEVM(&cfg, &types.Header{Number: big.NewInt(1), Time: 12}, evm); err != nil {
			t.Fatal(err)
		}
		got, left, err := evm.Call(caller, target, nil, 100000, new(big.Int))
		if err != nil {
			t.Fatal(err)
		}
		sdk, err := gethstate.New(gethtypes.EmptyRootHash, gethstate.NewDatabaseForTesting())
		if err != nil {
			t.Fatal(err)
		}
		sdk.SetCode(gethcommon.Address(target), code, tracing.CodeChangeUnspecified)
		sdk.SetNonce(gethcommon.Address(target), 1, tracing.NonceChangeUnspecified)
		reference := *gethtests.Forks["Amsterdam"]
		reference.ChainID = new(big.Int).Set(chain)
		want, wantLeft, err := runtime.Call(gethcommon.Address(target), nil, &runtime.Config{ChainConfig: &reference, State: sdk, Origin: gethcommon.Address(caller), GasLimit: 100000, BlockNumber: big.NewInt(1), Time: 12})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) || new(big.Int).SetBytes(got).Cmp(chain) != 0 || left != wantLeft {
			t.Fatalf("chain %s native=%x/%d SDK=%x/%d", chain, got, left, want, wantLeft)
		}
	}
}
