package canonicalproof

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/eth2030/eth2030/zkvm/riscv"
)

// ReferenceExecution independently runs the statement on the Go RV32IM CPU.
// The producer checks this result against the actual proved Rust guest result;
// verification itself still verifies the proof and never substitutes replay.
func ReferenceExecution(ctx context.Context, program, input []byte, cfg Config) (Statement, error) {
	var result Statement
	if err := validateInputs(program, input, cfg); err != nil {
		return result, err
	}
	cpu := riscv.NewRVCPU(cfg.MaxCycles)
	cpu.Memory.SetMaxPages(int(cfg.MemoryPages))
	if err := cpu.LoadProgram(program, cfg.ProgramBase, cfg.EntryPC); err != nil {
		return result, err
	}
	cpu.Regs[2] = cfg.StackBase
	cpu.InputBuf = append([]byte(nil), input...)
	if cfg.LoadInput {
		if len(input) > 0 {
			if err := cpu.Memory.LoadSegment(cfg.InputBase, input); err != nil {
				return result, err
			}
		}
		cpu.Regs[10] = cfg.InputBase
		cpu.Regs[11] = uint32(len(input))
	}
	for !cpu.Halted {
		if cpu.GasUsed&255 == 0 {
			if err := ctx.Err(); err != nil {
				return result, err
			}
		}
		if err := cpu.Step(); err != nil {
			return result, err
		}
		if len(cpu.OutputBuf) > MaxOutput {
			return result, ErrBounds
		}
	}
	if cpu.ExitCode != 0 {
		return result, fmt.Errorf("canonical guest exit code %d", cpu.ExitCode)
	}
	state := cpu.CanonicalStateDigest()
	return Statement{Version: 1, ProgramHash: hash(program), InputHash: hash(input), Output: hex.EncodeToString(cpu.OutputBuf), OutputHash: hash(cpu.OutputBuf), Cycles: cpu.GasUsed, ExitCode: cpu.ExitCode, FinalPC: cpu.PC, InputConsumed: uint64(cpu.InputPos()), StateHash: hex.EncodeToString(state[:]), Config: cfg}, nil
}
