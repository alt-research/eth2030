// Package payload: Glamsterdam-era payload types.
package payload

import (
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// GlamsterdamPayloadAttributes contains attributes for building a post-Glamsterdam
// payload. Extends V3 with targetBlobCount and slot number.
type GlamsterdamPayloadAttributes struct {
	Timestamp             uint64        `json:"timestamp"`
	PrevRandao            types.Hash    `json:"prevRandao"`
	SuggestedFeeRecipient types.Address `json:"suggestedFeeRecipient"`
	Withdrawals           []*Withdrawal `json:"withdrawals"`
	ParentBeaconBlockRoot types.Hash    `json:"parentBeaconBlockRoot"`
	TargetBlobCount       uint64        `json:"targetBlobCount"`
	SlotNumber            uint64        `json:"slotNumber"`
}

// BlobAndProofV2 represents a blob with its cell proofs (Osaka spec).
type BlobAndProofV2 struct {
	Blob   []byte
	Proofs [][]byte
}

// BlobsBundleV2 extends BlobsBundleV1 with cell proofs per EIP-7594.
type BlobsBundleV2 struct {
	Commitments [][]byte
	Proofs      [][]byte
	Blobs       [][]byte
}

// LegacyGlamsterdamPayloadResponse belongs to the historical standalone adapter.
// The production Engine V5 response is the canonical Osaka GetPayloadV5Response.
type LegacyGlamsterdamPayloadResponse struct {
	ExecutionPayload  *ExecutionPayloadV5 `json:"executionPayload"`
	BlockValue        *big.Int            `json:"blockValue"`
	BlobsBundle       *BlobsBundleV2      `json:"blobsBundle"`
	Override          bool                `json:"shouldOverrideBuilder"`
	ExecutionRequests [][]byte            `json:"executionRequests"`
}
