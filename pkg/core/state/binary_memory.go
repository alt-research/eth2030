package state

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/trie/bintrie"
)

var ErrBinaryMemoryState = errors.New("binary state: invalid or inconsistent memory state")

// BuildBinaryTreeFromMemory materializes the current MemoryStateDB, including
// its dirty storage overlay, into an EIP-7864 draft BLAKE3 trie. Self-destructed
// accounts and zero storage values are omitted. The source must not be mutated
// concurrently. This full-memory conversion is experimental: it is neither a
// StateDB implementation nor a fork activation or canonical root publication.
// Errors are returned explicitly rather than mapped to the valid empty root.
func BuildBinaryTreeFromMemory(source *MemoryStateDB) (*bintrie.BinaryTrie, error) {
	if source == nil {
		return nil, ErrBinaryMemoryState
	}
	tree := bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3)
	for address, obj := range source.stateObjects {
		if obj == nil {
			return nil, ErrBinaryMemoryState
		}
		if obj.selfDestructed {
			continue
		}
		if obj.account.Balance == nil || len(obj.account.CodeHash) != 32 || !bytes.Equal(obj.account.CodeHash, crypto.Keccak256(obj.code)) {
			return nil, fmt.Errorf("%w: account %x", ErrBinaryMemoryState, address)
		}
		account := &types.Account{Nonce: obj.account.Nonce, Balance: obj.account.Balance, CodeHash: obj.account.CodeHash}
		if err := tree.UpdateAccount(address, account, len(obj.code)); err != nil {
			return nil, fmt.Errorf("binary state account %x: %w", address, err)
		}
		if err := tree.UpdateContractCode(address, obj.code); err != nil {
			return nil, err
		}
		for slot, value := range mergeStorage(obj) {
			if err := tree.UpdateStorage(address, slot[:], value[:]); err != nil {
				return nil, err
			}
		}
	}
	return tree, nil
}
