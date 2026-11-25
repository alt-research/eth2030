package vm

import (
	"github.com/eth2030/eth2030/core/types"
)

// Amsterdam state access is installed separately from experimental later
// compute repricings. The canonical activation selects these costs by default.
func installAmsterdamGas(table *JumpTable) {
	table[SLOAD] = &operation{execute: opSload, constantGas: 100, dynamicGas: gasAmsterdamSload, minStack: 1, maxStack: 1024}
	table[BALANCE] = &operation{execute: opBalance, constantGas: 100, dynamicGas: gasAmsterdamAccount, minStack: 1, maxStack: 1024}
	table[EXTCODEHASH] = &operation{execute: opExtcodehash, constantGas: 100, dynamicGas: gasAmsterdamAccount, minStack: 1, maxStack: 1024}
	table[EXTCODESIZE] = &operation{execute: opExtcodesize, constantGas: 200, dynamicGas: gasAmsterdamAccount, minStack: 1, maxStack: 1024}
	table[EXTCODECOPY] = &operation{execute: opExtcodecopy, constantGas: 200, dynamicGas: gasAmsterdamExtCopy, minStack: 4, maxStack: 1024, memorySize: memoryExtcodecopy}
	table[SSTORE] = &operation{execute: opAmsterdamSstore, dynamicGas: gasAmsterdamSstore, minStack: 2, maxStack: 1024, writes: true}
	table[CALL] = &operation{execute: opCall, constantGas: 100, dynamicGas: gasAmsterdamCall, minStack: 7, maxStack: 1024, memorySize: memoryCall}
	table[CALLCODE] = &operation{execute: opCallCode, constantGas: 100, dynamicGas: gasAmsterdamCallCode, minStack: 7, maxStack: 1024, memorySize: memoryCallCode}
	table[DELEGATECALL] = &operation{execute: opDelegateCall, constantGas: 100, dynamicGas: gasAmsterdamDelegate, minStack: 6, maxStack: 1024, memorySize: memoryDelegateCall}
	table[STATICCALL] = &operation{execute: opStaticCall, constantGas: 100, dynamicGas: gasAmsterdamDelegate, minStack: 6, maxStack: 1024, memorySize: memoryStaticCall}
	table[CREATE] = &operation{execute: opCreate, constantGas: AmsterdamCreateAccess, dynamicGas: gasCreateDynamic, minStack: 3, maxStack: 1024, memorySize: memoryCreate, writes: true}
	table[CREATE2] = &operation{execute: opCreate2, constantGas: AmsterdamCreateAccess, dynamicGas: gasCreate2Dynamic, minStack: 4, maxStack: 1024, memorySize: memoryCreate2, writes: true}
	table[SELFDESTRUCT] = &operation{execute: opSelfdestruct, constantGas: GasSelfdestruct, dynamicGas: gasAmsterdamSelfdestruct, minStack: 1, maxStack: 1024, halts: true, writes: true}
}
func amsterdamAccountAccess(evm *EVM, address types.Address) uint64 {
	if evm.StateDB == nil || evm.StateDB.AddressInAccessList(address) {
		return 0
	}
	evm.StateDB.AddAddressToAccessList(address)
	return AmsterdamColdAccount - AmsterdamWarmAccess
}
func gasAmsterdamAccount(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	return amsterdamAccountAccess(evm, bigToAddress(s.Back(0))), nil
}
func gasAmsterdamExtCopy(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	memory, err := gasMemExpansion(evm, c, s, m, size)
	if err != nil {
		return 0, err
	}
	return safeAdd(safeAdd(amsterdamAccountAccess(evm, bigToAddress(s.Back(0))), safeMul(GasCopy, toWordSize(s.Back(3).Uint64()))), memory), nil
}
func gasAmsterdamSload(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	return gasEIP2929SlotCheck(evm, c.Address, bigToHash(s.Back(0))), nil
}
func gasAmsterdamSstore(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	if c.Gas <= CallStipend {
		return 0, ErrOutOfGas
	}
	if evm.StateDB == nil {
		return 100, nil
	}
	key := bigToHash(s.Back(0))
	_, warm := evm.StateDB.SlotInAccessList(c.Address, key)
	access := uint64(100)
	if !warm {
		access = 2100
		evm.StateDB.AddSlotToAccessList(c.Address, key)
	}
	current, original, next := evm.StateDB.GetState(c.Address, key), evm.StateDB.GetCommittedState(c.Address, key), bigToHash(s.Back(1))
	if evm.balTracker != nil {
		evm.balTracker.RecordStorageRead(c.Address, key, current)
	}
	if next != current && current == original {
		access += AmsterdamStorageWrite
	}
	return access, nil
}
func opAmsterdamSstore(pc *uint64, evm *EVM, c *Contract, m *Memory, s *Stack) ([]byte, error) {
	if evm.readOnly {
		return nil, ErrWriteProtection
	}
	key, value := bigToHash(s.Pop()), bigToHash(s.Pop())
	if evm.StateDB == nil {
		return nil, nil
	}
	current, original := evm.StateDB.GetState(c.Address, key), evm.StateDB.GetCommittedState(c.Address, key)
	zero := types.Hash{}
	// Read evidence survives a post-state OOG; rolled-back writes do not become
	// final BAL diffs. Charge state at the opcode boundary before publication.
	if evm.balTracker != nil {
		evm.balTracker.RecordStorageRead(c.Address, key, current)
	}
	if original == zero && current == zero && value != zero {
		if !c.ChargeStateGas(AmsterdamNewSlotStateGas) {
			return nil, ErrOutOfGas
		}
	}
	if original == zero && current != zero && value == zero {
		c.RefillStateGas(AmsterdamNewSlotStateGas)
	}
	if original != zero && current != zero && value == zero {
		evm.StateDB.AddRefund(AmsterdamStorageClearRefund)
	}
	if original != zero && current == zero && value != zero {
		evm.StateDB.SubRefund(AmsterdamStorageClearRefund)
	}
	if value == original && value != current {
		evm.StateDB.AddRefund(AmsterdamStorageWrite)
	}
	evm.StateDB.SetState(c.Address, key, value)
	if evm.balTracker != nil && current != value {
		evm.balTracker.RecordStorageChange(c.Address, key, current, value)
	}
	return nil, nil
}
func gasAmsterdamCallBase(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64, valueBearing, canCreate bool) (uint64, error) {
	address := bigToAddress(s.Back(1))
	cost := amsterdamAccountAccess(evm, address)
	c.pendingAccountStateGas = 0
	if valueBearing {
		cost = safeAdd(cost, AmsterdamAccountWrite+CallStipend)
	}
	memory, err := gasMemExpansion(evm, c, s, m, size)
	if err != nil {
		return 0, err
	}
	cost = safeAdd(cost, memory)
	if c.Gas < cost {
		return 0, ErrOutOfGas
	}
	cost, err = gasDelegationResolution(evm, c, address, cost)
	if err != nil {
		return 0, err
	}

	if canCreate && valueBearing && evm.StateDB != nil && evm.StateDB.Empty(address) {
		if !c.ChargeStateGas(AmsterdamNewAccountStateGas) {
			return 0, ErrOutOfGas
		}
		c.pendingAccountStateGas = AmsterdamNewAccountStateGas
	}
	// The account charge precedes the 63/64 split. The constant warm cost was
	// already paid by the interpreter before this dynamic-gas function.
	evm.callGasTemp = callGasForWord(c.Gas, cost, s.Back(0))
	return safeAdd(cost, evm.callGasTemp), nil
}
func gasAmsterdamCall(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	return gasAmsterdamCallBase(evm, c, s, m, size, s.Back(2).Sign() != 0, true)
}
func gasAmsterdamCallCode(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	return gasAmsterdamCallBase(evm, c, s, m, size, s.Back(2).Sign() != 0, false)
}
func gasAmsterdamDelegate(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	return gasAmsterdamCallBase(evm, c, s, m, size, false, false)
}
func gasAmsterdamSelfdestruct(evm *EVM, c *Contract, s *Stack, m *Memory, size uint64) (uint64, error) {
	address := bigToAddress(s.Back(0))
	cost := uint64(0)
	if evm.StateDB == nil {
		return 0, nil
	}
	if !evm.StateDB.AddressInAccessList(address) {
		evm.StateDB.AddAddressToAccessList(address)
		cost = AmsterdamColdAccount
	}
	if c.Gas < cost {
		return 0, ErrOutOfGas
	}
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(address)
	}
	if evm.StateDB.GetBalance(c.Address).Sign() > 0 && evm.StateDB.Empty(address) {
		cost += AmsterdamAccountWrite
		if !c.ChargeStateGas(AmsterdamNewAccountStateGas) {
			return 0, ErrOutOfGas
		}
	}
	return cost, nil
}

func (evm *EVM) executionCode(address types.Address) []byte {
	code := evm.StateDB.GetCode(address)
	if evm.frame8141 != nil || evm.forkRules.IsPrague || evm.forkRules.IsAmsterdam || evm.forkRules.IsGlamsterdan {
		if delegate, ok := types.ParseDelegation(code); ok {
			if evm.balTracker != nil {
				evm.balTracker.RecordAddressTouch(delegate)
			}
			return evm.StateDB.GetCode(delegate)
		}
	}
	return code
}
