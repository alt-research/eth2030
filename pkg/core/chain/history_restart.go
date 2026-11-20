package chain

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// restoreIndexedHead reuses only locally persisted execution validation and an
// independently authenticated original state root. It is a startup path, never
// a peer-import or forkchoice-selection shortcut.
func (bc *Blockchain) restoreIndexedHead() (bool, error) {
	if bc.config != nil && bc.config.BinaryOverlayTime != nil {
		return false, nil
	}
	headHash, err := rawdb.ReadHeadBlockHash(bc.db)
	if err != nil {
		return false, storageFailure(err)
	}
	if headHash == bc.genesis.Hash() {
		return false, nil
	}
	head := bc.GetBlock(headHash)
	if head == nil || head.Hash() != headHash {
		return false, fmt.Errorf("history recovery: missing original head")
	}
	available, err := state.HistoryStateAvailable(bc.db, head.Root())
	if err != nil {
		return false, storageFailure(err)
	}
	if !available {
		return false, nil
	}
	// A published but corrupt index is an error even if a legacy validation
	// marker is missing. Do not turn unreadable committed state into zero state.
	current, err := state.OpenTrieHistory(context.Background(), bc.db, head.Root())
	if err != nil {
		return false, storageFailure(err)
	}
	var headers []*types.Header
	var recent []*types.Block
	missingValidation := false
	for cursor := head; cursor.NumberU64() > 0; {
		number, hash := cursor.NumberU64(), cursor.Hash()
		canonical, err := rawdb.ReadCanonicalHash(bc.db, number)
		if err != nil {
			return false, storageFailure(err)
		}
		if canonical != hash {
			return false, fmt.Errorf("history recovery: noncanonical ancestor at %d", number)
		}
		marker, err := bc.db.Get(validatedKey(hash))
		if errors.Is(err, rawdb.ErrNotFound) {
			missingValidation = true
		} else if err != nil {
			return false, storageFailure(err)
		} else if !bytes.Equal(marker, []byte{1}) {
			return false, fmt.Errorf("history recovery: corrupt validation marker at %d", number)
		}
		parent := bc.GetBlock(cursor.ParentHash())
		if parent == nil || parent.Hash() != cursor.ParentHash() || parent.NumberU64()+1 != number {
			return false, fmt.Errorf("history recovery: disconnected ancestor at %d", number)
		}
		if err := bc.validator.ValidateHeader(cursor.Header(), parent.Header()); err != nil {
			return false, err
		}
		if err := bc.validator.ValidateBody(cursor); err != nil {
			return false, err
		}
		// Historical BAL pruning is permitted. Retained bytes still have to
		// match their original header commitment before serving them again.
		if _, err := bc.BlockAccessList(hash); err != nil {
			return false, err
		}
		headers = append(headers, cursor.Header())
		if len(recent) < bc.opts.BlockCacheSize {
			recent = append(recent, cursor)
		}
		if number == 1 && parent.Hash() != bc.genesis.Hash() {
			return false, fmt.Errorf("history recovery: wrong original genesis")
		}
		cursor = parent
	}
	if missingValidation {
		return false, nil
	}
	headers = append(headers, bc.genesis.Header())
	bc.mu.Lock()
	bc.adoptSnapPivot(head, current, headers)
	// The backwards verification walk cached old ancestors. Preserve the
	// normal body-cache capacity and chronological eviction order on return.
	bc.blockCache = make(map[types.Hash]*types.Block)
	bc.blockCacheOrder = nil
	for i := len(recent) - 1; i >= 0; i-- {
		bc.cacheBlockLocked(recent[i])
	}
	bc.mu.Unlock()
	final := rawdb.ReadFinalizedBlockHash(bc.db)
	if err := bc.ApplyForkchoice(head.Hash(), final, final); err != nil {
		return false, err
	}
	return true, nil
}
