package vm

import (
	"errors"
	"math"
	"math/big"

	"github.com/eth2030/eth2030/crypto"
)

// EIP7932DraftAlgorithm defines an algorithm for the pinned EIP-7932 draft.
// SignatureSize includes the algorithm prefix. A deployment must select a
// fixed registry for its fork and specify each additional algorithm separately.
type EIP7932DraftAlgorithm interface {
	AlgorithmID() byte
	SignatureSize() int
	GasCost(signingData []byte) uint64
	Validate(signature []byte) error
	Recover(signature, signingData []byte) ([]byte, error)
}

const EIP7932DraftBaseGas uint64 = 3000

var ErrEIP7932DraftRegistry = errors.New("eip7932 draft: invalid or duplicate algorithm")

// EIP7932DraftSigRecover implements the pinned draft's SIGRECOVER but is not
// registered at an EVM address. The pinned draft uses selector 0xff and
// address 0x12; both differ from other draft versions or reserved uses.
type EIP7932DraftSigRecover struct {
	algorithms map[byte]eip7932DraftEntry
}
type eip7932DraftEntry struct {
	algorithm EIP7932DraftAlgorithm
	size      int
}

// NewEIP7932DraftSigRecover fixes the registry at construction time. The
// pinned draft's secp256k1 algorithm is always registered; 0xfe is reserved.
func NewEIP7932DraftSigRecover(additional ...EIP7932DraftAlgorithm) (*EIP7932DraftSigRecover, error) {
	p := &EIP7932DraftSigRecover{algorithms: map[byte]eip7932DraftEntry{0xff: {algorithm: eip7932DraftSecp{}, size: 66}}}
	for _, algorithm := range additional {
		if algorithm == nil {
			return nil, ErrEIP7932DraftRegistry
		}
		id, size := algorithm.AlgorithmID(), algorithm.SignatureSize()
		if id == 0xfe || size < 1 {
			return nil, ErrEIP7932DraftRegistry
		}
		if _, exists := p.algorithms[id]; exists {
			return nil, ErrEIP7932DraftRegistry
		}
		p.algorithms[id] = eip7932DraftEntry{algorithm: algorithm, size: size}
	}
	return p, nil
}

func (p *EIP7932DraftSigRecover) RequiredGas(input []byte) uint64 {
	if len(input) == 0 {
		return EIP7932DraftBaseGas
	}
	entry, exists := p.algorithms[input[0]]
	if !exists || len(input) <= entry.size {
		return EIP7932DraftBaseGas
	}
	extra := entry.algorithm.GasCost(input[entry.size:])
	if extra > math.MaxUint64-EIP7932DraftBaseGas {
		return math.MaxUint64
	}
	return EIP7932DraftBaseGas + extra
}

func (p *EIP7932DraftSigRecover) Run(input []byte) ([]byte, error) {
	out := make([]byte, 32)
	if len(input) == 0 {
		return out, nil
	}
	entry, exists := p.algorithms[input[0]]
	if !exists || len(input) <= entry.size {
		return out, nil
	}
	sig, data := input[:entry.size], input[entry.size:]
	if err := entry.algorithm.Validate(sig); err != nil {
		return out, nil
	}
	key, err := entry.algorithm.Recover(sig, data)
	if err != nil || len(key) == 0 {
		return out, nil
	}
	var addressHash []byte
	if input[0] == 0xff {
		if len(key) != 65 || key[0] != 4 {
			return out, nil
		}
		addressHash = crypto.Keccak256(key[1:])
	} else {
		addressHash = crypto.Keccak256([]byte{input[0]}, key)
	}
	copy(out[12:], addressHash[12:])
	return out, nil
}

type eip7932DraftSecp struct{}

func (eip7932DraftSecp) AlgorithmID() byte  { return 0xff }
func (eip7932DraftSecp) SignatureSize() int { return 66 }
func (eip7932DraftSecp) GasCost(data []byte) uint64 {
	if len(data) == 32 {
		return 0
	}
	return 30 + 6*((uint64(len(data))+31)/32)
}
func (eip7932DraftSecp) Validate(sig []byte) error {
	if len(sig) != 66 || sig[0] != 0xff || !crypto.ValidateSignatureValues(sig[65], new(big.Int).SetBytes(sig[1:33]), new(big.Int).SetBytes(sig[33:65]), true) {
		return errors.New("eip7932 draft: invalid secp256k1 signature")
	}
	return nil
}
func (eip7932DraftSecp) Recover(sig, data []byte) ([]byte, error) {
	digest := data
	if len(data) != 32 {
		digest = crypto.Keccak256(data)
	}
	return crypto.Ecrecover(digest, sig[1:])
}
