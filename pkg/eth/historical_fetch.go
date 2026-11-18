package eth

import (
	"context"
	"fmt"
	"time"

	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/p2p"
)

func (h *Handler) historicalPeers(minVersion uint32) []*EthPeer {
	h.ethPeersMu.RLock()
	defer h.ethPeersMu.RUnlock()
	peers := make([]*EthPeer, 0, len(h.ethPeers))
	for _, peer := range h.ethPeers {
		if peer.Peer().Version() >= minVersion {
			peers = append(peers, peer)
		}
	}
	return peers
}

// FetchBlockAccessList restores a known block's committed BAL from a correlated
// eth/71 response. Unavailable or invalid data never becomes verified storage.
func (h *Handler) FetchBlockAccessList(ctx context.Context, hash types.Hash) ([]byte, error) {
	block := h.chain.GetBlock(hash)
	if block == nil || block.Header().BlockAccessListHash == nil {
		return nil, fmt.Errorf("known Amsterdam block required")
	}
	last := fmt.Errorf("no eth/71 peer has the requested BAL")
	for _, ep := range h.historicalPeers(ETH71) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		response, err := h.syncRequest(requestCtx, ep, p2p.GetBlockAccessListsMsg, func(id uint64) any {
			return &p2p.GetBlockAccessListsPacket{RequestID: id, BlockHashes: []types.Hash{hash}}
		})
		cancel()
		if err != nil {
			last = err
			continue
		}
		if len(response.bals) != 1 {
			continue
		}
		if len(response.bals[0]) == 1 && response.bals[0][0] == 0xc0 && *block.Header().BlockAccessListHash != crypto.Keccak256Hash(response.bals[0]) {
			last = ErrBALUnavailable
			continue
		}
		if err := ValidateBlockAccessListForHeader(response.bals[0], block.Header()); err != nil {
			last = err
			continue
		}
		return append([]byte(nil), response.bals[0]...), nil
	}
	return nil, last
}

// FetchReceipts downloads one complete receipt list, following eth/70 pages as
// needed. Partial input is bounded and remains untrusted until the root matches.
func (h *Handler) FetchReceipts(ctx context.Context, hash types.Hash) ([]*types.Receipt, error) {
	block := h.chain.GetBlock(hash)
	if block == nil {
		return nil, ErrReceiptsBlockUnknown
	}
	last := fmt.Errorf("no peer has the requested receipts")
	for _, ep := range h.historicalPeers(ETH68) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		receipts, err := h.fetchPeerReceipts(attemptCtx, ep, block)
		cancel()
		if err != nil {
			last = err
			continue
		}
		if consumer, ok := h.chain.(interface {
			StoreReceivedReceipts(types.Hash, []*types.Receipt) error
		}); ok {
			if err := consumer.StoreReceivedReceipts(hash, receipts); err != nil {
				return nil, err
			}
		}
		return receipts, nil
	}
	return nil, last
}

func (h *Handler) fetchPeerReceipts(ctx context.Context, ep *EthPeer, block *types.Block) ([]*types.Receipt, error) {
	var receipts []*types.Receipt
	var downloaded uint64
	for {
		requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		response, err := h.syncRequest(requestCtx, ep, p2p.GetReceiptsMsg, func(id uint64) any {
			if ep.Peer().Version() >= ETH70 {
				return &p2p.GetReceiptsPacketV70{RequestID: id, FirstBlockReceiptIndex: uint64(len(receipts)), Hashes: []types.Hash{block.Hash()}}
			}
			return &p2p.GetReceiptsPacket{RequestID: id, Hashes: []types.Hash{block.Hash()}}
		})
		cancel()
		if err != nil {
			return nil, err
		}
		if len(response.receipts) != 1 {
			return nil, fmt.Errorf("receipt response is unavailable or has extra pages")
		}
		page := response.receipts[0]
		if len(page) > len(block.Transactions())-len(receipts) {
			return nil, ErrReceiptsCountMismatch
		}
		for i, receipt := range page {
			if receipt == nil {
				return nil, fmt.Errorf("nil receipt")
			}
			encoded, err := receipt.EncodeRLP()
			if err != nil {
				return nil, err
			}
			size := uint64(len(encoded))
			if size > block.Transactions()[len(receipts)+i].Gas()/8 {
				return nil, ErrReceiptsTooLarge
			}
			if size > block.GasLimit() || downloaded > block.GasLimit()-size {
				return nil, ErrReceiptsTotalTooLarge
			}
			downloaded += size
		}
		receipts = append(receipts, page...)
		if response.incomplete {
			if len(page) == 0 || len(receipts) >= len(block.Transactions()) {
				return nil, fmt.Errorf("incomplete receipt response made no valid progress")
			}
			continue
		}
		if len(receipts) != len(block.Transactions()) {
			return nil, ErrReceiptsCountMismatch
		}
		if coreblock.ComputeReceiptsRoot(receipts) != block.Header().ReceiptHash {
			return nil, fmt.Errorf("receipt root mismatch")
		}
		return receipts, nil
	}
}
