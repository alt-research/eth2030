// The PQ blob validator checks the experimental Merkle commitment and its signature.
package pqblob

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/crypto/pqc"
)

// PQ blob validator algorithm name constants.
const (
	PQAlgDilithium = "dilithium"
	PQAlgFalcon    = "falcon"
	PQAlgSPHINCS   = "sphincs"
)

// Gas cost constants for PQ blob validation.
const (
	pqValidateBaseGas    = 3000
	pqValidatePerByteGas = 4
	pqDilithiumVerifyGas = 45000
	pqFalconVerifyGas    = 28000
	pqSPHINCSVerifyGas   = 120000
	pqMerkleProofGas     = 5000
	pqBatchOverheadGas   = 1500
)

// PQ blob validator errors.
var (
	ErrPQValidatorUnknownAlg   = errors.New("pq_validator: unknown algorithm")
	ErrPQValidatorNilBlob      = errors.New("pq_validator: nil blob data")
	ErrPQValidatorEmptyBlob    = errors.New("pq_validator: empty blob data")
	ErrPQValidatorNilCommit    = errors.New("pq_validator: nil commitment")
	ErrPQValidatorNilProof     = errors.New("pq_validator: nil proof")
	ErrPQValidatorMismatch     = errors.New("pq_validator: commitment does not match blob")
	ErrPQValidatorProofInvalid = errors.New("pq_validator: proof verification failed")
	ErrPQValidatorBatchEmpty   = errors.New("pq_validator: empty batch")
	ErrPQValidatorBatchLen     = errors.New("pq_validator: batch length mismatch")
	ErrPQValidatorBlobTooLarge = errors.New("pq_validator: blob exceeds max size")
)

// PQBlobProofV2 extends the base PQBlobProof with PQ-specific metadata for
// the DAS validation pipeline. It carries the signer's public key and a
// PQ signature over the Merkle root, making the proof verifiable without
// trusting the commitment source.
type PQBlobProofV2 struct {
	// Algorithm identifies the PQ signature scheme ("dilithium", "falcon", "sphincs").
	Algorithm string

	// Signature is the PQ signature over the Merkle root.
	Signature []byte

	// PublicKey is the signer's PQ public key.
	PublicKey []byte

	// MerkleRoot is the root of the lattice Merkle tree for the blob.
	MerkleRoot [PQCommitmentSize]byte

	// MerkleProof is the serialized proof path from leaf to root.
	MerkleProof []byte

	// BlobIndex is the index of the blob within the slot.
	BlobIndex uint64

	// SlotNumber is the beacon slot this blob belongs to.
	SlotNumber uint64
}

// PQBlobValidator validates PQ blob commitments within the DAS sampling
// pipeline. It combines Merkle commitment verification from pq_blobs.go
// with PQ signature verification to authenticate blob data.
type PQBlobValidator struct {
	mu        sync.RWMutex
	algorithm string
	signer    pqc.PQSigner
	algID     uint8
}

// NewPQBlobValidator creates a validator for the specified PQ algorithm.
// Supported: "dilithium", "falcon", "sphincs".
func NewPQBlobValidator(algorithm string) *PQBlobValidator {
	var signer pqc.PQSigner
	var algID uint8

	switch algorithm {
	case PQAlgDilithium:
		signer = &pqc.DilithiumSigner{}
		algID = PQBlobAlgMLDSA
	case PQAlgFalcon:
		signer = &pqc.FalconSigner{}
		algID = PQBlobAlgFalcon
	case PQAlgSPHINCS:
		signer = pqc.NewSPHINCSSigner()
		algID = PQBlobAlgSPHINCS
	default:
		return nil
	}

	return &PQBlobValidator{
		algorithm: algorithm,
		signer:    signer,
		algID:     algID,
	}
}

// Algorithm returns the configured PQ algorithm name.
func (v *PQBlobValidator) Algorithm() string {
	return v.algorithm
}

// ValidateBlobCommitment verifies that a PQ blob commitment matches the
// given blob data and requires the complete versioned-format digest.
func (v *PQBlobValidator) ValidateBlobCommitment(blob []byte, commitment []byte) error {
	if blob == nil {
		return ErrPQValidatorNilBlob
	}
	if len(blob) == 0 {
		return ErrPQValidatorEmptyBlob
	}
	if len(blob) > MaxBlobSize {
		return ErrPQValidatorBlobTooLarge
	}
	if commitment == nil || len(commitment) == 0 {
		return ErrPQValidatorNilCommit
	}

	recomputed, err := CommitBlob(blob)
	if err != nil {
		return err
	}
	if !bytes.Equal(commitment, recomputed.Digest[:]) {
		return ErrPQValidatorMismatch
	}
	return nil
}

