package vm

import (
	"crypto/sha3"
	"crypto/subtle"
	"encoding/binary"
	"io"
	"math/bits"

	"github.com/eth2030/eth2030/crypto"
)

const (
	mldsaQ      = 8380417
	mldsaDegree = 256
	mldsaGamma1 = 1 << 17
	mldsaAlpha  = (mldsaQ - 1) / 44
	mldsaOmega  = 80
)

type mldsaPoly [mldsaDegree]int64

// EIP8051DraftVerifierForVariant selects the expanded-key verifier for the
// pinned EIP-8051 byte layout. It returns nil for an unsupported selector.
func EIP8051DraftVerifierForVariant(variant byte) EIP8051DraftVerifier {
	switch variant {
	case EIP8051DraftNIST, EIP8051DraftETH:
		return func(message, signature, key []byte) bool {
			return verifyEIP8051Expanded(message, signature, key, variant == EIP8051DraftETH)
		}
	default:
		return nil
	}
}

// NewEIP8051ExpandedPrecompile constructs the draft contract without assigning
// it an address. Callers must select the variant explicitly.
func NewEIP8051ExpandedPrecompile(variant byte) (*EIP8051DraftPrecompile, error) {
	verify := EIP8051DraftVerifierForVariant(variant)
	if verify == nil {
		return nil, ErrEIP8051DraftInvalidAlgorithm
	}
	return NewEIP8051DraftPrecompile(verify), nil
}

// NewEIP8051ExpandedAlgorithm binds a draft registry entry to the
// corresponding expanded-key verifier and the caller's key lookup.
func NewEIP8051ExpandedAlgorithm(variant byte, lookup EIP8051DraftPublicKeyLookup) (EIP7932DraftAlgorithm, error) {
	return NewEIP8051DraftAlgorithm(variant, lookup, EIP8051DraftVerifierForVariant(variant))
}

func verifyEIP8051Expanded(message, signature, key []byte, eth bool) bool {
	if len(message) != EIP8051DraftMessageSize || len(signature) != EIP8051DraftSignatureSize || len(key) != EIP8051DraftPublicKeySize {
		return false
	}
	var z [4]mldsaPoly
	for i := range z {
		packed := signature[32+576*i : 32+576*(i+1)]
		for j := range z[i] {
			offset, shift := 18*j/8, uint(18*j%8)
			word := uint32(packed[offset]) | uint32(packed[offset+1])<<8 | uint32(packed[offset+2])<<16
			coefficient := mldsaGamma1 - int64((word>>shift)&0x3ffff)
			if coefficient <= -(mldsaGamma1-78) || coefficient >= mldsaGamma1-78 {
				return false
			}
			z[i][j] = mldsaMod(coefficient)
		}
	}
	hints, ok := mldsaHints(signature[2336:])
	if !ok {
		return false
	}

	// FIPS 204 SampleInBall, with the XOF selected by the draft variant.
	xof := mldsaXOF(eth, signature[:32])
	var signBytes [8]byte
	_, _ = io.ReadFull(xof, signBytes[:])
	signs := binary.LittleEndian.Uint64(signBytes[:])
	var challenge mldsaPoly
	for i := mldsaDegree - 39; i < mldsaDegree; i++ {
		var index [1]byte
		for {
			_, _ = io.ReadFull(xof, index[:])
			if int(index[0]) <= i {
				break
			}
		}
		j := int(index[0])
		challenge[i] = challenge[j]
		challenge[j] = mldsaMod(1 - 2*int64(signs&1))
		signs >>= 1
	}
	mldsaNTT(&challenge, false)
	for i := range z {
		mldsaNTT(&z[i], false)
	}

	// Draft wire order: row-major A_hat, 32-byte tr, then NTT(t1).
	// The ETH variant supplies NTT(2^13 * t1) in the key instead.
	const trOffset = 4 * 4 * mldsaDegree * 4
	const t1Offset = trOffset + 32
	var packedW [768]byte
	for row := range 4 {
		var w mldsaPoly
		for i := range w {
			t := int64(binary.BigEndian.Uint32(key[t1Offset+4*(row*mldsaDegree+i):]))
			if t >= mldsaQ {
				return false
			}
			if !eth {
				t = mldsaMod(t << 13)
			}
			sum := -challenge[i] * t
			for col := range 4 {
				a := int64(binary.BigEndian.Uint32(key[4*((row*4+col)*mldsaDegree+i):]))
				if a >= mldsaQ {
					return false
				}
				sum += a * z[col][i]
			}
			w[i] = mldsaMod(sum)
		}
		mldsaNTT(&w, true)
		for i := 0; i < mldsaDegree; i += 4 {
			var word uint32
			for j := range 4 {
				value := w[i+j]
				high := (value + mldsaAlpha/2 - 1) / mldsaAlpha
				low := value - high*mldsaAlpha
				if high == 44 {
					high, low = 0, value-mldsaQ
				}
				if hints[row][i+j] {
					if low > 0 {
						high = (high + 1) % 44
					} else {
						high = (high + 43) % 44
					}
				}
				word |= uint32(high) << (6 * j)
			}
			offset := row*192 + i/4*3
			packedW[offset], packedW[offset+1], packedW[offset+2] = byte(word), byte(word>>8), byte(word>>16)
		}
	}

	var mu [64]byte
	_, _ = io.ReadFull(mldsaXOF(eth, key[trOffset:t1Offset], message), mu[:])
	var expected [32]byte
	_, _ = io.ReadFull(mldsaXOF(eth, mu[:], packedW[:]), expected[:])
	return subtle.ConstantTimeCompare(signature[:32], expected[:]) == 1
}

