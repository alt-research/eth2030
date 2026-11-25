// Package canonicalproof proves and verifies the versioned canonical RV32IM
// execution statement using the fixed RISC Zero guest image. Synthetic trace
// commitments from the older research helpers are not accepted as proofs.
package canonicalproof

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ProvingTimeout bounds CPU proving on modest hosts. Verification retains its
// separate short deadline; cancellation and node shutdown still interrupt work.
const ProvingTimeout = 15 * time.Minute

const (
	MaxProgram    = 1 << 20
	MaxInput      = 1 << 20
	MaxOutput     = 1 << 20
	MaxCycles     = 1 << 24
	MaxProof      = 16 << 20
	DefaultBinary = "eth2030-prover"
)

var (
	ErrUnavailable  = errors.New("canonical execution prover is not installed")
	ErrInvalidProof = errors.New("invalid canonical execution proof")
	ErrBounds       = errors.New("canonical execution resource bounds exceeded")
)

// Config is part of the proved statement, including the entire initial machine
// layout. It cannot be supplied by an untrusted proof in place of caller intent.
type Config struct {
	MaxCycles   uint64 `json:"max_cycles"`
	MemoryPages uint32 `json:"memory_pages"`
	ProgramBase uint32 `json:"program_base"`
	EntryPC     uint32 `json:"entry_pc"`
	StackBase   uint32 `json:"stack_base"`
	InputBase   uint32 `json:"input_base"`
	LoadInput   bool   `json:"load_input"`
}

func DefaultConfig() Config {
	return Config{MaxCycles: 65536, MemoryPages: 4096, StackBase: 0x80000000}
}

func (c Config) Validate() error {
	if c.MaxCycles == 0 || c.MaxCycles > MaxCycles || c.MemoryPages == 0 || c.MemoryPages > 16384 || c.ProgramBase >= 0xf0000000 || c.EntryPC&3 != 0 {
		return ErrBounds
	}
	return nil
}

type Statement struct {
	Version       uint32 `json:"version"`
	ProgramHash   string `json:"program_hash"`
	InputHash     string `json:"input_hash"`
	Output        string `json:"output"`
	OutputHash    string `json:"output_hash"`
	Cycles        uint64 `json:"cycles"`
	ExitCode      uint32 `json:"exit_code"`
	FinalPC       uint32 `json:"final_pc"`
	InputConsumed uint64 `json:"input_consumed"`
	StateHash     string `json:"state_hash"`
	Config        Config `json:"config"`
}

type Proof struct {
	Version   uint32    `json:"version"`
	ImageID   string    `json:"image_id"`
	Receipt   string    `json:"receipt"`
	Statement Statement `json:"statement"`
}

type Expected struct {
	Program []byte
	Input   []byte
	Config  Config
	// A non-nil output and cycles bind legacy execution-result callers as well.
	Output    []byte
	Cycles    *uint64
	StateHash string
}

type Metadata struct {
	System  string `json:"system"`
	SDK     string `json:"sdk"`
	Version uint32 `json:"version"`
	ImageID string `json:"image_id"`
}

// Prover selects a locally installed, fixed-image prover/verifier binary. It
// never accepts a verification key or executable path from a proof/RPC request.
type Prover struct{ Binary string }

func (p Prover) path() (string, error) {
	name := p.Binary
	if name == "" {
		if executable, err := os.Executable(); err == nil {
			sibling := filepath.Join(filepath.Dir(executable), DefaultBinary)
			if info, err := os.Stat(sibling); err == nil && !info.IsDir() {
				return sibling, nil
			}
		}
		name = DefaultBinary
	}
	resolved, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return resolved, nil
}

func (p Prover) Available() bool { _, err := p.path(); return err == nil }

type command struct {
	Action   string     `json:"action"`
	Program  string     `json:"program,omitempty"`
	Input    *string    `json:"input,omitempty"`
	Config   *Config    `json:"config,omitempty"`
	Proof    *Proof     `json:"proof,omitempty"`
	Expected *Statement `json:"expected,omitempty"`
}

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, ErrBounds
	}
	return b.Buffer.Write(data)
}

func DecodeProof(data []byte) (*Proof, error) {
	if len(data) == 0 || len(data) > MaxProof {
		return nil, ErrInvalidProof
	}
	var result Proof
	if err := decode(data, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProof, err)
	}
	if result.Version != 1 || len(result.ImageID) != 64 || result.Receipt == "" {
		return nil, ErrInvalidProof
	}
	return &result, nil
}

func decode(data []byte, result any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing proof response")
	}
	return nil
}

