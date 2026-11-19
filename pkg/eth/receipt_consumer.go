package eth

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/p2p/ethproto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func (h *Handler) receiptBounds(packet any) ([][]uint64, error) {
	var hashes []types.Hash
	var offset uint64
	switch p := packet.(type) {
	case *p2p.GetReceiptsPacketV70:
		hashes, offset = p.Hashes, p.FirstBlockReceiptIndex
	case *p2p.GetReceiptsPacket:
		hashes = []types.Hash(p.Hashes)
	default:
		return nil, nil
	}
	if len(hashes) > MaxBodies {
		return nil, fmt.Errorf("too many receipt blocks")
	}
	limits := make([][]uint64, 0, len(hashes))
	for i, hash := range hashes {
		block := h.chain.GetBlock(hash)
		if block == nil {
			return nil, ErrReceiptsBlockUnknown
		}
		start := uint64(0)
		if i == 0 {
			start = offset
		}
		if start > uint64(len(block.Transactions())) {
			return nil, ErrReceiptsCountMismatch
		}
		page := make([]uint64, 0, len(block.Transactions())-int(start))
		for _, tx := range block.Transactions()[start:] {
			page = append(page, tx.Gas()/8)
		}
		limits = append(limits, page)
	}
	return limits, nil
}

func (h *Handler) consumeReceipts(ep *EthPeer, msg p2p.Msg) error {
	stream := gethrlp.NewStream(bytes.NewReader(msg.Payload), uint64(len(msg.Payload)))
	if _, err := stream.List(); err != nil {
		return err
	}
	id, err := stream.Uint64()
	if err != nil {
		return err
	}
	h.syncMu.Lock()
	request := h.syncRequests[syncRequestKey{ep, id}]
	h.syncMu.Unlock()
	if request == nil || request.code != p2p.ReceiptsMsg {
		return nil
	} // late/unsolicited: no allocation or storage
	_, incomplete, pages, err := ethproto.DecodeReceiptsLimited(msg.Payload, ep.Peer().Version(), request.receiptLimits)
	if errors.Is(err, ethproto.ErrReceiptSizeLimit) {
		err = ErrReceiptsTooLarge
	}
	if errors.Is(err, ethproto.ErrReceiptCountLimit) {
		err = ErrReceiptsCountMismatch
	}
	h.deliverSyncResponse(ep, id, p2p.ReceiptsMsg, syncResponse{receipts: pages, incomplete: incomplete, err: err})
	return nil
}
