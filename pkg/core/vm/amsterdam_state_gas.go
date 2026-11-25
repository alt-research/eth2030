package vm

import (
	"errors"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
)

// Fixed parameters of EIPs 2780/8037/8038 at EIPs commit 3db38a3. Keep the
// state-byte price independent of block gas-limit changes in this revision.
const (
	AmsterdamCPSB               uint64 = 1530
	AmsterdamNewAccountStateGas uint64 = 120 * AmsterdamCPSB
	AmsterdamNewSlotStateGas    uint64 = 64 * AmsterdamCPSB
	AmsterdamDelegationStateGas uint64 = 23 * AmsterdamCPSB
	AmsterdamMaxExecutionGas    uint64 = 1 << 24
	AmsterdamMaxTotalGas        uint64 = 1<<32 - 1
	AmsterdamTxBase             uint64 = 12000
	AmsterdamTxValue            uint64 = 6000
	AmsterdamColdAccount        uint64 = 3000
	AmsterdamWarmAccess         uint64 = 100
	AmsterdamAccountWrite       uint64 = 9000
	AmsterdamStorageWrite       uint64 = 10000
	AmsterdamCreateAccess       uint64 = 12000
	AmsterdamAuthorizationBase  uint64 = 7816
	AmsterdamStorageClearRefund uint64 = 11616
	AmsterdamAccessListAddress  uint64 = 2900
	AmsterdamAccessListStorage  uint64 = 2000
)

// ChargeStateGas is atomic on failure: state gas drains the reservoir before
// borrowing execution gas. Regular opcode charges continue using UseGas.
func (c *Contract) ChargeStateGas(amount uint64) bool {
	fromExecution := uint64(0)
	if amount > c.StateGasReservoir {
		fromExecution = amount - c.StateGasReservoir
		if fromExecution > c.Gas {
			return false
		}
	}
	c.Gas -= fromExecution
	c.StateGasFromGasLeft += fromExecution
	if amount > c.StateGasReservoir {
		c.StateGasReservoir = 0
	} else {
		c.StateGasReservoir -= amount
	}
	c.StateGasUsed += int64(amount)
	return true
}

// RefillStateGas undoes a state creation, repaying this frame's most recent
// execution borrowing before crediting the state reservoir.
func (c *Contract) RefillStateGas(amount uint64) {
	repaid := min(amount, c.StateGasFromGasLeft)
	c.Gas += repaid
	c.StateGasFromGasLeft -= repaid
	c.StateGasReservoir += amount - repaid
	c.StateGasUsed -= int64(amount)
}

// CommitStateGas preserves authorization state across a later top-level frame
// rollback. It moves no gas and does not change the amount of state consumed.
func (c *Contract) CommitStateGas() {
	c.StateGasCommitted += c.StateGasFromGasLeft
	c.StateGasBaseline = c.StateGasReservoir
	c.StateGasFromGasLeft = 0
}

// RollbackStateGas restores the frame-entry reservoir. Refilling net usage is
// insufficient when a child credited state gas that its ancestor paid for.
func (c *Contract) RollbackStateGas(exceptional bool) {
	net := int64(c.StateGasBaseline) - int64(c.StateGasReservoir) + int64(c.StateGasFromGasLeft)
	c.StateGasUsed -= net
	c.Gas += c.StateGasFromGasLeft
	c.StateGasFromGasLeft = 0
	c.StateGasReservoir = c.StateGasBaseline
	if exceptional {
		c.Gas = 0
	}
}

// RestorePreExecutionStateGas also reverses committed authorization charges
// when execution never entered its first frame. The caller rolls back state.
func (c *Contract) RestorePreExecutionStateGas(initialReservoir uint64) {
	c.Gas += c.StateGasFromGasLeft + c.StateGasCommitted
	c.StateGasFromGasLeft = 0
	c.StateGasCommitted = 0
	c.StateGasReservoir = initialReservoir
	c.StateGasBaseline = initialReservoir
	c.StateGasUsed = 0
}

