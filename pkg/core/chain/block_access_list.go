package chain

import (
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

func blockAccessListKey(hash types.Hash) []byte {
	return append([]byte("BAL/"), hash[:]...)
}

// BlockAccessList returns the independently stored, execution-validated BAL.
// A missing/pruned BAL is nil; it does not make the block body unavailable.
func (bc *Blockchain) BlockAccessList(hash types.Hash) ([]byte, error) {
	block := bc.GetBlock(hash)
	if block == nil || block.Header().BlockAccessListHash == nil {
		return nil, nil
	}
	expected := *block.Header().BlockAccessListHash
	if block.NumberU64() == 0 && expected == crypto.Keccak256Hash([]byte{0xc0}) {
		return []byte{0xc0}, nil
	}
	encoded, err := bc.db.Get(blockAccessListKey(hash))
	if errors.Is(err, rawdb.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, storageFailure(err)
	}
	if crypto.Keccak256Hash(encoded) != expected {
		return nil, fmt.Errorf("stored block access list hash mismatch for %s", hash.Hex())
	}
	return encoded, nil
}

// StoreBlockAccessListRLP restores historical BAL data only for a locally
// execution-validated block or authenticated finalized snap ancestor, and only
// when it matches that block's commitment. It never grants execution authority.
func (bc *Blockchain) StoreBlockAccessListRLP(hash types.Hash, encoded []byte) error {
	block := bc.GetBlock(hash)
	if block == nil {
		return fmt.Errorf("BAL target is unknown")
	}
	if !bc.HasValidatedBlock(hash) {
		anchored, err := bc.snapHistoryBlock(hash)
		if err != nil {
			return err
		}
		if !anchored {
			return fmt.Errorf("BAL target is not execution-validated or a finalized pivot ancestor")
		}
	}
	expected := block.Header().BlockAccessListHash
	if expected == nil || crypto.Keccak256Hash(encoded) != *expected {
		return fmt.Errorf("received block access list hash mismatch")
	}
	if err := bc.db.Put(blockAccessListKey(hash), append([]byte(nil), encoded...)); err != nil {
		return storageFailure(err)
	}
	return nil
}
