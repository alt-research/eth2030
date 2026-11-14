package chain

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/trie/bintrie"
)

const (
	overlayCheckpointVersion = 1
	overlayNodePrefix        = "eth2030:overlay-nodes:"
)

func overlayCheckpointKey(hash types.Hash) []byte {
	return append([]byte("eth2030:binary-overlay-checkpoint:"), hash[:]...)
}

func writeOverlayCheckpoint(db rawdb.Database, batch rawdb.Batch, blk *types.Block, overlay *state.BinaryOverlayState) error {
	// The batch publishes the cursor alongside the block and canonical indices.
	// Unchanged content-addressed nodes are already present in the database.
	nodes := overlayNodeBatch{rawdb.NewTable(db, overlayNodePrefix), batch}
	checkpoint, err := overlay.PersistCheckpoint(nodes, nodes.db.Has)
	if err != nil {
		return err
	}
	if checkpoint.OverlayRoot != blk.Root() {
		return ErrBinaryOverlayRoot
	}
	var encoded [81]byte
	encoded[0] = overlayCheckpointVersion
	copy(encoded[1:33], checkpoint.FrozenRoot[:])
	copy(encoded[33:65], checkpoint.OverlayRoot[:])
	binary.BigEndian.PutUint64(encoded[65:73], checkpoint.Account)
	binary.BigEndian.PutUint64(encoded[73:81], checkpoint.Slot)
	return batch.Put(overlayCheckpointKey(blk.Hash()), encoded[:])
}

// overlayNodeBatch provides the node store with a view of committed nodes and
// stages writes in the same database batch as the block.
type overlayNodeBatch struct {
	db    *rawdb.Table
	batch rawdb.Batch
}

func (s overlayNodeBatch) Get(key []byte) ([]byte, error) { return s.db.Get(key) }

func (s overlayNodeBatch) Put(key, value []byte) error {
	return s.batch.Put(append([]byte(overlayNodePrefix), key...), value)
}

func (bc *Blockchain) verifyOverlayCheckpoint(blk *types.Block, replayed state.StateDB) error {
	data, err := bc.db.Get(overlayCheckpointKey(blk.Hash()))
	if errors.Is(err, rawdb.ErrNotFound) {
		// Databases created before checkpoints continue to recover by replay.
		return nil
	}
	if err != nil {
		return err
	}
	overlay, ok := replayed.(*state.BinaryOverlayState)
	if !ok || len(data) != 81 || data[0] != overlayCheckpointVersion {
		return state.ErrBinaryConversionCheckpoint
	}
	checkpoint := state.BinaryConversionCheckpoint{
		FrozenRoot:  types.BytesToHash(data[1:33]),
		OverlayRoot: types.BytesToHash(data[33:65]),
		Account:     binary.BigEndian.Uint64(data[65:73]),
		Slot:        binary.BigEndian.Uint64(data[73:81]),
	}
	if checkpoint != overlay.Checkpoint() || checkpoint.OverlayRoot != blk.Root() {
		return state.ErrBinaryConversionCheckpoint
	}
	if _, err := bintrie.OpenPersisted(rawdb.NewTable(bc.db, overlayNodePrefix), bintrie.HashFunctionBlake3, checkpoint.OverlayRoot); err != nil {
		return fmt.Errorf("load binary overlay checkpoint nodes: %w", err)
	}
	return nil
}
