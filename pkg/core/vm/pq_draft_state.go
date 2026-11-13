package vm

import (
	"errors"
	"fmt"
	"maps"

	"github.com/eth2030/eth2030/core/types"
)

var (
	ErrPQDraftAddress = errors.New("PQ draft precompile address is zero, duplicated, or reserved")
	ErrPQDraftState   = errors.New("PQ draft precompiles require world state")
)

// PQDraftPrecompiles installs the explicitly addressed private-chain profile.
// Key accounts hold STOP followed by the raw expanded public key. The account
// address is the last 20 bytes of Keccak256 of the key.
func (evm *EVM) PQDraftPrecompiles(base map[types.Address]PrecompiledContract, recover, nist, eth types.Address, reserved ...types.Address) (map[types.Address]PrecompiledContract, error) {
	if evm.StateDB == nil {
		return nil, ErrPQDraftState
	}
	addresses := [3]types.Address{recover, nist, eth}
	for i, address := range addresses {
		if address.IsZero() {
			return nil, ErrPQDraftAddress
		}
		if _, found := base[address]; found {
			return nil, fmt.Errorf("%w: %s", ErrPQDraftAddress, address)
		}
		for _, used := range addresses[:i] {
			if used == address {
				return nil, fmt.Errorf("%w: %s", ErrPQDraftAddress, address)
			}
		}
		for _, used := range reserved {
			if used == address {
				return nil, fmt.Errorf("%w: %s", ErrPQDraftAddress, address)
			}
		}
	}
	lookup := func(id [20]byte) ([]byte, error) {
		address := types.Address(id)
		evm.StateDB.AddAddressToAccessList(address)
		if evm.balTracker != nil {
			evm.balTracker.RecordAddressTouch(address)
		}
		code := evm.StateDB.GetCode(address)
		if len(code) != EIP8051DraftPublicKeySize+1 || code[0] != byte(STOP) {
			return nil, errors.New("PQ draft public key account is missing or malformed")
		}
		return code[1:], nil
	}
	nistAlgorithm, err := NewEIP8051ExpandedAlgorithm(EIP8051DraftNIST, lookup)
	if err != nil {
		return nil, err
	}
	ethAlgorithm, err := NewEIP8051ExpandedAlgorithm(EIP8051DraftETH, lookup)
	if err != nil {
		return nil, err
	}
	registry, err := NewEIP7932DraftSigRecover(nistAlgorithm, ethAlgorithm)
	if err != nil {
		return nil, err
	}
	contracts := make(map[types.Address]PrecompiledContract, len(base)+3)
	maps.Copy(contracts, base)
	contracts[recover] = statePQRecovery{evm: evm, EIP7932DraftSigRecover: registry}
	contracts[nist], err = NewEIP8051ExpandedPrecompile(EIP8051DraftNIST)
	if err != nil {
		return nil, err
	}
	contracts[eth], err = NewEIP8051ExpandedPrecompile(EIP8051DraftETH)
	if err != nil {
		return nil, err
	}
	return contracts, nil
}

// statePQRecovery charges key lookup separately from the draft verification
// penalty. Access pricing follows the active EVM's account access schedule.
// The private profile charges fixed copy gas for the full expanded key.
type statePQRecovery struct {
	evm *EVM
	*EIP7932DraftSigRecover
}

func (p statePQRecovery) RequiredGas(input []byte) uint64 {
	cost := p.EIP7932DraftSigRecover.RequiredGas(input)
	if len(input) != EIP8051DraftSignatureInfoSize+EIP8051DraftMessageSize ||
		(input[0] != EIP8051DraftNIST && input[0] != EIP8051DraftETH) {
		return cost
	}
	address := types.BytesToAddress(input[EIP8051DraftSignatureInfoSize-20 : EIP8051DraftSignatureInfoSize])
	warm, cold := WarmStorageReadCost, ColdAccountAccessCost
	if p.evm.forkRules.IsGlamsterdan {
		warm, cold = WarmStorageReadGlamst, ColdAccountAccessGlamst
	}
	accessCost := cold
	if p.evm.StateDB.AddressInAccessList(address) {
		accessCost = warm
	}
	return cost + accessCost + GasCopy*((EIP8051DraftPublicKeySize+1+31)/32)
}
