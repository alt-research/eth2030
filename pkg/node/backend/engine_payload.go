package backend

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"log/slog"
	"math/big"
	"time"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/engine/payload"
	enginepayload "github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/rlp"
)

// GetPayloadByID retrieves a previously requested payload by its ID.
func (b *EngineBackend) GetPayloadByID(id payload.PayloadID) (*payload.GetPayloadResponse, error) {
	p, err := b.awaitPayload(id)
	if err != nil {
		return nil, err
	}
	return b.payloadResponse(id, p)
}

func (b *EngineBackend) awaitPayload(id payload.PayloadID) (*pendingPayload, error) {
	slog.Debug("engine_getPayload", "payloadID", id)

	b.mu.Lock()
	p, ok := b.payloads[id]
	b.mu.Unlock()

	if !ok {
		slog.Warn("engine_getPayload: payload not found", "payloadID", id)
		return nil, fmt.Errorf("payload %v not found", id)
	}

	select {
	case <-p.done:
	case <-time.After(8 * time.Second):
		slog.Warn("engine_getPayload: build timed out", "payloadID", id)
		return nil, fmt.Errorf("payload %v build timed out", id)
	}

	if p.err != nil {
		return nil, p.err
	}

	return p, nil
}

func (b *EngineBackend) payloadResponse(id payload.PayloadID, p *pendingPayload) (*payload.GetPayloadResponse, error) {
	blk := p.block
	header := blk.Header()

	var blobGasUsed, excessBlobGas uint64
	if header.BlobGasUsed != nil {
		blobGasUsed = *header.BlobGasUsed
	}
	if header.ExcessBlobGas != nil {
		excessBlobGas = *header.ExcessBlobGas
	}
	execPayload := &payload.ExecutionPayloadV4{
		ExecutionPayloadV3: payload.ExecutionPayloadV3{
			DependencyWitnesses: blk.DependencyWitnesses8504(),
			ExecutionPayloadV2: payload.ExecutionPayloadV2{
				ExecutionPayloadV1: payload.ExecutionPayloadV1{
					ParentHash:    header.ParentHash,
					FeeRecipient:  header.Coinbase,
					StateRoot:     header.Root,
					ReceiptsRoot:  header.ReceiptHash,
					LogsBloom:     header.Bloom,
					PrevRandao:    header.MixDigest,
					BlockNumber:   blk.NumberU64(),
					GasLimit:      header.GasLimit,
					GasUsed:       header.GasUsed,
					Timestamp:     header.Time,
					ExtraData:     header.Extra,
					BaseFeePerGas: header.BaseFee,
					BlockHash:     blk.Hash(),
					Transactions:  encodeTxsRLP(blk.Transactions()),
				},
			},
			BlobGasUsed:   blobGasUsed,
			ExcessBlobGas: excessBlobGas,
		},
	}

	if p.result != nil {
		execPayload.ExecutionRequests = p.result.Requests.Encode()
	}
	if ws := blk.Withdrawals(); ws != nil {
		execPayload.Withdrawals = make([]*payload.Withdrawal, 0, len(ws))
		for _, w := range ws {
			execPayload.Withdrawals = append(execPayload.Withdrawals, &payload.Withdrawal{
				Index:          w.Index,
				ValidatorIndex: w.ValidatorIndex,
				Address:        w.Address,
				Amount:         w.Amount,
			})
		}
	}

	blockValue := new(big.Int)
	for i, receipt := range p.receipts {
		if i < len(blk.Transactions()) {
			if frame, ok := blk.Transactions()[i].Inner().(*types.FrameTx8141); ok {
				blobFee := gas.BlobBaseFeeForHeader(b.node.Blockchain().Config(), p.block.Header())
				tip, err := types.FrameProposerTip(frame, receipt.GasUsed, header.BaseFee, blobFee)
				if err == nil {
					blockValue.Add(blockValue, tip)
				}
				continue
			}
		}
		if receipt.EffectiveGasPrice != nil && header.BaseFee != nil {
			tip := new(big.Int).Sub(receipt.EffectiveGasPrice, header.BaseFee)
			if tip.Sign() > 0 {
				tipTotal := new(big.Int).Mul(tip, new(big.Int).SetUint64(receipt.GasUsed))
				blockValue.Add(blockValue, tipTotal)
			}
		}
	}

	slog.Debug("engine_getPayload: returning payload",
		"payloadID", id,
		"blockNumber", blk.NumberU64(),
		"blockHash", blk.Hash(),
		"txCount", len(blk.Transactions()),
		"blockValue", blockValue,
	)

	blobsBundle, err := payload.BuildBlobsBundleV1(blk.Transactions())
	if err != nil {
		return nil, err
	}

	// Cache blobs for engine_getBlobsV2 (PeerDAS).
	// When the CL calls engine_getBlobsV2, it needs to find blobs in the cache
	// because engine_newPayload doesn't receive blob data.
	if len(blobsBundle.Blobs) > 0 {
		b.blobCacheMu.Lock()
		// Clear old cache entries to avoid stale data
		b.blobCache = make(map[types.Hash]*payload.BlobAndProofV2)
		for _, tx := range blk.Transactions() {
			if len(tx.BlobHashes()) == 0 {
				continue
			}
			sc := tx.BlobSidecar()
			if sc == nil {
				slog.Warn("engine_getPayload: BlobSidecar is nil for blob tx")
				continue
			}
			// The cache is consumed by PeerDAS V2/V3 retrieval only. Legacy
			// sidecars must not be silently upgraded by a later query.
			_, frame := tx.Inner().(*types.FrameTx8141)
			if sc.Version != 1 && !frame {
				continue
			}
			blobHashes := tx.BlobHashes()
			slog.Debug("engine_getPayload: caching blob hashes",
				"blobHashes", len(blobHashes),
				"scBlobs", len(sc.Blobs),
			)
			for i, hash := range blobHashes {
				if i < len(sc.Blobs) && len(sc.Blobs[i]) > 0 {
					if len(sc.Proofs) < (i+1)*128 {
						continue
					}
					b.blobCache[hash] = copyBlobCellProofs(sc.Blobs[i], sc.Proofs[i*128:(i+1)*128])
					slog.Debug("engine_getPayload: cached blob", "hash", hash.Hex(), "index", i)
				}
			}
		}
		b.blobCacheMu.Unlock()
		slog.Debug("engine_getPayload: cached blobs for getBlobsV2",
			"blobCount", len(b.blobCache),
			"blockNumber", blk.NumberU64(),
		)
	}

	return &payload.GetPayloadResponse{
		ExecutionPayload: execPayload,
		BlockValue:       blockValue,
		BlobsBundle:      blobsBundle,
		Override:         false,
	}, nil
}

