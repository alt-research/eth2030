package pq

import (
	"crypto/sha256"
	"errors"
	"slices"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/pqc"
	"github.com/eth2030/eth2030/proofs"
)

// STARK signature aggregation errors.
var (
	ErrSTARKAggNoSigs       = errors.New("stark_sig_aggregation: no signatures to aggregate")
	ErrSTARKAggInvalidProof = errors.New("stark_sig_aggregation: invalid aggregate proof")
	ErrSTARKAggVerifyFailed = errors.New("stark_sig_aggregation: verification failed")
	ErrSTARKAggNilResult    = errors.New("stark_sig_aggregation: nil aggregation result")
	ErrSTARKAggMismatch     = errors.New("stark_sig_aggregation: committee root mismatch")
)

// STARKSignatureAggregation carries the original signed attestations.
// The legacy proof field is reserved for a future statement-bound provider.
type STARKSignatureAggregation struct {
	// Signatures are the original PQ attestations being aggregated.
	Signatures []PQAttestation
	// AggregateProof is reserved. The individual-signature verifier rejects it.
	AggregateProof *proofs.STARKProofData
	// CommitteeRoot is the Merkle root of the participating validator public keys.
	CommitteeRoot types.Hash
	// Message is the beacon block root being attested to.
	Message types.Hash
	// NumValidators is the number of validators in this aggregation.
	NumValidators int
	// Slot is the slot this aggregation covers.
	Slot uint64
	// TargetEpoch is the target epoch for finality.
	TargetEpoch uint64
}

// STARKSignatureAggregator validates each signature in the batch.
type STARKSignatureAggregator struct{}

func NewSTARKSignatureAggregator() *STARKSignatureAggregator { return &STARKSignatureAggregator{} }

func (sa *STARKSignatureAggregator) Aggregate(attestations []PQAttestation) (*STARKSignatureAggregation, error) {
	if len(attestations) == 0 {
		return nil, ErrSTARKAggNoSigs
	}
	pubkeys, err := verifiedAttestationKeys(attestations)
	if err != nil {
		return nil, err
	}
	signed := slices.Clone(attestations)
	for i := range signed {
		signed[i].PQPublicKey = slices.Clone(signed[i].PQPublicKey)
		signed[i].PQSignature = slices.Clone(signed[i].PQSignature)
		signed[i].ClassicSignature = slices.Clone(signed[i].ClassicSignature)
	}
	return &STARKSignatureAggregation{
		Signatures: signed, CommitteeRoot: computeCommitteeRoot(pubkeys),
		Message: signed[0].BeaconBlockRoot, NumValidators: len(signed),
		Slot: signed[0].Slot, TargetEpoch: signed[0].TargetEpoch,
	}, nil
}

func (sa *STARKSignatureAggregator) Verify(agg *STARKSignatureAggregation) (bool, error) {
	if agg == nil {
		return false, ErrSTARKAggNilResult
	}
	if agg.AggregateProof != nil {
		return false, ErrSTARKAggInvalidProof
	}
	if len(agg.Signatures) == 0 || agg.NumValidators != len(agg.Signatures) {
		return false, ErrSTARKAggVerifyFailed
	}
	first := agg.Signatures[0]
	if first.Slot != agg.Slot || first.TargetEpoch != agg.TargetEpoch || first.BeaconBlockRoot != agg.Message {
		return false, ErrSTARKAggVerifyFailed
	}
	pubkeys, err := verifiedAttestationKeys(agg.Signatures)
	if err != nil {
		return false, err
	}
	if computeCommitteeRoot(pubkeys) != agg.CommitteeRoot {
		return false, ErrSTARKAggMismatch
	}
	return true, nil
}

// The caller must check committee membership against consensus state.
func verifiedAttestationKeys(attestations []PQAttestation) ([][]byte, error) {
	first := attestations[0]
	indices := make(map[uint64]struct{}, len(attestations))
	keys := make(map[string]struct{}, len(attestations))
	pubkeys := make([][]byte, 0, len(attestations))
	for _, att := range attestations {
		if att.Slot != first.Slot || att.CommitteeIndex != first.CommitteeIndex ||
			att.SourceEpoch != first.SourceEpoch || att.TargetEpoch != first.TargetEpoch ||
			att.BeaconBlockRoot != first.BeaconBlockRoot {
			return nil, ErrSTARKAggVerifyFailed
		}
		if _, exists := indices[att.ValidatorIndex]; exists {
			return nil, ErrSTARKAggVerifyFailed
		}
		key := string(att.PQPublicKey)
		if _, exists := keys[key]; exists {
			return nil, ErrSTARKAggVerifyFailed
		}
		message := attestationMessage(att.Slot, att.CommitteeIndex, att.BeaconBlockRoot, att.SourceEpoch, att.TargetEpoch)
		if !pqc.VerifyDilithium(att.PQPublicKey, message, att.PQSignature) {
			return nil, ErrSTARKAggVerifyFailed
		}
		indices[att.ValidatorIndex], keys[key] = struct{}{}, struct{}{}
		pubkeys = append(pubkeys, att.PQPublicKey)
	}
	return pubkeys, nil
}

