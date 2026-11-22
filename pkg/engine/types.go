// Package engine defines types for the Engine API (CL-EL communication).
package engine

import (
	"math/big"

	"github.com/eth2030/eth2030/core/types"
	engapi "github.com/eth2030/eth2030/engine/api"
	engblobs "github.com/eth2030/eth2030/engine/blobsbundle"
	engconvert "github.com/eth2030/eth2030/engine/convert"
	"github.com/eth2030/eth2030/engine/payload"
)

// Re-exported type aliases for backward compatibility.
// The canonical definitions live in engine/payload.
type (
	PayloadID            = payload.PayloadID
	Withdrawal           = payload.Withdrawal
	ExecutionPayloadV1   = payload.ExecutionPayloadV1
	ExecutionPayloadV2   = payload.ExecutionPayloadV2
	ExecutionPayloadV3   = payload.ExecutionPayloadV3
	ExecutionPayloadV4   = payload.ExecutionPayloadV4
	ExecutionPayloadV5   = payload.ExecutionPayloadV5
	PayloadAttributesV1  = payload.PayloadAttributesV1
	PayloadAttributesV2  = payload.PayloadAttributesV2
	PayloadAttributesV3  = payload.PayloadAttributesV3
	PayloadAttributesV4  = payload.PayloadAttributesV4
	GetPayloadV3Response = payload.GetPayloadV3Response
	GetPayloadV4Response = payload.GetPayloadV4Response
	GetPayloadV6Response = payload.GetPayloadV6Response
	GetPayloadResponse   = payload.GetPayloadResponse
	BlobsBundleV1        = payload.BlobsBundleV1
)

// Re-exported type aliases — status/forkchoice/payload types from engine/payload;
// handler types from engine/api.
type (
	// Status/forkchoice — canonical in engine/payload.
	PayloadStatusV1         = payload.PayloadStatusV1
	ForkchoiceStateV1       = payload.ForkchoiceStateV1
	ForkchoiceUpdatedResult = payload.ForkchoiceUpdatedResult
	// Blobs — canonical in engine/payload.
	BlobAndProofV1 = payload.BlobAndProofV1
	// Glamsterdam — canonical in engine/payload.
	BlobAndProofV2                   = payload.BlobAndProofV2
	BlobsBundleV2                    = payload.BlobsBundleV2
	GlamsterdamPayloadAttributes     = payload.GlamsterdamPayloadAttributes
	LegacyGlamsterdamPayloadResponse = payload.LegacyGlamsterdamPayloadResponse
	GetPayloadV5Response             = payload.GetPayloadV5Response
	// V7 — canonical in engine/payload.
	DALayerConfig        = payload.DALayerConfig
	ProofRequirements    = payload.ProofRequirements
	PayloadAttributesV7  = payload.PayloadAttributesV7
	ExecutionPayloadV7   = payload.ExecutionPayloadV7
	GetPayloadV7Response = payload.GetPayloadV7Response
	// Handler types from engine/api.
	ClientVersionV2   = engapi.ClientVersionV2
	EngineGlamsterdam = engapi.EngineGlamsterdam
	// From api/v4.go
	DepositRequest       = engapi.DepositRequest
	WithdrawalRequest    = engapi.WithdrawalRequest
	ConsolidationRequest = engapi.ConsolidationRequest
	ExecutionRequestsV4  = engapi.ExecutionRequestsV4
	GetPayloadV4Result   = engapi.GetPayloadV4Result
	EngV4                = engapi.EngV4
	// From api/uncoupled.go
	InclusionProof           = engapi.InclusionProof
	UncoupledPayloadEnvelope = engapi.UncoupledPayloadEnvelope
	UncoupledPayloadHandler  = engapi.UncoupledPayloadHandler
	// Note: EngineV7 is NOT aliased here; engine_v7.go defines engineV7Wrapper
	// which wraps engapi.EngineV7 and exposes backend for package-internal tests.
	// From api/epbs.go
)

// PayloadStatus values.
const (
	StatusValid            = "VALID"
	StatusInvalid          = "INVALID"
	StatusSyncing          = "SYNCING"
	StatusAccepted         = "ACCEPTED"
	StatusInvalidBlockHash = "INVALID_BLOCK_HASH"
	// StatusInclusionListUnsatisfied is returned when a valid IL tx is absent
	// from the block with sufficient remaining gas (EIP-7805 §engine-api).
	StatusInclusionListUnsatisfied = "INCLUSION_LIST_UNSATISFIED"
)

