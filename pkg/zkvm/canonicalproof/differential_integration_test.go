//go:build canonicalproof_integration

package canonicalproof

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/eth2030/eth2030/zkvm/riscv"
)

func TestIndependentRV32IMExecutionVectors(t *testing.T) {
	prover := Prover{}
	if !prover.Available() {
		t.Fatal("real verifier/interpreter binary required")
	}
	cfg := Config{MaxCycles: 100, MemoryPages: 4, StackBase: 0x80000000}
	finish := []uint32{riscv.EncodeRType(0x33, 10, 0, 4, 0, 0), riscv.EncodeIType(0x13, 17, 0, 0, 1), 0x73, riscv.EncodeIType(0x13, 10, 0, 0, 0), riscv.EncodeIType(0x13, 17, 0, 0, 0), 0x73}
	check := func(t *testing.T, words []uint32, input []byte) {
		t.Helper()
		var program []byte
		for _, word := range words {
			program = binary.LittleEndian.AppendUint32(program, word)
		}
		goResult, err := ReferenceExecution(context.Background(), program, input, cfg)
		if err != nil {
			t.Fatal(err)
		}
		rustResult, err := prover.ExecuteReference(context.Background(), program, input, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if goResult != rustResult {
			t.Fatalf("independent machine state differs\nGo: %+v\nRust: %+v", goResult, rustResult)
		}
	}
	for _, a := range []int32{-1, -2048, 0, 17} {
		for _, b := range []int32{-1, 0, 3} {
			for _, encoding := range [][2]uint32{{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}, {0x20, 0}, {0x20, 5}, {1, 0}, {1, 1}, {1, 2}, {1, 3}, {1, 4}, {1, 5}, {1, 6}, {1, 7}} {
				t.Run(fmt.Sprintf("register/%d/%d/%x/%d", a, b, encoding[0], encoding[1]), func(t *testing.T) {
					words := []uint32{riscv.EncodeIType(0x13, 1, 0, 0, a), riscv.EncodeIType(0x13, 3, 0, 0, b), riscv.EncodeRType(0x33, 4, encoding[1], 1, 3, encoding[0])}
					check(t, append(words, finish...), nil)
				})
			}
		}
	}
	for _, f3 := range []uint32{0, 2, 3, 4, 6, 7} {
		t.Run(fmt.Sprintf("immediate/%d", f3), func(t *testing.T) {
			check(t, append([]uint32{riscv.EncodeIType(0x13, 1, 0, 0, -17), riscv.EncodeIType(0x13, 4, f3, 1, -9)}, finish...), nil)
		})
	}
	for _, imm := range []int32{0, 1, 31, 0x400, 0x41f} {
		f3 := uint32(5)
		if imm < 32 {
			f3 = 1
		}
		t.Run(fmt.Sprintf("shift/%d", imm), func(t *testing.T) {
			check(t, append([]uint32{riscv.EncodeIType(0x13, 1, 0, 0, -17), riscv.EncodeIType(0x13, 4, f3, 1, imm)}, finish...), nil)
		})
	}
	for _, f3 := range []uint32{0, 1, 4, 5, 6, 7} {
		for _, other := range []int32{-2, 2} {
			t.Run(fmt.Sprintf("branch/%d/%d", f3, other), func(t *testing.T) {
				words := []uint32{riscv.EncodeIType(0x13, 1, 0, 0, -1), riscv.EncodeIType(0x13, 3, 0, 0, other), riscv.EncodeBType(0x63, f3, 1, 3, 8), riscv.EncodeIType(0x13, 4, 0, 0, 42)}
				check(t, append(words, finish...), nil)
			})
		}
	}
	for _, f3 := range []uint32{0, 1, 2, 4, 5} {
		t.Run(fmt.Sprintf("memory/%d", f3), func(t *testing.T) {
			words := []uint32{riscv.EncodeUType(0x37, 1, 0x1000), riscv.EncodeIType(0x13, 1, 0, 1, -1), riscv.EncodeIType(0x13, 3, 0, 0, -17), riscv.EncodeSType(0x23, 2, 1, 3, 0), riscv.EncodeIType(0x03, 4, f3, 1, 0)}
			check(t, append(words, finish...), nil)
		})
	}
	t.Run("jumps-upper-fence", func(t *testing.T) {
		words := []uint32{riscv.EncodeUType(0x37, 1, 0x80000000), riscv.EncodeUType(0x17, 3, 0x1000), riscv.EncodeJType(0x6f, 4, 8), 0, riscv.EncodeIType(0x13, 5, 0, 0, 28), riscv.EncodeIType(0x67, 5, 0, 5, 0), 0, 0x0f}
		check(t, append(words, finish...), nil)
	})
	t.Run("io-and-eof", func(t *testing.T) {
		words := []uint32{riscv.EncodeIType(0x13, 17, 0, 0, 2), 0x73, 0x73, riscv.EncodeRType(0x33, 4, 0, 10, 0, 0)}
		check(t, append(words, finish...), []byte{42})
	})
}
