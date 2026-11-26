package execution

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

type assertionSlot7906 struct {
	address types.Address
	key     types.Hash
}

// assertionState7906 retains pre-values before the first transaction write.
// Its observations survive child reverts; net changes are materialized only on
// the first ASSERT read. Storage wipes do not enumerate untouched slots.
type assertionState7906 struct {
	state.StateDB
	balances map[types.Address]*big.Int
	nonces   map[types.Address]uint64
	codes    map[types.Address]types.Hash
	slots    map[assertionSlot7906]types.Hash
	trace    *vm.AssertionTrace7906
	txHash   types.Hash
}

func newAssertionState7906(db state.StateDB, hash types.Hash) *assertionState7906 {
	return &assertionState7906{StateDB: db, balances: make(map[types.Address]*big.Int), nonces: make(map[types.Address]uint64), codes: make(map[types.Address]types.Hash), slots: make(map[assertionSlot7906]types.Hash), txHash: hash}
}
func (s *assertionState7906) balance(a types.Address) {
	if _, ok := s.balances[a]; !ok {
		s.balances[a] = new(big.Int).Set(s.StateDB.GetBalance(a))
	}
}
func (s *assertionState7906) nonce(a types.Address) {
	if _, ok := s.nonces[a]; !ok {
		s.nonces[a] = s.StateDB.GetNonce(a)
	}
}
func normalizedCode7906(db state.StateDB, a types.Address) types.Hash {
	h := db.GetCodeHash(a)
	if h == (types.Hash{}) {
		return types.EmptyCodeHash
	}
	return h
}
func (s *assertionState7906) code(a types.Address) {
	if _, ok := s.codes[a]; !ok {
		s.codes[a] = normalizedCode7906(s.StateDB, a)
	}
}
func (s *assertionState7906) AddBalance(a types.Address, v *big.Int) {
	s.balance(a)
	s.StateDB.AddBalance(a, v)
}
func (s *assertionState7906) SubBalance(a types.Address, v *big.Int) {
	s.balance(a)
	s.StateDB.SubBalance(a, v)
}
func (s *assertionState7906) SetNonce(a types.Address, v uint64) {
	s.nonce(a)
	s.StateDB.SetNonce(a, v)
}
func (s *assertionState7906) SetCode(a types.Address, v []byte) { s.code(a); s.StateDB.SetCode(a, v) }
func (s *assertionState7906) CreateAccount(a types.Address) {
	s.balance(a)
	s.nonce(a)
	s.code(a)
	s.StateDB.CreateAccount(a)
}
func (s *assertionState7906) SelfDestruct(a types.Address) { s.balance(a); s.StateDB.SelfDestruct(a) }
func (s *assertionState7906) SetState(a types.Address, k, v types.Hash) {
	key := assertionSlot7906{a, k}
	if _, ok := s.slots[key]; !ok {
		s.slots[key] = s.StateDB.GetState(a, k)
	}
	s.StateDB.SetState(a, k, v)
}
func (s *assertionState7906) Error() error { return state.ReadError(s.StateDB) }
func (s *assertionState7906) FinalizePreState() {
	if db, ok := s.StateDB.(interface{ FinalizePreState() }); ok {
		db.FinalizePreState()
	}
}
func (s *assertionState7906) RevertToSnapshot(id int) { s.StateDB.RevertToSnapshot(id); s.trace = nil }

func (s *assertionState7906) index() *vm.AssertionTrace7906 {
	if s.trace != nil {
		return s.trace
	}
	t := new(vm.AssertionTrace7906)
	for a, old := range s.balances {
		next := s.StateDB.GetBalance(a)
		if old.Cmp(next) != 0 {
			t.Balances = append(t.Balances, vm.AssertionChange7906{Address: a, Before: new(big.Int).Set(old), After: new(big.Int).Set(next)})
		}
	}
	for a, old := range s.nonces {
		next := s.StateDB.GetNonce(a)
		if old != next {
			t.Nonces = append(t.Nonces, vm.AssertionChange7906{Address: a, Before: new(big.Int).SetUint64(old), After: new(big.Int).SetUint64(next)})
		}
	}
	for a, old := range s.codes {
		next := normalizedCode7906(s.StateDB, a)
		if old != next {
			t.Codes = append(t.Codes, vm.AssertionChange7906{Address: a, Before: new(big.Int).SetBytes(old[:]), After: new(big.Int).SetBytes(next[:])})
		}
	}
	for key, old := range s.slots {
		next := s.StateDB.GetState(key.address, key.key)
		if old != next {
			t.Slots = append(t.Slots, vm.AssertionChange7906{Address: key.address, Key: key.key, Before: new(big.Int).SetBytes(old[:]), After: new(big.Int).SetBytes(next[:])})
		}
	}
	for _, table := range [][]vm.AssertionChange7906{t.Balances, t.Nonces, t.Codes, t.Slots} {
		sort.Slice(table, func(i, j int) bool {
			if cmp := bytes.Compare(table[i].Address[:], table[j].Address[:]); cmp != 0 {
				return cmp < 0
			}
			return bytes.Compare(table[i].Key[:], table[j].Key[:]) < 0
		})
	}
	for _, log := range s.StateDB.GetLogs(s.txHash) {
		copied := *log
		copied.Topics = append([]types.Hash(nil), log.Topics...)
		copied.Data = bytes.Clone(log.Data)
		t.Events = append(t.Events, &copied)
	}
	s.trace = t
	return t
}