// GetPayloadV4ByID retrieves a previously built payload for getPayloadV4 (Prague).
func (b *EngineBackend) GetPayloadV4ByID(id payload.PayloadID) (*payload.GetPayloadV4Response, error) {
	resp, err := b.GetPayloadByID(id)
	if err != nil {
		return nil, err
	}
	return &payload.GetPayloadV4Response{
		ExecutionPayload:  &resp.ExecutionPayload.ExecutionPayloadV3,
		BlockValue:        resp.BlockValue,
		BlobsBundle:       resp.BlobsBundle,
		ExecutionRequests: resp.ExecutionPayload.ExecutionRequests,
	}, nil
}

// GetPayloadV6ByID retrieves a previously built payload for getPayloadV6 (Amsterdam).
func (b *EngineBackend) GetPayloadV6ByID(id payload.PayloadID) (*payload.GetPayloadV6Response, error) {
	p, err := b.awaitPayload(id)
	if err != nil {
		return nil, err
	}
	resp, err := b.payloadResponse(id, p)
	if err != nil {
		return nil, err
	}
	blobsBundleV2, err := payload.BuildBlobsBundleV2(p.block.Transactions())
	if err != nil {
		return nil, err
	}

	var blockBAL *bal.BlockAccessList
	if p.result != nil {
		blockBAL = p.result.BlockAccessList
	}
	v5 := enginepayload.BlockToPayloadV5(p.block, p.block.Header().MixDigest, resp.ExecutionPayload.Withdrawals, blockBAL, p.slot)
	v5.ExecutionRequests = resp.ExecutionPayload.ExecutionRequests
	return &payload.GetPayloadV6Response{
		ExecutionPayload:  v5,
		BlockValue:        resp.BlockValue,
		BlobsBundle:       blobsBundleV2,
		ExecutionRequests: resp.ExecutionPayload.ExecutionRequests,
	}, nil
}

// GetPayloadBodiesByHash returns payload bodies for the given block hashes.
func (b *EngineBackend) GetPayloadBodiesByHash(hashes []types.Hash) ([]*payload.ExecutionPayloadBodyV2, error) {
	results := make([]*payload.ExecutionPayloadBodyV2, len(hashes))
	for i, h := range hashes {
		blk := b.node.Blockchain().GetBlock(h)
		if blk == nil {
			results[i] = nil
			continue
		}
		results[i] = enginepayload.BlockToPayloadBodyV2(blk)
		encoded, err := b.node.Blockchain().BlockAccessList(h)
		if err != nil {
			return nil, err
		}
		results[i].BlockAccessList = encoded
	}
	return results, nil
}

