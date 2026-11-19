package eth

import (
	"fmt"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
)

const blockRangeUpdateMsg = p2p.BlockRangeUpdateMsg

// status69 is the eth/69+ seven-field status. It intentionally omits TD.
type status69 struct {
	Version    uint32
	Network    uint64
	Genesis    types.Hash
	ForkID     p2p.ForkID
	Earliest   uint64
	Latest     uint64
	LatestHash types.Hash
}

type blockRange struct {
	Earliest, Latest uint64
	LatestHash       types.Hash
}

func (ep *EthPeer) handshake69(local status69) (*p2p.StatusData, error) {
	if err := ep.sendMessage(p2p.StatusMsg, &local); err != nil {
		return nil, err
	}
	msg, err := ep.transport.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.Code != p2p.StatusMsg {
		return nil, fmt.Errorf("expected eth status")
	}
	var remote status69
	if err := decodeMsg(msg, &remote); err != nil {
		return nil, err
	}
	if remote.Version != local.Version {
		return nil, ErrIncompatibleVersion
	}
	if remote.Network != local.Network {
		return nil, ErrNetworkIDMismatch
	}
	if remote.Genesis != local.Genesis {
		return nil, ErrGenesisMismatch
	}
	if remote.Earliest > remote.Latest {
		return nil, fmt.Errorf("invalid peer block range")
	}
	ep.requestMu.Lock()
	ep.blockRange = blockRange{remote.Earliest, remote.Latest, remote.LatestHash}
	ep.localRange = blockRange{local.Earliest, local.Latest, local.LatestHash}
	ep.requestMu.Unlock()
	ep.peer.SetVersion(remote.Version)
	ep.peer.SetHead(remote.LatestHash, new(big.Int))
	return &p2p.StatusData{ProtocolVersion: remote.Version, NetworkID: remote.Network, Genesis: remote.Genesis, ForkID: remote.ForkID, Head: remote.LatestHash, TD: new(big.Int)}, nil
}

func (h *Handler) handleBlockRange(ep *EthPeer, msg p2p.Msg) error {
	if ep.Peer().Version() < ETH69 {
		return fmt.Errorf("block range update requires eth/69")
	}
	var update blockRange
	if err := decodeMsg(msg, &update); err != nil {
		return err
	}
	if update.Earliest > update.Latest {
		return fmt.Errorf("invalid peer block range")
	}
	ep.requestMu.Lock()
	ep.blockRange = update
	ep.requestMu.Unlock()
	ep.peer.SetHead(update.LatestHash, new(big.Int))
	return nil
}

// NotifyBlockRange queues the newest local range without making canonical
// head processing wait for a peer's network writer.
func (h *Handler) NotifyBlockRange() {
	head := h.chain.CurrentBlock()
	if head == nil {
		return
	}
	earliest := uint64(0)
	if history, ok := h.chain.(interface{ HistoryOldestBlock() uint64 }); ok {
		earliest = history.HistoryOldestBlock()
	}
	next := blockRange{earliest, head.NumberU64(), head.Hash()}
	h.ethPeersMu.RLock()
	defer h.ethPeersMu.RUnlock()
	for _, ep := range h.ethPeers {
		ep.queueRange(next)
	}
}
func (ep *EthPeer) queueRange(next blockRange) {
	if ep.peer.Version() < ETH69 {
		return
	}
	ep.requestMu.Lock()
	old := ep.localRange
	ep.requestMu.Unlock()
	if next == old {
		return
	}
	if next.Earliest == old.Earliest && next.Latest >= old.Latest && next.Latest-old.Latest < 32 {
		return
	}
	select {
	case ep.rangeUpdates <- next:
	default:
		select {
		case <-ep.rangeUpdates:
		default:
		}
		select {
		case ep.rangeUpdates <- next:
		default:
		}
	}
}
func (ep *EthPeer) runRangeUpdates() {
	for {
		select {
		case <-ep.done:
			return
		case next := <-ep.rangeUpdates:
			if err := ep.sendMessage(blockRangeUpdateMsg, next); err != nil {
				ep.transport.Close()
				return
			}
			ep.requestMu.Lock()
			ep.localRange = next
			ep.requestMu.Unlock()
		}
	}
}
