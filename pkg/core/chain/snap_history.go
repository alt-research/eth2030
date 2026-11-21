package chain

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

var snapHistoryCursorKey = []byte("snap-history-cursor")

func snapHistoryBlockKey(hash types.Hash) []byte {
	return append([]byte("snap-history-block/"), hash[:]...)
}

// snapHistoryCursor is a storage/download frontier, never execution authority.
// The immutable anchor is the original CL-finalized, state-authenticated pivot.
func (bc *Blockchain) snapHistoryCursor() (anchor, cursor *types.Block, err error) {
	data, err := bc.db.Get(snapPivotKey)
	if errors.Is(err, rawdb.ErrNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, storageFailure(err)
	}
	if len(data) != 32 {
		return nil, nil, fmt.Errorf("snap history: corrupt pivot")
	}
	anchor = bc.GetBlock(types.BytesToHash(data))
	final := bc.CurrentFinalBlock()
	if anchor == nil || !bc.HasValidatedBlock(anchor.Hash()) || final == nil || final.NumberU64() < anchor.NumberU64() {
		return nil, nil, fmt.Errorf("snap history: pivot is not finalized and validated")
	}
	canonical, err := rawdb.ReadCanonicalHash(bc.db, anchor.NumberU64())
	if err != nil {
		return nil, nil, storageFailure(err)
	}
	if canonical != anchor.Hash() {
		return nil, nil, fmt.Errorf("snap history: original pivot is not canonical")
	}
	cursor = anchor
	data, err = bc.db.Get(snapHistoryCursorKey)
	if errors.Is(err, rawdb.ErrNotFound) {
		return anchor, cursor, nil
	}
	if err != nil {
		return nil, nil, storageFailure(err)
	}
	if len(data) != 64 || !bytes.Equal(data[:32], anchor.Hash().Bytes()) {
		return nil, nil, fmt.Errorf("snap history: corrupt frontier anchor")
	}
	cursor = bc.GetBlock(types.BytesToHash(data[32:]))
	if cursor == nil || cursor.NumberU64() > anchor.NumberU64() {
		return nil, nil, fmt.Errorf("snap history: missing frontier block")
	}
	return anchor, cursor, nil
}

// SnapHistoryCursor authenticates the saved prefix on worker startup/resume.
// Hash-linked historical bytes remain distinct from execution-validated state.
func (bc *Blockchain) SnapHistoryCursor(ctx context.Context) (*types.Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	anchor, cursor, err := bc.snapHistoryCursor()
	if err != nil || anchor == nil {
		return nil, err
	}
	for block := anchor; block.Hash() != cursor.Hash(); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if block.NumberU64() <= cursor.NumberU64() {
			return nil, fmt.Errorf("snap history: frontier is not an original pivot ancestor")
		}
		parent := bc.GetBlock(block.ParentHash())
		if parent == nil || parent.NumberU64()+1 != block.NumberU64() {
			return nil, fmt.Errorf("snap history: disconnected saved prefix")
		}
		if err := bc.validator.ValidateHeader(block.Header(), parent.Header()); err != nil {
			return nil, err
		}
		canonical, err := rawdb.ReadCanonicalHash(bc.db, parent.NumberU64())
		if err != nil || canonical != parent.Hash() {
			return nil, fmt.Errorf("snap history: noncanonical saved prefix")
		}
		if parent.NumberU64() != 0 {
			if err := bc.validator.ValidateBody(parent); err != nil {
				return nil, err
			}
		} else if parent.Hash() != bc.genesis.Hash() {
			return nil, fmt.Errorf("snap history: wrong saved genesis")
		}
		if parent.Header().BlockAccessListHash != nil {
			data, err := bc.BlockAccessList(parent.Hash())
			if err != nil {
				return nil, err
			}
			if data == nil {
				return nil, fmt.Errorf("snap history: incomplete committed BAL prefix")
			}
		}
		block = parent
	}
	return cursor, nil
}

