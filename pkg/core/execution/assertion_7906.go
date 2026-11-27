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

type assertionSnapshot7906 struct {
	id      int
	version uint64
}

// assertionState7906 retains pre-values before the first transaction write.
// Its observations survive child reverts; net changes are materialized only on
// the first ASSERT read. Storage wipes do not enumerate untouched slots.
type assertionState7906 struct {
	state.StateDB
	balances  map[types.Address]*big.Int
	nonces    map[types.Address]uint64
	codes     map[types.Address]types.Hash
	slots     map[assertionSlot7906]types.Hash
	trace     *vm.AssertionTrace7906
	txHash    types.Hash
	version   uint64
	snapshots []assertionSnapshot7906
}

func newAssertionState7906(db state.StateDB, hash types.Hash) *assertionState7906 {
	return &assertionState7906{StateDB: db, balances: make(map[types.Address]*big.Int), nonces: make(map[types.Address]uint64), codes: make(map[types.Address]types.Hash), slots: make(map[assertionSlot7906]types.Hash), txHash: hash}
}

func (s *assertionState7906) changed() { s.version++; s.trace = nil }

func (s *assertionState7906) Snapshot() int {
	id := s.StateDB.Snapshot()
	s.snapshots = append(s.snapshots, assertionSnapshot7906{id, s.version})
	return id
}

func (s *assertionState7906) RevertToSnapshot(id int) {
	// Native journals issue ordered revision IDs. Binary lookup and suffix
	// truncation avoid scanning every successful earlier call on each revert.
	at := sort.Search(len(s.snapshots), func(i int) bool { return s.snapshots[i].id >= id })
	unchanged := at < len(s.snapshots) && s.snapshots[at].id == id && s.snapshots[at].version == s.version
	s.StateDB.RevertToSnapshot(id)
	// ASSERT subcalls cannot write the observable diff. Catching a child
	// revert must retain the index: otherwise a flat-priced TXTRACE loop can
	// repeatedly rebuild every touched account, slot and log. Body rollback
	// and snapshots predating this journal still invalidate conservatively.
	if !unchanged {
		s.changed()
	}
	s.snapshots = s.snapshots[:at]
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
	if v.Sign() != 0 {
		s.changed()
	}
	s.balance(a)
	s.StateDB.AddBalance(a, v)
}
func (s *assertionState7906) SubBalance(a types.Address, v *big.Int) {
	if v.Sign() != 0 {
		s.changed()
	}
	s.balance(a)
	s.StateDB.SubBalance(a, v)
}
func (s *assertionState7906) SetNonce(a types.Address, v uint64) {
	s.changed()
	s.nonce(a)
	s.StateDB.SetNonce(a, v)
}
func (s *assertionState7906) SetCode(a types.Address, v []byte) {
	s.changed()
	s.code(a)
	s.StateDB.SetCode(a, v)
}
func (s *assertionState7906) CreateAccount(a types.Address) {
	s.changed()
	s.balance(a)
	s.nonce(a)
	s.code(a)
	s.StateDB.CreateAccount(a)
}
func (s *assertionState7906) SelfDestruct(a types.Address) {
	s.changed()
	s.balance(a)
	s.StateDB.SelfDestruct(a)
}
func (s *assertionState7906) AddLog(log *types.Log) {
	s.changed()
	s.StateDB.AddLog(log)
}
func (s *assertionState7906) SetState(a types.Address, k, v types.Hash) {
	s.changed()
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

func (s *assertionState7906) FrameRecoveryCode8151(a types.Address) []byte {
	if reader, ok := s.StateDB.(interface{ FrameRecoveryCode8151(types.Address) []byte }); ok {
		return reader.FrameRecoveryCode8151(a)
	}
	return s.StateDB.GetCode(a)
}
