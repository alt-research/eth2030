package state

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/trie/bintrie"
)

var ErrBinaryOverlayState = errors.New("binary overlay requires complete memory state")

// BinaryOverlayState keeps the logical world state in memory for EVM reads.
// Its journal determines which writes enter the BLAKE3 overlay at the end
// of each block. The frozen snapshot is owned by the conversion cursor.
type BinaryOverlayState struct {
	*MemoryStateDB
	tree          *bintrie.BinaryTrie
	originalSlots map[binaryOverlaySlot]types.Hash
	conversion    *BinaryOverlayConversion
	converted     bool
}

type binaryOverlaySlot struct {
	address types.Address
	key     types.Hash
}

var _ StateDB = (*BinaryOverlayState)(nil)

func NewBinaryOverlayState(source *MemoryStateDB) (*BinaryOverlayState, error) {
	if source == nil {
		return nil, ErrBinaryOverlayState
	}
	conversion, err := NewBinaryOverlayConversion(source)
	if err != nil {
		return nil, err
	}
	memory := source.Copy()
	for address, object := range memory.stateObjects {
		if object.selfDestructed {
			delete(memory.stateObjects, address)
		}
	}
	memory.finalizeStorage()
	return &BinaryOverlayState{
		MemoryStateDB: memory,
		tree:          bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3),
		conversion:    conversion,
	}, nil
}

func (s *BinaryOverlayState) Dup() StateDB {
	return &BinaryOverlayState{
		MemoryStateDB: s.MemoryStateDB.Copy(),
		tree:          s.tree.Copy(),
		conversion:    s.conversion.Copy(),
		converted:     s.converted,
	}
}

// SetTxContext resets the original-slot cache for the next EVM transaction.
func (s *BinaryOverlayState) SetTxContext(hash types.Hash, index int) {
	s.MemoryStateDB.SetTxContext(hash, index)
	s.originalSlots = make(map[binaryOverlaySlot]types.Hash)
}

func (s *BinaryOverlayState) GetCommittedState(address types.Address, key types.Hash) types.Hash {
	if s.originalSlots == nil {
		return s.MemoryStateDB.GetCommittedState(address, key)
	}
	if value, exists := s.originalSlots[binaryOverlaySlot{address, key}]; exists {
		return value
	}
	return s.GetState(address, key)
}

func (s *BinaryOverlayState) SetState(address types.Address, key, value types.Hash) {
	if s.originalSlots != nil {
		slot := binaryOverlaySlot{address, key}
		if _, exists := s.originalSlots[slot]; !exists {
			s.originalSlots[slot] = s.GetState(address, key)
		}
	}
	s.MemoryStateDB.SetState(address, key, value)
}

func (s *BinaryOverlayState) Convert(stride int) error {
	if s.converted {
		return nil
	}
	tree, _, complete, err := s.conversion.Step(s.tree, stride)
	if err != nil {
		return err
	}
	s.tree, s.converted = tree, complete
	return nil
}

// GetRoot returns the last finalized overlay root.
func (s *BinaryOverlayState) GetRoot() types.Hash { return s.tree.Hash() }

// Commit finalizes EVM writes in a separate tree. An error leaves the prior
// root and the journal available to the caller.
func (s *BinaryOverlayState) Commit() (types.Hash, error) {
	headers := make(map[types.Address]struct{})
	slots := make(map[types.Address]map[types.Hash]struct{})
	addSlot := func(address types.Address, key types.Hash) {
		if slots[address] == nil {
			slots[address] = make(map[types.Hash]struct{})
		}
		slots[address][key] = struct{}{}
	}
	resetSlots := func(address types.Address, previous *stateObject) {
		if previous == nil {
			return
		}
		for key := range previous.committedStorage {
			addSlot(address, key)
		}
		for key := range previous.dirtyStorage {
			addSlot(address, key)
		}
	}
	for _, entry := range s.journal.entries {
		switch change := entry.(type) {
		case createAccountChange:
			headers[change.addr] = struct{}{}
			resetSlots(change.addr, change.prev)
		case selfDestructChange:
			headers[change.addr] = struct{}{}
			resetSlots(change.addr, s.stateObjects[change.addr])
		case balanceChange:
			headers[change.addr] = struct{}{}
		case nonceChange:
			headers[change.addr] = struct{}{}
		case codeChange:
			headers[change.addr] = struct{}{}
		case storageChange:
			addSlot(change.addr, change.key)
		}
	}

	tree := s.tree.Copy()
	for address := range headers {
		if err := updateBinaryOverlayAccount(tree, address, s.stateObjects[address]); err != nil {
			return types.Hash{}, fmt.Errorf("binary overlay account %s: %w", address, err)
		}
	}
	for address, keys := range slots {
		for key := range keys {
			var value types.Hash
			if !s.HasSelfDestructed(address) {
				value = s.GetState(address, key)
			}
			path := bintrie.GetTreeKeyForStorageSlot(address, new(big.Int).SetBytes(key[:]))
			// A zero leaf shadows the old MPT slot and blocks later conversion.
			if err := tree.Put(path[:], value[:]); err != nil {
				return types.Hash{}, err
			}
		}
	}
	s.tree = tree
	for address, object := range s.stateObjects {
		if object.selfDestructed {
			delete(s.stateObjects, address)
		}
	}
	s.MemoryStateDB.finalizeStorage()
	s.journal = newJournal()
	s.originalSlots = nil
	return s.GetRoot(), nil
}

func updateBinaryOverlayAccount(tree *bintrie.BinaryTrie, address types.Address, object *stateObject) error {
	basicKey, codeKey := bintrie.GetTreeKeyForBasicData(address), bintrie.GetTreeKeyForCodeHash(address)
	previous, err := tree.Get(basicKey[:])
	if err != nil {
		return err
	}
	var code []byte
	if object != nil && !object.selfDestructed {
		code = object.code
		if err := tree.UpdateAccount(address, &object.account, len(code)); err != nil {
			return err
		}
		if err := tree.UpdateContractCode(address, code); err != nil {
			return err
		}
	} else {
		var zero types.Hash
		if err := tree.Put(basicKey[:], zero[:]); err != nil {
			return err
		}
		if err := tree.Put(codeKey[:], zero[:]); err != nil {
			return err
		}
	}
	if len(previous) == 32 {
		oldLength := int(previous[5])<<16 | int(previous[6])<<8 | int(previous[7])
		for chunk := (len(code) + 30) / 31; chunk < (oldLength+30)/31; chunk++ {
			key := bintrie.GetTreeKeyForCodeChunk(address, uint64(chunk))
			if err := tree.Delete(key[:]); err != nil {
				return err
			}
		}
	}
	return nil
}
