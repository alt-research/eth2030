package node

import (
	"context"
	"errors"
	"fmt"

	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	p2psnap "github.com/eth2030/eth2030/p2p/snap"
	"github.com/eth2030/eth2030/sync/statesync"
)

func (n *Node) snap2Clients() []*p2psnap.Client {
	var clients []*p2psnap.Client
	for _, client := range n.snapHandler.Clients() {
		if client.Version() >= 2 && n.ethHandler.Peers().Peer(client.ID()) != nil {
			clients = append(clients, client)
		}
	}
	return clients
}

func (n *Node) snap2Peer(ctx context.Context) *snapFailoverPeer {
	peer := &snapFailoverPeer{ctx: ctx}
	for _, verified := range p2psnap.VerifiedPeers(ctx, n.snap2Clients()) {
		peer.peers = append(peer.peers, verified)
	}
	return peer
}

func (n *Node) snapBlock(ctx context.Context, hash types.Hash) (*types.Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if block := n.blockchain.GetBlock(hash); block != nil {
		return block, nil
	}
	return n.ethHandler.FetchBlock(ctx, hash)
}

func (n *Node) syncFinalizedPivot(ctx context.Context, pivot *types.Block) (bool, error) {
	if len(n.snap2Clients()) == 0 {
		return n.syncFinalizedPivotV1(ctx, pivot)
	}
	if n.blockchain.CurrentBlock().NumberU64() != 0 {
		return false, nil
	}
	return n.syncFinalizedPivot2(ctx, pivot)
}

func (n *Node) fetchSnapBALPath(ctx context.Context, headers []*types.Header) ([]statesync.BALStep, error) {
	var result []statesync.BALStep
	for len(result) < len(headers) {
		remaining := headers[len(result):]
		if len(remaining) > p2psnap.MaxBlockAccessLists {
			remaining = remaining[:p2psnap.MaxBlockAccessLists]
		}
		progress := false
		for _, client := range n.snap2Clients() {
			data, err := client.RequestBlockAccessLists(ctx, remaining, p2psnap.BALSoftResponseLimit)
			if err != nil {
				continue
			}
			for i, item := range data {
				if item == nil {
					break
				}
				result = append(result, statesync.BALStep{Header: remaining[i], Encoded: item})
				progress = true
			}
			if progress {
				break
			}
		}
		if !progress {
			return nil, fmt.Errorf("snap/2 required BAL unavailable")
		}
	}
	return result, nil
}

func reverseSnapHeaders(headers []*types.Header) {
	for i, j := 0, len(headers)-1; i < j; i, j = i+1, j-1 {
		headers[i], headers[j] = headers[j], headers[i]
	}
}

func (n *Node) resolveSnapPivot(ctx context.Context, old *types.Header) (*statesync.PivotUpdate, error) {
	target, _, _ := n.executionSync.Status()
	if target.Finalized == (types.Hash{}) || target.Finalized == old.Hash() {
		return nil, nil
	}
	latest, err := n.snapBlock(ctx, target.Finalized)
	if err != nil {
		return nil, err
	}
	left, right := old, latest.Header()
	var rollback, forward []*types.Header
	validator := coreblock.NewBlockValidator(n.blockchain.Config())
	parent := func(header *types.Header) (*types.Header, error) {
		if header.Number == nil || !header.Number.IsUint64() || header.Number.Sign() == 0 {
			return nil, fmt.Errorf("snap/2 pivot has no common ancestor")
		}
		block, err := n.snapBlock(ctx, header.ParentHash)
		if err != nil {
			return nil, err
		}
		if err := validator.ValidateHeader(header, block.Header()); err != nil {
			return nil, err
		}
		return block.Header(), nil
	}
	for left.Hash() != right.Hash() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if left.Number == nil || right.Number == nil || !left.Number.IsUint64() || !right.Number.IsUint64() {
			return nil, fmt.Errorf("snap/2 invalid pivot ancestry")
		}
		if left.Number.Cmp(right.Number) >= 0 {
			rollback = append(rollback, left)
			left, err = parent(left)
		} else {
			forward = append(forward, right)
			right, err = parent(right)
		}
		if err != nil {
			return nil, &statesync.PivotRestart{Header: latest.Header()}
		}
	}
	reverseSnapHeaders(rollback)
	reverseSnapHeaders(forward)
	oldPath, err := n.fetchSnapBALPath(ctx, rollback)
	if err != nil {
		return nil, &statesync.PivotRestart{Header: latest.Header()}
	}
	newPath, err := n.fetchSnapBALPath(ctx, forward)
	if err != nil {
		return nil, &statesync.PivotRestart{Header: latest.Header()}
	}
	return &statesync.PivotUpdate{Common: left, Header: latest.Header(), Rollback: oldPath, Forward: newPath, Peer: n.snap2Peer(ctx)}, nil
}

func (n *Node) snapPivotHistory(ctx context.Context, pivot *types.Block) ([]*types.Header, error) {
	var history []*types.Header
	child := pivot
	validator := coreblock.NewBlockValidator(n.blockchain.Config())
	for len(history) < 256 && child.NumberU64() > 0 {
		parent, err := n.snapBlock(ctx, child.ParentHash())
		if err != nil {
			return nil, err
		}
		if err := validator.ValidateHeader(child.Header(), parent.Header()); err != nil {
			return nil, err
		}
		history = append(history, parent.Header())
		child = parent
	}
	return history, nil
}

func (n *Node) syncFinalizedPivot2(ctx context.Context, pivot *types.Block) (bool, error) {
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		var writer *statesync.SnapshotWriter
		snapshot, err := state.LoadStateSnapshot(n.db, pivot.Root())
		if err == nil {
			// Reconstruct local nodes from the authenticated complete snapshot.
			writer = statesync.NewSnapshotWriterFromSnapshot(n.db, snapshot)
			if err := writer.Finalize(pivot.Root()); err != nil {
				return false, err
			}
		} else if errors.Is(err, rawdb.ErrNotFound) {
			writer = statesync.NewSnapshotWriter(n.db)
		} else {
			return false, err
		}
		scheduler := statesync.NewStateSyncScheduler(writer, nil)
		if err := scheduler.SetPivot(pivot.Header()); err != nil {
			return false, err
		}
		if err := scheduler.SetPivotResolver(n.resolveSnapPivot); err != nil {
			return false, err
		}
		if snapshot == nil {
			if err := scheduler.RunContext(ctx, n.snap2Peer(ctx)); err != nil {
				var restart *statesync.PivotRestart
				if !errors.As(err, &restart) {
					return false, err
				}
				pivot, err = n.snapBlock(ctx, restart.Header.Hash())
				if err != nil {
					return false, err
				}
				continue
			}
			if scheduler.PivotHeader().Hash() != pivot.Hash() {
				pivot, err = n.snapBlock(ctx, scheduler.PivotHeader().Hash())
				if err != nil {
					return false, err
				}
			}
		}
		history, err := n.snapPivotHistory(ctx, pivot)
		if err != nil {
			return false, err
		}
		if err := n.executionSync.AdoptFinalized(ctx, pivot, func() error { return n.blockchain.ImportSnapPivot(pivot, history) }); err != nil {
			return false, err
		}
		return true, nil
	}
}
