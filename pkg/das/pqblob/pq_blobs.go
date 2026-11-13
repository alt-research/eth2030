// Package das implements PeerDAS data structures and verification logic.
//
// pq_blobs.go implements the experimental ordered Merkle blob commitment.
package pqblob

import (
	"encoding/binary"
	"errors"
	"math/bits"

	"github.com/eth2030/eth2030/core/types"
	"sync"

	"github.com/eth2030/eth2030/crypto"
)

// Post-quantum blob constants.
const (
	// PQCommitmentSize is the byte size of a PQ blob commitment.
	PQCommitmentSize = 64

	// LatticeDimension is the lattice dimension used for commitments.
	LatticeDimension = 256

	// LatticeModulus is the prime modulus for the lattice ring.
	LatticeModulus = 12289

	// ChunkSize is the byte size of each blob chunk for proof generation.
	ChunkSize = 32

	// MaxBlobSize is the maximum blob size supported (128 KiB).
	MaxBlobSize = 128 * 1024
)

// PQ blob errors.
var (
	ErrPQBlobEmpty         = errors.New("pq_blob: empty data")
	ErrPQBlobTooLarge      = errors.New("pq_blob: data exceeds max size")
	ErrPQBlobNilCommitment = errors.New("pq_blob: nil commitment")
	ErrPQBlobNilProof      = errors.New("pq_blob: nil proof")
	ErrPQBlobIndexOOB      = errors.New("pq_blob: chunk index out of bounds")
	ErrPQBlobMismatch      = errors.New("pq_blob: commitment mismatch")
	ErrPQBlobBatchEmpty    = errors.New("pq_blob: empty batch")
	ErrPQBlobBatchMismatch = errors.New("pq_blob: batch length mismatch")
)

// PQBlobCommitment binds the Merkle root to the original byte length.
type PQBlobCommitment struct {
	// Digest is a 64-byte Merkle commitment digest.
	Digest [PQCommitmentSize]byte

	// NumChunks records how many chunks the original blob was divided into.
	NumChunks uint32

	// DataSize is the original data size in bytes.
	DataSize uint32
}

// PQBlobProof is a Merkle proof for a specific chunk within a blob.
// It demonstrates that a particular chunk is consistent with the commitment
// without revealing the entire blob.
type PQBlobProof struct {
	// ChunkIndex is the index of the chunk this proof covers.
	ChunkIndex uint32

	// ChunkHash is the Keccak256 hash of the chunk data.
	ChunkHash [32]byte

	// Siblings contains the ordered authentication path from leaf to root.
	Siblings []types.Hash

	// CommitmentDigest links back to the commitment this proof is for.
	CommitmentDigest [PQCommitmentSize]byte
}

// CommitBlob commits the ordered 32-byte chunks and the original byte length.
func CommitBlob(data []byte) (*PQBlobCommitment, error) {
	if len(data) == 0 {
		return nil, ErrPQBlobEmpty
	}
	if len(data) > MaxBlobSize {
		return nil, ErrPQBlobTooLarge
	}

	numChunks := chunkCount(len(data))
	chunkHashes := computeChunkHashes(data, numChunks)
	digest := blobDigest(blobMerkleRoot(chunkHashes), uint32(len(data)))

	return &PQBlobCommitment{
		Digest: digest, NumChunks: uint32(numChunks), DataSize: uint32(len(data)),
	}, nil
}

// VerifyBlobCommitment checks whether a commitment is consistent with the
// given data. Returns true if the commitment matches.
func VerifyBlobCommitment(commitment *PQBlobCommitment, data []byte) bool {
	if commitment == nil || len(data) == 0 {
		return false
	}
	if uint32(len(data)) != commitment.DataSize {
		return false
	}

	recomputed, err := CommitBlob(data)
	if err != nil {
		return false
	}
	return *recomputed == *commitment
}

// GenerateBlobProof creates a Merkle proof that a specific chunk at
// the given index is part of the committed blob.
func GenerateBlobProof(data []byte, index uint32) (*PQBlobProof, error) {
	if len(data) == 0 {
		return nil, ErrPQBlobEmpty
	}
	if len(data) > MaxBlobSize {
		return nil, ErrPQBlobTooLarge
	}

	numChunks := uint32(chunkCount(len(data)))
	if index >= numChunks {
		return nil, ErrPQBlobIndexOOB
	}

	leaves := computeChunkHashes(data, int(numChunks))
	proof := &PQBlobProof{ChunkIndex: index, ChunkHash: leaves[index]}
	position := int(index)
	for len(leaves) > 1 {
		proof.Siblings = append(proof.Siblings, leaves[position^1])
		leaves = blobMerkleLevel(leaves)
		position >>= 1
	}
	proof.CommitmentDigest = blobDigest(leaves[0], uint32(len(data)))
	return proof, nil
}

