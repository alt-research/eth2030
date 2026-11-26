package bal

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

type IndexedChange[T any] struct {
	Index uint64
	Value T
}

type SlotChanges struct {
	Slot    *big.Int
	Changes []IndexedChange[*big.Int]
}

type AccountChanges struct {
	adoptedCodeHashes map[uint64]types.Hash
	Address           types.Address
	StorageChanges    []SlotChanges
	StorageReads      []*big.Int
	BalanceChanges    []IndexedChange[*big.Int]
	NonceChanges      []IndexedChange[uint64]
	CodeChanges       []IndexedChange[[]byte]
}

// EncodeRLP encodes the EIP-7928 account-grouped wire format. Internal
// entries retain per-phase pre-values and read values for execution tooling.
func (list *BlockAccessList) EncodeRLP() ([]byte, error) {
	entries := append([]AccessEntry(nil), list.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		if cmp := bytes.Compare(entries[i].Address[:], entries[j].Address[:]); cmp != 0 {
			return cmp < 0
		}
		return entries[i].AccessIndex < entries[j].AccessIndex
	})
	if err := ValidateBALOrdering(&BlockAccessList{Entries: entries}); err != nil {
		return nil, err
	}
	accounts := make([]AccountChanges, 0)
	for first := 0; first < len(entries); {
		last := first + 1
		for last < len(entries) && entries[last].Address == entries[first].Address {
			last++
		}
		account, err := encodeAccountChanges(entries[first:last])
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
		first = last
	}
	return EncodeAccountChanges(accounts)
}

func encodeAccountChanges(entries []AccessEntry) (AccountChanges, error) {
	account := AccountChanges{Address: entries[0].Address}
	writes := make(map[types.Hash][]IndexedChange[*big.Int])
	reads := make(map[types.Hash]bool)
	for _, entry := range entries {
		idx := entry.AccessIndex
		if idx > math.MaxUint32 {
			return account, fmt.Errorf("BAL access index %d exceeds uint32", idx)
		}
		for _, read := range entry.StorageReads {
			reads[read.Slot] = true
		}
		for _, write := range entry.StorageChanges {
			if write.OldValue == write.NewValue {
				reads[write.Slot] = true
				continue
			}
			writes[write.Slot] = append(writes[write.Slot], IndexedChange[*big.Int]{idx, new(big.Int).SetBytes(write.NewValue[:])})
		}
		if change := entry.BalanceChange; change != nil {
			account.BalanceChanges = append(account.BalanceChanges, IndexedChange[*big.Int]{idx, change.NewValue})
		}
		if change := entry.NonceChange; change != nil {
			account.NonceChanges = append(account.NonceChanges, IndexedChange[uint64]{idx, change.NewValue})
		}
		if change := entry.CodeChange; change != nil {
			if change.NewCodeHash != nil {
				if account.adoptedCodeHashes == nil {
					account.adoptedCodeHashes = make(map[uint64]types.Hash)
				}
				account.adoptedCodeHashes[idx] = *change.NewCodeHash
				account.CodeChanges = append(account.CodeChanges, IndexedChange[[]byte]{idx, nil})
			} else {
				account.CodeChanges = append(account.CodeChanges, IndexedChange[[]byte]{idx, change.NewCode})
			}
		}
	}
	for slot, changes := range writes {
		account.StorageChanges = append(account.StorageChanges, SlotChanges{new(big.Int).SetBytes(slot[:]), changes})
		delete(reads, slot)
	}
	for slot := range reads {
		account.StorageReads = append(account.StorageReads, new(big.Int).SetBytes(slot[:]))
	}
	sort.Slice(account.StorageChanges, func(i, j int) bool { return account.StorageChanges[i].Slot.Cmp(account.StorageChanges[j].Slot) < 0 })
	sort.Slice(account.StorageReads, func(i, j int) bool { return account.StorageReads[i].Cmp(account.StorageReads[j]) < 0 })
	return account, nil
}

// Hash computes the Keccak256 hash of the RLP-encoded BlockAccessList.
func (list *BlockAccessList) Hash() types.Hash {
	encoded, err := list.EncodeRLP()
	if err != nil {
		return types.Hash{}
	}
	return crypto.Keccak256Hash(encoded)
}