func (p Prover) run(ctx context.Context, request command, result any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.path()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if len(encoded) > 32<<20 {
		return ErrBounds
	}
	cmd := exec.CommandContext(ctx, path)
	configureProcess(cmd)
	cmd.WaitDelay = 2 * time.Second
	cmd.Stdin = bytes.NewReader(encoded)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if key != "RISC0_DEV_MODE" && key != "BONSAI_API_KEY" && key != "BONSAI_API_URL" {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "RISC0_DEV_MODE=0")
	output := &boundedBuffer{limit: MaxProof}
	stderr := &boundedBuffer{limit: 16384}
	cmd.Stdout, cmd.Stderr = output, stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("canonical prover failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return decode(output.Bytes(), result)
}

func (p Prover) Metadata(ctx context.Context) (Metadata, error) {
	var result Metadata
	err := p.run(ctx, command{Action: "metadata"}, &result)
	if err == nil && (result.System != "risc0" || result.SDK != "3.1.0" || result.Version != 1 || len(result.ImageID) != 64) {
		err = ErrInvalidProof
	}
	return result, err
}

func hash(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

func validateInputs(program, input []byte, cfg Config) error {
	if len(program) == 0 || len(program) > MaxProgram || len(input) > MaxInput {
		return ErrBounds
	}
	return cfg.Validate()
}

func matchStatement(proof *Proof, expected Expected) error {
	if proof == nil || proof.Version != 1 || proof.Statement.Version != 1 || len(proof.ImageID) != 64 || proof.Receipt == "" || len(proof.Receipt) > MaxProof {
		return ErrInvalidProof
	}
	if err := validateInputs(expected.Program, expected.Input, expected.Config); err != nil {
		return err
	}
	s := proof.Statement
	output, err := hex.DecodeString(s.Output)
	state, stateErr := hex.DecodeString(s.StateHash)
	if err != nil || len(output) > MaxOutput || stateErr != nil || len(state) != 32 || s.ProgramHash != hash(expected.Program) || s.InputHash != hash(expected.Input) || s.OutputHash != hash(output) || s.Config != expected.Config || s.ExitCode != 0 || s.Cycles == 0 || s.Cycles > s.Config.MaxCycles || s.InputConsumed > uint64(len(expected.Input)) {
		return ErrInvalidProof
	}
	if expected.Output != nil && !bytes.Equal(expected.Output, output) || expected.Cycles != nil && *expected.Cycles != s.Cycles || expected.StateHash != "" && expected.StateHash != s.StateHash {
		return ErrInvalidProof
	}
	return nil
}

func (p Prover) Verify(ctx context.Context, proof *Proof, expected Expected) error {
	if err := matchStatement(proof, expected); err != nil {
		return err
	}
	return p.VerifyClaim(ctx, proof, proof.Statement)
}

// VerifyClaim verifies an explicitly expected public journal. Callers that have
// raw program/input bytes should use Verify to derive their digests directly.
func (p Prover) VerifyClaim(ctx context.Context, proof *Proof, expected Statement) error {
	if proof == nil || proof.Version != 1 || proof.Statement != expected || expected.Version != 1 || expected.ExitCode != 0 || expected.Config.Validate() != nil || expected.Cycles == 0 || expected.Cycles > expected.Config.MaxCycles || len(proof.Receipt) > MaxProof {
		return ErrInvalidProof
	}
	output, err := hex.DecodeString(expected.Output)
	if err != nil || len(output) > MaxOutput || hash(output) != expected.OutputHash {
		return ErrInvalidProof
	}
	for _, value := range []string{expected.ProgramHash, expected.InputHash, expected.StateHash, proof.ImageID} {
		decoded, err := hex.DecodeString(value)
		if err != nil || len(decoded) != 32 {
			return ErrInvalidProof
		}
	}
	var result struct {
		Valid     bool      `json:"valid"`
		Statement Statement `json:"statement"`
	}
	if err := p.run(ctx, command{Action: "verify", Proof: proof, Expected: &expected}, &result); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidProof, err)
	}
	if !result.Valid || result.Statement != proof.Statement {
		return ErrInvalidProof
	}
	return nil
}

func (p Prover) Prove(ctx context.Context, program, input []byte, cfg Config) (*Proof, error) {
	if err := validateInputs(program, input, cfg); err != nil {
		return nil, err
	}
	reference, err := ReferenceExecution(ctx, program, input, cfg)
	if err != nil {
		return nil, err
	}
	encodedInput := hex.EncodeToString(input)
	var proof Proof
	if err := p.run(ctx, command{Action: "prove", Program: hex.EncodeToString(program), Input: &encodedInput, Config: &cfg}, &proof); err != nil {
		return nil, err
	}
	if err := p.Verify(ctx, &proof, Expected{Program: program, Input: input, Config: cfg}); err != nil {
		return nil, err
	}
	if proof.Statement != reference {
		return nil, fmt.Errorf("%w: independent RV32IM execution disagrees with proved statement", ErrInvalidProof)
	}
	return &proof, nil
}

// ExecuteReference invokes the same RV32IM machine natively without generating
// a proof. It is only a differential-testing tool, never a verification result.
func (p Prover) ExecuteReference(ctx context.Context, program, input []byte, cfg Config) (Statement, error) {
	var result Statement
	if err := validateInputs(program, input, cfg); err != nil {
		return result, err
	}
	encodedInput := hex.EncodeToString(input)
	err := p.run(ctx, command{Action: "execute", Program: hex.EncodeToString(program), Input: &encodedInput, Config: &cfg}, &result)
	return result, err
}
