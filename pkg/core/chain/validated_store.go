package chain

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

var ErrStorageFailure = errors.New("chain storage failure")

func storageFailure(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrStorageFailure, err)
}

// CheckStorage preserves a fenced/unreadable database error before callers use
// cache-only or boolean lookup paths. A missing durable head is also a local
// storage error, not evidence that a peer needs to provide an unknown block.
func (bc *Blockchain) CheckStorage() error {
	_, err := rawdb.ReadHeadBlockHash(bc.db)
	return storageFailure(err)
}

func validatedKey(hash types.Hash) []byte { return append([]byte("validated-block/"), hash[:]...) }

// HasValidatedBlock distinguishes execution-validated blocks from headers and
// bodies that merely happen to be present in the database or block cache.
func (bc *Blockchain) HasValidatedBlock(hash types.Hash) bool {
	if hash == bc.genesis.Hash() {
		return true
	}
	value, err := bc.db.Get(validatedKey(hash))
	return err == nil && bytes.Equal(value, []byte{1})
}

func writeValidatedBlock(batch rawdb.KeyValueWriter, blk *types.Block, receipts []*types.Receipt) error {
	header, err := blk.Header().EncodeRLP()
	if err != nil {
		return err
	}
	body, err := encodeBlockBody(blk.Body())
	if err != nil {
		return err
	}
	if err := rawdb.WriteHeader(batch, blk.NumberU64(), blk.Hash(), header); err != nil {
		return err
	}
	if err := rawdb.WriteBody(batch, blk.NumberU64(), blk.Hash(), body); err != nil {
		return err
	}
	var encoded []byte
	for _, receipt := range receipts {
		if receipt == nil {
			return fmt.Errorf("nil validated receipt")
		}
		data, err := receipt.EncodeRLP()
		if err != nil {
			return err
		}
		encoded = append(encoded, data...)
	}
	if err := rawdb.WriteReceipts(batch, blk.NumberU64(), blk.Hash(), encoded); err != nil {
		return err
	}
	return batch.Put(validatedKey(blk.Hash()), []byte{1})
}

func (bc *Blockchain) persistValidatedBlock(blk *types.Block, receipts []*types.Receipt) error {
	batch := bc.db.NewBatch()
	if err := writeValidatedBlock(batch, blk, receipts); err != nil {
		return err
	}
	return batch.Write()
}

// BranchHashFn serves BLOCKHASH from the candidate's own ancestors, including
// the header-only window retained below a snap pivot. A concurrent canonical
// head change cannot change this block's execution environment.
func (bc *Blockchain) BranchHashFn(parent *types.Header) func(uint64) types.Hash {
	hashes := make(map[uint64]types.Hash)
	for count := 0; parent != nil && parent.Number != nil && count < 256; count++ {
		number := parent.Number.Uint64()
		hashes[number] = parent.Hash()
		if number == 0 {
			break
		}
		hash := parent.ParentHash
		block := bc.GetBlock(hash)
		if block != nil {
			parent = block.Header()
			continue
		}
		encoded, err := rawdb.ReadHeader(bc.db, number-1, hash)
		if err != nil {
			break
		}
		header, err := types.DecodeHeaderRLP(encoded)
		if err != nil || header.Hash() != hash || header.Number == nil || header.Number.Uint64() != number-1 {
			break
		}
		parent = header
	}
	return func(number uint64) types.Hash { return hashes[number] }
}
