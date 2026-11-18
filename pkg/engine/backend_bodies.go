package engine

import (
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	enginepayload "github.com/eth2030/eth2030/engine/payload"
)

// GetHeadHash returns the current canonical head block hash.
func (b *EngineBackend) GetHeadHash() types.Hash {
	hash, err := b.getActorHeadHash()
	if err != nil {
		return b.getHeadHash()
	}
	return hash
}

// GetSafeHash returns the current safe (justified) block hash.
func (b *EngineBackend) GetSafeHash() types.Hash {
	hash, err := b.getActorSafeHash()
	if err != nil {
		_, safeHash, _ := b.getForkchoiceState()
		return safeHash
	}
	return hash
}

// GetFinalizedHash returns the current finalized block hash.
func (b *EngineBackend) GetFinalizedHash() types.Hash {
	hash, err := b.getActorFinalHash()
	if err != nil {
		_, _, finalHash := b.getForkchoiceState()
		return finalHash
	}
	return hash
}

// GetPayloadBodiesByHash returns payload bodies for the given block hashes.
// Entries for unknown or out-of-retention-window blocks are nil.
// Implements backendapi.PayloadBodiesBackend.
// P1: Uses fine-grained locks for better concurrency.
func (b *EngineBackend) GetPayloadBodiesByHash(hashes []types.Hash) ([]*enginepayload.ExecutionPayloadBodyV2, error) {
	return b.loadPayloadBodiesByHash(hashes), nil
}

// GetPayloadBodiesByRange returns payload bodies for count blocks starting at start.
// Entries outside the retention window are nil.
// Implements backendapi.PayloadBodiesBackend.
// P1: Uses fine-grained locks for better concurrency.
func (b *EngineBackend) GetPayloadBodiesByRange(start, count uint64) ([]*enginepayload.ExecutionPayloadBodyV2, error) {
	return b.loadPayloadBodiesByRange(start, count), nil
}

// isBALRetainedForBlocks uses actual beacon slots when both are known.
// A block-number fallback conservatively keeps more data across missed slots.
func isBALRetainedForBlocks(head, block *types.Block) bool {
	if head == nil {
		return true
	}
	h, b := head.Header(), block.Header()
	if h.SlotNumber != nil && b.SlotNumber != nil {
		return rawdb.IsBALRetained(*h.SlotNumber, *b.SlotNumber)
	}
	return rawdb.IsBALRetained(head.NumberU64(), block.NumberU64())
}
