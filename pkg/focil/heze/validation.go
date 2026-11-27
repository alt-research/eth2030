package heze

import (
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

type Disposition uint8

const (
	Ignore Disposition = iota
	Reject
)

type ValidationError struct {
	Disposition Disposition
	Reason      string
}

func (e *ValidationError) Error() string { return "heze: " + e.Reason }

var ErrContextUnavailable = errors.New("heze: authenticated consensus context unavailable")

type Clock struct {
	Slot           uint64
	IntoSlotMS     uint64
	SlotDurationMS uint64
}

// DependentState must be obtained from the requested dependent beacon block,
// processed through the shuffling lookahead start. PossibleDependent includes
// canonical/finality ancestry checks, not merely presence in a block map.
type DependentState struct {
	KnownBlock             bool
	StateAvailable         bool
	BlockSlot              uint64
	ShufflingDependentSlot uint64
	PossibleDependent      bool
	Committee              Committee
	ValidatorPubkeys       map[uint64][48]byte
	ForkVersion            [4]byte
	GenesisValidatorsRoot  [32]byte
}

type ConsensusView interface {
	Clock() (Clock, error)
	DependentState(slot uint64, root types.Hash) (DependentState, error)
}

func authenticate(s SignedInclusionList, view ConsensusView) (DependentState, error) {
	if view == nil {
		return DependentState{}, ErrContextUnavailable
	}
	if err := CheckTransactions(s.Message.Transactions); err != nil {
		return DependentState{}, &ValidationError{Reject, err.Error()}
	}
	state, err := view.DependentState(s.Message.Slot, s.Message.DependentRoot)
	if err != nil {
		return state, fmt.Errorf("%w: %v", ErrContextUnavailable, err)
	}
	if !state.KnownBlock || !state.StateAvailable {
		return state, &ValidationError{Ignore, "dependent block/state unavailable"}
	}
	if state.BlockSlot > state.ShufflingDependentSlot {
		return state, &ValidationError{Reject, "dependent block after shuffling dependent slot"}
	}
	if !state.PossibleDependent {
		return state, &ValidationError{Ignore, "impossible dependent block"}
	}
	member := false
	for _, index := range state.Committee {
		if index == s.Message.ValidatorIndex {
			member = true
			break
		}
	}
	if !member {
		return state, &ValidationError{Reject, "validator not in committee"}
	}
	pubkey, known := state.ValidatorPubkeys[s.Message.ValidatorIndex]
	if !known {
		return state, ErrContextUnavailable
	}
	root := s.Message.SigningRoot(state.ForkVersion, state.GenesisValidatorsRoot)
	if !crypto.BLSVerify(pubkey, root[:], s.Signature) {
		return state, &ValidationError{Reject, "invalid BLS inclusion-list signature"}
	}
	return state, nil
}

func checkedClock(view ConsensusView) (Clock, error) {
	if view == nil {
		return Clock{}, ErrContextUnavailable
	}
	clock, err := view.Clock()
	if err != nil {
		return Clock{}, err
	}
	if clock.SlotDurationMS == 0 || clock.IntoSlotMS >= clock.SlotDurationMS {
		return Clock{}, ErrContextUnavailable
	}
	return clock, nil
}
func timely(clock Clock, slot uint64) bool {
	// Integer millisecond clocks compare against the rational deadline using
	// its ceiling. This also handles configured durations not divisible by 4.
	remainder := clock.SlotDurationMS % 10000 * DueBasisPoints
	due := clock.SlotDurationMS/10000*DueBasisPoints + (remainder+9999)/10000
	return clock.Slot == slot && clock.IntoSlotMS < due
}

// Import executes on_inclusion_list for a previously authenticated or fetched
// message. Unlike gossip, this permits the specified prior-slot request window.
func (s *Store) Import(il SignedInclusionList, view ConsensusView) error {
	clock, err := checkedClock(view)
	if err != nil {
		return err
	}
	if il.Message.Slot > clock.Slot || clock.Slot-il.Message.Slot > MinRequestSlots {
		return &ValidationError{Reject, "IL outside import retention window"}
	}
	if _, err := authenticate(il, view); err != nil {
		return err
	}
	return s.process(il, timely(clock, il.Message.Slot), false)
}

// Gossip executes validate_inclusion_list_gossip and on_inclusion_list. No
// member, key, branch, domain or signature check is optional. Failed validation
// never consumes the two-message allowance or changes an attester's view.
func (s *Store) Gossip(raw []byte, view ConsensusView) error {
	var il SignedInclusionList
	if err := il.UnmarshalSSZ(raw); err != nil {
		return &ValidationError{Reject, err.Error()}
	}
	if s.gossipCount(il.Message) >= 2 {
		return &ValidationError{Ignore, "already seen two valid inclusion lists"}
	}
	clock, err := checkedClock(view)
	if err != nil {
		return err
	}
	if il.Message.Slot != clock.Slot {
		return &ValidationError{Ignore, "IL not for current slot"}
	}
	total := 0
	for _, tx := range il.Message.Transactions {
		total += len(tx)
	}
	if total == 0 {
		return &ValidationError{Ignore, "empty IL"}
	}
	if _, err := authenticate(il, view); err != nil {
		return err
	}
	return s.process(il, timely(clock, il.Message.Slot), true)
}
