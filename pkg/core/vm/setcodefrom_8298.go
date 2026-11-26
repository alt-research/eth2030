package vm

import (
	"bytes"
	"maps"

	"github.com/eth2030/eth2030/core/types"
)

// SETCODEFROM adopts existing regular deployed code (EIP-8298). The gas
// parameters are those of its required EIP-8038 profile.
const SETCODEFROM OpCode = 0xa7

type codeAdoptionJournal struct {
	StateDB
	origins   map[types.Address]types.Hash
	snapshots map[int]map[types.Address]types.Hash
}

func (s *codeAdoptionJournal) SetCode(a types.Address, code []byte) {
	delete(s.origins, a)
	s.StateDB.SetCode(a, code)
}
func (s *codeAdoptionJournal) CreateAccount(a types.Address) {
	delete(s.origins, a)
	s.StateDB.CreateAccount(a)
}
func (s *codeAdoptionJournal) Snapshot() int {
	id := s.StateDB.Snapshot()
	s.snapshots[id] = maps.Clone(s.origins)
	return id
}
func (s *codeAdoptionJournal) RevertToSnapshot(id int) {
	s.StateDB.RevertToSnapshot(id)
	s.origins = maps.Clone(s.snapshots[id])
	for key := range s.snapshots {
		if key >= id {
			delete(s.snapshots, key)
		}
	}
}
func (s *codeAdoptionJournal) StorageRoot(a types.Address) types.Hash {
	if roots, ok := s.StateDB.(interface {
		StorageRoot(types.Address) types.Hash
	}); ok {
		return roots.StorageRoot(a)
	}
	return types.Hash{}
}

// System contracts are the fork-specific accounts called by the protocol from
// SYSTEM_ADDRESS. The deposit contract is deliberately absent.
func (evm *EVM) SetSystemContracts8298(addresses []types.Address) {
	evm.systemContracts8298 = make(map[types.Address]bool, len(addresses))
	for _, a := range addresses {
		evm.systemContracts8298[a] = true
	}
}

func (evm *EVM) installCodeAdoptionJournal() {
	if evm.forkRules.IsEIP8298 && evm.StateDB != nil && evm.codeAdoptions == nil {
		journal := &codeAdoptionJournal{StateDB: evm.StateDB, origins: make(map[types.Address]types.Hash), snapshots: make(map[int]map[types.Address]types.Hash)}
		evm.codeAdoptions, evm.StateDB = journal, journal
	}
}

func gasSetCodeFrom8298(evm *EVM, contract *Contract, stack *Stack, _ *Memory, _ uint64) (uint64, error) {
	// Static and system-account guards are also checked before the interpreter's
	// gas calculator, so an exceptional halt cannot read or warm the source.
	source := types.BytesToAddress(stack.Back(0).Bytes())
	cost := uint64(200)
	if !evm.StateDB.AddressInAccessList(source) {
		cost = 3100
	}
	if contract.Gas < cost {
		return 0, ErrOutOfGas
	}
	evm.StateDB.AddAddressToAccessList(source)
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(source)
	}
	hash := evm.StateDB.GetCodeHash(source)
	code := evm.StateDB.GetCode(source) // Raw code; never follow 7702 delegation.
	valid := evm.StateDB.Exist(source) && hash != (types.Hash{}) && hash != types.EmptyCodeHash && len(code) != 0 && code[0] != 0xef && !evm.createdInTransaction(source)
	if !valid {
		return cost, nil
	}
	cost += 100
	if contract.Gas < cost {
		return 0, ErrOutOfGas
	}
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(contract.Address)
	}
	if hash != evm.StateDB.GetCodeHash(contract.Address) {
		cost += 9000
	}
	return cost, nil
}

func opSetCodeFrom8298(_ *uint64, evm *EVM, contract *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	source := types.BytesToAddress(stack.Peek().Bytes())
	hash, code := evm.StateDB.GetCodeHash(source), evm.StateDB.GetCode(source)
	if !evm.StateDB.Exist(source) || hash == (types.Hash{}) || hash == types.EmptyCodeHash || len(code) == 0 || code[0] == 0xef || evm.createdInTransaction(source) {
		stack.Peek().SetUint64(0)
		return nil, nil
	}
	old := evm.StateDB.GetCode(contract.Address)
	if !bytes.Equal(old, code) {
		evm.StateDB.SetCode(contract.Address, bytes.Clone(code))
		if tracker, ok := evm.balTracker.(interface {
			RecordCodeAdoption(types.Address, []byte, types.Hash)
		}); ok {
			tracker.RecordCodeAdoption(contract.Address, old, hash)
		}
	}
	// Even an unchanged hash sets the final origin for any earlier code change.
	evm.codeAdoptions.origins[contract.Address] = hash
	stack.Peek().SetUint64(1)
	return nil, nil
}

func (evm *EVM) adoptedCreationCode(address types.Address) ([]byte, bool) {
	if !evm.forkRules.IsEIP8298 {
		return nil, false
	}
	hash := evm.StateDB.GetCodeHash(address)
	if hash == (types.Hash{}) || hash == types.EmptyCodeHash {
		return nil, false
	}
	return bytes.Clone(evm.StateDB.GetCode(address)), true
}

func (evm *EVM) finalizeCodeAdoptions8298() {
	if evm.codeAdoptions == nil {
		return
	}
	if tracker, ok := evm.balTracker.(interface {
		ReconcileCodeOrigins8298(map[types.Address]types.Hash)
	}); ok {
		tracker.ReconcileCodeOrigins8298(evm.codeAdoptions.origins)
	}
}

// HasOpcode8298 scans instructions, never PUSH immediates. This is shared by
// public admission and dependency classification.
func HasOpcode8298(code []byte, wanted OpCode) bool {
	for pc := 0; pc < len(code); pc++ {
		op := OpCode(code[pc])
		if op == wanted {
			return true
		}
		if op >= PUSH1 && op <= PUSH32 {
			pc += int(op-PUSH1) + 1
		}
	}
	return false
}

// Preserve the paid ecRecover signer-class dependency through code adoption.
func (s *codeAdoptionJournal) FrameRecoveryCode8151(a types.Address) []byte {
	if reader, ok := s.StateDB.(interface{ FrameRecoveryCode8151(types.Address) []byte }); ok {
		return reader.FrameRecoveryCode8151(a)
	}
	return s.StateDB.GetCode(a)
}