// ValidateBlobProof verifies a PQBlobProofV2 against the expected commitment.
// It checks:
//  1. The proof's Merkle root matches the commitment
//  2. The PQ signature over the Merkle root is valid
//  3. The Merkle proof path is consistent
func (v *PQBlobValidator) ValidateBlobProof(blob []byte, proof *PQBlobProofV2, commitment []byte) error {
	if blob == nil {
		return ErrPQValidatorNilBlob
	}
	if len(blob) == 0 {
		return ErrPQValidatorEmptyBlob
	}
	if proof == nil {
		return ErrPQValidatorNilProof
	}
	if commitment == nil || len(commitment) == 0 {
		return ErrPQValidatorNilCommit
	}

	// Step 1: Verify the blob matches the commitment.
	if err := v.ValidateBlobCommitment(blob, commitment); err != nil {
		return fmt.Errorf("pq_validator: blob-commitment check failed: %w", err)
	}

	if proof.Algorithm != v.algorithm || !bytes.Equal(proof.MerkleRoot[:], commitment) {
		return ErrPQValidatorProofInvalid
	}
	chunkProof, err := GenerateBlobProof(blob, 0)
	if err != nil || !bytes.Equal(proof.MerkleProof, serializeChunkProof(chunkProof)) {
		return ErrPQValidatorProofInvalid
	}

	// Step 3: Verify the PQ signature over the Merkle root.
	if len(proof.Signature) == 0 || len(proof.PublicKey) == 0 {
		return ErrPQValidatorProofInvalid
	}

	v.mu.RLock()
	signer := v.signer
	v.mu.RUnlock()

	sigMsg := pqValidatorSigningMessage(proof.MerkleRoot[:], proof.BlobIndex, proof.SlotNumber)
	if !signer.Verify(proof.PublicKey, sigMsg, proof.Signature) {
		return ErrPQValidatorProofInvalid
	}

	return nil
}

// GenerateCommitmentProof creates a PQBlobProofV2 for the given blob.
// It commits the blob, signs the commitment with the validator's PQ key,
// and packages the result into a verifiable proof.
func (v *PQBlobValidator) GenerateCommitmentProof(blob []byte) (*PQBlobProofV2, error) {
	if blob == nil {
		return nil, ErrPQValidatorNilBlob
	}
	if len(blob) == 0 {
		return nil, ErrPQValidatorEmptyBlob
	}
	if len(blob) > MaxBlobSize {
		return nil, ErrPQValidatorBlobTooLarge
	}

	// Compute the ordered Merkle commitment.
	commitment, err := CommitBlob(blob)
	if err != nil {
		return nil, fmt.Errorf("pq_validator: commit failed: %w", err)
	}

	// Generate a key pair for signing.
	v.mu.Lock()
	kp, err := v.signer.GenerateKey()
	v.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("pq_validator: keygen failed: %w", err)
	}

	proof := &PQBlobProofV2{
		Algorithm:  v.algorithm,
		BlobIndex:  0,
		SlotNumber: 0,
	}
	copy(proof.MerkleRoot[:], commitment.Digest[:])

	// Sign the Merkle root with PQ scheme.
	sigMsg := pqValidatorSigningMessage(proof.MerkleRoot[:], proof.BlobIndex, proof.SlotNumber)
	sig, err := v.signer.Sign(kp.SecretKey, sigMsg)
	if err != nil {
		return nil, fmt.Errorf("pq_validator: sign failed: %w", err)
	}

	proof.Signature = sig
	proof.PublicKey = make([]byte, len(kp.PublicKey))
	copy(proof.PublicKey, kp.PublicKey)

	// Generate the Merkle proof path for chunk 0.
	chunkProof, err := GenerateBlobProof(blob, 0)
	if err != nil {
		return nil, fmt.Errorf("pq_validator: merkle proof failed: %w", err)
	}
	proof.MerkleProof = serializeChunkProof(chunkProof)

	return proof, nil
}