// GetPayloadBodiesByRange returns payload bodies for a range of block numbers.
func (b *EngineBackend) GetPayloadBodiesByRange(start, count uint64) ([]*payload.ExecutionPayloadBodyV2, error) {
	head := b.node.Blockchain().CurrentBlock().NumberU64()
	if start > head {
		return []*payload.ExecutionPayloadBodyV2{}, nil
	}
	if remaining := head - start; count > remaining && count-remaining > 1 {
		count = remaining + 1
	}
	results := make([]*payload.ExecutionPayloadBodyV2, count)
	for i := uint64(0); i < count; i++ {
		num := start + i
		blk := b.node.Blockchain().GetBlockByNumber(num)
		if blk == nil {
			results[i] = nil
			continue
		}
		results[i] = enginepayload.BlockToPayloadBodyV2(blk)
		encoded, err := b.node.Blockchain().BlockAccessList(blk.Hash())
		if err != nil {
			return nil, err
		}
		results[i].BlockAccessList = encoded
	}
	return results, nil
}

// GetBlobsByVersionedHashes returns blobs by versioned hashes.
func (b *EngineBackend) GetBlobsByVersionedHashes(hashes []types.Hash) []*payload.BlobAndProofV1 {
	if b.node.TxPool() == nil {
		return make([]*payload.BlobAndProofV1, len(hashes))
	}
	raw := b.node.TxPool().GetBlobsByVersionedHashes(hashes)
	result := make([]*payload.BlobAndProofV1, len(raw))
	for i, r := range raw {
		if r != nil {
			result[i] = &payload.BlobAndProofV1{
				Blob:       r.Blob,
				Commitment: r.Commitment,
				Proof:      r.Proof,
			}
		}
	}
	return result
}

// GetBlobsV2ByVersionedHashes returns blobs with cell proofs.
func (b *EngineBackend) GetBlobsV2ByVersionedHashes(hashes []types.Hash) []*payload.BlobAndProofV2 {
	if !b.beginBlobPoolRead() {
		return nil
	}
	defer b.workers.Done()
	return b.computeBlobsV2(hashes)
}

// GetBlobsV3ByVersionedHashes returns blobs with cell proofs (sparse).
func (b *EngineBackend) GetBlobsV3ByVersionedHashes(hashes []types.Hash) []*payload.BlobAndProofV2 {
	if !b.beginBlobPoolRead() {
		return nil
	}
	defer b.workers.Done()
	return b.computeBlobsV2(hashes)
}

// A successful read joins shutdown ownership; syncing never exposes cached data.
func (b *EngineBackend) beginBlobPoolRead() bool {
	if b.beginWork() != nil {
		return false
	}
	if source, ok := b.node.(interface {
		SyncProgress() (bool, uint64, uint64, uint64)
	}); ok {
		if syncing, _, _, _ := source.SyncProgress(); syncing {
			b.workers.Done()
			return false
		}
	}
	return true
}

func (b *EngineBackend) computeBlobsV2(hashes []types.Hash) []*payload.BlobAndProofV2 {
	result := make([]*payload.BlobAndProofV2, len(hashes))
	var pending []types.Hash
	var positions []int
	b.blobCacheMu.RLock()
	for i, hash := range hashes {
		if cached := b.blobCache[hash]; cached != nil {
			result[i] = copyBlobCellProofs(cached.Blob, cached.Proofs)
		} else {
			pending = append(pending, hash)
			positions = append(positions, i)
		}
	}
	b.blobCacheMu.RUnlock()
	if len(pending) != 0 && b.node.TxPool() != nil {
		for i, item := range b.node.TxPool().GetBlobsWithCellProofs(pending) {
			if item != nil {
				result[positions[i]] = copyBlobCellProofs(item.Blob, item.CellProofs)
			}
		}
	}
	return result
}

// Responses own their bytes: a caller cannot mutate pool or published-cache data.
func copyBlobCellProofs(blob []byte, proofs [][]byte) *payload.BlobAndProofV2 {
	out := &payload.BlobAndProofV2{Blob: bytes.Clone(blob), Proofs: make([][]byte, len(proofs))}
	for i, proof := range proofs {
		out.Proofs[i] = bytes.Clone(proof)
	}
	return out
}