func mldsaHints(encoded []byte) ([4][mldsaDegree]bool, bool) {
	var hints [4][mldsaDegree]bool
	start := 0
	for row := range hints {
		end := int(encoded[mldsaOmega+row])
		if end < start || end > mldsaOmega {
			return hints, false
		}
		for i := start; i < end; i++ {
			if i > start && encoded[i-1] >= encoded[i] {
				return hints, false
			}
			hints[row][encoded[i]] = true
		}
		start = end
	}
	for _, padding := range encoded[start:mldsaOmega] {
		if padding != 0 {
			return hints, false
		}
	}
	return hints, true
}

func mldsaMod(x int64) int64 {
	x %= mldsaQ
	if x < 0 {
		x += mldsaQ
	}
	return x
}

var mldsaRoots, mldsaInverseRoots = func() ([256]int64, [256]int64) {
	var forward, inverse [256]int64
	power := int64(1)
	for i := range 256 {
		index := bits.Reverse8(uint8(i))
		forward[index] = power
		power = mldsaMod(power * 1753)
	}
	// 1753^256 = -1 mod q, so its negative powers follow from the forward table.
	for i := 1; i < 256; i++ {
		inverse[bits.Reverse8(uint8(i))] = mldsaQ - forward[bits.Reverse8(uint8(256-i))]
	}
	return forward, inverse
}()

// FIPS 204 NTT and inverse NTT over the negacyclic ring.
func mldsaNTT(poly *mldsaPoly, inverse bool) {
	if !inverse {
		for width := 128; width >= 1; width /= 2 {
			base := 128 / width
			for group := range base {
				root := mldsaRoots[base+group]
				start := 2 * width * group
				for i := start; i < start+width; i++ {
					u, v := poly[i], mldsaMod(root*poly[i+width])
					poly[i], poly[i+width] = mldsaMod(u+v), mldsaMod(u-v)
				}
			}
		}
		return
	}
	for width := 1; width <= 128; width *= 2 {
		base := 128 / width
		for group := range base {
			root := mldsaInverseRoots[base+group]
			start := 2 * width * group
			for i := start; i < start+width; i++ {
				u, v := poly[i], poly[i+width]
				poly[i], poly[i+width] = mldsaMod(u+v), mldsaMod(root*(u-v))
			}
		}
	}
	const inverseDegree = mldsaQ - (mldsaQ-1)/mldsaDegree
	for i := range poly {
		poly[i] = mldsaMod(poly[i] * inverseDegree)
	}
}

func mldsaXOF(eth bool, parts ...[]byte) io.Reader {
	if eth {
		return &mldsaKeccakPRNG{seed: crypto.Keccak256Hash(parts...)}
	}
	xof := sha3.NewSHAKE256()
	for _, part := range parts {
		_, _ = xof.Write(part)
	}
	return xof
}

type mldsaKeccakPRNG struct {
	seed    [32]byte
	counter uint64
	pool    []byte
}

func (p *mldsaKeccakPRNG) Read(out []byte) (int, error) {
	for written := 0; written < len(out); {
		if len(p.pool) == 0 {
			var counter [8]byte
			binary.BigEndian.PutUint64(counter[:], p.counter)
			p.pool = crypto.Keccak256(p.seed[:], counter[:])
			p.counter++
		}
		n := copy(out[written:], p.pool)
		written += n
		p.pool = p.pool[n:]
	}
	return len(out), nil
}