// BatchValidateCommitments validates multiple blobs against their commitments.
// Returns a boolean slice indicating which validations passed, and an error
// if the inputs are malformed.
func (v *PQBlobValidator) BatchValidateCommitments(blobs [][]byte, commitments [][]byte) ([]bool, error) {
	if len(blobs) == 0 {
		return nil, ErrPQValidatorBatchEmpty
	}
	if len(blobs) != len(commitments) {
		return nil, ErrPQValidatorBatchLen
	}

	n := len(blobs)
	results := make([]bool, n)

	if n <= 4 {
		for i := 0; i < n; i++ {
			err := v.ValidateBlobCommitment(blobs[i], commitments[i])
			results[i] = err == nil
		}
		return results, nil
	}

	// Parallel validation for larger batches.
	var wg sync.WaitGroup
	workers := 4
	if n < workers {
		workers = n
	}
	chunkSz := (n + workers - 1) / workers

	for w := 0; w < workers; w++ {
		start := w * chunkSz
		end := start + chunkSz
		if end > n {
			end = n
		}
		if start >= n {
			break
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for i := s; i < e; i++ {
				err := v.ValidateBlobCommitment(blobs[i], commitments[i])
				results[i] = err == nil
			}
		}(start, end)
	}

	wg.Wait()
	return results, nil
}

// EstimateValidationGas returns the estimated gas cost for validating a
// PQ blob commitment with the given algorithm and blob size.
func EstimateValidationGas(algorithm string, blobSize int) uint64 {
	base := uint64(pqValidateBaseGas)
	perByte := uint64(blobSize) * uint64(pqValidatePerByteGas)

	var sigGas uint64
	switch algorithm {
	case PQAlgDilithium:
		sigGas = pqDilithiumVerifyGas
	case PQAlgFalcon:
		sigGas = pqFalconVerifyGas
	case PQAlgSPHINCS:
		sigGas = pqSPHINCSVerifyGas
	default:
		sigGas = pqDilithiumVerifyGas
	}

	return base + perByte + sigGas + pqMerkleProofGas
}

// SupportedPQAlgorithms returns the list of supported PQ algorithm names.
func SupportedPQAlgorithms() []string {
	return []string{PQAlgDilithium, PQAlgFalcon, PQAlgSPHINCS}
}

// pqValidatorSigningMessage constructs the message signed in a PQBlobProofV2.
// Format: domain_tag || merkle_root || blob_index(8) || slot_number(8).
func pqValidatorSigningMessage(merkleRoot []byte, blobIndex, slotNumber uint64) []byte {
	domain := []byte("pq-blob-proof-v2")
	msg := make([]byte, len(domain)+len(merkleRoot)+16)
	offset := copy(msg, domain)
	offset += copy(msg[offset:], merkleRoot)
	msg[offset] = byte(blobIndex >> 56)
	msg[offset+1] = byte(blobIndex >> 48)
	msg[offset+2] = byte(blobIndex >> 40)
	msg[offset+3] = byte(blobIndex >> 32)
	msg[offset+4] = byte(blobIndex >> 24)
	msg[offset+5] = byte(blobIndex >> 16)
	msg[offset+6] = byte(blobIndex >> 8)
	msg[offset+7] = byte(blobIndex)
	msg[offset+8] = byte(slotNumber >> 56)
	msg[offset+9] = byte(slotNumber >> 48)
	msg[offset+10] = byte(slotNumber >> 40)
	msg[offset+11] = byte(slotNumber >> 32)
	msg[offset+12] = byte(slotNumber >> 24)
	msg[offset+13] = byte(slotNumber >> 16)
	msg[offset+14] = byte(slotNumber >> 8)
	msg[offset+15] = byte(slotNumber)
	return msg
}

// serializeChunkProof serializes a PQBlobProof into bytes for inclusion
// in a PQBlobProofV2's MerkleProof field.
func serializeChunkProof(proof *PQBlobProof) []byte {
	if proof == nil {
		return nil
	}
	// version || index || chunk hash || commitment digest || siblings.
	buf := make([]byte, 1+4+32+PQCommitmentSize, 1+4+32+PQCommitmentSize+len(proof.Siblings)*32)
	buf[0] = 1
	binary.BigEndian.PutUint32(buf[1:5], proof.ChunkIndex)
	copy(buf[5:37], proof.ChunkHash[:])
	copy(buf[37:], proof.CommitmentDigest[:])
	for _, sibling := range proof.Siblings {
		buf = append(buf, sibling[:]...)
	}
	return buf
}

// commitmentDigestBytes extracts the commitment digest bytes from a
// PQBlobCommitment for use in validation comparisons.
func commitmentDigestBytes(c *PQBlobCommitment) []byte {
	if c == nil {
		return nil
	}
	digest := make([]byte, PQCommitmentSize)
	copy(digest, c.Digest[:])
	return digest
}

// pqValidatorHash computes a domain-separated hash for the validator.
func pqValidatorHash(domain string, data []byte) []byte {
	input := make([]byte, 0, len(domain)+len(data))
	input = append(input, []byte(domain)...)
	input = append(input, data...)
	return crypto.Keccak256(input)
}