// StageSnapHistoryParent authenticates a historical body against the next
// required parent hash. It neither executes it nor changes head/finality/state.
func (bc *Blockchain) StageSnapHistoryParent(childHash types.Hash, parent *types.Block) error {
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	anchor, child, err := bc.snapHistoryCursor()
	if err != nil {
		return err
	}
	if anchor == nil || child == nil || child.Hash() != childHash || parent == nil || parent.Header().Number == nil || !parent.Header().Number.IsUint64() || child.NumberU64() == 0 || parent.NumberU64()+1 != child.NumberU64() || parent.Hash() != child.ParentHash() {
		return fmt.Errorf("snap history: disconnected requested parent")
	}
	if err := bc.validator.ValidateHeader(child.Header(), parent.Header()); err != nil {
		return err
	}
	if parent.NumberU64() == 0 && parent.Hash() != bc.genesis.Hash() {
		return fmt.Errorf("snap history: wrong genesis")
	}
	if parent.NumberU64() == 0 {
		// Genesis has its own initialization rules and is already known
		// locally. Never persist a peer-supplied alternative genesis body.
		parent = bc.genesis
	} else if err := bc.validator.ValidateBody(parent); err != nil {
		return err
	}
	header, err := parent.Header().EncodeRLP()
	if err != nil {
		return err
	}
	body, err := encodeBlockBody(parent.Body())
	if err != nil {
		return err
	}
	batch := bc.db.NewBatch()
	for _, write := range []func() error{
		func() error { return rawdb.WriteHeader(batch, parent.NumberU64(), parent.Hash(), header) },
		func() error { return rawdb.WriteBody(batch, parent.NumberU64(), parent.Hash(), body) },
		func() error { return rawdb.WriteCanonicalHash(batch, parent.NumberU64(), parent.Hash()) },
		func() error { return batch.Put(snapHistoryBlockKey(parent.Hash()), anchor.Hash().Bytes()) },
	} {
		if err := write(); err != nil {
			return storageFailure(err)
		}
	}
	return storageFailure(batch.Write())
}

func (bc *Blockchain) snapHistoryBlock(hash types.Hash) (bool, error) {
	data, err := bc.db.Get(snapHistoryBlockKey(hash))
	if errors.Is(err, rawdb.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, storageFailure(err)
	}
	anchor, _, err := bc.snapHistoryCursor()
	if err != nil || anchor == nil {
		return false, err
	}
	if !bytes.Equal(data, anchor.Hash().Bytes()) {
		return false, fmt.Errorf("snap history: wrong body anchor")
	}
	return true, nil
}

// CompleteSnapHistoryBlock advances only after the exact committed BAL is
// durable. Interrupted body/BAL acquisition resumes the same authenticated step.
func (bc *Blockchain) CompleteSnapHistoryBlock(hash types.Hash) error {
	bc.insertMu.Lock()
	defer bc.insertMu.Unlock()
	anchor, cursor, err := bc.snapHistoryCursor()
	if err != nil {
		return err
	}
	if anchor == nil || cursor == nil || (hash != cursor.Hash() && hash != cursor.ParentHash()) {
		return fmt.Errorf("snap history: unexpected completed block")
	}
	block := bc.GetBlock(hash)
	if block == nil {
		return fmt.Errorf("snap history: missing completed body")
	}
	if hash != anchor.Hash() {
		ready, err := bc.snapHistoryBlock(hash)
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("snap history: unauthenticated completed body")
		}
	}
	if block.Header().BlockAccessListHash != nil {
		data, err := bc.BlockAccessList(hash)
		if err != nil {
			return err
		}
		if data == nil {
			return fmt.Errorf("snap history: incomplete completed BAL")
		}
	}
	return storageFailure(bc.db.Put(snapHistoryCursorKey, append(anchor.Hash().Bytes(), hash.Bytes()...)))
}
