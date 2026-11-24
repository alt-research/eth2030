package riscv

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestCanonicalABIRejectsReservedInstructionForms(t *testing.T) {
	for name, instruction := range map[string]uint32{
		"jalr funct3":              EncodeIType(0x67, 1, 1, 0, 0),
		"slli high bits":           EncodeIType(0x13, 1, 1, 0, 0x400),
		"srli high bits":           EncodeIType(0x13, 1, 5, 0, 0x200),
		"register funct7":          EncodeRType(0x33, 1, 4, 0, 0, 0x20),
		"reserved system":          0x00200073,
		"reserved ecall registers": 0x000000f3,
	} {
		t.Run(name, func(t *testing.T) {
			cpu := NewRVCPU(4)
			var program [4]byte
			binary.LittleEndian.PutUint32(program[:], instruction)
			if err := cpu.LoadProgram(program[:], 0, 0); err != nil {
				t.Fatal(err)
			}
			if err := cpu.Step(); !errors.Is(err, ErrRVInvalidInstruction) {
				t.Fatal("reserved instruction was accepted", err)
			}
		})
	}
}

func TestCanonicalFenceAndMisalignedControlTransfer(t *testing.T) {
	cpu := NewRVCPU(4)
	var program [4]byte
	binary.LittleEndian.PutUint32(program[:], 0x0f)
	if err := cpu.LoadProgram(program[:], 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := cpu.Step(); err != nil || cpu.PC != 4 || cpu.GasUsed != 1 {
		t.Fatal("FENCE is not a metered no-op", err)
	}
	binary.LittleEndian.PutUint32(program[:], EncodeJType(0x6f, 1, 2))
	cpu = NewRVCPU(4)
	if err := cpu.LoadProgram(program[:], 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := cpu.Step(); !errors.Is(err, ErrRVMemUnaligned) || cpu.Regs[1] != 0 || cpu.GasUsed != 0 {
		t.Fatal("misaligned jump changed state", err)
	}
}

func TestCanonicalSignedDivisionOverflow(t *testing.T) {
	for _, f3 := range []uint32{4, 6} {
		cpu := NewRVCPU(4)
		cpu.Regs[1] = 0x80000000
		cpu.Regs[3] = 0xffffffff
		var program [4]byte
		binary.LittleEndian.PutUint32(program[:], EncodeRType(0x33, 4, f3, 1, 3, 1))
		if err := cpu.LoadProgram(program[:], 0, 0); err != nil {
			t.Fatal(err)
		}
		if err := cpu.Step(); err != nil {
			t.Fatal(err)
		}
		want := uint32(0)
		if f3 == 4 {
			want = 0x80000000
		}
		if cpu.Regs[4] != want {
			t.Fatalf("overflow result %x, want %x", cpu.Regs[4], want)
		}
	}
}