// VerifyBlobProof checks an ordered authentication path against the
// commitment's exact chunk count and original byte length.
func VerifyBlobProof(proof *PQBlobProof, commitment *PQBlobCommitment) bool {
	if ValidatePQBlobProof(proof) != nil || ValidatePQBlob(commitment) != nil ||
		proof.CommitmentDigest != commitment.Digest || proof.ChunkIndex >= commitment.NumChunks ||
		len(proof.Siblings) != bits.Len32(commitment.NumChunks-1) {
		return false
	}
	root := types.Hash(proof.ChunkHash)
	position := proof.ChunkIndex
	for _, sibling := range proof.Siblings {
		if position&1 == 0 {
			root = blobBranch(root, sibling)
		} else {
			root = blobBranch(sibling, root)
		}
		position >>= 1
	}
	return blobDigest(root, commitment.DataSize) == commitment.Digest
}

// BatchVerifyProofs verifies multiple proofs against their corresponding
// commitments. Returns true only if all proofs are valid. The proofs and
// commitments slices must have the same length.
func BatchVerifyProofs(proofs []*PQBlobProof, commitments []*PQBlobCommitment) bool {
	if len(proofs) == 0 || len(commitments) == 0 {
		return false
	}
	if len(proofs) != len(commitments) {
		return false
	}

	// Use parallel verification for batches larger than 4.
	if len(proofs) > 4 {
		return batchVerifyParallel(proofs, commitments)
	}

	for i := range proofs {
		if !VerifyBlobProof(proofs[i], commitments[i]) {
			return false
		}
	}
	return true
}

// batchVerifyParallel verifies proofs in parallel using goroutines.
func batchVerifyParallel(proofs []*PQBlobProof, commitments []*PQBlobCommitment) bool {
	var (
		wg      sync.WaitGroup
		valid   = true
		mu      sync.Mutex
		n       = len(proofs)
		workers = 4
	)
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
				if !VerifyBlobProof(proofs[i], commitments[i]) {
					mu.Lock()
					valid = false
					mu.Unlock()
					return
				}
			}
		}(start, end)
	}

	wg.Wait()
	return valid
}

// ValidatePQBlob checks that a PQ blob commitment is well-formed:
// non-zero digest, valid chunk count, and consistent data size.
func ValidatePQBlob(commitment *PQBlobCommitment) error {
	if commitment == nil {
		return ErrPQBlobNilCommitment
	}
	allZero := true
	for _, b := range commitment.Digest {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return errors.New("pq_blob: zero commitment digest")
	}
	if commitment.DataSize == 0 {
		return ErrPQBlobEmpty
	}
	if commitment.DataSize > MaxBlobSize {
		return ErrPQBlobTooLarge
	}
	expectedChunks := uint32(chunkCount(int(commitment.DataSize)))
	if commitment.NumChunks != expectedChunks {
		return errors.New("pq_blob: chunk count inconsistent with data size")
	}
	return nil
}

// ValidatePQBlobProof checks that a PQ blob proof has valid structure.
func ValidatePQBlobProof(proof *PQBlobProof) error {
	if proof == nil {
		return ErrPQBlobNilProof
	}
	if len(proof.Siblings) > bits.Len32(MaxBlobSize/ChunkSize-1) {
		return ErrPQBlobMismatch
	}
	return nil
}

// chunkCount returns the number of ChunkSize-byte chunks needed for data.
func chunkCount(dataLen int) int {
	return (dataLen + ChunkSize - 1) / ChunkSize
}

// extractChunk extracts the chunk at the given index, zero-padding if needed.
func extractChunk(data []byte, index uint32) []byte {
	start := int(index) * ChunkSize
	end := start + ChunkSize
	if start >= len(data) {
		return make([]byte, ChunkSize)
	}
	if end > len(data) {
		chunk := make([]byte, ChunkSize)
		copy(chunk, data[start:])
		return chunk
	}
	chunk := make([]byte, ChunkSize)
	copy(chunk, data[start:end])
	return chunk
}

// computeChunkHashes returns leaf hashes padded to a power of two.
func computeChunkHashes(data []byte, count int) []types.Hash {
	leaves := make([]types.Hash, 1<<bits.Len(uint(count-1)))
	for i := range count {
		leaves[i] = crypto.Keccak256Hash(extractChunk(data, uint32(i)))
	}
	return leaves
}

func blobBranch(left, right types.Hash) types.Hash {
	return crypto.Keccak256Hash([]byte{1}, left[:], right[:])
}

func blobMerkleLevel(level []types.Hash) []types.Hash {
	next := make([]types.Hash, len(level)/2)
	for i := range next {
		next[i] = blobBranch(level[2*i], level[2*i+1])
	}
	return next
}

func blobMerkleRoot(level []types.Hash) types.Hash {
	for len(level) > 1 {
		level = blobMerkleLevel(level)
	}
	return level[0]
}

func blobDigest(root types.Hash, size uint32) [PQCommitmentSize]byte {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], size)
	binding := crypto.Keccak256Hash([]byte("eth2030-pq-blob-merkle-v1"), length[:], root[:])
	var digest [PQCommitmentSize]byte
	copy(digest[:32], root[:])
	copy(digest[32:], binding[:])
	return digest
}
