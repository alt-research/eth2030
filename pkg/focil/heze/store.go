package heze

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/eth2030/eth2030/core/types"
)

type Key struct {
	Slot          uint64
	DependentRoot types.Hash
}
type member struct {
	First       SignedInclusionList
	Second      *SignedInclusionList
	Timely      bool
	GossipCount uint8
}
type Store struct {
	mu      sync.RWMutex
	entries map[Key]map[uint64]*member
}

func NewStore() *Store        { return &Store{entries: make(map[Key]map[uint64]*member)} }
func key(m InclusionList) Key { return Key{m.Slot, m.DependentRoot} }
func equal(a, b InclusionList) bool {
	if a.Slot != b.Slot || a.DependentRoot != b.DependentRoot || a.ValidatorIndex != b.ValidatorIndex || len(a.Transactions) != len(b.Transactions) {
		return false
	}
	for i := range a.Transactions {
		if !bytes.Equal(a.Transactions[i], b.Transactions[i]) {
			return false
		}
	}
	return true
}
func (s *Store) gossipCount(m InclusionList) uint8 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e := s.entries[key(m)][m.ValidatorIndex]; e != nil {
		return e.GossipCount
	}
	return 0
}

// process follows Heze process_inclusion_list: first arrival fixes timeliness;
// a later distinct message proves equivocation even after the view deadline.
// It is only called after the complete consensus validation boundary.
func (s *Store) process(il SignedInclusionList, timely, gossip bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(il.Message)
	if s.entries[k] == nil {
		s.entries[k] = make(map[uint64]*member)
	}
	e := s.entries[k][il.Message.ValidatorIndex]
	if e != nil && gossip && e.GossipCount >= 2 {
		return &ValidationError{Ignore, "already seen two valid inclusion lists"}
	}
	if e == nil {
		e = &member{First: il.Clone(), Timely: timely}
		s.entries[k][il.Message.ValidatorIndex] = e
	} else if e.Second == nil && !equal(e.First.Message, il.Message) {
		copied := il.Clone()
		e.Second = &copied
	}
	if gossip {
		e.GossipCount++
	}
	return nil
}

func (s *Store) Transactions(k Key, onlyTimely bool) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[string]bool)
	out := make([][]byte, 0)
	indices := make([]uint64, 0, len(s.entries[k]))
	for i := range s.entries[k] {
		indices = append(indices, i)
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })
	for _, i := range indices {
		e := s.entries[k][i]
		if e.Second != nil || onlyTimely && !e.Timely {
			continue
		}
		for _, tx := range e.First.Message.Transactions {
			if !seen[string(tx)] {
				seen[string(tx)] = true
				out = append(out, bytes.Clone(tx))
			}
		}
	}
	return out
}
func (s *Store) Bits(k Key, committee Committee, onlyTimely bool) Bits {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out Bits
	for i, v := range committee {
		e := s.entries[k][v]
		if e != nil && e.Second == nil && (!onlyTimely || e.Timely) {
			out.Set(i)
		}
	}
	return out
}
func (s *Store) Evidence(k Key, index uint64) (SignedInclusionList, *SignedInclusionList, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e := s.entries[k][index]
	if e == nil {
		return SignedInclusionList{}, nil, false
	}
	var second *SignedInclusionList
	if e.Second != nil {
		v := e.Second.Clone()
		second = &v
	}
	return e.First.Clone(), second, true
}
func (s *Store) ByIndices(req Request, committee Committee, currentSlot, forkSlot uint64) ([]SignedInclusionList, error) {
	minimum := uint64(0)
	if currentSlot > MinRequestSlots {
		minimum = currentSlot - MinRequestSlots
	}
	if minimum < forkSlot {
		minimum = forkSlot
	}
	if req.Slot < minimum || req.Slot > currentSlot {
		return nil, ErrResourceUnavailable
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SignedInclusionList, 0)
	seen := make(map[uint64]bool)
	for i, v := range committee {
		if !req.Indices.Has(i) || seen[v] {
			continue
		}
		seen[v] = true
		e := s.entries[Key{req.Slot, req.DependentRoot}][v]
		if e != nil && e.Second == nil {
			out = append(out, e.First.Clone())
		}
	}
	return out, nil
}

// Prune retains N through N+MIN_REQUEST_SLOTS+1 inclusive, including the extra
// payload-decision slot. Branches sharing a slot remain independent.
func (s *Store) Prune(currentSlot uint64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	if currentSlot <= MinRequestSlots+1 {
		return n
	}
	minimum := currentSlot - MinRequestSlots - 1
	for k, v := range s.entries {
		if k.Slot < minimum {
			n += len(v)
			delete(s.entries, k)
		}
	}
	return n
}

type diskMember struct {
	First       SignedInclusionList
	Second      *SignedInclusionList
	Timely      bool
	GossipCount uint8
}
type diskStore struct {
	Version int
	Entries []diskMember
}

func (s *Store) MarshalBinary() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := diskStore{Version: 1, Entries: make([]diskMember, 0)}
	for _, v := range s.entries {
		for _, e := range v {
			d.Entries = append(d.Entries, diskMember{e.First, e.Second, e.Timely, e.GossipCount})
		}
	}
	sort.Slice(d.Entries, func(i, j int) bool {
		a, b := d.Entries[i].First.Message, d.Entries[j].First.Message
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		if a.DependentRoot != b.DependentRoot {
			return bytes.Compare(a.DependentRoot[:], b.DependentRoot[:]) < 0
		}
		return a.ValidatorIndex < b.ValidatorIndex
	})
	return json.Marshal(d)
}

// Restore reauthenticates persisted messages against the supplied consensus
// view. An unavailable beacon state is an error, never an omission excuse.
func Restore(data []byte, view ConsensusView) (*Store, error) {
	var d diskStore
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	if d.Version != 1 {
		return nil, errors.New("heze: unknown store version")
	}
	clock, err := checkedClock(view)
	if err != nil {
		return nil, err
	}
	minimum := uint64(0)
	if clock.Slot > MinRequestSlots+1 {
		minimum = clock.Slot - MinRequestSlots - 1
	}
	s := NewStore()
	for _, e := range d.Entries {
		// Expired cache entries have no remaining consumer. Do not require a
		// CL to recover already-pruned historical state after a long shutdown.
		if e.First.Message.Slot < minimum {
			continue
		}
		if e.First.Message.Slot > clock.Slot {
			return nil, errors.New("heze: future persisted inclusion list")
		}
		if e.GossipCount > 2 {
			return nil, errors.New("heze: corrupt gossip count")
		}
		if _, err := authenticate(e.First, view); err != nil {
			return nil, err
		}
		k := key(e.First.Message)
		if s.entries[k] == nil {
			s.entries[k] = make(map[uint64]*member)
		}
		if s.entries[k][e.First.Message.ValidatorIndex] != nil {
			return nil, errors.New("heze: duplicate persisted member")
		}
		if e.Second != nil {
			if _, err := authenticate(*e.Second, view); err != nil {
				return nil, err
			}
			if key(e.Second.Message) != k || e.Second.Message.ValidatorIndex != e.First.Message.ValidatorIndex || equal(e.First.Message, e.Second.Message) {
				return nil, errors.New("heze: corrupt equivocation evidence")
			}
		}
		first := e.First.Clone()
		var second *SignedInclusionList
		if e.Second != nil {
			v := e.Second.Clone()
			second = &v
		}
		s.entries[k][first.Message.ValidatorIndex] = &member{first, second, e.Timely, e.GossipCount}
	}
	return s, nil
}
