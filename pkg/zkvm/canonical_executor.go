// canonical_executor.go wires canonical guest execution to the real RISC-V
// CPU emulator (riscv_cpu.go). It connects registered guest programs from
// GuestRegistry to actual RVCPU execution with witness collection and ZK
// proof generation via the proof backend.
//
// Part of the K+ roadmap for canonical RISC-V guest execution.
package zkvm

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/zkvm/canonicalproof"
)

// CanonicalExecutor errors.
var (
	ErrCanonExecNilRegistry     = errors.New("canonical_exec: nil guest registry")
	ErrCanonExecProgramNotFound = errors.New("canonical_exec: program not found")
	ErrCanonExecGasExhausted    = errors.New("canonical_exec: gas limit exhausted")
	ErrCanonExecCPUFault        = errors.New("canonical_exec: cpu execution fault")
	ErrCanonExecBadExitCode     = errors.New("canonical_exec: non-zero exit code")
	ErrCanonExecProofFailed     = errors.New("canonical_exec: proof generation failed")
	ErrCanonExecVerifyFailed    = errors.New("canonical_exec: proof verification failed")
	ErrCanonExecNilProof        = errors.New("canonical_exec: nil proof data")
	ErrCanonExecEmptyInput      = errors.New("canonical_exec: empty program ID")
)

// CanonicalExecutorConfig holds configuration for the executor.
type CanonicalExecutorConfig struct {
	// GasLimit is the maximum gas (instruction count) for guest execution.
	GasLimit uint64

	// MemoryPages is the maximum number of 4KB pages the guest may allocate.
	MemoryPages int

	// StackBase is the initial stack pointer address.
	StackBase uint32

	// ProgramBase is the base address for loading program code.
	ProgramBase uint32

	// InputBase is the base address for loading guest input data.
	InputBase uint32

	// CollectWitness enables witness collection for proof generation.
	CollectWitness bool
}

// DefaultCanonicalExecutorConfig returns sensible defaults.
func DefaultCanonicalExecutorConfig() CanonicalExecutorConfig {
	return CanonicalExecutorConfig{
		GasLimit:       1 << 24, // ~16M instructions
		MemoryPages:    4096,    // 16 MiB
		StackBase:      0x80000000,
		ProgramBase:    0x00010000,
		InputBase:      0x40000000,
		CollectWitness: true,
	}
}

// GuestOutput holds the result of a canonical guest execution.
type GuestOutput struct {
	// Output is the bytes written to the output buffer by the guest.
	Output []byte

	// ExitCode is the ECALL halt exit code (0 = success).
	ExitCode uint32

	// GasUsed is the total instructions executed.
	GasUsed uint64

	// Steps is the total CPU step count.
	Steps uint64

	// ProgramHash is the SHA-256 hash of the executed program.
	ProgramHash [32]byte
}

// GuestProof holds a proof of correct guest execution.
type GuestProof struct {
	// ProgramID is the trusted registry key, distinct from the SHA-256 image
	// input digest used by the execution statement.
	ProgramID types.Hash
	// ProofResult is the raw proof from the proof backend.
	ProofResult *ProofResult

	// ProgramHash identifies which guest program was executed.
	ProgramHash [32]byte

	// PublicInputs are the publicly verifiable inputs.
	PublicInputs []byte

	// OutputHash is SHA-256(output).
	OutputHash [32]byte
}

// CanonicalExecutor manages execution of registered guest programs on RVCPU.
type CanonicalExecutor struct {
	registry *GuestRegistry
	config   CanonicalExecutorConfig
	mu       sync.Mutex
}

// NewCanonicalExecutor creates a new executor bound to a guest registry.
func NewCanonicalExecutor(registry *GuestRegistry, config CanonicalExecutorConfig) (*CanonicalExecutor, error) {
	if registry == nil {
		return nil, ErrCanonExecNilRegistry
	}
	return &CanonicalExecutor{
		registry: registry,
		config:   config,
	}, nil
}

