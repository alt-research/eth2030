package vm

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

// The fixtures use ZKNOX's independent Python polynomial implementation with
// the pinned draft's 32-byte tr, raw message and big-endian field encoding.
func TestEIP8051ExpandedReferenceVectors(t *testing.T) {
	file, err := os.Open("testdata/eip8051_expanded.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var vectors []struct{ Name, Message, Signature, Key string }
	if err := json.NewDecoder(reader).Decode(&vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 {
		t.Fatalf("want both variants, got %d vectors", len(vectors))
	}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			decode := func(value string) []byte {
				data, err := hex.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			msg, sig, key := decode(vector.Message), decode(vector.Signature), decode(vector.Key)
			variant, other := EIP8051DraftNIST, EIP8051DraftETH
			if vector.Name == "Keccak" {
				variant, other = other, variant
			}
			precompile, err := NewEIP8051ExpandedPrecompile(variant)
			if err != nil {
				t.Fatal(err)
			}
			evm, _ := newIntegrationEVM()
			draftAddress := types.Address{19: 0x12 + variant - EIP8051DraftNIST}
			evm.SetPrecompiles(map[types.Address]PrecompiledContract{draftAddress: precompile})
			input := bytes.Join([][]byte{msg, sig, key}, nil)
			want := make([]byte, 32)
			want[31] = 1
			out, left, err := evm.StaticCall(types.Address{}, draftAddress, input, 4500)
			if err != nil || left != 0 || !bytes.Equal(out, want) {
				t.Fatalf("known-answer precompile execution: %x, gas %d, error %v", out, left, err)
			}
			if _, _, err := evm.StaticCall(types.Address{}, draftAddress, input, 4499); !errors.Is(err, ErrOutOfGas) {
				t.Fatalf("underfunded EVM call: %v", err)
			}
			if EIP8051DraftVerifierForVariant(other)(msg, sig, key) {
				t.Fatal("accepted a signature from the other variant")
			}

			digest := crypto.Keccak256(key)
			alg, err := NewEIP8051ExpandedAlgorithm(variant, func([20]byte) ([]byte, error) {
				return key, nil // The adapter authenticates the returned key.
			})
			if err != nil {
				t.Fatal(err)
			}
			registry, err := NewEIP7932DraftSigRecover(alg)
			if err != nil {
				t.Fatal(err)
			}
			envelope := bytes.Join([][]byte{{variant}, sig, digest[12:], msg}, nil)
			address := crypto.Keccak256([]byte{variant}, key)
			registryAddress := types.Address{19: 0x12}
			evm.SetPrecompiles(map[types.Address]PrecompiledContract{registryAddress: registry})
			out, left, err = evm.StaticCall(types.Address{}, registryAddress, envelope, 6000)
			if err != nil || left != 0 || len(out) != 32 || !bytes.Equal(out[:12], make([]byte, 12)) || !bytes.Equal(out[12:], address[12:]) {
				t.Fatalf("known-answer SIGRECOVER: %x, gas %d, error %v", out, left, err)
			}

			for _, mutation := range []struct {
				name string
				edit func([]byte)
			}{
				{"message", func(b []byte) { b[0] ^= 1 }},
				{"challenge", func(b []byte) { b[32] ^= 1 }},
				{"tr", func(b []byte) { b[32+2420+16384] ^= 1 }},
				{"noncanonical A", func(b []byte) {
					field := b[32+2420:]
					binary.BigEndian.PutUint32(field, binary.BigEndian.Uint32(field)+8380417)
				}},
				{"noncanonical t1", func(b []byte) {
					field := b[32+2420+16416:]
					binary.BigEndian.PutUint32(field, binary.BigEndian.Uint32(field)+8380417)
				}},
				{"unordered hints", func(b []byte) {
					hints := b[32+2336:]
					start := 0
					for _, end := range hints[80:84] {
						if int(end)-start >= 2 {
							hints[start], hints[start+1] = hints[start+1], hints[start]
							return
						}
						start = int(end)
					}
					t.Fatal("fixture lacks two hints in one polynomial")
				}},
			} {
				t.Run(mutation.name, func(t *testing.T) {
					changed := bytes.Clone(input)
					mutation.edit(changed)
					out, left, err := runPrecompile(precompile, changed, 4500)
					if err != nil || left != 0 || !bytes.Equal(out, make([]byte, 32)) {
						t.Fatalf("tampered input: %x, gas %d, error %v", out, left, err)
					}
				})
			}
		})
	}
}

// A zero matrix and t1 make w independent of z. This isolates the strict
// norm check from challenge verification at the boundary.
func TestEIP8051ExpandedStrictNorm(t *testing.T) {
	message := make([]byte, 32)
	key := make([]byte, 20512)
	for _, variant := range []byte{EIP8051DraftNIST, EIP8051DraftETH} {
		eth := variant == EIP8051DraftETH
		mu := make([]byte, 64)
		_, _ = io.ReadFull(mldsaXOF(eth, make([]byte, 64)), mu)
		sig := make([]byte, 2420)
		_, _ = io.ReadFull(mldsaXOF(eth, mu, make([]byte, 768)), sig[:32])
		for i := 32; i < 2336; i += 9 {
			// Four 18-bit encodings of gamma1 represent four zero coefficients.
			copy(sig[i:i+9], []byte{0, 0, 2, 0, 8, 0, 32, 0, 128})
		}
		verify := EIP8051DraftVerifierForVariant(variant)
		sig[32] = 79 // z[0] = gamma1 - beta - 1.
		sig[34] &^= 3
		if !verify(message, sig, key) {
			t.Fatalf("variant %x rejected coefficient inside bound", variant)
		}
		sig[32] = 78 // z[0] = gamma1 - beta.
		if verify(message, sig, key) {
			t.Fatalf("variant %x accepted coefficient at bound", variant)
		}
	}
}
