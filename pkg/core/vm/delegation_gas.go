package vm

import (
	"github.com/eth2030/eth2030/core/types"
	"math/big"
)

// callGasForWord clamps the original 256-bit request before narrowing. Taking
// Uint64 first can turn a valid large request into a tiny or zero budget.
func callGasForWord(available, base uint64, requested *big.Int) uint64 {
	limit := ^uint64(0)
	if requested.IsUint64() {
		limit = requested.Uint64()
	}
	return callGasEIP150(available, base, limit)
}

// Delegation is resolved after the recipient's known pre-state costs pass,
// before balance/depth checks. The extra target has its own access gas check.
func gasDelegationResolution(evm *EVM, c *Contract, address types.Address, cost uint64) (uint64, error) {
	if evm.StateDB == nil || !(evm.frame8141 != nil || evm.forkRules.IsPrague || evm.forkRules.IsAmsterdam || evm.forkRules.IsGlamsterdan) {
		return cost, nil
	}
	if c.Gas < cost {
		return 0, ErrOutOfGas
	}
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(address)
	}
	target, delegated := types.ParseDelegation(evm.StateDB.GetCode(address))
	if !delegated {
		return cost, nil
	}
	cold, warm := uint64(2600), uint64(100)
	if evm.frame8141 != nil || evm.forkRules.IsAmsterdam {
		cold = AmsterdamColdAccount
	} else if evm.forkRules.IsGlamsterdan {
		cold, warm = ColdAccountAccessGlamst, WarmStorageReadGlamst
	}
	extra := warm
	if !evm.StateDB.AddressInAccessList(target) {
		extra = cold
	}
	cost = safeAdd(cost, extra)
	if c.Gas < cost {
		return 0, ErrOutOfGas
	}
	evm.StateDB.AddAddressToAccessList(target)
	_ = evm.StateDB.GetCode(target)
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(target)
	}
	return cost, nil
}
