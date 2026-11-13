package vm

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/crypto"
	"testing"
)

// This tests the draft envelope and gas only. The injected callback is a stub,
// not an ML-DSA or ML-DSA-ETH cryptographic verifier.
func TestEIP8051DraftGasAndEnvelope(t *testing.T) {
	calls := 0
	verifier := func(message, signature, key []byte) bool {
		calls++
		if len(message) != 32 || len(signature) != 2420 || len(key) != 20512 {
			t.Fatal("incorrect argument boundaries")
		}
		return message[0] == 1 && signature[0] == 2 && key[0] == 3
	}
	p := NewEIP8051DraftPrecompile(verifier)
	input := make([]byte, 22964)
	input[0], input[32], input[32+2420] = 1, 2, 3
	for _, data := range [][]byte{nil, input[:len(input)-1], input, append(bytes.Clone(input), 0)} {
		if gas := p.RequiredGas(data); gas != 4500 {
			t.Fatalf("gas %d, want 4500", gas)
		}
		if _, _, err := runPrecompile(p, data, 4499); !errors.Is(err, ErrOutOfGas) {
			t.Fatalf("insufficient gas: %v", err)
		}
	}
	for _, data := range [][]byte{nil, input[:len(input)-1], append(bytes.Clone(input), 0)} {
		out, left, err := runPrecompile(p, data, 4500)
		if err != nil || left != 0 || !bytes.Equal(out, make([]byte, 32)) || calls != 0 {
			t.Fatalf("invalid length: %x %d %v calls=%d", out, left, err, calls)
		}
	}
	out, left, err := runPrecompile(p, input, 4501)
	want := make([]byte, 32)
	want[31] = 1
	if err != nil || left != 1 || !bytes.Equal(out, want) || calls != 1 {
		t.Fatalf("accepted: %x %d %v calls=%d", out, left, err, calls)
	}
	input[32] = 4
	out, _, err = runPrecompile(p, input, 4500)
	if err != nil || !bytes.Equal(out, make([]byte, 32)) || calls != 2 {
		t.Fatalf("rejected: %x %v calls=%d", out, err, calls)
	}
	unconfigured := NewEIP8051DraftPrecompile(nil)
	if _, err := unconfigured.Run(input); !errors.Is(err, ErrEIP8051DraftNoVerifier) {
		t.Fatalf("unconfigured provider: %v", err)
	}
}

// The callbacks below only test registry plumbing; they do not implement the
// draft's ML-DSA algorithms or establish cryptographic conformance.
func TestEIP8051DraftRegistryAdapter(t *testing.T) {
	key := make([]byte, EIP8051DraftPublicKeySize)
	key[0] = 9
	digest := crypto.Keccak256(key)
	var identifier [20]byte
	copy(identifier[:], digest[12:])
	lookup := func(id [20]byte) ([]byte, error) {
		if id != identifier {
			return nil, errors.New("not found")
		}
		return bytes.Clone(key), nil
	}
	verify := func(message, signature, publicKey []byte) bool {
		return len(message) == 32 && message[0] == 5 && signature[0] == 7 && bytes.Equal(publicKey, key)
	}
	for _, id := range []byte{EIP8051DraftNIST, EIP8051DraftETH} {
		t.Run(fmt.Sprintf("%x", id), func(t *testing.T) {
			alg, err := NewEIP8051DraftAlgorithm(id, lookup, verify)
			if err != nil {
				t.Fatal(err)
			}
			p, err := NewEIP7932DraftSigRecover(alg)
			if err != nil {
				t.Fatal(err)
			}
			input := make([]byte, EIP8051DraftSignatureInfoSize+32)
			input[0], input[1], input[EIP8051DraftSignatureInfoSize] = id, 7, 5
			copy(input[1+EIP8051DraftSignatureSize:], identifier[:])
			out, left, err := runPrecompile(p, input, 6000)
			addressHash := crypto.Keccak256([]byte{id}, key)
			if err != nil || left != 0 || !bytes.Equal(out[12:], addressHash[12:]) {
				t.Fatalf("registry output %x, gas %d, err %v", out, left, err)
			}
			if _, _, err := runPrecompile(p, input, 5999); !errors.Is(err, ErrOutOfGas) {
				t.Fatalf("insufficient gas: %v", err)
			}
			invalid := bytes.Clone(input)
			invalid[1] ^= 1
			out, err = p.Run(invalid)
			if err != nil || !bytes.Equal(out, make([]byte, 32)) {
				t.Fatal("accepted invalid signature")
			}
			invalid = bytes.Clone(input)
			invalid[1+EIP8051DraftSignatureSize] ^= 1
			out, err = p.Run(invalid)
			if err != nil || !bytes.Equal(out, make([]byte, 32)) {
				t.Fatal("accepted unknown public key")
			}
			out, err = p.Run(input[:len(input)-1])
			if err != nil || !bytes.Equal(out, make([]byte, 32)) {
				t.Fatal("accepted non-32-byte payload hash")
			}
		})
	}
	if _, err := NewEIP8051DraftAlgorithm(0xd3, lookup, verify); !errors.Is(err, ErrEIP8051DraftInvalidAlgorithm) {
		t.Fatal("accepted invalid variant")
	}
	if _, err := NewEIP8051DraftAlgorithm(EIP8051DraftNIST, nil, verify); !errors.Is(err, ErrEIP8051DraftInvalidAlgorithm) {
		t.Fatal("accepted missing lookup")
	}
	if _, err := NewEIP8051DraftAlgorithm(EIP8051DraftNIST, lookup, nil); !errors.Is(err, ErrEIP8051DraftInvalidAlgorithm) {
		t.Fatal("accepted missing verifier")
	}
	wrongLookup := func([20]byte) ([]byte, error) { wrong := bytes.Clone(key); wrong[0] ^= 1; return wrong, nil }
	alg, _ := NewEIP8051DraftAlgorithm(EIP8051DraftNIST, wrongLookup, func(_, _, _ []byte) bool { return true })
	info := make([]byte, EIP8051DraftSignatureInfoSize)
	info[0] = EIP8051DraftNIST
	copy(info[1+EIP8051DraftSignatureSize:], identifier[:])
	if _, err := alg.Recover(info, make([]byte, 32)); err == nil {
		t.Fatal("accepted a lookup result with the wrong key hash")
	}
}
