package vm

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/crypto"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// These cases reproduce refs/EIPs/assets/eip-7932/precompile_test_cases.py.
func TestEIP7932DraftReferenceCases(t *testing.T) {
	p, err := NewEIP7932DraftSigRecover()
	if err != nil {
		t.Fatal(err)
	}
	key, err := gethcrypto.HexToECDSA("1f7627096fa44f0b850f5d9a859d271723ee856e526b947d0d4b011168bdcac1")
	if err != nil {
		t.Fatal(err)
	}
	address := gethcrypto.PubkeyToAddress(key.PublicKey)
	if address.Hex() != "0xd3eF791e8a9c9BD26787D262e66e673FE8E7262A" {
		t.Fatalf("unexpected reference key: %s", address.Hex())
	}
	zeroSig, err := gethcrypto.Sign(make([]byte, 32), key)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte{0xde, 0xad, 0xbe, 0xef}
	messageSig, err := gethcrypto.Sign(crypto.Keccak256(msg), key)
	if err != nil {
		t.Fatal(err)
	}
	validInput := func(sig, data []byte) []byte { out := append([]byte{0xff}, sig...); return append(out, data...) }
	success := make([]byte, 32)
	copy(success[12:], address[:])
	zero := make([]byte, 32)
	cases := []struct {
		name  string
		input []byte
		gas   uint64
		want  []byte
	}{
		{"empty", nil, 3000, zero},
		{"reserved", []byte{0xfe}, 3000, zero},
		{"unknown at secp size", append(append([]byte{0xfe}, bytes.Repeat([]byte{1}, 65)...), make([]byte, 32)...), 3000, zero},
		{"unknown above secp size", append(append([]byte{0xfe}, bytes.Repeat([]byte{1}, 66)...), make([]byte, 32)...), 3000, zero},
		{"selector only", []byte{0xff}, 3000, zero},
		{"short no message", append([]byte{0xff}, make([]byte, 65)...), 3000, zero},
		{"reference short signature", append(append([]byte{0xff}, bytes.Repeat([]byte{0xfe}, 64)...), make([]byte, 32)...), 3036, zero},
		{"invalid signature 30 byte message", validInput(bytes.Repeat([]byte{0xfe}, 65), make([]byte, 30)), 3036, zero},
		{"reference extra signature bytes", append(append([]byte{0xff}, bytes.Repeat([]byte{0xfe}, 67)...), make([]byte, 32)...), 3042, zero},
		{"invalid signature 32 byte message", validInput(bytes.Repeat([]byte{0xfe}, 65), make([]byte, 32)), 3000, zero},
		{"valid digest", validInput(zeroSig, make([]byte, 32)), 3000, success},
		{"valid raw message", validInput(messageSig, msg), 3036, success},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if gas := p.RequiredGas(tc.input); gas != tc.gas {
				t.Fatalf("gas %d want %d", gas, tc.gas)
			}
			got, left, err := runPrecompile(p, tc.input, tc.gas+1)
			if err != nil || left != 1 || !bytes.Equal(got, tc.want) {
				t.Fatalf("result %x, gas left %d, err %v", got, left, err)
			}
			if _, _, err := runPrecompile(p, tc.input, tc.gas-1); !errors.Is(err, ErrOutOfGas) {
				t.Fatalf("insufficient gas: %v", err)
			}
		})
	}
	// Mathematically equivalent high-s signatures are forbidden by the draft.
	high := append([]byte(nil), zeroSig...)
	s := new(big.Int).SetBytes(high[32:64])
	s.Sub(gethcrypto.S256().Params().N, s)
	s.FillBytes(high[32:64])
	high[64] ^= 1
	got, err := p.Run(validInput(high, make([]byte, 32)))
	if err != nil || !bytes.Equal(got, zero) {
		t.Fatalf("accepted high-s signature: %x %v", got, err)
	}
}

type testEIP7932Algorithm struct {
	id   byte
	size int
	gas  uint64
}

func (a testEIP7932Algorithm) AlgorithmID() byte     { return a.id }
func (a testEIP7932Algorithm) SignatureSize() int    { return a.size }
func (a testEIP7932Algorithm) GasCost([]byte) uint64 { return a.gas }
func (a testEIP7932Algorithm) Validate(sig []byte) error {
	if sig[1] != 7 {
		return errors.New("invalid")
	}
	return nil
}
func (a testEIP7932Algorithm) Recover(_ []byte, _ []byte) ([]byte, error) {
	return []byte("public key"), nil
}

func TestEIP7932DraftRegistry(t *testing.T) {
	for _, algorithms := range [][]EIP7932DraftAlgorithm{{testEIP7932Algorithm{id: 0xfe, size: 2}}, {testEIP7932Algorithm{id: 0xff, size: 2}}, {testEIP7932Algorithm{id: 1, size: 0}}, {testEIP7932Algorithm{id: 1, size: 2}, testEIP7932Algorithm{id: 1, size: 2}}} {
		if _, err := NewEIP7932DraftSigRecover(algorithms...); !errors.Is(err, ErrEIP7932DraftRegistry) {
			t.Fatalf("accepted invalid registry: %v", err)
		}
	}
	p, err := NewEIP7932DraftSigRecover(testEIP7932Algorithm{id: 1, size: 2, gas: 43}, testEIP7932Algorithm{id: 2, size: 2, gas: math.MaxUint64})
	if err != nil {
		t.Fatal(err)
	}
	input := []byte{1, 7, 9}
	if got := p.RequiredGas(input); got != 3043 {
		t.Fatalf("gas %d", got)
	}
	out, err := p.Run(input)
	hash := crypto.Keccak256([]byte{1}, []byte("public key"))
	if err != nil || !bytes.Equal(out[12:], hash[12:]) || !bytes.Equal(out[:12], make([]byte, 12)) {
		t.Fatalf("address %x, err %v", out, err)
	}
	out, err = p.Run([]byte{1, 8, 9})
	if err != nil || !bytes.Equal(out, make([]byte, 32)) {
		t.Fatal("accepted invalid signature")
	}
	if p.RequiredGas([]byte{2, 7, 9}) != math.MaxUint64 {
		t.Fatal("gas overflow did not saturate")
	}
}
