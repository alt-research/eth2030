package vm

import (
	"bytes"
	"errors"

	"github.com/eth2030/eth2030/crypto"
)

// The pinned EIP-8051 draft describes these input sizes for both variants.
const (
	EIP8051DraftGas           uint64 = 4500
	EIP8051DraftMessageSize          = 32
	EIP8051DraftSignatureSize        = 2420
	EIP8051DraftPublicKeySize        = 20512
	EIP8051DraftInputSize            = EIP8051DraftMessageSize + EIP8051DraftSignatureSize + EIP8051DraftPublicKeySize
)

var ErrEIP8051DraftNoVerifier = errors.New("eip8051 draft: expanded-key verifier is not configured")

// EIP8051DraftVerifier must verify the chosen draft variant with its complete
// expanded-key encoding, including field, hint and norm checks. It is not the
// compact FIPS-204 verifier exposed by ordinary ML-DSA libraries.
type EIP8051DraftVerifier func(message, signature, expandedPublicKey []byte) bool

// EIP8051DraftPrecompile provides the pinned draft's gas and input envelope.
// No expanded-key verifier or EVM address is configured by this package.
type EIP8051DraftPrecompile struct{ verify EIP8051DraftVerifier }

func NewEIP8051DraftPrecompile(verify EIP8051DraftVerifier) *EIP8051DraftPrecompile {
	return &EIP8051DraftPrecompile{verify: verify}
}

func (*EIP8051DraftPrecompile) RequiredGas([]byte) uint64 { return EIP8051DraftGas }

func (p *EIP8051DraftPrecompile) Run(input []byte) ([]byte, error) {
	output := make([]byte, 32)
	if len(input) != EIP8051DraftInputSize {
		return output, nil
	}
	if p == nil || p.verify == nil {
		return nil, ErrEIP8051DraftNoVerifier
	}
	message := input[:EIP8051DraftMessageSize]
	signature := input[EIP8051DraftMessageSize : EIP8051DraftMessageSize+EIP8051DraftSignatureSize]
	publicKey := input[EIP8051DraftMessageSize+EIP8051DraftSignatureSize:]
	if p.verify(message, signature, publicKey) {
		output[31] = 1
	}
	return output, nil
}

const (
	EIP8051DraftNIST              byte = 0xd1
	EIP8051DraftETH               byte = 0xd2
	EIP8051DraftSignatureInfoSize      = 1 + EIP8051DraftSignatureSize + 20
	// The draft recommends approximately 3000 pending benchmarking.
	EIP8051DraftRegistryPenalty uint64 = 3000
)

var ErrEIP8051DraftInvalidAlgorithm = errors.New("eip8051 draft: invalid signature algorithm configuration")

// EIP8051DraftPublicKeyLookup resolves the 20-byte identifier in signature_info.
// The registry adapter independently checks that it hashes to this identifier.
type EIP8051DraftPublicKeyLookup func([20]byte) ([]byte, error)

type eip8051DraftAlgorithm struct {
	id     byte
	lookup EIP8051DraftPublicKeyLookup
	verify EIP8051DraftVerifier
}

// NewEIP8051DraftAlgorithm creates an optional EIP-7932 registry entry for
// one pinned EIP-8051 draft variant. It requires an expanded-key verifier and
// state-backed key lookup supplied by the caller; neither is installed by default.
func NewEIP8051DraftAlgorithm(id byte, lookup EIP8051DraftPublicKeyLookup, verify EIP8051DraftVerifier) (EIP7932DraftAlgorithm, error) {
	if (id != EIP8051DraftNIST && id != EIP8051DraftETH) || lookup == nil || verify == nil {
		return nil, ErrEIP8051DraftInvalidAlgorithm
	}
	return &eip8051DraftAlgorithm{id: id, lookup: lookup, verify: verify}, nil
}

func (a *eip8051DraftAlgorithm) AlgorithmID() byte   { return a.id }
func (*eip8051DraftAlgorithm) SignatureSize() int    { return EIP8051DraftSignatureInfoSize }
func (*eip8051DraftAlgorithm) GasCost([]byte) uint64 { return EIP8051DraftRegistryPenalty }
func (a *eip8051DraftAlgorithm) Validate(signature []byte) error {
	if len(signature) != EIP8051DraftSignatureInfoSize || signature[0] != a.id {
		return ErrEIP8051DraftInvalidAlgorithm
	}
	return nil
}
func (a *eip8051DraftAlgorithm) Recover(signature, signingData []byte) ([]byte, error) {
	if err := a.Validate(signature); err != nil || len(signingData) != 32 {
		return nil, ErrEIP8051DraftInvalidAlgorithm
	}
	var id [20]byte
	copy(id[:], signature[1+EIP8051DraftSignatureSize:])
	key, err := a.lookup(id)
	if err != nil || len(key) != EIP8051DraftPublicKeySize {
		return nil, ErrEIP8051DraftInvalidAlgorithm
	}
	digest := crypto.Keccak256(key)
	if !bytes.Equal(digest[12:], id[:]) || !a.verify(signingData, signature[1:1+EIP8051DraftSignatureSize], key) {
		return nil, ErrEIP8051DraftInvalidAlgorithm
	}
	return key, nil
}
