package state

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/trie/bintrie"
)

var (
	ErrBinaryConversionStride     = errors.New("binary conversion stride must be positive")
	ErrBinaryConversionCheckpoint = errors.New("invalid binary overlay conversion checkpoint")
)

type binaryConversionSlot struct {
	key, order, value types.Hash
}

type binaryConversionAccount struct {
	address types.Address
	order   types.Hash
	account types.Account
	code    []byte
	slots   []binaryConversionSlot
}

// BinaryOverlayConversion implements the pinned EIP-7748 conversion order
// over a frozen memory snapshot for the pinned EIP-7864 BLAKE3 tree.
// Callers supply their current overlay, which may contain earlier writes.
type BinaryOverlayConversion struct {
	accounts   []binaryConversionAccount
	frozenRoot types.Hash
	account    int
	slot       int
}

func NewBinaryOverlayConversion(frozen *MemoryStateDB) (*BinaryOverlayConversion, error) {
	if frozen == nil {
		return nil, ErrBinaryMemoryState
	}
	conversion := new(BinaryOverlayConversion)
	for address, object := range frozen.stateObjects {
		if object == nil {
			return nil, ErrBinaryMemoryState
		}
		if object.selfDestructed {
			continue
		}
		account := object.account
		code := object.code
		if account.Balance == nil || account.Balance.Sign() < 0 || account.Balance.BitLen() > 128 ||
			len(code) > 0xffffff || !bytes.Equal(account.CodeHash, crypto.Keccak256(code)) {
			return nil, fmt.Errorf("%w: account %s", ErrBinaryMemoryState, address)
		}
		if account.Nonce == 0 && len(code) == 0 && account.Balance.Sign() == 0 {
			continue // Empty EIP-161 accounts consume no conversion units.
		}
		account.Balance = frozen.GetBalance(address)
		account.CodeHash = bytes.Clone(account.CodeHash)
		entry := binaryConversionAccount{
			address: address, order: crypto.Keccak256Hash(address[:]),
			account: account, code: bytes.Clone(code),
		}
		if account.Nonce != 0 || len(code) != 0 {
			for slot, value := range mergeStorage(object) {
				entry.slots = append(entry.slots, binaryConversionSlot{
					key: slot, order: crypto.Keccak256Hash(slot[:]), value: value,
				})
			}
			sort.Slice(entry.slots, func(i, j int) bool {
				return bytes.Compare(entry.slots[i].order[:], entry.slots[j].order[:]) < 0
			})
		}
		conversion.accounts = append(conversion.accounts, entry)
	}
	sort.Slice(conversion.accounts, func(i, j int) bool {
		return bytes.Compare(conversion.accounts[i].order[:], conversion.accounts[j].order[:]) < 0
	})
	conversion.frozenRoot = frozen.GetRoot()
	return conversion, nil
}

// BinaryConversionCheckpoint identifies the frozen MPT, the overlay root and
// the next conversion unit. The caller associates it with its block or branch.
type BinaryConversionCheckpoint struct {
	FrozenRoot  types.Hash
	OverlayRoot types.Hash
	Account     uint64
	Slot        uint64
}

// Persist writes the overlay nodes before returning a checkpoint for publication.
func (c *BinaryOverlayConversion) Persist(overlay *bintrie.BinaryTrie, store bintrie.NodeStore) (BinaryConversionCheckpoint, error) {
	if overlay == nil || !overlay.IsDraftBlake3() {
		return BinaryConversionCheckpoint{}, bintrie.ErrDraftStateHashFunction
	}
	root, err := overlay.Persist(store)
	if err != nil {
		return BinaryConversionCheckpoint{}, err
	}
	return BinaryConversionCheckpoint{
		FrozenRoot: c.frozenRoot, OverlayRoot: root,
		Account: uint64(c.account), Slot: uint64(c.slot),
	}, nil
}

// ResumeBinaryOverlayConversion authenticates both trees and restores the
// cursor from a checkpoint supplied by the caller.
func ResumeBinaryOverlayConversion(frozen *MemoryStateDB, store bintrie.NodeStore, checkpoint BinaryConversionCheckpoint) (*BinaryOverlayConversion, *bintrie.BinaryTrie, error) {
	conversion, err := NewBinaryOverlayConversion(frozen)
	if err != nil {
		return nil, nil, err
	}
	if checkpoint.FrozenRoot != conversion.frozenRoot || checkpoint.Account > uint64(len(conversion.accounts)) {
		return nil, nil, ErrBinaryConversionCheckpoint
	}
	if checkpoint.Account == uint64(len(conversion.accounts)) {
		if checkpoint.Slot != 0 {
			return nil, nil, ErrBinaryConversionCheckpoint
		}
	} else if checkpoint.Slot > uint64(len(conversion.accounts[checkpoint.Account].slots)) {
		return nil, nil, ErrBinaryConversionCheckpoint
	}
	tree, err := bintrie.OpenPersisted(store, bintrie.HashFunctionBlake3, checkpoint.OverlayRoot)
	if err != nil {
		return nil, nil, err
	}
	conversion.account, conversion.slot = int(checkpoint.Account), int(checkpoint.Slot)
	return conversion, tree, nil
}

// Copy shares the immutable frozen snapshot and gives the caller a separate
// cursor for block building or reorg handling.
func (c *BinaryOverlayConversion) Copy() *BinaryOverlayConversion {
	clone := *c
	return &clone
}

// Step applies up to stride conversion units to a copy of overlay. An
// existing leaf wins over the frozen value, including a 32-byte zero value.
// The input tree and cursor remain usable if any write fails.
func (c *BinaryOverlayConversion) Step(overlay *bintrie.BinaryTrie, stride int) (tree *bintrie.BinaryTrie, used int, complete bool, err error) {
	if stride <= 0 {
		return nil, 0, false, ErrBinaryConversionStride
	}
	if overlay == nil || !overlay.IsDraftBlake3() {
		return nil, 0, false, bintrie.ErrDraftStateHashFunction
	}
	if c.account == len(c.accounts) {
		return overlay, 0, true, nil
	}
	next := overlay.Copy()
	account, slot := c.account, c.slot
	for used < stride && account < len(c.accounts) {
		entry := &c.accounts[account]
		if slot < len(entry.slots) {
			item := entry.slots[slot]
			existing, err := next.GetStorage(entry.address, item.key[:])
			if err != nil {
				return nil, 0, false, err
			}
			if existing == nil {
				if err := next.UpdateStorage(entry.address, item.key[:], item.value[:]); err != nil {
					return nil, 0, false, err
				}
			}
			slot++
		} else {
			// An existing header owns its code and replaces the frozen account.
			key := bintrie.GetTreeKeyForBasicData(entry.address)
			existing, err := next.Get(key[:])
			if err != nil {
				return nil, 0, false, err
			}
			if existing == nil {
				if err := next.UpdateAccount(entry.address, &entry.account, len(entry.code)); err != nil {
					return nil, 0, false, err
				}
				if err := next.UpdateContractCode(entry.address, entry.code); err != nil {
					return nil, 0, false, err
				}
			}
			account++
			slot = 0
		}
		used++
	}
	c.account, c.slot = account, slot
	return next, used, account == len(c.accounts), nil
}
