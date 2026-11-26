// Package vm implements the Ethereum Virtual Machine.
//
// This file implements EIP-7610: Revert creation in case of non-empty storage.
// EIP-7610 extends the existing CREATE/CREATE2 collision check to also reject
// deployment to addresses that have non-empty storage, in addition to the
// pre-existing checks for non-zero nonce and non-empty code.
//
// Spec: https://eips.ethereum.org/EIPS/eip-7610
package vm

import (
	"errors"

	"github.com/eth2030/eth2030/core/types"
)

// ErrContractCreationCollision is returned when a CREATE or CREATE2 targets
// an address that already has a non-zero nonce, non-empty code, or (per
// EIP-7610) non-empty storage.
var ErrContractCreationCollision = errors.New("contract creation collision: address already in use")

// CollisionCheck7610 performs EIP-7610 contract creation collision checks.
type CollisionCheck7610 struct {
	// Enabled controls whether the EIP-7610 storage check is active.
	// When false, only the legacy nonce/code checks are performed.
	Enabled bool
}

// NewCollisionCheck7610 returns a CollisionCheck7610 with the given
// enabled state.
func NewCollisionCheck7610(enabled bool) *CollisionCheck7610 {
	return &CollisionCheck7610{Enabled: enabled}
}

// HasNonEmptyStorage checks the complete current storage root, including dirty
// writes. Probing slots cannot establish emptiness: storage keys span 256 bits.
// All core/state.StateDB implementations expose StorageRoot. A VM-only adapter
// without that capability must fail closed instead of permitting storage loss.
func HasNonEmptyStorage(stateDB StateDB, addr types.Address) bool {
	if roots, ok := stateDB.(interface {
		StorageRoot(types.Address) types.Hash
	}); ok {
		return roots.StorageRoot(addr) != types.EmptyRootHash
	}
	return true
}

// CheckCreateCollision checks whether deploying a contract at addr would
// collide with existing state. It returns ErrContractCreationCollision if
// the address has:
//   - a non-zero nonce, OR
//   - non-empty code (code hash differs from the empty code hash), OR
//   - non-empty storage (EIP-7610, only when c.Enabled is true)
//
// An address that only has a non-zero balance is acceptable per EIP-7610
// and returns nil.
func (c *CollisionCheck7610) CheckCreateCollision(stateDB StateDB, addr types.Address) error {
	// Check nonce.
	if stateDB.GetNonce(addr) != 0 {
		return ErrContractCreationCollision
	}

	// Check code. An address with deployed code is not a valid target.
	codeHash := stateDB.GetCodeHash(addr)
	if codeHash != (types.Hash{}) && codeHash != types.EmptyCodeHash {
		return ErrContractCreationCollision
	}

	// EIP-7610: check storage.
	if c.Enabled {
		if HasNonEmptyStorage(stateDB, addr) {
			return ErrContractCreationCollision
		}
	}

	return nil
}
