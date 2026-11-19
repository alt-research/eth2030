package bal

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
)

// DecodeForHeader authenticates and decodes the account-grouped EIP-7928
// representation. Wire records have post-values, not invented execution pre-values.
func DecodeForHeader(encoded []byte, header *types.Header) ([]AccountChanges, error) {
	if header == nil || header.BlockAccessListHash == nil {
		return nil, fmt.Errorf("BAL requires a committed header")
	}
	if crypto.Keccak256Hash(encoded) != *header.BlockAccessListHash {
		return nil, fmt.Errorf("BAL header commitment mismatch")
	}
	accounts, err := DecodeRLP(encoded)
	if err != nil {
		return nil, err
	}
	var count uint64
	for _, account := range accounts {
		count += 1 + uint64(len(account.StorageChanges)+len(account.StorageReads))
	}
	if count > header.GasLimit/BALItemCost {
		return nil, ErrBALSizeExceeded
	}
	return accounts, nil
}

func validWord(value *big.Int) bool {
	return value != nil && value.Sign() >= 0 && value.BitLen() <= 256
}

func validIndices[T any](changes []IndexedChange[T]) bool {
	for i, change := range changes {
		if change.Index > math.MaxUint32 || (i > 0 && changes[i-1].Index >= change.Index) {
			return false
		}
	}
	return true
}

// DecodeRLP validates canonical structure/order before any state consumer uses it.
func DecodeRLP(encoded []byte) ([]AccountChanges, error) {
	var accounts []AccountChanges
	if err := rlp.DecodeBytes(encoded, &accounts); err != nil {
		return nil, err
	}
	for i, account := range accounts {
		if i > 0 && bytes.Compare(accounts[i-1].Address[:], account.Address[:]) >= 0 {
			return nil, fmt.Errorf("BAL account order")
		}
		written := make(map[types.Hash]bool, len(account.StorageChanges))
		for j, slot := range account.StorageChanges {
			if !validWord(slot.Slot) || (j > 0 && account.StorageChanges[j-1].Slot.Cmp(slot.Slot) >= 0) || len(slot.Changes) == 0 || !validIndices(slot.Changes) {
				return nil, fmt.Errorf("BAL storage change order/width")
			}
			written[types.BytesToHash(slot.Slot.Bytes())] = true
			for _, change := range slot.Changes {
				if !validWord(change.Value) {
					return nil, fmt.Errorf("BAL storage value width")
				}
			}
		}
		for j, slot := range account.StorageReads {
			if !validWord(slot) || (j > 0 && account.StorageReads[j-1].Cmp(slot) >= 0) || written[types.BytesToHash(slot.Bytes())] {
				return nil, fmt.Errorf("BAL storage read order/overlap")
			}
		}
		if !validIndices(account.BalanceChanges) || !validIndices(account.NonceChanges) || !validIndices(account.CodeChanges) {
			return nil, fmt.Errorf("BAL field change order/width")
		}
		for _, change := range account.BalanceChanges {
			if !validWord(change.Value) {
				return nil, fmt.Errorf("BAL balance width")
			}
		}
	}
	return accounts, nil
}
