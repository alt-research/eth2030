package execution

import (
	"bytes"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"sort"
)

// FrameDependency identifies a pre-state read made by validation, including
// reads in reverted child calls. Values are always taken from the original
// snapshot, never from deployment/escrow effects in the disposable simulation.
type FrameDependency struct {
	Address types.Address
	Kind    uint8
	Slot    types.Hash
}

const (
	FrameBalanceDependency uint8 = iota
	FrameNonceDependency
	FrameCodeDependency
	FrameExistenceDependency
	FrameStorageDependency
)

func (d FrameDependency) Value(db state.StateDB) string {
	switch d.Kind {
	case FrameBalanceDependency:
		return db.GetBalance(d.Address).String()
	case FrameNonceDependency:
		return new(big.Int).SetUint64(db.GetNonce(d.Address)).String()
	case FrameCodeDependency:
		return string(db.GetCode(d.Address))
	case FrameExistenceDependency:
		if db.Exist(d.Address) {
			return "1"
		}
		return "0"
	case FrameStorageDependency:
		v := db.GetState(d.Address, d.Slot)
		return string(v[:])
	default:
		panic("unknown Frame dependency")
	}
}

type FramePrefixValidation struct {
	Profile      string // caller-selected execution/activation profile
	Payer        types.Address
	Dependencies map[FrameDependency]string
}

func (v *FramePrefixValidation) Unchanged(db state.StateDB) bool {
	if v == nil {
		return false
	}
	for dep, value := range v.Dependencies {
		if dep.Value(db) != value {
			return false
		}
	}
	return state.ReadError(db) == nil
}

type frameReadSet struct {
	state.StateDB
	reads map[FrameDependency]struct{}
}

func (s *frameReadSet) record(addr types.Address, kind uint8, slot types.Hash) {
	s.reads[FrameDependency{addr, kind, slot}] = struct{}{}
}
func (s *frameReadSet) account(addr types.Address) {
	for _, kind := range []uint8{FrameBalanceDependency, FrameNonceDependency, FrameCodeDependency, FrameExistenceDependency} {
		s.record(addr, kind, types.Hash{})
	}
}
func (s *frameReadSet) GetBalance(addr types.Address) *big.Int {
	s.record(addr, FrameBalanceDependency, types.Hash{})
	return s.StateDB.GetBalance(addr)
}
func (s *frameReadSet) GetNonce(addr types.Address) uint64 {
	s.record(addr, FrameNonceDependency, types.Hash{})
	return s.StateDB.GetNonce(addr)
}
func (s *frameReadSet) GetCode(addr types.Address) []byte {
	s.record(addr, FrameCodeDependency, types.Hash{})
	return s.StateDB.GetCode(addr)
}
func (s *frameReadSet) GetCodeHash(addr types.Address) types.Hash {
	s.record(addr, FrameCodeDependency, types.Hash{})
	return s.StateDB.GetCodeHash(addr)
}
func (s *frameReadSet) GetCodeSize(addr types.Address) int {
	s.record(addr, FrameCodeDependency, types.Hash{})
	return s.StateDB.GetCodeSize(addr)
}
func (s *frameReadSet) Exist(addr types.Address) bool {
	s.record(addr, FrameExistenceDependency, types.Hash{})
	return s.StateDB.Exist(addr)
}
func (s *frameReadSet) Empty(addr types.Address) bool { s.account(addr); return s.StateDB.Empty(addr) }
func (s *frameReadSet) GetState(addr types.Address, slot types.Hash) types.Hash {
	s.record(addr, FrameStorageDependency, slot)
	return s.StateDB.GetState(addr, slot)
}
func (s *frameReadSet) GetCommittedState(addr types.Address, slot types.Hash) types.Hash {
	s.record(addr, FrameStorageDependency, slot)
	return s.StateDB.GetCommittedState(addr, slot)
}
func (s *frameReadSet) FinalizePreState() {
	if f, ok := s.StateDB.(interface{ FinalizePreState() }); ok {
		f.FinalizePreState()
	}
}
func (s *frameReadSet) Error() error { return state.ReadError(s.StateDB) }

// OrderedFrameDependencies provides deterministic index/evidence iteration.
func OrderedFrameDependencies(v *FramePrefixValidation) []FrameDependency {
	keys := make([]FrameDependency, 0, len(v.Dependencies))
	for d := range v.Dependencies {
		keys = append(keys, d)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.Address != b.Address {
			return bytes.Compare(a.Address[:], b.Address[:]) < 0
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return bytes.Compare(a.Slot[:], b.Slot[:]) < 0
	})
	return keys
}
