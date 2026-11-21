package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// EIP-7928's selected HISTORY_PRUNE_EPOCHS, in actual beacon slots. A missed
// slot does not consume a block number, so block count is never this boundary.
const snapHistoryRetentionSlots = uint64(14299 * 32)

var errSnapHistoryWaiting = errors.New("awaiting authenticated snap pivot")

func (n *Node) startSnapHistory() {
	ctx, cancel := context.WithCancel(context.Background())
	n.historyCancel, n.historyDone = cancel, make(chan struct{})
	go func() {
		defer close(n.historyDone)
		for {
			err := n.backfillSnapHistory(ctx)
			if err == nil || errors.Is(err, context.Canceled) {
				return
			}
			if !errors.Is(err, errSnapHistoryWaiting) {
				slog.Warn("snap historical body/BAL backfill paused", "err", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}()
}

func (n *Node) backfillSnapHistory(ctx context.Context) error {
	cursor, err := n.blockchain.SnapHistoryCursor(ctx)
	if err != nil {
		return err
	}
	if cursor == nil {
		if n.config.SyncMode != "snap" {
			return nil
		}
		return errSnapHistoryWaiting
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		final := n.blockchain.CurrentFinalBlock()
		if final == nil {
			return fmt.Errorf("snap historical anchor is no longer finalized")
		}
		if slot, last := cursor.Header().SlotNumber, final.Header().SlotNumber; slot != nil && last != nil && *last > snapHistoryRetentionSlots && *slot < *last-snapHistoryRetentionSlots {
			return nil
		}
		if cursor.Header().BlockAccessListHash != nil {
			data, err := n.blockchain.BlockAccessList(cursor.Hash())
			if err != nil {
				return err
			}
			if data == nil {
				data, err = n.ethHandler.FetchBlockAccessList(ctx, cursor.Hash())
				if err != nil {
					return err
				}
				if err := n.blockchain.StoreBlockAccessListRLP(cursor.Hash(), data); err != nil {
					return err
				}
			}
		}
		if err := n.blockchain.CompleteSnapHistoryBlock(cursor.Hash()); err != nil {
			return err
		}
		if cursor.NumberU64() == 0 {
			slog.Info("snap historical bodies and BALs backfilled to genesis")
			return nil
		}
		parent := n.blockchain.Genesis()
		if cursor.NumberU64() != 1 {
			parent, err = n.ethHandler.FetchBlock(ctx, cursor.ParentHash())
			if err != nil {
				return err
			}
		}
		if err := n.blockchain.StageSnapHistoryParent(cursor.Hash(), parent); err != nil {
			return err
		}
		cursor = parent
	}
}
