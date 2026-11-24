package zkvm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/eth2030/eth2030/zkvm/canonicalproof"
)

// Verification errors.
var (
	ErrNilVerificationKey = errors.New("zkvm: nil verification key")
	ErrNilProof           = errors.New("zkvm: nil proof")
	ErrEmptyProofData     = errors.New("zkvm: empty proof data")
	ErrEmptyVKData        = errors.New("zkvm: empty verification key data")
	ErrInvalidProof       = errors.New("zkvm: invalid proof")
)

// VerifyProof validates a proof against a verification key.
// It returns true if the proof is valid, false otherwise.
func VerifyProof(vk *VerificationKey, proof *Proof) (bool, error) {
	if vk == nil {
		return false, ErrNilVerificationKey
	}
	if proof == nil {
		return false, ErrNilProof
	}
	if len(vk.Data) == 0 {
		return false, ErrEmptyVKData
	}
	if len(proof.Data) == 0 {
		return false, ErrEmptyProofData
	}

	return (&RISCZeroVerifier{}).Verify(vk, proof)
}

// RISCZeroVerifier is the production backend. The trusted key identifies the
// fixed interpreter image and the expected SHA-256 digest of the guest program.
type RISCZeroVerifier struct{}

func (*RISCZeroVerifier) Name() string { return "risc0-3.1.0" }

func (*RISCZeroVerifier) Prove(program *GuestProgram, input []byte) (*Proof, error) {
	if program == nil || len(program.Code) == 0 {
		return nil, ErrGuestEmptyProgram
	}
	ctx, cancel := context.WithTimeout(context.Background(), canonicalproof.ProvingTimeout)
	defer cancel()
	receipt, err := (canonicalproof.Prover{}).Prove(ctx, program.Code, input, canonicalproof.DefaultConfig())
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return nil, err
	}
	return &Proof{Data: data, PublicInputs: append([]byte(nil), input...)}, nil
}

func (*RISCZeroVerifier) Verify(vk *VerificationKey, proof *Proof) (bool, error) {
	if vk == nil {
		return false, ErrNilVerificationKey
	}
	if proof == nil {
		return false, ErrNilProof
	}
	if len(vk.Data) == 0 {
		return false, ErrEmptyVKData
	}
	if len(proof.Data) == 0 {
		return false, ErrEmptyProofData
	}
	receipt, err := canonicalproof.DecodeProof(proof.Data)
	if err != nil {
		return false, nil
	}
	if receipt.ImageID != hex.EncodeToString(vk.Data) {
		return false, nil
	}
	expected := receipt.Statement
	expected.ProgramHash = hex.EncodeToString(vk.ProgramHash[:])
	inputHash := sha256.Sum256(proof.PublicInputs)
	expected.InputHash = hex.EncodeToString(inputHash[:])
	expected.Config = canonicalproof.DefaultConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = (canonicalproof.Prover{}).VerifyClaim(ctx, receipt, expected)
	return err == nil, err
}

// MockVerifier is an explicitly selected test double. It does not verify
// computational integrity and is never the production VerifyProof backend.
type MockVerifier struct{}

// Name returns the verifier backend name.
func (v *MockVerifier) Name() string {
	return "mock"
}

// Prove generates a deterministic test fixture, not an execution proof.
func (v *MockVerifier) Prove(program *GuestProgram, input []byte) (*Proof, error) {
	if program == nil || len(program.Code) == 0 {
		return nil, errors.New("zkvm: nil or empty program")
	}

	// Generate proof data bound to the program and input via SHA-256.
	programHash := sha256.Sum256(program.Code)
	inputHash := sha256.Sum256(input)
	proofBinding := sha256.Sum256(append(programHash[:], inputHash[:]...))

	return &Proof{
		Data:         append([]byte("mock-proof:"), proofBinding[:]...),
		PublicInputs: input,
	}, nil
}

// Verify accepts nonzero structural fixtures for tests only.
func (v *MockVerifier) Verify(vk *VerificationKey, proof *Proof) (bool, error) {
	if vk == nil {
		return false, ErrNilVerificationKey
	}
	if proof == nil {
		return false, ErrNilProof
	}
	if len(vk.Data) == 0 {
		return false, ErrEmptyVKData
	}
	if len(proof.Data) == 0 {
		return false, ErrEmptyProofData
	}
	if len(proof.PublicInputs) == 0 {
		return false, nil
	}

	// Structural verification: proof data must be non-trivial.
	allZero := true
	for _, b := range proof.Data {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return false, nil
	}

	return true, nil
}

// RejectingVerifier is a testing verifier that rejects all proofs.
type RejectingVerifier struct{}

// Name returns the verifier backend name.
func (v *RejectingVerifier) Name() string {
	return "rejecting"
}

// Prove always returns an error.
func (v *RejectingVerifier) Prove(program *GuestProgram, input []byte) (*Proof, error) {
	return nil, errors.New("zkvm: rejecting verifier cannot prove")
}

// Verify always returns false.
func (v *RejectingVerifier) Verify(vk *VerificationKey, proof *Proof) (bool, error) {
	if vk == nil {
		return false, ErrNilVerificationKey
	}
	if proof == nil {
		return false, ErrNilProof
	}
	return false, nil
}
