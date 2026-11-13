// validation_frame_circuit.go implements STARK-based validation frame circuits.
// These circuits prove that EIP-8141 validation frames executed correctly,
// enabling frame calldata stripping for block compression.
//
// Part of the EL roadmap: proof aggregation and mandatory 3-of-5 proofs (K+).
package prover

import (
	"crypto/sha256"
	"errors"
)

// Validation frame circuit errors.
var (
	ErrFrameReverted         = errors.New("validation_frame: frame reverted")
	ErrFrameNilOutput        = errors.New("validation_frame: nil output")
	ErrFrameEmptyBatch       = errors.New("validation_frame: empty batch")
	ErrFrameProofUnavailable = errors.New("validation_frame: no execution-statement proof provider configured")
)

// ValidationFrameProver proves that validation frames executed correctly.
type ValidationFrameProver interface {
	ProveValidationFrame(frameCalldata, output []byte) (*STARKProofData, error)
	ProveAllValidationFrames(frames [][]byte) (*STARKProofData, error)
	Verify(proof *STARKProofData) bool
}

// StubValidationFrameProver is a stub that always succeeds for testing.
type StubValidationFrameProver struct{}

func (s *StubValidationFrameProver) ProveValidationFrame(frameCalldata, output []byte) (*STARKProofData, error) {
	if output == nil {
		return nil, ErrFrameNilOutput
	}
	if len(output) > 0 && output[0] == 0 {
		return nil, ErrFrameReverted
	}
	return &STARKProofData{
		TraceCommitment: sha256.Sum256(frameCalldata),
		TraceLength:     1,
		BlowupFactor:    DefaultBlowupFactor,
		NumQueries:      1,
		FieldModulus:    GoldilocksModulus,
		ConstraintCount: 1,
	}, nil
}

func (s *StubValidationFrameProver) ProveAllValidationFrames(frames [][]byte) (*STARKProofData, error) {
	if len(frames) == 0 {
		return nil, ErrFrameEmptyBatch
	}
	h := sha256.New()
	for _, f := range frames {
		h.Write(f)
	}
	var commit [32]byte
	copy(commit[:], h.Sum(nil))
	return &STARKProofData{
		TraceCommitment: commit,
		TraceLength:     uint64(len(frames)),
		BlowupFactor:    DefaultBlowupFactor,
		NumQueries:      1,
		FieldModulus:    GoldilocksModulus,
		ConstraintCount: 1,
	}, nil
}

func (s *StubValidationFrameProver) Verify(proof *STARKProofData) bool {
	return proof != nil
}

// STARKValidationFrameProver is retained for API compatibility. The former
// trace committed to calldata and a constant, not EVM execution or APPROVE.
// No execution-statement prover is supplied by this package.
type STARKValidationFrameProver struct{}

func NewSTARKValidationFrameProver() *STARKValidationFrameProver {
	return &STARKValidationFrameProver{}
}
func (*STARKValidationFrameProver) ProveValidationFrame(_ []byte, output []byte) (*STARKProofData, error) {
	if output == nil {
		return nil, ErrFrameNilOutput
	}
	if len(output) == 0 || output[0] == 0 {
		return nil, ErrFrameReverted
	}
	return nil, ErrFrameProofUnavailable
}
func (*STARKValidationFrameProver) ProveAllValidationFrames(frames [][]byte) (*STARKProofData, error) {
	if len(frames) == 0 {
		return nil, ErrFrameEmptyBatch
	}
	return nil, ErrFrameProofUnavailable
}
func (*STARKValidationFrameProver) Verify(*STARKProofData) bool { return false }

// ValidationFrameProofSize returns the approximate serialized proof size.
func ValidationFrameProofSize(proof *STARKProofData) int {
	return proof.ProofSize()
}
