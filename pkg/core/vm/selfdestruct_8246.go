package vm

import (
	"bytes"
	"math/big"
	"slices"

	"github.com/eth2030/eth2030/core/types"
)

func (evm *EVM) isEIP8246() bool {
	return evm.forkRules.IsAmsterdam || evm.forkRules.IsGlamsterdan
}

func (evm *EVM) markContractCreated(address types.Address) {
	if evm.frame8141 != nil {
		evm.frame8141.created[address] = true
		return
	}
	if !evm.isEIP8246() && !evm.forkRules.IsCancun {
		return
	}
	if evm.createdContracts == nil {
		evm.createdContracts = make(map[types.Address]bool)
	}
	if !evm.createdContracts[address] {
		evm.createdContracts[address] = true
		evm.createdOrder = append(evm.createdOrder, address)
	}
}

func (evm *EVM) createdInTransaction(address types.Address) bool {
	if evm.frame8141 != nil {
		return evm.frame8141.created[address]
	}
	return evm.createdContracts[address]
}

func (evm *EVM) revertCreationSnapshot(snapshot, createdSnapshot int) {
	evm.StateDB.RevertToSnapshot(snapshot)
	for _, address := range evm.createdOrder[createdSnapshot:] {
		delete(evm.createdContracts, address)
	}
	evm.createdOrder = evm.createdOrder[:createdSnapshot]
}

// FinalizeSelfDestructs applies EIP-8246 after all transaction credits (including
// fee settlement), before state finalization and BAL reconciliation. SELFDESTRUCT
// itself leaves code, storage and nonce observable until this boundary.
func (evm *EVM) FinalizeSelfDestructs() {
	defer evm.finalizeCodeAdoptions8298()
	if !evm.isEIP8246() || evm.StateDB == nil {
		return
	}
	addresses := slices.Clone(evm.createdOrder)
	if evm.frame8141 != nil {
		for address := range evm.frame8141.created {
			addresses = append(addresses, address)
		}
	}
	slices.SortFunc(addresses, func(a, b types.Address) int { return bytes.Compare(a[:], b[:]) })
	for _, address := range addresses {
		if !evm.StateDB.HasSelfDestructed(address) {
			continue
		}
		balance := new(big.Int).Set(evm.StateDB.GetBalance(address))
		// CreateAccount resets nonce, code and all storage in every native backend.
		// Retain a deletion marker only for an empty resulting account (EIP-161).
		evm.StateDB.CreateAccount(address)
		if balance.Sign() != 0 {
			evm.StateDB.AddBalance(address, balance)
		} else {
			evm.StateDB.SelfDestruct(address)
		}
	}
}

func (evm *EVM) recordCreationNonce(address types.Address, old, new uint64) {
	if tracker, ok := evm.balTracker.(interface {
		RecordNonceChange(types.Address, uint64, uint64)
	}); ok {
		tracker.RecordNonceChange(address, old, new)
	}
}
