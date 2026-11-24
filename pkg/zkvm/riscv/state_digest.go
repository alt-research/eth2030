package riscv

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
)

// CanonicalStateDigest binds the complete terminal machine state of the v1
// canonical guest ABI. Call only while the CPU is not being mutated.
func (cpu *RVCPU) CanonicalStateDigest() [32]byte {
	h := sha256.New()
	_, _ = h.Write([]byte("ETH2030_CANONICAL_RV32_STATE_V1"))
	var buf [8]byte
	write32 := func(v uint32) { binary.LittleEndian.PutUint32(buf[:4], v); _, _ = h.Write(buf[:4]) }
	write64 := func(v uint64) { binary.LittleEndian.PutUint64(buf[:], v); _, _ = h.Write(buf[:]) }
	write32(cpu.PC)
	write64(cpu.GasUsed)
	write64(uint64(cpu.inputPos))
	for _, v := range cpu.Regs {
		write32(v)
	}
	pages := make([]uint32, 0, len(cpu.Memory.pages))
	for index := range cpu.Memory.pages {
		pages = append(pages, index)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i] < pages[j] })
	write32(uint32(len(pages)))
	for _, index := range pages {
		write32(index)
		_, _ = h.Write(cpu.Memory.pages[index])
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest
}