// AggregateCount returns the number of signatures in an aggregation.
func (agg *STARKSignatureAggregation) AggregateCount() int {
	if agg == nil {
		return 0
	}
	return agg.NumValidators
}

// GasSavings reports zero because the current verifier checks each signature.
func (agg *STARKSignatureAggregation) GasSavings() uint64 { return 0 }

// computeCommitteeRoot computes a Merkle root over validator public keys.
func computeCommitteeRoot(pubkeys [][]byte) types.Hash {
	if len(pubkeys) == 0 {
		return types.Hash{}
	}

	// Hash each public key into a leaf.
	leaves := make([][32]byte, len(pubkeys))
	for i, pk := range pubkeys {
		h := sha256.New()
		h.Write(pk)
		copy(leaves[i][:], h.Sum(nil))
	}

	// Pad to next power of two.
	n := len(leaves)
	target := 1
	for target < n {
		target <<= 1
	}
	padded := make([][32]byte, target)
	copy(padded, leaves)

	// Build Merkle tree bottom-up.
	layer := padded
	for len(layer) > 1 {
		next := make([][32]byte, len(layer)/2)
		for i := range next {
			h := sha256.New()
			h.Write(layer[2*i][:])
			h.Write(layer[2*i+1][:])
			copy(next[i][:], h.Sum(nil))
		}
		layer = next
	}

	var root types.Hash
	copy(root[:], layer[0][:])
	return root
}

// BatchVerifyPQAttestations verifies every ML-DSA attestation and binds the
// ordered public keys to the returned committee commitment.
func BatchVerifyPQAttestations(attestations []PQAttestation) (*STARKSignatureAggregation, error) {
	return NewSTARKSignatureAggregator().Aggregate(attestations)
}

// MerklePQAggregation holds a lightweight Merkle-only aggregation of PQ
// attestations, used in lean available chain mode where STARK proofs are skipped.
type MerklePQAggregation struct {
	MerkleRoot    types.Hash
	NumValidators int
	Slot          uint64
}

// MerkleAggregatePQAttestations computes a SHA-256 Merkle tree over the
// attestation public key hashes and returns the root. No STARK proof is produced.
func MerkleAggregatePQAttestations(attestations []PQAttestation) (*MerklePQAggregation, error) {
	if len(attestations) == 0 {
		return nil, ErrSTARKAggNoSigs
	}
	pubkeys := make([][]byte, len(attestations))
	for i, att := range attestations {
		pubkeys[i] = att.PQPublicKey
	}
	root := computeCommitteeRoot(pubkeys)
	return &MerklePQAggregation{
		MerkleRoot:    root,
		NumValidators: len(attestations),
		Slot:          attestations[0].Slot,
	}, nil
}

// AggregateWithConfig verifies individual signatures in both modes.
func (sa *STARKSignatureAggregator) AggregateWithConfig(attestations []PQAttestation, _ *LeanConfig) (*STARKSignatureAggregation, error) {
	return sa.Aggregate(attestations)
}

// AggregationStats holds statistics about a STARK signature aggregation.
type AggregationStats struct {
	NumSignatures     int
	ProofSizeBytes    int
	CommitteeRoot     types.Hash
	Slot              uint64
	TargetEpoch       uint64
	EstimatedGasSaved uint64
}

// Stats returns statistics about this aggregation.
func (agg *STARKSignatureAggregation) Stats() AggregationStats {
	if agg == nil {
		return AggregationStats{}
	}
	proofSize := 0
	if agg.AggregateProof != nil {
		proofSize = agg.AggregateProof.ProofSize()
	}
	return AggregationStats{
		NumSignatures:     agg.NumValidators,
		ProofSizeBytes:    proofSize,
		CommitteeRoot:     agg.CommitteeRoot,
		Slot:              agg.Slot,
		TargetEpoch:       agg.TargetEpoch,
		EstimatedGasSaved: agg.GasSavings(),
	}
}
