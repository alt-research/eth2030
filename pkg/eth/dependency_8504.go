package eth

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/p2p"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

// hgtp/1 is an explicitly native experimental capability. EIP-8504 assigns no
// official eth codes. It reuses the pinned ethrex devnet wrapper message bodies,
// with a separate namespace and a chain/profile handshake, so it cannot collide
// with eth/72 cell exchange or claim interoperability with an assigned standard.
const (
	dependencyStatus8504 = iota
	dependencyAnnounce8504
	dependencyRequest8504
	dependencyResponse8504
	maxDependencyAnnouncements8504 = 4096
	maxDependencyRequests8504      = 256
)

var dependencyProfile8504 = crypto.Keccak256Hash([]byte("eth2030/hgtp/1:8504@96da3c17047ad31608c40bfe75049db78dc4417d:EthLib@c4474ed422d83b6b735ef802ee43960abe1b788f:7740:495360:3000"))

type dependencyAnnouncement8504 struct {
	WrapperHash, TransactionHash types.Hash
	Length                       uint64
}
type dependencyRequestPacket8504 struct {
	ID     uint64
	Hashes []types.Hash
}
type dependencyResponsePacket8504 struct {
	ID       uint64
	Wrappers []gethrlp.RawValue
}
type dependencyStatusPacket8504 struct {
	Network          uint64
	Genesis, Profile types.Hash
}
type dependencyRequestState8504 struct {
	announced map[types.Hash]dependencyAnnouncement8504
	expires   time.Time
}
type dependencyPeer8504 struct {
	id        string
	transport p2p.Transport
	send      sync.Mutex
}

func (p *dependencyPeer8504) write(code uint64, value any) error {
	raw, err := gethrlp.EncodeToBytes(value)
	if err != nil {
		return err
	}
	p.send.Lock()
	defer p.send.Unlock()
	return p.transport.WriteMsg(p2p.Msg{Code: code, Size: uint32(len(raw)), Payload: raw})
}

func (h *Handler) EnableDependencyProtocol8504(enabled bool) { h.dependencyEnabled8504 = enabled }
func (h *Handler) dependencyProtocol8504() p2p.Protocol {
	return p2p.Protocol{Name: "hgtp", Version: 1, Length: 4, Run: func(peer *p2p.Peer, t p2p.Transport) error { return h.runDependencyPeer8504(peer.ID(), t) }}
}

func (h *Handler) admittedDependencyAnnouncements8504(txs []*types.Transaction) []dependencyAnnouncement8504 {
	var out []dependencyAnnouncement8504
	for _, candidate := range txs {
		tx := h.txPool.Get(candidate.Hash())
		if tx == nil || !tx.HasDependencies8504() {
			continue
		}
		raw, err := types.EncodeDependencyWrapper8504(tx)
		if err != nil {
			continue
		}
		out = append(out, dependencyAnnouncement8504{crypto.Keccak256Hash(raw), tx.Hash(), uint64(len(raw))})
		if len(out) == maxDependencyAnnouncements8504 {
			break
		}
	}
	return out
}
func (h *Handler) announceDependencies8504(txs []*types.Transaction, exclude string) {
	if !h.dependencyEnabled8504 {
		return
	}
	ann := h.admittedDependencyAnnouncements8504(txs)
	if len(ann) == 0 {
		return
	}
	h.dependencyMu8504.RLock()
	peers := make([]*dependencyPeer8504, 0, len(h.dependencyPeers8504))
	for id, p := range h.dependencyPeers8504 {
		if id != exclude {
			peers = append(peers, p)
		}
	}
	h.dependencyMu8504.RUnlock()
	for _, p := range peers {
		if err := p.write(dependencyAnnounce8504, ann); err != nil {
			_ = p.transport.Close()
		}
	}
}

