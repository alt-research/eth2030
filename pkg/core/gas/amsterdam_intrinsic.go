package gas

import (
	"errors"
	"fmt"
	"math/big"
	"math/bits"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

type AmsterdamIntrinsic struct {
	Gas   uint64
	Floor uint64
}

func addAmsterdamCost(total, count, cost uint64) (uint64, error) {
	hi, low := bits.Mul64(count, cost)
	sum, carry := bits.Add64(total, low, 0)
	if hi != 0 || carry != 0 {
		return 0, errors.New("Amsterdam intrinsic gas overflow")
	}
	return sum, nil
}

// CalculateAmsterdamIntrinsic reads transaction fields only. New-account and
// delegation costs belong to the runtime pre-execution phase, never validity.
func CalculateAmsterdamIntrinsic(from types.Address, to *types.Address, value *big.Int, data []byte, access types.AccessList, authorizations uint64) (AmsterdamIntrinsic, error) {
	base := vm.AmsterdamTxBase
	if to == nil {
		base += vm.AmsterdamCreateAccess
	} else if *to != from {
		base += vm.AmsterdamColdAccount
		if value != nil && value.Sign() > 0 {
			base += vm.AmsterdamTxValue
		}
	}
	result := AmsterdamIntrinsic{Gas: base, Floor: base}
	var err error
	add := func(target *uint64, count, cost uint64) {
		if err == nil {
			*target, err = addAmsterdamCost(*target, count, cost)
		}
	}
	var zeros uint64
	for _, b := range data {
		if b == 0 {
			zeros++
		}
	}
	length := uint64(len(data))
	add(&result.Gas, zeros, 4)
	add(&result.Gas, length-zeros, 16)
	add(&result.Floor, length, 64)
	if to == nil {
		add(&result.Gas, (length+31)/32, 2)
	}
	for _, entry := range access {
		keys := uint64(len(entry.StorageKeys))
		add(&result.Gas, 1, vm.AmsterdamAccessListAddress+20*64)
		add(&result.Gas, keys, vm.AmsterdamAccessListStorage+32*64)
		add(&result.Floor, 1, 20*64)
		add(&result.Floor, keys, 32*64)
	}
	add(&result.Gas, authorizations, vm.AmsterdamAuthorizationBase)
	return result, err
}
func (g AmsterdamIntrinsic) Validate(limit uint64) error {
	if limit > vm.AmsterdamMaxTotalGas {
		return fmt.Errorf("Amsterdam transaction gas limit exceeds total cap: %d", limit)
	}
	required := max(g.Gas, g.Floor)
	if required > vm.AmsterdamMaxExecutionGas {
		return fmt.Errorf("Amsterdam intrinsic/floor exceeds execution cap: %d", required)
	}
	if limit < required {
		return fmt.Errorf("intrinsic gas too low: have %d want %d", limit, required)
	}
	return nil
}

// ExecutionBudget bounds the execution dimension after paying intrinsic gas.
func (g AmsterdamIntrinsic) ExecutionBudget(limit uint64) (execution, reservoir uint64, err error) {
	if err = g.Validate(limit); err != nil {
		return 0, 0, err
	}
	evmGas := limit - g.Gas
	execution = min(vm.AmsterdamMaxExecutionGas-g.Gas, evmGas)
	return execution, evmGas - execution, nil
}
