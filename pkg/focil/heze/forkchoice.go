package heze

import (
	"errors"
	"sync"

	"github.com/eth2030/eth2030/core/types"
)

type ExecutionState uint8

const (
	NotValidated ExecutionState = iota
	Valid
	Invalidated
)

type payloadDecision struct {
	State     ExecutionState
	Satisfied bool
	Available bool
}

// PayloadDecisions is the Heze inclusion-satisfaction part of a CL fork-choice
// store. It never changes execution validity or ancestor decisions. Other Gloas
// fork-choice conditions are explicit inputs to ShouldExtend.
type PayloadDecisions struct {
	mu        sync.RWMutex
	decisions map[types.Hash]payloadDecision
}

func NewPayloadDecisions(anchor types.Hash) *PayloadDecisions {
	return &PayloadDecisions{decisions: map[types.Hash]payloadDecision{anchor: {Valid, true, true}}}
}

func (p *PayloadDecisions) Record(root types.Hash, status ExecutionState, satisfied *bool, available bool) error {
	if status > Invalidated || status == Valid && satisfied == nil {
		return errors.New("heze: missing/invalid execution satisfaction response")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if old, ok := p.decisions[root]; ok && old.State == Valid && status != Invalidated {
		old.Available = available
		p.decisions[root] = old
		return nil
	}
	value := status == NotValidated
	if status == Valid {
		value = *satisfied
	}
	p.decisions[root] = payloadDecision{status, value, available}
	return nil
}
func (p *PayloadDecisions) IsSatisfied(root types.Hash) (bool, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d, known := p.decisions[root]
	return known && d.Available && d.Satisfied, known
}

type ExtensionContext struct {
	PayloadSlot          uint64
	CurrentSlot          uint64
	PayloadTimely        bool
	PayloadDataAvailable bool
	ProposerBoostRoot    types.Hash
	BoostParentRoot      types.Hash
	BoostParentFull      bool
}

func (p *PayloadDecisions) ShouldExtend(root types.Hash, c ExtensionContext) bool {
	if c.CurrentSlot == 0 || c.PayloadSlot != c.CurrentSlot-1 {
		return false
	}
	satisfied, known := p.IsSatisfied(root)
	if !known || !satisfied {
		return false
	}
	return c.PayloadTimely && c.PayloadDataAvailable || c.ProposerBoostRoot == (types.Hash{}) || c.BoostParentRoot != root || c.BoostParentFull
}