// ExecuteGuest loads a registered program by its hash, runs it on RVCPU with
// the given input, and returns the output along with the witness trace.
func (ce *CanonicalExecutor) ExecuteGuest(programID types.Hash, input []byte) (*GuestOutput, *RVWitnessCollector, error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Look up the registered program.
	program, err := ce.registry.GetGuest(programID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrCanonExecProgramNotFound, err)
	}

	// Set up the CPU.
	cpu := NewRVCPU(ce.config.GasLimit)
	cpu.Memory.SetMaxPages(ce.config.MemoryPages)

	// Attach witness collector if enabled.
	var witness *RVWitnessCollector
	if ce.config.CollectWitness {
		witness = NewRVWitnessCollector()
		cpu.Witness = witness
	}

	// Load the program at ProgramBase.
	if err := cpu.LoadProgram(program, ce.config.ProgramBase, ce.config.ProgramBase); err != nil {
		return nil, nil, fmt.Errorf("%w: load program: %v", ErrCanonExecCPUFault, err)
	}

	// Load input data into memory at InputBase.
	if len(input) > 0 {
		if err := cpu.Memory.LoadSegment(ce.config.InputBase, input); err != nil {
			return nil, nil, fmt.Errorf("%w: load input: %v", ErrCanonExecCPUFault, err)
		}
	}

	// Set up the input buffer for ECALL-based I/O as well.
	cpu.InputBuf = input

	// Initialize stack pointer (x2/sp) to StackBase.
	cpu.Regs[2] = ce.config.StackBase

	// Store input length in a1 (x11) and input base address in a0 (x10)
	// so the guest can find its input.
	cpu.Regs[10] = ce.config.InputBase
	cpu.Regs[11] = uint32(len(input))

	// Run the CPU.
	runErr := cpu.Run()

	programHash := HashProgram(program)

	output := &GuestOutput{
		Output:      make([]byte, len(cpu.OutputBuf)),
		ExitCode:    cpu.ExitCode,
		GasUsed:     cpu.GasUsed,
		Steps:       cpu.Steps,
		ProgramHash: programHash,
	}
	copy(output.Output, cpu.OutputBuf)

	// If the CPU halted normally (ExitCode=0, Halted=true), that is success.
	// Gas exhaustion is a hard error.
	if runErr != nil {
		if errors.Is(runErr, ErrRVGasExhausted) {
			return output, witness, ErrCanonExecGasExhausted
		}
		// Non-gas errors that result in a halt with exit code are acceptable.
		if !cpu.Halted {
			return output, witness, fmt.Errorf("%w: %v", ErrCanonExecCPUFault, runErr)
		}
	}

	return output, witness, nil
}

// ExecuteAndProve runs a guest program and generates a ZK proof of execution.
func (ce *CanonicalExecutor) ExecuteAndProve(programID types.Hash, input []byte) (*GuestOutput, *GuestProof, error) {
	ctx, cancel := context.WithTimeout(context.Background(), canonicalproof.ProvingTimeout)
	defer cancel()
	return ce.ExecuteAndProveContext(ctx, programID, input)
}

func (ce *CanonicalExecutor) proofConfig() canonicalproof.Config {
	return canonicalproof.Config{MaxCycles: ce.config.GasLimit, MemoryPages: uint32(ce.config.MemoryPages), ProgramBase: ce.config.ProgramBase, EntryPC: ce.config.ProgramBase, StackBase: ce.config.StackBase, InputBase: ce.config.InputBase, LoadInput: true}
}

