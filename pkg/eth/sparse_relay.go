package eth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/txpool/sparse"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

type relayKey struct {
	peer string
	id   uint64
}
type relayRequest struct {
	code    uint64
	hashes  []types.Hash
	mask    types.CellMask
	expires time.Time
}
type relayAnnouncement struct {
	mask  types.CellMask
	retry time.Time
}
type relayRecord struct {
	provider bool
	peers    map[string]relayAnnouncement
	expires  time.Time
}
type sparseRelay struct {
	mu                sync.Mutex
	store             *sparse.Store
	records           map[types.Hash]*relayRecord
	requests          map[relayKey]relayRequest
	seed              [32]byte
	custody           types.CellMask
	eager             bool
	custodyGeneration uint64
}

func newSparseRelay(pool TxPool) *sparseRelay {
	var validate func(*types.Transaction) error
	if v, ok := pool.(interface {
		ValidateSparseBlobMetadata(*types.Transaction) error
	}); ok {
		validate = v.ValidateSparseBlobMetadata
	}
	r := &sparseRelay{store: sparse.New(validate), records: make(map[types.Hash]*relayRecord), requests: make(map[relayKey]relayRequest)}
	// A failed entropy source cannot silently create predictable role/noise.
	if _, err := rand.Read(r.seed[:]); err != nil {
		r.eager = true
	}
	return r
}
func (r *sparseRelay) prune() {
	now := time.Now()
	for k, v := range r.requests {
		if !now.Before(v.expires) {
			delete(r.requests, k)
		}
	}
	for k, v := range r.records {
		if !now.Before(v.expires) {
			delete(r.records, k)
		}
	}
}
func (r *sparseRelay) role(hash types.Hash) bool {
	input := append(append([]byte(nil), r.seed[:]...), hash[:]...)
	sum := sha256.Sum256(input)
	return binary.BigEndian.Uint64(sum[:8]) < ^uint64(0)/100*15
}
func (h *Handler) relayTransaction(hash types.Hash) *types.Transaction {
	if tx := h.txPool.Get(hash); tx != nil {
		return tx
	}
	return h.relay.store.Get(hash)
}
func (h *Handler) relayMask(hash types.Hash) types.CellMask {
	if tx := h.txPool.Get(hash); tx != nil && types.HasFullBlobSidecar(tx) {
		return types.AllCells
	}
	return h.relay.store.Mask(hash)
}
func (h *Handler) sendRelayAnnouncement(ep *EthPeer, txs []*types.Transaction) error {
	if ep.Peer().Version() < ETH72 {
		return ep.SendNewPooledTransactionHashes(txs)
	}
	for _, tx := range txs {
		raw, err := tx.EncodeRLP()
		if err != nil {
			return err
		}
		packet := NewPooledTxHashesMsg72{Types: []byte{tx.Type()}, Sizes: []uint32{uint32(len(raw))}, Hashes: []types.Hash{tx.Hash()}}
		if tx.Type() == types.BlobTxType {
			mask := h.relayMask(tx.Hash())
			if mask != (types.CellMask{}) && mask != types.AllCells {
				h.relay.mu.Lock()
				mask = mask.Intersection(h.relay.custody)
				h.relay.mu.Unlock()
			}
			packet.Mask = append([]byte(nil), mask[:]...)
		}
		if err := ep.sendMessage(p2p.NewPooledTransactionHashesMsg, &packet); err != nil {
			return err
		}
	}
	return nil
}
func (h *Handler) handleSparseAnnouncement(ep *EthPeer, msg p2p.Msg) error {
	var ann NewPooledTxHashesMsg72
	if err := decodeMsg(msg, &ann); err != nil {
		return err
	}
	n := len(ann.Hashes)
	if n > 4096 || len(ann.Types) != n || len(ann.Sizes) != n {
		return errors.New("invalid eth/72 announcement dimensions")
	}
	hasBlob := false
	for _, kind := range ann.Types {
		hasBlob = hasBlob || kind == types.BlobTxType
	}
	if hasBlob && len(ann.Mask) != 16 || !hasBlob && len(ann.Mask) != 0 && len(ann.Mask) != 16 {
		return errors.New("invalid announcement cell mask")
	}
	var mask types.CellMask
	copy(mask[:], ann.Mask)
	var ordinary []types.Hash
	for i, hash := range ann.Hashes {
		if ann.Sizes[i] == 0 || ann.Sizes[i] > MaxMessageSize {
			return errors.New("invalid transaction announcement size")
		}
		if ann.Types[i] != types.BlobTxType {
			if h.txPool.Get(hash) == nil {
				ordinary = append(ordinary, hash)
			}
			continue
		}
		if h.txPool.Get(hash) != nil {
			continue
		}
		r := h.relay
		r.mu.Lock()
		r.prune()
		record := r.records[hash]
		if record == nil && len(r.records) < sparse.MaxEntries {
			record = &relayRecord{provider: r.role(hash), peers: make(map[string]relayAnnouncement), expires: time.Now().Add(sparse.Lifetime)}
			r.records[hash] = record
		}
		if record != nil && len(record.peers) < 128 {
			record.peers[ep.ID()] = relayAnnouncement{mask: mask}
		}
		r.mu.Unlock()
		if err := h.scheduleBlob(hash); err != nil {
			return err
		}
	}
	if len(ordinary) > 0 {
		return h.requestRelay(ep, p2p.PooledTransactionsMsg, ordinary, types.CellMask{})
	}
	return nil
}
func (h *Handler) requestRelay(ep *EthPeer, code uint64, hashes []types.Hash, mask types.CellMask) error {
	r := h.relay
	r.mu.Lock()
	r.prune()
	if len(r.requests) >= 128 {
		r.mu.Unlock()
		return nil
	}
	for _, req := range r.requests {
		for _, pending := range req.hashes {
			for _, hash := range hashes {
				if pending == hash {
					r.mu.Unlock()
					return nil
				}
			}
		}
	}
	id := ep.nextRequestID()
	key := relayKey{ep.ID(), id}
	r.requests[key] = relayRequest{code: code, hashes: append([]types.Hash(nil), hashes...), mask: mask, expires: time.Now().Add(5 * time.Second)}
	r.mu.Unlock()
	var err error
	if code == CellsMsg {
		err = ep.sendMessage(GetCellsMsg, &GetCellsPacket{RequestID: id, Hashes: hashes, Mask: mask})
	} else {
		err = ep.sendMessage(p2p.GetPooledTransactionsMsg, &p2p.GetPooledTransactionsPacket{RequestID: id, Hashes: hashes})
	}
	if err != nil {
		r.mu.Lock()
		delete(r.requests, key)
		r.mu.Unlock()
	}
	return err
}
func (h *Handler) takeRelayRequest(ep *EthPeer, id, code uint64) (relayRequest, error) {
	r := h.relay
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prune()
	key := relayKey{ep.ID(), id}
	req, ok := r.requests[key]
	if !ok || req.code != code {
		return relayRequest{}, errors.New("unsolicited sparse response")
	}
	delete(r.requests, key)
	for _, hash := range req.hashes {
		if record := r.records[hash]; record != nil {
			ann := record.peers[ep.ID()]
			ann.retry = time.Now().Add(2 * time.Second)
			record.peers[ep.ID()] = ann
		}
	}
	return req, nil
}
func (h *Handler) scheduleBlob(hash types.Hash) error {
	if h.txPool.Get(hash) != nil {
		return nil
	}
	r := h.relay
	hasMetadata := r.store.Has(hash)
	available := r.store.Mask(hash)
	r.mu.Lock()
	r.prune()
	record := r.records[hash]
	if record == nil {
		r.mu.Unlock()
		return nil
	}
	for _, req := range r.requests {
		for _, pending := range req.hashes {
			if pending == hash {
				r.mu.Unlock()
				return nil
			}
		}
	}
	full := record.provider || r.eager
	wanted := r.custody.Difference(available)
	if full {
		wanted = types.AllCells.Difference(available)
	}
	providers := 0
	for _, ann := range record.peers {
		if ann.mask == types.AllCells {
			providers++
		}
	}
	var peerID string
	var mask types.CellMask
	code := uint64(p2p.PooledTransactionsMsg)
	for id, ann := range record.peers {
		if time.Now().Before(ann.retry) {
			continue
		}
		if !hasMetadata {
			peerID = id
			break
		}
		if !full && providers < 2 {
			continue
		}
		if record.provider && ann.mask != types.AllCells && providers > 0 {
			continue
		}
		mask = wanted.Intersection(ann.mask)
		if r.eager && !record.provider && mask.OneCount() > 64 {
			indices := mask.Indices()[:64]
			mask = types.CellMask{}
			for _, index := range indices {
				mask[index/8] |= 1 << uint(index%8)
			}
		}
		if mask.OneCount() == 0 {
			continue
		}
		if !full && ann.mask == types.AllCells {
			var noise [1]byte
			if _, err := rand.Read(noise[:]); err != nil {
				r.mu.Unlock()
				return err
			}
			for i := 0; i < 128; i++ {
				index := uint64((int(noise[0]) + i) % 128)
				if !r.custody.IsSet(index) {
					mask[index/8] |= 1 << uint(index%8)
					break
				}
			}
		}
		peerID = id
		code = CellsMsg
		break
	}
	r.mu.Unlock()
	if peerID == "" {
		return nil
	}
	h.ethPeersMu.RLock()
	ep := h.ethPeers[peerID]
	h.ethPeersMu.RUnlock()
	if ep == nil {
		return nil
	}
	return h.requestRelay(ep, code, []types.Hash{hash}, mask)
}
func (h *Handler) handleSparsePooled(ep *EthPeer, msg p2p.Msg) error {
	var packet pooledTransactions72
	if err := decodeMsg(msg, &packet); err != nil {
		return err
	}
	req, err := h.takeRelayRequest(ep, packet.RequestID, p2p.PooledTransactionsMsg)
	if err != nil {
		return err
	}
	position := 0
	for _, tx := range packet.Transactions {
		for position < len(req.hashes) && req.hashes[position] != tx.Hash() {
			position++
		}
		if position == len(req.hashes) {
			return errors.New("unrequested or reordered pooled transaction")
		}
		position++
		if tx.Type() == types.BlobTxType {
			if err := h.relay.store.Stage(tx); err != nil {
				return err
			}
			if err := h.scheduleBlob(tx.Hash()); err != nil {
				return err
			}
		} else if err := h.txPool.AddRemote(tx); err == nil {
			h.announceTransactions([]*types.Transaction{tx}, ep.ID())
		}
	}
	return nil
}
func (h *Handler) handleGetCells(ep *EthPeer, msg p2p.Msg) error {
	if ep.Peer().Version() < ETH72 {
		return errors.New("cell request requires eth/72")
	}
	var req GetCellsPacket
	if err := decodeMsg(msg, &req); err != nil {
		return err
	}
	if len(req.Hashes) > MaxCellHashes {
		return errors.New("too many cell hashes")
	}
	reply := CellsPacket{RequestID: req.RequestID}
	for _, hash := range req.Hashes {
		var cells [][]byte
		mask := req.Mask.Intersection(h.relayMask(hash))
		if mask.OneCount() == 0 {
			continue
		}
		if tx := h.txPool.Get(hash); tx != nil && types.HasFullBlobSidecar(tx) {
			for _, blob := range tx.BlobSidecar().Blobs {
				all, err := bls.DefaultKZGBackend().ComputeCells(blob)
				if err != nil {
					return err
				}
				for _, index := range mask.Indices() {
					cells = append(cells, append([]byte(nil), all[index][:]...))
				}
			}
		} else {
			cells, mask = h.relay.store.Cells(hash, mask)
		}
		if len(cells)*2048 > CellResponseLimit {
			return errors.New("cell response exceeds budget")
		}
		reply.Hashes = []types.Hash{hash}
		reply.Cells = [][][]byte{cells}
		reply.Mask = mask
		break
	}
	return ep.sendMessage(CellsMsg, &reply)
}
func (h *Handler) handleCells(ep *EthPeer, msg p2p.Msg) error {
	if ep.Peer().Version() < ETH72 {
		return errors.New("cell response requires eth/72")
	}
	var response CellsPacket
	if err := decodeMsg(msg, &response); err != nil {
		return err
	}
	req, err := h.takeRelayRequest(ep, response.RequestID, CellsMsg)
	if err != nil {
		return err
	}
	if len(response.Hashes) != len(response.Cells) || len(response.Hashes) > len(req.hashes) || response.Mask.Difference(req.mask).OneCount() != 0 {
		return errors.New("unrequested cell dimensions")
	}
	position := 0
	for i, hash := range response.Hashes {
		for position < len(req.hashes) && req.hashes[position] != hash {
			position++
		}
		if position == len(req.hashes) {
			return errors.New("unrequested or reordered cell transaction")
		}
		position++
		full, err := h.relay.store.AddCells(hash, response.Mask, response.Cells[i])
		if err != nil {
			h.relay.store.Remove(hash)
			return err
		}
		if full != nil {
			if err = h.txPool.AddRemote(full); err != nil {
				return fmt.Errorf("reconstructed blob promotion: %w", err)
			}
			h.relay.store.Remove(hash)
			h.announceTransactions([]*types.Transaction{full}, ep.ID())
		} else if tx := h.relay.store.Get(hash); tx != nil {
			h.announceTransactions([]*types.Transaction{tx}, ep.ID())
		}
	}
	return nil
}
func (h *Handler) runSparseRelay(ep *EthPeer) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var generation uint64
	// Initial pool exchange uses the negotiated announcement schema.
	var pending []*types.Transaction
	for _, txs := range h.txPool.Pending() {
		pending = append(pending, txs...)
	}
	_ = h.sendRelayAnnouncement(ep, pending)
	for {
		select {
		case <-ep.done:
			return
		case <-ticker.C:
			h.relay.mu.Lock()
			h.relay.prune()
			changed := generation != h.relay.custodyGeneration
			generation = h.relay.custodyGeneration
			var hashes []types.Hash
			for hash := range h.relay.records {
				hashes = append(hashes, hash)
			}
			h.relay.mu.Unlock()
			if changed {
				for _, hash := range h.relay.store.Hashes() {
					if tx := h.relay.store.Get(hash); tx != nil {
						_ = h.sendRelayAnnouncement(ep, []*types.Transaction{tx})
					}
				}
			}
			for _, hash := range hashes {
				_ = h.scheduleBlob(hash)
			}
		}
	}
}

