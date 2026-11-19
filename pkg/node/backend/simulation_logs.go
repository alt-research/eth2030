package backend

import (
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math/big"
)

// simulationLogState observes execution order, including virtual transfer logs.
// Only actual AddLog calls reach StateDB, so virtual events never enter receipts.
type simulationLogState struct {
	state.StateDB
	synthetic bool
	events    []*types.Log
	snapshots map[int]int
	hash      types.Hash
	index     int
}

func (s *simulationLogState) beginCall() { s.events = nil; s.snapshots = make(map[int]int) }
func (s *simulationLogState) Snapshot() int {
	id := s.StateDB.Snapshot()
	s.snapshots[id] = len(s.events)
	return id
}
func (s *simulationLogState) RevertToSnapshot(id int) {
	s.StateDB.RevertToSnapshot(id)
	if count, ok := s.snapshots[id]; ok {
		s.events = s.events[:count]
	}
	for saved := range s.snapshots {
		if saved >= id {
			delete(s.snapshots, saved)
		}
	}
}
func (s *simulationLogState) AddLog(log *types.Log) {
	s.StateDB.AddLog(log)
	s.events = append(s.events, log)
}
func (s *simulationLogState) SetTxContext(hash types.Hash, index int) {
	s.StateDB.SetTxContext(hash, index)
	s.hash = hash
	s.index = index
}
func (s *simulationLogState) FinalizePreState() {
	if target, ok := s.StateDB.(interface{ FinalizePreState() }); ok {
		target.FinalizePreState()
	}
}
func (s *simulationLogState) FinalizeTransaction() {
	if target, ok := s.StateDB.(interface{ FinalizeTransaction() }); ok {
		target.FinalizeTransaction()
	}
}
func (s *simulationLogState) RecordETHTransfer(from, to types.Address, value *big.Int) {
	if !s.synthetic {
		return
	}
	amount := make([]byte, 32)
	value.FillBytes(amount)
	source, destination := types.Hash{}, types.Hash{}
	copy(source[12:], from[:])
	copy(destination[12:], to[:])
	s.events = append(s.events, &types.Log{Address: types.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"), Topics: []types.Hash{vm.TransferEventTopic, source, destination}, Data: amount, TxHash: s.hash, TxIndex: uint(s.index)})
}