func (ce *CanonicalExecutor) ExecuteAndProveContext(ctx context.Context, programID types.Hash, input []byte) (*GuestOutput, *GuestProof, error) {
	output, _, err := ce.ExecuteGuest(programID, input)
	if err != nil {
		return output, nil, err
	}

	// Build public inputs: programHash || inputHash || outputHash.
	inputHash := sha256.Sum256(input)
	outputHash := sha256.Sum256(output.Output)

	publicInputs := make([]byte, 0, 96)
	publicInputs = append(publicInputs, output.ProgramHash[:]...)
	publicInputs = append(publicInputs, inputHash[:]...)
	publicInputs = append(publicInputs, outputHash[:]...)

	program, err := ce.registry.GetGuest(programID)
	if err != nil {
		return output, nil, err
	}
	prover := canonicalproof.Prover{}
	receipt, err := prover.Prove(ctx, program, input, ce.proofConfig())
	if err != nil {
		return output, nil, fmt.Errorf("%w: %v", ErrCanonExecProofFailed, err)
	}
	if err := prover.Verify(ctx, receipt, canonicalproof.Expected{Program: program, Input: input, Output: output.Output, Config: ce.proofConfig(), Cycles: &output.GasUsed}); err != nil {
		return output, nil, err
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return output, nil, err
	}
	key, _ := hex.DecodeString(receipt.ImageID)
	state, _ := hex.DecodeString(receipt.Statement.StateHash)
	var stateHash [32]byte
	copy(stateHash[:], state)
	proofResult := &ProofResult{ProofBytes: encoded, VerificationKey: key, TraceCommitment: stateHash, PublicInputsHash: sha256.Sum256(publicInputs)}

	proof := &GuestProof{
		ProgramID:    programID,
		ProofResult:  proofResult,
		ProgramHash:  output.ProgramHash,
		PublicInputs: publicInputs,
		OutputHash:   outputHash,
	}

	return output, proof, nil
}

// VerifyGuestProof verifies a guest execution proof against the claimed
// program hash and public inputs.
func (ce *CanonicalExecutor) VerifyGuestProof(proof *GuestProof) error {
	if proof == nil || proof.ProofResult == nil {
		return ErrCanonExecNilProof
	}

	if len(proof.PublicInputs) != 96 {
		return ErrCanonExecVerifyFailed
	}
	program, err := ce.registry.GetGuest(proof.ProgramID)
	if err != nil || HashProgram(program) != proof.ProgramHash {
		return ErrCanonExecVerifyFailed
	}
	receipt, err := canonicalproof.DecodeProof(proof.ProofResult.ProofBytes)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCanonExecVerifyFailed, err)
	}
	expectedPIHash := sha256.Sum256(proof.PublicInputs)
	if expectedPIHash != proof.ProofResult.PublicInputsHash {
		return fmt.Errorf("%w: public inputs hash mismatch", ErrCanonExecVerifyFailed)
	}
	if hex.EncodeToString(proof.PublicInputs[:32]) != hex.EncodeToString(proof.ProgramHash[:]) || hex.EncodeToString(proof.PublicInputs[64:]) != hex.EncodeToString(proof.OutputHash[:]) || hex.EncodeToString(proof.ProofResult.VerificationKey) != receipt.ImageID || hex.EncodeToString(proof.ProofResult.TraceCommitment[:]) != receipt.Statement.StateHash {
		return ErrCanonExecVerifyFailed
	}
	expected := receipt.Statement
	expected.ProgramHash = hex.EncodeToString(proof.ProgramHash[:])
	expected.InputHash = hex.EncodeToString(proof.PublicInputs[32:64])
	expected.OutputHash = hex.EncodeToString(proof.OutputHash[:])
	expected.Config = ce.proofConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return (canonicalproof.Prover{}).VerifyClaim(ctx, receipt, expected)
}

// ComputeProgramID computes the program ID (hash) for a given program binary.
// This is the same as the GuestRegistry key.
func ComputeProgramID(program []byte) types.Hash {
	return crypto.Keccak256Hash(program)
}

// buildPublicInputsFromOutput constructs the standard public inputs encoding
// from a GuestOutput and the original input.
func buildPublicInputsFromOutput(output *GuestOutput, input []byte) []byte {
	inputHash := sha256.Sum256(input)
	outputHash := sha256.Sum256(output.Output)

	buf := make([]byte, 0, 32+32+32+8)
	buf = append(buf, output.ProgramHash[:]...)
	buf = append(buf, inputHash[:]...)
	buf = append(buf, outputHash[:]...)

	var exitBuf [8]byte
	binary.LittleEndian.PutUint64(exitBuf[:], uint64(output.ExitCode))
	buf = append(buf, exitBuf[:]...)

	return buf
}
