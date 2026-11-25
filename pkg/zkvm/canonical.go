package zkvm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/zkvm/canonicalproof"
)

// sha256Hash computes SHA-256 of data and returns the fixed-size array.
func sha256Hash(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// Canonical zkVM Guest Execution (K+ roadmap)
//
// Implements the canonical guest framework for zkVM execution. This enables
// mandatory proof-carrying blocks where the state transition is verified via
// a RISC-V zkVM guest program. The guest program executes the Ethereum STF
// inside a zkVM and produces a proof of correct execution.

// CanonicalGuestPrecompileAddr is the precompile address for guest execution.
// Assigned at 0x0200 in the K+ precompile range.
var CanonicalGuestPrecompileAddr = types.BytesToAddress([]byte{0x02, 0x00})

// Guest execution errors.
var (
	ErrGuestEmptyProgram      = errors.New("zkvm: empty guest program")
	ErrGuestCycleLimit        = errors.New("zkvm: cycle limit exceeded")
	ErrGuestMemoryLimit       = errors.New("zkvm: memory limit exceeded")
	ErrGuestNotRegistered     = errors.New("zkvm: guest program not registered")
	ErrGuestAlreadyRegistered = errors.New("zkvm: guest program already registered")
	ErrGuestInvalidProof      = errors.New("zkvm: guest proof verification failed")
	ErrGuestInputTooShort     = errors.New("zkvm: precompile input too short")
)

// Default configuration values.
const (
	DefaultMaxCycles   = 1 << 24                 // ~16M cycles
	DefaultMemoryLimit = RVMaxPages * RVPageSize // 64 MiB, matching the machine limit
	DefaultProofSystem = "risc0"

	// Gas costs for the canonical guest precompile.
	GuestPrecompileBaseGas     = 100_000
	GuestPrecompilePerCycleGas = 1 // 1 gas per cycle
)

// CanonicalGuestConfig holds configuration for guest execution.
type CanonicalGuestConfig struct {
	MaxCycles   uint64
	MemoryLimit uint64
	ProofSystem string
}

// DefaultGuestConfig returns the default configuration.
func DefaultGuestConfig() CanonicalGuestConfig {
	return CanonicalGuestConfig{
		MaxCycles:   DefaultMaxCycles,
		MemoryLimit: DefaultMemoryLimit,
		ProofSystem: DefaultProofSystem,
	}
}

// RiscVGuest represents a RISC-V guest program ready for execution.
type RiscVGuest struct {
	program []byte
	input   []byte
	config  CanonicalGuestConfig
}

// GuestExecution holds the result of a guest program execution.
type GuestExecution struct {
	Output      []byte
	Cycles      uint64
	ProofData   []byte
	Success     bool
	StateHash   string
	ProofConfig *canonicalproof.Config
}

// NewRiscVGuest creates a new RISC-V guest with the given program, input, and config.
func NewRiscVGuest(program, input []byte, config CanonicalGuestConfig) *RiscVGuest {
	return &RiscVGuest{
		program: append([]byte(nil), program...),
		input:   append([]byte(nil), input...),
		config:  config,
	}
}

// Execute runs the guest locally. A successful execution is not a proof:
// ProofData remains empty until ExecuteAndProve produces a real receipt.
func (g *RiscVGuest) Execute() (*GuestExecution, error) {
	if len(g.program) == 0 {
		return nil, ErrGuestEmptyProgram
	}

	// Check memory limit: each page is 4KB, estimate pages from program size.
	memPages := (uint64(len(g.program)) + RVPageSize - 1) / RVPageSize
	if memPages*RVPageSize > g.config.MemoryLimit || g.config.MemoryLimit > DefaultMemoryLimit {
		return &GuestExecution{
			Cycles:  0,
			Success: false,
		}, ErrGuestMemoryLimit
	}

	// Create CPU with the cycle limit as gas limit.
	cpu := NewRVCPU(g.config.MaxCycles)
	if g.config.MaxCycles == 0 || g.config.MaxCycles > canonicalproof.MaxCycles {
		return nil, ErrGuestCycleLimit
	}
	cpu.Memory.SetMaxPages(int(g.config.MemoryLimit / RVPageSize))

	// Set up input buffer for ECALL-based I/O.
	cpu.InputBuf = g.input

	// Load program at address 0 with entry point 0.
	if err := cpu.LoadProgram(g.program, 0, 0); err != nil {
		return nil, fmt.Errorf("zkvm: load program: %w", err)
	}

	// Initialize stack pointer (x2/sp).
	cpu.Regs[2] = 0x80000000

	// Run the CPU.
	runErr := cpu.Run()

	cycles := cpu.GasUsed

	// Check if we ran out of gas (cycles).
	if runErr != nil {
		if errors.Is(runErr, ErrRVGasExhausted) {
			return &GuestExecution{
				Cycles:  cycles,
				Success: false,
			}, ErrGuestCycleLimit
		}
		// If the CPU didn't halt cleanly, treat it as a failure.
		if !cpu.Halted {
			return &GuestExecution{
				Cycles:  cycles,
				Success: false,
			}, fmt.Errorf("zkvm: cpu fault: %w", runErr)
		}
	}
	if cpu.ExitCode != 0 {
		return nil, fmt.Errorf("canonical guest: nonzero exit code %d", cpu.ExitCode)
	}

	// Collect output from the CPU's output buffer.
	output := make([]byte, len(cpu.OutputBuf))
	copy(output, cpu.OutputBuf)

	return &GuestExecution{
		Output:  output,
		Cycles:  cycles,
		Success: true,
	}, nil
}

// ExecuteAndProve binds the independent local execution result to a real,
// fixed-image RV32IM execution receipt. Missing proving software is an error.
func (g *RiscVGuest) ExecuteAndProve(ctx context.Context) (*GuestExecution, error) {
	execution, err := g.Execute()
	if err != nil {
		return execution, err
	}
	if g.config.ProofSystem != DefaultProofSystem {
		return execution, ErrGuestInvalidProof
	}
	cfg := canonicalproof.Config{MaxCycles: g.config.MaxCycles, MemoryPages: uint32(g.config.MemoryLimit / RVPageSize), StackBase: 0x80000000}
	prover := canonicalproof.Prover{}
	proof, err := prover.Prove(ctx, g.program, g.input, cfg)
	if err != nil {
		return execution, err
	}
	if err := prover.Verify(ctx, proof, canonicalproof.Expected{Program: g.program, Input: g.input, Config: cfg, Output: execution.Output, Cycles: &execution.Cycles}); err != nil {
		return execution, err
	}
	encoded, err := json.Marshal(proof)
	if err != nil {
		return execution, err
	}
	execution.ProofData, execution.StateHash, execution.ProofConfig = encoded, proof.Statement.StateHash, &cfg
	return execution, nil
}

// VerifyExecution verifies the receipt and the caller's program, input, output,
// cycle count, resource/layout configuration and final machine-state digest.
func VerifyExecution(execution *GuestExecution, program, input []byte) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return VerifyExecutionContext(ctx, execution, program, input) == nil
}

