package vm

import (
	"errors"
	"math"
	"math/big"
)

// osakaModExp applies EIP-7883 pricing and EIP-7823's independent execution
// limit. Gas calculation remains defined for oversized headers; execution
// rejects them before any input-sized allocation.
type osakaModExp struct{}

func modexpLengths(input []byte) [3]*big.Int {
	var lengths [3]*big.Int
	for i := range lengths {
		lengths[i] = new(big.Int).SetBytes(getDataSlice(input, uint64(i*32), 32))
	}
	return lengths
}
func (*osakaModExp) RequiredGas(input []byte) uint64 {
	lengths := modexpLengths(input)
	base, exponent, modulus := lengths[0], lengths[1], lengths[2]
	maxLen := new(big.Int).Set(base)
	if modulus.Cmp(maxLen) > 0 {
		maxLen.Set(modulus)
	}
	complexity := big.NewInt(16)
	if maxLen.Cmp(big.NewInt(32)) > 0 {
		words := new(big.Int).Add(maxLen, big.NewInt(7))
		words.Rsh(words, 3)
		complexity.Mul(words, words)
		complexity.Lsh(complexity, 1)
	}
	headBits := 0
	if base.IsUint64() && base.Uint64() <= uint64(len(input)) {
		size := uint64(32)
		if exponent.Cmp(big.NewInt(32)) < 0 {
			size = exponent.Uint64()
		}
		// The header is excluded before applying the base offset.
		data := input[min(len(input), 96):]
		headBits = new(big.Int).SetBytes(getDataSlice(data, base.Uint64(), size)).BitLen()
	}
	iterations := new(big.Int)
	if exponent.Cmp(big.NewInt(32)) > 0 {
		iterations.Sub(exponent, big.NewInt(32))
		iterations.Lsh(iterations, 4)
	}
	if headBits > 0 {
		iterations.Add(iterations, big.NewInt(int64(headBits-1)))
	}
	if iterations.Sign() == 0 {
		iterations.SetUint64(1)
	}
	total := new(big.Int).Mul(complexity, iterations)
	if !total.IsUint64() {
		return math.MaxUint64
	}
	return max(uint64(500), total.Uint64())
}
func (*osakaModExp) Run(input []byte) ([]byte, error) {
	for _, length := range modexpLengths(input) {
		if !length.IsUint64() || length.Uint64() > 1024 {
			return nil, errors.New("modexp input length exceeds EIP-7823 limit")
		}
	}
	return (&bigModExp{}).Run(input)
}