// Blobsbundle re-exports — canonical definitions live in engine/blobsbundle.
const (
	BlobSize             = engblobs.BlobSize
	KZGCommitmentSize    = engblobs.KZGCommitmentSize
	KZGProofSize         = engblobs.KZGProofSize
	MaxBlobsPerBundle    = engblobs.MaxBlobsPerBundle
	VersionedHashVersion = engblobs.VersionedHashVersion
)

// Blob bundle error re-exports.
var (
	ErrBlobBundleEmpty        = engblobs.ErrBlobBundleEmpty
	ErrBlobBundleMismatch     = engblobs.ErrBlobBundleMismatch
	ErrBlobBundleTooMany      = engblobs.ErrBlobBundleTooMany
	ErrBlobInvalidSize        = engblobs.ErrBlobInvalidSize
	ErrCommitmentInvalidSize  = engblobs.ErrCommitmentInvalidSize
	ErrProofInvalidSize       = engblobs.ErrProofInvalidSize
	ErrVersionedHashMismatch  = engblobs.ErrVersionedHashMismatch
	ErrBlobBundleSidecarIndex = engblobs.ErrBlobBundleSidecarIndex
)

type (
	KZGVerifier        = engblobs.KZGVerifier
	BlobSidecar        = engblobs.BlobSidecar
	BlobsBundleBuilder = engblobs.BlobsBundleBuilder
)

var (
	NewBlobsBundleBuilder   = engblobs.NewBlobsBundleBuilder
	ValidateBundle          = engblobs.ValidateBundle
	VersionedHash           = engblobs.VersionedHash
	DeriveVersionedHashes   = engblobs.DeriveVersionedHashes
	ValidateVersionedHashes = engblobs.ValidateVersionedHashes
	PrepareSidecars         = engblobs.PrepareSidecars
	GetSidecar              = engblobs.GetSidecar
)

// TransitionConfigurationV1 for Engine API transition configuration exchange.
type TransitionConfigurationV1 struct {
	TerminalTotalDifficulty *big.Int   `json:"terminalTotalDifficulty"`
	TerminalBlockHash       types.Hash `json:"terminalBlockHash"`
	TerminalBlockNumber     uint64     `json:"terminalBlockNumber"`
}

// Convert sub-package re-exports for backward compatibility.
// The canonical definitions live in engine/convert.
const (
	PayloadV1 = engconvert.PayloadV1
	PayloadV2 = engconvert.PayloadV2
	PayloadV3 = engconvert.PayloadV3
	PayloadV4 = engconvert.PayloadV4
	PayloadV5 = engconvert.PayloadV5
)

type (
	PayloadVersion     = engconvert.PayloadVersion
	ForkTimestamps     = engconvert.ForkTimestamps
	WithdrawalsSummary = engconvert.WithdrawalsSummary
)

var (
	PayloadToHeaderV1           = engconvert.PayloadToHeaderV1
	PayloadToHeaderV2           = engconvert.PayloadToHeaderV2
	PayloadToHeaderV3           = engconvert.PayloadToHeaderV3
	PayloadToHeaderV5           = engconvert.PayloadToHeaderV5
	HeaderToPayloadV2           = engconvert.HeaderToPayloadV2
	HeaderToPayloadV3           = engconvert.HeaderToPayloadV3
	ExtractVersionedHashes      = engconvert.ExtractVersionedHashes
	VersionedHashFromCommitment = engconvert.VersionedHashFromCommitment
	BlobSidecarFromBundle       = engconvert.BlobSidecarFromBundle
	ProcessWithdrawalsExt       = engconvert.ProcessWithdrawalsExt
	CoreWithdrawalsFromPayload  = engconvert.CoreWithdrawalsFromPayload
	DeterminePayloadVersion     = engconvert.DeterminePayloadVersion
	ConvertV1ToV2               = engconvert.ConvertV1ToV2
	ConvertV2ToV3               = engconvert.ConvertV2ToV3
	ConvertV3ToV4               = engconvert.ConvertV3ToV4
	ConvertV4ToV5               = engconvert.ConvertV4ToV5
	ValidatePayloadConsistency  = engconvert.ValidatePayloadConsistency
	SummarizeWithdrawals        = engconvert.SummarizeWithdrawals
)