func VerifyExecutionContext(ctx context.Context, execution *GuestExecution, program, input []byte) error {
	if execution == nil || !execution.Success || execution.ProofConfig == nil || len(execution.StateHash) != 64 {
		return ErrGuestInvalidProof
	}
	proof, err := canonicalproof.DecodeProof(execution.ProofData)
	if err != nil {
		return err
	}
	return (canonicalproof.Prover{}).Verify(ctx, proof, canonicalproof.Expected{Program: program, Input: input, Config: *execution.ProofConfig, Output: execution.Output, Cycles: &execution.Cycles, StateHash: execution.StateHash})
}

// GuestRegistry manages registered guest programs by their program hash.
type GuestRegistry struct {
	mu       sync.RWMutex
	programs map[types.Hash][]byte // programHash -> program bytes
}

// NewGuestRegistry creates a new empty guest registry.
func NewGuestRegistry() *GuestRegistry {
	return &GuestRegistry{
		programs: make(map[types.Hash][]byte),
	}
}

// RegisterGuest registers a guest program by its hash.
func (r *GuestRegistry) RegisterGuest(program []byte) (types.Hash, error) {
	if len(program) == 0 {
		return types.Hash{}, ErrGuestEmptyProgram
	}

	h := crypto.Keccak256Hash(program)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.programs[h]; exists {
		return h, ErrGuestAlreadyRegistered
	}

	// Store a copy.
	stored := make([]byte, len(program))
	copy(stored, program)
	r.programs[h] = stored

	return h, nil
}

// GetGuest retrieves a registered guest program by hash.
func (r *GuestRegistry) GetGuest(programHash types.Hash) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	program, exists := r.programs[programHash]
	if !exists {
		return nil, ErrGuestNotRegistered
	}

	// Return a copy.
	result := make([]byte, len(program))
	copy(result, program)
	return result, nil
}

// Count returns the number of registered guest programs.
func (r *GuestRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.programs)
}

// ValidateGuestProgram checks that a guest program binary is well-formed and
// consistent with the registry:
//   - Program must be non-empty
//   - Program hash must match the expected hash (if provided)
//   - Program must be registered in the registry (if registry is provided)
func ValidateGuestProgram(program []byte, expectedHash types.Hash, registry *GuestRegistry) error {
	if len(program) == 0 {
		return ErrGuestEmptyProgram
	}
	actualHash := crypto.Keccak256Hash(program)
	if expectedHash != (types.Hash{}) && actualHash != expectedHash {
		return fmt.Errorf("zkvm: program hash mismatch: got %x, want %x", actualHash[:8], expectedHash[:8])
	}
	if registry != nil {
		_, err := registry.GetGuest(actualHash)
		if err != nil {
			return fmt.Errorf("zkvm: program not registered: %w", err)
		}
	}
	return nil
}

// CanonicalGuestPrecompile implements the precompile interface for canonical
// guest invocation at address 0x0200.
//
// Input format: programHash(32) || input(variable)
// Output: execution output bytes
type CanonicalGuestPrecompile struct {
	Registry *GuestRegistry
	Config   CanonicalGuestConfig
}

// RequiredGas returns the gas cost for running the canonical guest precompile.
func (p *CanonicalGuestPrecompile) RequiredGas(input []byte) uint64 {
	if len(input) < 32 {
		return GuestPrecompileBaseGas
	}
	// Base gas + per-byte cost for input.
	inputLen := uint64(len(input) - 32)
	return GuestPrecompileBaseGas + inputLen*GuestPrecompilePerCycleGas
}

// Run executes the canonical guest precompile.
func (p *CanonicalGuestPrecompile) Run(input []byte) ([]byte, error) {
	if len(input) < 32 {
		return nil, ErrGuestInputTooShort
	}

	// Parse program hash from first 32 bytes.
	var programHash types.Hash
	copy(programHash[:], input[:32])

	// Look up registered program.
	program, err := p.Registry.GetGuest(programHash)
	if err != nil {
		return nil, fmt.Errorf("canonical guest: %w", err)
	}

	// Execute the guest.
	guestInput := input[32:]
	guest := NewRiscVGuest(program, guestInput, p.Config)
	execution, err := guest.Execute()
	if err != nil {
		return nil, fmt.Errorf("canonical guest execution: %w", err)
	}

	if !execution.Success {
		return nil, errors.New("canonical guest: execution failed")
	}

	return execution.Output, nil
}