func generatePayloadID(parentHash types.Hash, attrs *block.BuildBlockAttributes, slot uint64) (payload.PayloadID, error) {
	data, err := rlp.EncodeToBytes([]any{
		parentHash, attrs.Timestamp, attrs.FeeRecipient, attrs.Random,
		attrs.Withdrawals, attrs.BeaconRoot, attrs.GasLimit, attrs.InclusionListTxs, slot,
	})
	if err != nil {
		return payload.PayloadID{}, err
	}
	hash := crypto.Keccak256Hash(data)
	var id payload.PayloadID
	copy(id[:], hash[:len(id)])
	return id, nil
}

func encodeTxsRLP(txs []*types.Transaction) [][]byte {
	encoded := make([][]byte, 0, len(txs))
	for _, tx := range txs {
		raw, err := tx.EncodeRLP()
		if err != nil {
			continue
		}
		encoded = append(encoded, raw)
	}
	return encoded
}

// GetPayloadV7ByID retrieves a previously built payload for getPayloadV7.
// When EIP-7807 SSZ blocks are active at the payload timestamp, the response
// additionally carries the SSZ payload encoding and its hash tree root —
// the SSZ execution block hash the CL commits to.
func (b *EngineBackend) GetPayloadV7ByID(id payload.PayloadID) (*payload.GetPayloadV6Response, error) {
	resp, err := b.GetPayloadV6ByID(id)
	if err != nil {
		return nil, err
	}
	p, err := b.awaitPayload(id)
	if err != nil {
		return resp, nil
	}
	if p.block == nil || !b.node.Blockchain().Config().IsSSZBlock(p.block.Time()) {
		return resp, nil
	}
	requests := types.Requests(nil)
	balRLP := []byte(nil)
	if p.result != nil {
		requests = p.result.Requests
		if p.result.BlockAccessList != nil {
			balRLP, _ = p.result.BlockAccessList.EncodeRLP()
		}
	}
	receipts := p.receipts
	excessRegular := uint64(0)
	if p.block.Header().CalldataExcessGas != nil {
		excessRegular = *p.block.Header().CalldataExcessGas
	}
	ctx := types.ExecutionSSZContext{
		BlobGasLimit:     gas.FusakaMaxBlobGasPerBlock,
		BlobBaseFee:      gas.BlobBaseFeeForHeader(b.node.Blockchain().Config(), p.block.Header()),
		ExcessRegularGas: excessRegular,
		Requests:         requests,
		BlockAccessList:  balRLP,
		SlotNumber:       p.slot,
	}
	payloadSSZ, err := types.NewExecutionPayloadSSZ(p.block, receipts, ctx)
	if err != nil {
		slog.Debug("GetPayloadV7ByID: ssz build failed", "err", err, "receipts", len(receipts), "balRLP", len(balRLP))
		return resp, nil
	}
	enc, err := payloadSSZ.MarshalSSZ()
	if err != nil {
		return resp, nil
	}
	root, err := payloadSSZ.HashTreeRoot()
	if err != nil {
		return resp, nil
	}
	resp.SSZPayload = json.RawMessage(`"` + "0x" + hex.EncodeToString(enc) + `"`)
	resp.SSZBlockHash = &root
	return resp, nil
}

func (b *EngineBackend) SetCustodyColumns(mask types.CellMask) {
	if h, ok := b.node.EthHandler().(interface{ SetCustodyColumns(types.CellMask) }); ok {
		h.SetCustodyColumns(mask)
	}
}
func (b *EngineBackend) GetBlobsV4ByVersionedHashes(hashes []types.Hash, mask types.CellMask) []*payload.BlobCellsAndProofsV1 {
	if !b.beginBlobPoolRead() {
		return nil
	}
	defer b.workers.Done()
	h, ok := b.node.EthHandler().(interface {
		GetBlobCells([]types.Hash, types.CellMask) []types.BlobCells
	})
	if !ok {
		return nil
	}
	cells := h.GetBlobCells(hashes, mask)
	if cells == nil {
		return nil
	}
	result := make([]*payload.BlobCellsAndProofsV1, len(hashes))
	for i := range result {
		if i >= len(cells) || cells[i].Cells == nil {
			continue
		}
		item := &payload.BlobCellsAndProofsV1{BlobCells: make([]*hexutil.Bytes, mask.OneCount()), Proofs: make([]*hexutil.Bytes, mask.OneCount())}
		if i < len(cells) {
			for j := range item.BlobCells {
				if j < len(cells[i].Cells) && cells[i].Cells[j] != nil {
					value := hexutil.Bytes(cells[i].Cells[j])
					item.BlobCells[j] = &value
				}
				if item.BlobCells[j] != nil && j < len(cells[i].Proofs) && cells[i].Proofs[j] != nil {
					value := hexutil.Bytes(cells[i].Proofs[j])
					item.Proofs[j] = &value
				}
			}
		}
		result[i] = item
	}
	return result
}
