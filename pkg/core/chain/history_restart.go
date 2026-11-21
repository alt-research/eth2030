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
func (bc *Blockchain) restoreIndexedHead(anchor *types.Block, ancestry []*types.Header) (bool, error) {
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
	if err := state.AuditTrieHistory(context.Background(), bc.db, head.Root()); err != nil {
		return false, storageFailure(err)
	}
	current, err := state.OpenTrieHistory(context.Background(), bc.db, head.Root())
	if err != nil {
		return false, storageFailure(err)
	}
	var headers []*types.Header
	var recent []*types.Block
	missingValidation := false
	final, err := rawdb.ReadFinalizedBlockHashChecked(bc.db)
	if err != nil {
		return false, storageFailure(err)
	}
	finalSeen := final == anchor.Hash() || (anchor.NumberU64() == 0 && final == (types.Hash{}))
	cursor := head
	for cursor.NumberU64() > anchor.NumberU64() {
		number, hash := cursor.NumberU64(), cursor.Hash()
		if hash == final {
			finalSeen = true
		}
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
	if cursor.Hash() != anchor.Hash() {
		return false, fmt.Errorf("history recovery: original checkpoint is not a saved-head ancestor")
	}
	canonical, err := rawdb.ReadCanonicalHash(bc.db, anchor.NumberU64())
	if err != nil {
		return false, storageFailure(err)
	}
	if canonical != anchor.Hash() {
		return false, fmt.Errorf("history recovery: noncanonical original checkpoint")
	}
	if anchor.NumberU64() != 0 {
		marker, err := bc.db.Get(validatedKey(anchor.Hash()))
		if errors.Is(err, rawdb.ErrNotFound) {
			missingValidation = true
		} else if err != nil {
			return false, storageFailure(err)
		} else if !bytes.Equal(marker, []byte{1}) {
			return false, fmt.Errorf("history recovery: corrupt checkpoint validation marker")
		}
	}
	if !finalSeen {
		return false, fmt.Errorf("history recovery: finality is outside the authenticated saved prefix")
	}
	if missingValidation {
		return false, nil
	}
	if len(recent) < bc.opts.BlockCacheSize {
		recent = append(recent, anchor)
	}
	headers = append(headers, ancestry...)
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
	if err := bc.ApplyForkchoice(head.Hash(), final, final); err != nil {
		return false, err
	}
	return true, nil
}