func (h *Handler) runDependencyPeer8504(id string, t p2p.Transport) error {
	peer := &dependencyPeer8504{id: id, transport: t}
	status := dependencyStatusPacket8504{h.networkID, h.chain.Genesis().Hash(), dependencyProfile8504}
	if err := peer.write(dependencyStatus8504, status); err != nil {
		return err
	}
	msg, err := t.ReadMsg()
	if err != nil {
		return err
	}
	var remote dependencyStatusPacket8504
	if msg.Code != dependencyStatus8504 || len(msg.Payload) > 256 || gethrlp.DecodeBytes(msg.Payload, &remote) != nil || remote != status {
		return errors.New("hgtp/1 chain or pinned profile mismatch")
	}
	h.dependencyMu8504.Lock()
	if h.dependencyPeers8504 == nil {
		h.dependencyPeers8504 = make(map[string]*dependencyPeer8504)
	}
	h.dependencyPeers8504[id] = peer
	h.dependencyMu8504.Unlock()
	defer func() { h.dependencyMu8504.Lock(); delete(h.dependencyPeers8504, id); h.dependencyMu8504.Unlock() }()
	var pending []*types.Transaction
	for _, list := range h.txPool.Pending() {
		pending = append(pending, list...)
	}
	if ann := h.admittedDependencyAnnouncements8504(pending); len(ann) > 0 {
		if err := peer.write(dependencyAnnounce8504, ann); err != nil {
			return err
		}
	}
	requests := make(map[uint64]dependencyRequestState8504)
	requestedTx := make(map[types.Hash]bool)
	var requestedOrder []types.Hash
	var next uint64
	missed := 0
	var queue []dependencyAnnouncement8504
	queuedTx := make(map[types.Hash]bool)
	drain := func() error {
		for len(queue) > 0 && len(requests) < 16 {
			want := make(map[types.Hash]dependencyAnnouncement8504)
			var hashes []types.Hash
			size := uint64(0)
			for len(queue) > 0 && len(hashes) < maxDependencyRequests8504 {
				a := queue[0]
				if len(hashes) > 0 && size+a.Length > 2*1024*1024 {
					break
				}
				queue = queue[1:]
				delete(queuedTx, a.TransactionHash)
				if h.txPool.Get(a.TransactionHash) != nil || requestedTx[a.TransactionHash] {
					continue
				}
				if _, duplicate := want[a.WrapperHash]; duplicate {
					continue
				}
				hashes = append(hashes, a.WrapperHash)
				want[a.WrapperHash] = a
				size += a.Length
				requestedTx[a.TransactionHash] = true
				requestedOrder = append(requestedOrder, a.TransactionHash)
				if len(requestedOrder) > 16384 {
					delete(requestedTx, requestedOrder[0])
					requestedOrder = requestedOrder[1:]
				}
			}
			if len(hashes) != 0 {
				next++
				requests[next] = dependencyRequestState8504{want, time.Now().Add(15 * time.Second)}
				if err := peer.write(dependencyRequest8504, dependencyRequestPacket8504{next, hashes}); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for {
		msg, err := t.ReadMsg()
		if err != nil {
			return err
		}
		if len(msg.Payload) > 4*1024*1024 || int(msg.Size) != len(msg.Payload) {
			return errors.New("oversized or inconsistent dependency message")
		}
		for key, r := range requests {
			if time.Now().After(r.expires) {
				missed += len(r.announced)
				delete(requests, key)
			}
		}
		if missed >= 4096 {
			return errors.New("too many undelivered dependency wrappers")
		}
		switch msg.Code {
		case dependencyAnnounce8504:
			if len(msg.Payload) > maxDependencyAnnouncements8504*80+64 {
				return errors.New("oversized dependency announcement")
			}
			var ann []dependencyAnnouncement8504
			if err := gethrlp.DecodeBytes(msg.Payload, &ann); err != nil {
				return err
			}
			if len(ann) > maxDependencyAnnouncements8504 {
				return errors.New("too many dependency announcements")
			}
			for _, a := range ann {
				if a.Length == 0 || a.Length > types.MaxDependencyWrapperBytes8504 {
					return errors.New("invalid dependency announced length")
				}
				if h.txPool.Get(a.TransactionHash) != nil || requestedTx[a.TransactionHash] || queuedTx[a.TransactionHash] {
					continue
				}
				if len(queue) == maxDependencyAnnouncements8504 {
					return errors.New("dependency announcement queue limit")
				}
				queue = append(queue, a)
				queuedTx[a.TransactionHash] = true
			}

		case dependencyRequest8504:
			if len(msg.Payload) > maxDependencyRequests8504*33+64 {
				return errors.New("oversized dependency request")
			}
			var req dependencyRequestPacket8504
			if err := gethrlp.DecodeBytes(msg.Payload, &req); err != nil {
				return err
			}
			if len(req.Hashes) > maxDependencyRequests8504 {
				return errors.New("too many requested dependency wrappers")
			}
			response := dependencyResponsePacket8504{ID: req.ID}
			seen := make(map[types.Hash]bool)
			size := 0
			if source, ok := h.txPool.(interface{ DependencyWrapper8504(types.Hash) []byte }); ok {
				for _, hash := range req.Hashes {
					if seen[hash] {
						continue
					}
					seen[hash] = true
					raw := source.DependencyWrapper8504(hash)
					if len(raw) == 0 {
						continue
					}
					if size+len(raw) > 2*1024*1024 {
						break
					}
					response.Wrappers = append(response.Wrappers, raw)
					size += len(raw)
				}
			}
			if err := peer.write(dependencyResponse8504, response); err != nil {
				return err
			}
		case dependencyResponse8504:
			var response dependencyResponsePacket8504
			if err := gethrlp.DecodeBytes(msg.Payload, &response); err != nil {
				return err
			}
			req, ok := requests[response.ID]
			if !ok {
				return errors.New("unsolicited dependency wrapper response")
			}
			delete(requests, response.ID)
			if len(response.Wrappers) > maxDependencyRequests8504 {
				return errors.New("too many dependency response wrappers")
			}
			for _, raw := range response.Wrappers {
				hash := crypto.Keccak256Hash(raw)
				ann, asked := req.announced[hash]
				if !asked || ann.Length != uint64(len(raw)) {
					return errors.New("unrequested dependency hash or length")
				}
				delete(req.announced, hash)
				tx, err := types.DecodeDependencyWrapper8504(raw)
				if err != nil {
					return err
				}
				if tx.Hash() != ann.TransactionHash {
					return errors.New("dependency transaction announcement mismatch")
				}
				if h.txPool.Get(tx.Hash()) != nil {
					continue
				}
				if err := h.txPool.AddRemote(tx); err != nil {
					if h.txPool.Get(tx.Hash()) != nil {
						continue
					}
					return fmt.Errorf("dependency admission: %w", err)
				}
				h.announceTransactions([]*types.Transaction{tx}, id)
			}
			missed += len(req.announced)
		default:
			return errors.New("unknown hgtp/1 message code")
		}
		if err := drain(); err != nil {
			return err
		}
	}
}