// MergeStateGas receives a completed child. Failed children must first restore
// their baseline; successful children's borrowing follows their merged gas.
func (c *Contract) MergeStateGas(child *Contract) {
	c.Gas += child.Gas
	c.StateGasReservoir = child.StateGasReservoir
	c.StateGasFromGasLeft += child.StateGasFromGasLeft
	c.StateGasUsed += child.StateGasUsed
	repaid := min(c.StateGasReservoir, c.StateGasFromGasLeft)
	c.Gas += repaid
	c.StateGasReservoir -= repaid
	c.StateGasFromGasLeft -= repaid
}

func (evm *EVM) amsterdamStateGasActive() bool {
	return evm.forkRules.IsAmsterdam && evm.frame8141 == nil && evm.FrameCtx == nil
}
func stateGasCopy(c *Contract) *Contract {
	return &Contract{Gas: c.Gas, StateGasReservoir: c.StateGasReservoir, StateGasFromGasLeft: c.StateGasFromGasLeft, StateGasBaseline: c.StateGasBaseline, StateGasCommitted: c.StateGasCommitted, StateGasUsed: c.StateGasUsed}
}

// SetTransactionStateGas carries pre-execution charges and its authorization
// commit into the top-level frame. Fork activation, not a runtime toggle, selects
// this accounting. The caller has already checked the transaction gas limits.
func (evm *EVM) SetTransactionStateGas(c *Contract) { evm.initialStateGas = stateGasCopy(c) }
func (evm *EVM) TransactionStateGas() *Contract {
	if evm.returnedStateGas == nil {
		return nil
	}
	return stateGasCopy(evm.returnedStateGas)
}
func (evm *EVM) beginStateGasFrame(caller, address types.Address, value *big.Int, gas uint64) *Contract {
	if !evm.amsterdamStateGasActive() {
		return nil
	}
	c := NewContract(caller, address, value, gas)
	if evm.depth == 0 && evm.initialStateGas != nil {
		initial := evm.initialStateGas
		evm.initialStateGas = nil
		c.StateGasReservoir = initial.StateGasReservoir
		c.StateGasFromGasLeft = initial.StateGasFromGasLeft
		c.StateGasBaseline = initial.StateGasBaseline
		c.StateGasCommitted = initial.StateGasCommitted
		c.StateGasUsed = initial.StateGasUsed
	} else {
		c.StateGasReservoir = evm.callReservoirForward
		c.StateGasBaseline = c.StateGasReservoir
	}
	evm.callReservoirForward = 0
	return c
}
func (evm *EVM) finishStateGasFrame(c *Contract, left *uint64, err error) {
	c.Gas = *left
	if err != nil {
		c.RollbackStateGas(!errors.Is(err, ErrExecutionReverted))
	}
	*left = c.Gas
	evm.callReservoirReturn = c.StateGasReservoir
	evm.returnedStateGas = stateGasCopy(c)
}
func (evm *EVM) forwardStateGas(c *Contract) {
	evm.callReservoirForward = c.StateGasReservoir
	evm.callReservoirReturn = c.StateGasReservoir
	evm.returnedStateGas = &Contract{StateGasReservoir: c.StateGasReservoir, StateGasBaseline: c.StateGasReservoir}
	c.StateGasReservoir = 0
}
func (evm *EVM) absorbStateGas(c *Contract, returned uint64, err error) {
	child := evm.returnedStateGas
	if child == nil {
		child = &Contract{StateGasReservoir: evm.callReservoirReturn}
	}
	child.Gas = returned
	c.MergeStateGas(child)
	if err != nil && c.pendingAccountStateGas != 0 {
		c.RefillStateGas(c.pendingAccountStateGas)
	}
	c.pendingAccountStateGas = 0
	evm.callReservoirForward = 0
	evm.callReservoirReturn = 0
	evm.returnedStateGas = nil
}
