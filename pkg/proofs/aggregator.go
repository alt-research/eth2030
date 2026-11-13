package proofs

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/ssz"
)

// Aggregation errors.
var (
	ErrNoProofs     = errors.New("proofs: no proofs to aggregate")
	ErrNilProof     = errors.New("proofs: nil aggregated proof")
	ErrProofInvalid = errors.New("proofs: proof verification failed")
)

// ProofAggregator defines the interface for aggregating and verifying proofs.
type ProofAggregator interface {
	// Aggregate combines multiple execution proofs into an aggregated proof.
	Aggregate(proofs []ExecutionProof) (*AggregatedProof, error)

	// Verify checks the validity of an aggregated proof.
	Verify(proof *AggregatedProof) (bool, error)
}

// SimpleAggregator aggregates proofs by computing a merkle root over
// individual proof hashes, using SHA-256.
type SimpleAggregator struct{}

// NewSimpleAggregator creates a new SimpleAggregator.
func NewSimpleAggregator() *SimpleAggregator {
	return &SimpleAggregator{}
}

// Aggregate computes a merkle root over individual proof hashes.
func (a *SimpleAggregator) Aggregate(proofs []ExecutionProof) (*AggregatedProof, error) {
	if len(proofs) == 0 {
		return nil, ErrNoProofs
	}

	// Hash each proof to get leaf nodes.
	leaves := make([][32]byte, len(proofs))
	for i, p := range proofs {
		leaves[i] = hashProof(&p)
	}

	// Use SSZ merkleization to compute the root.
	root := ssz.Merkleize(leaves, 0)

	var aggregateRoot types.Hash
	copy(aggregateRoot[:], root[:])

	return &AggregatedProof{
		Proofs:        proofs,
		AggregateRoot: aggregateRoot,
		Valid:         true,
	}, nil
}

// Verify re-computes the merkle root and compares it to the stored aggregate root.
func (a *SimpleAggregator) Verify(proof *AggregatedProof) (bool, error) {
	if proof == nil {
		return false, ErrNilProof
	}
	if len(proof.Proofs) == 0 {
		return false, ErrNoProofs
	}

	leaves := make([][32]byte, len(proof.Proofs))
	for i, p := range proof.Proofs {
		leaves[i] = hashProof(&p)
	}

	root := ssz.Merkleize(leaves, 0)

	var expected types.Hash
	copy(expected[:], root[:])

	return expected == proof.AggregateRoot, nil
}

// hashProof hashes an ExecutionProof into a 32-byte leaf.
func hashProof(p *ExecutionProof) [32]byte {
	h := sha256.New()
	h.Write([]byte("eth2030-execution-proof-v2"))
	h.Write(p.StateRoot[:])
	h.Write(p.BlockHash[:])
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(p.ProofData)))
	h.Write(length[:])
	h.Write(p.ProofData)
	binary.BigEndian.PutUint64(length[:], uint64(len(p.ProverID)))
	h.Write(length[:])
	h.Write([]byte(p.ProverID))
	var typeBuf [4]byte
	binary.LittleEndian.PutUint32(typeBuf[:], uint32(p.Type))
	h.Write(typeBuf[:])
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// STARKAggregateProvider proves and verifies the validity of the ordered
// execution proofs, bound to the canonical aggregate root. Implementations
// must verify the individual execution statements, not just their hashes.
type STARKAggregateProvider interface {
	Prove(proofs []ExecutionProof, root types.Hash) ([]byte, error)
	Verify(proofs []ExecutionProof, root types.Hash, aggregateProof []byte) (bool, error)
}

var ErrSTARKAggregateNoProvider = errors.New("proofs: STARK aggregation provider is not configured")

// STARKAggregator requires an explicit provider. No local placeholder prover
// is installed by default.
type STARKAggregator struct{ provider STARKAggregateProvider }

func NewSTARKAggregator() *STARKAggregator { return &STARKAggregator{} }

// NewSTARKAggregatorWithProvider configures the provider for the concrete
// aggregation statement and proof format selected by the caller.
func NewSTARKAggregatorWithProvider(provider STARKAggregateProvider) *STARKAggregator {
	return &STARKAggregator{provider: provider}
}

func (sa *STARKAggregator) Aggregate(proofs []ExecutionProof) (*AggregatedProof, error) {
	if len(proofs) == 0 {
		return nil, ErrNoProofs
	}
	if sa == nil || sa.provider == nil {
		return nil, ErrSTARKAggregateNoProvider
	}
	frozen := cloneExecutionProofs(proofs)
	root := starkAggregateRoot(frozen)
	artifact, err := sa.provider.Prove(cloneExecutionProofs(frozen), root)
	if err != nil {
		return nil, err
	}
	if len(artifact) == 0 {
		return nil, ErrProofInvalid
	}
	valid, err := sa.provider.Verify(cloneExecutionProofs(frozen), root, bytes.Clone(artifact))
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrProofInvalid
	}
	return &AggregatedProof{Proofs: frozen, AggregateRoot: root, AggregateProof: bytes.Clone(artifact), Valid: true}, nil
}

func (sa *STARKAggregator) Verify(proof *AggregatedProof) (bool, error) {
	if proof == nil {
		return false, ErrNilProof
	}
	if len(proof.Proofs) == 0 {
		return false, ErrNoProofs
	}
	if sa == nil || sa.provider == nil {
		return false, ErrSTARKAggregateNoProvider
	}
	if len(proof.AggregateProof) == 0 {
		return false, ErrProofInvalid
	}
	frozen := cloneExecutionProofs(proof.Proofs)
	root := starkAggregateRoot(frozen)
	if root != proof.AggregateRoot {
		return false, nil
	}
	return sa.provider.Verify(frozen, root, bytes.Clone(proof.AggregateProof))
}

func cloneExecutionProofs(proofs []ExecutionProof) []ExecutionProof {
	out := make([]ExecutionProof, len(proofs))
	for i, proof := range proofs {
		out[i] = proof
		out[i].ProofData = bytes.Clone(proof.ProofData)
	}
	return out
}

// starkAggregateRoot binds ordered fields with unambiguous lengths. This is a
// local artifact identifier, not a claim that its inputs are valid proofs.
func starkAggregateRoot(proofs []ExecutionProof) types.Hash {
	h := sha256.New()
	h.Write([]byte("eth2030-stark-aggregate-v1"))
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(proofs)))
	h.Write(n[:])
	for _, proof := range proofs {
		h.Write(proof.StateRoot[:])
		h.Write(proof.BlockHash[:])
		h.Write([]byte{byte(proof.Type)})
		binary.BigEndian.PutUint64(n[:], uint64(len(proof.ProofData)))
		h.Write(n[:])
		h.Write(proof.ProofData)
		binary.BigEndian.PutUint64(n[:], uint64(len(proof.ProverID)))
		h.Write(n[:])
		h.Write([]byte(proof.ProverID))
	}
	var root types.Hash
	copy(root[:], h.Sum(nil))
	return root
}
