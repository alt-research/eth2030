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

var (
	ErrProofBackendNilTrace          = errors.New("proof: nil witness trace")
	ErrProofBackendEmptyTrace        = errors.New("proof: empty witness trace")
	ErrProofBackendNilRequest        = errors.New("proof: nil proof request")
	ErrProofBackendNilResult         = errors.New("proof: nil proof result")
	ErrProofBackendInvalidProof      = errors.New("proof: verification failed")
	ErrProofBackendBadLength         = errors.New("proof: invalid proof encoding")
	ErrProofBackendStatementRequired = errors.New("proof: executable program, input and machine statement required; a trace commitment is not an execution proof")
)

// Kept only to identify and reject the old 256-byte synthetic fixture format.
const groth16ProofSize = 256

// ProofRequest proves a concrete execution. Trace alone is not an authoritative
// statement and is never converted into a cryptographic proof.
type ProofRequest struct {
	Trace        *RVWitnessCollector
	PublicInputs []byte
	ProgramHash  [32]byte
	Program      []byte
	Input        []byte
	Config       *canonicalproof.Config
}

// ProofResult carries a real fixed-image receipt. The legacy metadata fields
// now bind the image ID, final machine-state digest and public digest tuple.
type ProofResult struct {
	ProofBytes       []byte
	VerificationKey  []byte
	TraceCommitment  [32]byte
	PublicInputsHash [32]byte
}

func receiptPublicInputs(proof *canonicalproof.Proof) ([]byte, error) {
	var inputs []byte
	for _, value := range []string{proof.Statement.ProgramHash, proof.Statement.InputHash, proof.Statement.OutputHash} {
		part, err := hex.DecodeString(value)
		if err != nil || len(part) != 32 {
			return nil, ErrProofBackendInvalidProof
		}
		inputs = append(inputs, part...)
	}
	return inputs, nil
}

func ProveExecution(req *ProofRequest) (*ProofResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), canonicalproof.ProvingTimeout)
	defer cancel()
	return ProveExecutionContext(ctx, req)
}

func ProveExecutionContext(ctx context.Context, req *ProofRequest) (*ProofResult, error) {
	if req == nil {
		return nil, ErrProofBackendNilRequest
	}
	if len(req.Program) == 0 {
		return nil, ErrProofBackendStatementRequired
	}
	programHash := HashProgram(req.Program)
	if req.ProgramHash != ([32]byte{}) && req.ProgramHash != programHash {
		return nil, ErrProofBackendInvalidProof
	}
	cfg := canonicalproof.DefaultConfig()
	if req.Config != nil {
		cfg = *req.Config
	}
	proof, err := (canonicalproof.Prover{}).Prove(ctx, req.Program, req.Input, cfg)
	if err != nil {
		return nil, err
	}
	inputs, err := receiptPublicInputs(proof)
	if err != nil {
		return nil, err
	}
	if len(req.PublicInputs) > 0 && sha256.Sum256(req.PublicInputs) != sha256.Sum256(inputs) {
		return nil, ErrProofBackendInvalidProof
	}
	encoded, err := json.Marshal(proof)
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(proof.ImageID)
	if err != nil || len(key) != 32 {
		return nil, ErrProofBackendInvalidProof
	}
	state, err := hex.DecodeString(proof.Statement.StateHash)
	if err != nil || len(state) != 32 {
		return nil, ErrProofBackendInvalidProof
	}
	var stateHash [32]byte
	copy(stateHash[:], state)
	return &ProofResult{ProofBytes: encoded, VerificationKey: key, TraceCommitment: stateHash, PublicInputsHash: sha256.Sum256(inputs)}, nil
}

// VerifyExecProof verifies only real receipts. No length/structure/hash-shaped
// fallback is permitted. Consumers must also compare PublicInputsHash to their
// expected program/input/output digest tuple.
func VerifyExecProof(result *ProofResult, programHash [32]byte) (bool, error) {
	if result == nil {
		return false, ErrProofBackendNilResult
	}
	proof, err := canonicalproof.DecodeProof(result.ProofBytes)
	if err != nil {
		return false, ErrProofBackendBadLength
	}
	if proof.Statement.ProgramHash != hex.EncodeToString(programHash[:]) || proof.ImageID != hex.EncodeToString(result.VerificationKey) || proof.Statement.StateHash != hex.EncodeToString(result.TraceCommitment[:]) {
		return false, ErrProofBackendInvalidProof
	}
	inputs, err := receiptPublicInputs(proof)
	if err != nil {
		return false, err
	}
	if sha256.Sum256(inputs) != result.PublicInputsHash {
		return false, ErrProofBackendInvalidProof
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = (canonicalproof.Prover{}).VerifyClaim(ctx, proof, proof.Statement)
	return err == nil, err
}

func HashProgram(program []byte) [32]byte { return sha256.Sum256(program) }

// VerifyWithBackend cannot infer an authenticated execution from an arbitrary
// trace and opaque public bytes. Call ProveExecution with an explicit statement.
func VerifyWithBackend(trace *RVWitnessCollector, program, publicInputs []byte) (*ProofResult, error) {
	if trace == nil {
		return nil, ErrProofBackendNilTrace
	}
	if trace.StepCount() == 0 {
		return nil, ErrProofBackendEmptyTrace
	}
	return nil, ErrProofBackendStatementRequired
}
