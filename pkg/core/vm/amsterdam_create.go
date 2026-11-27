package vm

import (
	"errors"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
)

// createAmsterdam keeps the parent reservation outside the child frame. In
// particular, code-deposit OOG cannot consume the parent's EIP-150 retained gas.
func (evm *EVM) createAmsterdam(caller types.Address, code []byte, gas uint64, value *big.Int, address types.Address, kind OpCode) (ret []byte, created types.Address, left uint64, err error) {
	if len(code) > MaxInitCodeSizeForFork(evm.forkRules) {
		return nil, types.Address{}, gas, ErrMaxInitCodeSizeExceeded
	}
	parent := evm.activeContract
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(address)
	}
	existent := !evm.StateDB.Empty(address)
	hash := evm.StateDB.GetCodeHash(address)
	collision := evm.StateDB.GetNonce(address) != 0 || (hash != (types.Hash{}) && hash != types.EmptyCodeHash)
	if !collision && (evm.forkRules.IsHogota || evm.forkRules.IsEIP8298) {
		collision = HasNonEmptyStorage(evm.StateDB, address)
	}
	evm.StateDB.AddAddressToAccessList(address)
	if parent != nil {
		parent.Gas = gas
		if !existent {
			if !parent.ChargeStateGas(AmsterdamNewAccountStateGas) {
				return nil, types.Address{}, 0, ErrFrameStateGas
			}
			parent.pendingAccountStateGas = AmsterdamNewAccountStateGas
		}
		gas = parent.Gas
		parent.Gas = 0
		evm.forwardStateGas(parent)
	}
	retained := uint64(0)
	if parent != nil {
		retained = gas / CallGasFraction
		gas -= retained
	}
	if finish := evm.traceFrame(kind, caller, address, code, gas, value); finish != nil {
		defer func() {
			childLeft := left
			if childLeft >= retained {
				childLeft -= retained
			}
			finish(ret, childLeft, err)
		}()
	}
	frame := evm.beginStateGasFrame(caller, address, value, gas)
	// Top-level callers outside transaction processing still pay the state they
	// create. The transaction processor supplies its already charged carrier.
	if parent == nil && frame.StateGasUsed == 0 && !existent {
		if !frame.ChargeStateGas(AmsterdamNewAccountStateGas) {
			evm.finishStateGasFrame(frame, &left, ErrOutOfGas)
			return nil, types.Address{}, 0, ErrOutOfGas
		}
	}
	defer func() {
		childLeft := left
		evm.finishStateGasFrame(frame, &childLeft, err)
		left = retained + childLeft
	}()
	if collision {
		return nil, types.Address{}, 0, errors.New("contract address collision")
	}
	snapshot := evm.StateDB.Snapshot()
	createdSnapshot := len(evm.createdOrder)
	previousBalance := evm.StateDB.GetBalance(address)
	evm.StateDB.CreateAccount(address)
	if previousBalance.Sign() > 0 {
		evm.StateDB.AddBalance(address, previousBalance)
	}
	evm.markContractCreated(address)
	evm.StateDB.SetNonce(address, 1)
	evm.recordCreationNonce(address, 0, 1)
	if value != nil && value.Sign() > 0 {
		var oldCaller, oldTarget *big.Int
		if evm.balTracker != nil {
			oldCaller = new(big.Int).Set(evm.StateDB.GetBalance(caller))
			oldTarget = new(big.Int).Set(evm.StateDB.GetBalance(address))
		}
		evm.StateDB.SubBalance(caller, value)
		evm.StateDB.AddBalance(address, value)
		if evm.balTracker != nil {
			evm.balTracker.RecordBalanceChange(caller, oldCaller, evm.StateDB.GetBalance(caller))
			evm.balTracker.RecordBalanceChange(address, oldTarget, evm.StateDB.GetBalance(address))
		}
		evm.recordETHTransfer(caller, address, value)
	}
	frame.Code = code
	evm.depth++
	ret, err = evm.Run(frame, nil)
	evm.depth--
	if err == nil {
		if adopted, ok := evm.adoptedCreationCode(address); ok {
			return adopted, address, frame.Gas, nil
		}
		if len(ret) > MaxCodeSizeForFork(evm.forkRules) {
			err = errors.New("max code size exceeded")
		} else if len(ret) > 0 && ret[0] == 0xef {
			err = errors.New("invalid code prefix")
		}
		if err == nil && len(ret) > 0 {
			if !frame.ChargeStateGas(uint64(len(ret))*AmsterdamCPSB) || !frame.UseGas(6*toWordSize(uint64(len(ret)))) {
				err = ErrOutOfGas
			}
		}
	}
	left = frame.Gas
	if err != nil {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		if !errors.Is(err, ErrExecutionReverted) {
			ret = nil
		}
		return ret, types.Address{}, left, err
	}
	evm.StateDB.SetCode(address, ret)
	if tracker, ok := evm.balTracker.(interface {
		RecordCodeChange(types.Address, []byte, []byte)
	}); ok {
		tracker.RecordCodeChange(address, nil, ret)
	}
	return ret, address, left, nil
}
