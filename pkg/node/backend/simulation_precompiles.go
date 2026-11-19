package backend

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"maps"
	"sort"
	"strings"
)

// Derive a request-owned map; never delete or move entries in the fork globals.
func simulationPrecompiles(cfg *config.ChainConfig, header *types.Header, db state.StateDB, overrides rpctypes.StateOverride) (map[types.Address]vm.PrecompiledContract, error) {
	evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, db)
	active, err := execution.ConfigureBlockEVM(cfg, header, evm)
	if err != nil {
		return nil, err
	}
	runtime := maps.Clone(active)
	names := make([]string, 0, len(overrides))
	for name := range overrides {
		names = append(names, name)
	}
	sort.Strings(names)
	destinations := make(map[types.Address]bool)
	for _, name := range names {
		raw, err := hexutil.Decode(name)
		if err != nil || len(raw) != 20 {
			return nil, simulationError(-32602, "invalid override address")
		}
		address := types.BytesToAddress(raw)
		ov := overrides[name]
		if destinations[address] {
			return nil, simulationError(-32602, "precompile destination is also overridden")
		}
		precompile, exists := runtime[address]
		if ov.MovePrecompileToAddress != nil {
			to, err := hexutil.Decode(*ov.MovePrecompileToAddress)
			if err != nil || len(to) != 20 {
				return nil, simulationError(-32602, "invalid precompile destination")
			}
			destination := types.BytesToAddress(to)
			if destination == address {
				return nil, simulationError(-38022, "precompile destination references itself")
			}
			if !exists {
				return nil, simulationError(-32602, "cannot move an inactive precompile")
			}
			for _, other := range names {
				if strings.EqualFold(destination.Hex(), other) {
					return nil, simulationError(-32602, "precompile destination is also overridden")
				}
			}
			runtime[destination] = precompile
			destinations[destination] = true
		}
		if exists {
			delete(runtime, address)
		}
	}
	return runtime, nil
}