// SetCustodyColumns consumes the CL bitarray. Null updates never call this.
// Peer lifecycle workers dispatch expansion requests and updated announcements.
func (h *Handler) SetCustodyColumns(mask types.CellMask) {
	r := h.relay
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.custody == mask {
		return
	}
	r.custody = mask
	r.custodyGeneration++
}

// PrepareBlobBuilding keeps builders in eager mode permanently after first use.
// Incomplete staged entries remain absent from the ordinary executable pool.
func (h *Handler) PrepareBlobBuilding() { h.relay.mu.Lock(); h.relay.eager = true; h.relay.mu.Unlock() }

// GetBlobCells projects requested versioned hashes and indices without exposing
// missing data as zeros. Complete ordinary data and authenticated sparse data
// share this production consumer.
func (h *Handler) GetBlobCells(hashes []types.Hash, mask types.CellMask) []types.BlobCells {
	result := make([]types.BlobCells, len(hashes))
	indices := mask.Indices()

	var txs []*types.Transaction
	if source, ok := h.txPool.(interface{ BlobTransactions() []*types.Transaction }); ok {
		txs = source.BlobTransactions()
	} else {
		for _, list := range h.txPool.Pending() {
			txs = append(txs, list...)
		}
	}
	for _, hash := range h.relay.store.Hashes() {
		if tx := h.relay.store.Get(hash); tx != nil {
			txs = append(txs, tx)
		}
	}
	for _, tx := range txs {
		for b, hash := range tx.BlobHashes() {
			for i, wanted := range hashes {
				if wanted != hash {
					continue
				}
				side := tx.BlobSidecar()
				if side == nil {
					continue
				}
				// Full-pool and staged entries can overlap during promotion,
				// and distinct transactions may reference the same blob. Merge
				// available cells instead of erasing an earlier authenticated row.
				if result[i].Cells == nil {
					result[i].Cells = make([][]byte, len(indices))
					result[i].Proofs = make([][]byte, len(indices))
				}
				var all [][2048]byte
				var proofs [][]byte
				if types.HasFullBlobSidecar(tx) {
					var err error
					all, err = bls.DefaultKZGBackend().ComputeCells(side.Blobs[b])
					if err != nil {
						continue
					}
					if side.Version == 1 {
						proofs = side.Proofs[b*128 : (b+1)*128]
					} else {
						_, generated, err := bls.DefaultKZGBackend().ComputeCellsAndProofs(side.Blobs[b])
						if err != nil {
							continue
						}
						for _, p := range generated {
							proofs = append(proofs, append([]byte(nil), p[:]...))
						}
					}
					for j, index := range indices {
						result[i].Cells[j] = append([]byte(nil), all[index][:]...)
						result[i].Proofs[j] = append([]byte(nil), proofs[index]...)
					}
				} else {
					cells, available := h.relay.store.Cells(tx.Hash(), mask)
					served := available.Indices()
					for k, index := range served {
						for j, requested := range indices {
							if index == requested {
								result[i].Cells[j] = cells[b*len(served)+k]
								result[i].Proofs[j] = append([]byte(nil), side.Proofs[b*128+int(index)]...)
							}
						}
					}
				}
			}
		}
	}
	return result
}

func (h *Handler) handleGetSparsePooled(ep *EthPeer, msg p2p.Msg) error {
	var req p2p.GetPooledTransactionsPacket
	if err := gethrlp.DecodeBytes(msg.Payload, &req); err != nil {
		return err
	}
	if len(req.Hashes) > 256 {
		return errors.New("too many pooled transaction hashes")
	}
	reply := pooledTransactions72{RequestID: req.RequestID}
	size := 64
	for _, hash := range req.Hashes {
		tx := h.relayTransaction(hash)
		if tx == nil {
			continue
		}
		meta, err := sparseMetadata(tx)
		if err != nil {
			return err
		}
		raw, err := types.EncodePooledTransaction(meta)
		if err != nil {
			return err
		}
		if size+len(raw)+8 > CellResponseLimit {
			break
		}
		reply.Transactions = append(reply.Transactions, meta)
		size += len(raw) + 8
	}
	return ep.sendMessage(p2p.PooledTransactionsMsg, &reply)
}
