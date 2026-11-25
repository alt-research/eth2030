package eftest

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"os"
	"strconv"
	"testing"
)

// This consumes an independent Python oracle. It does not use the old native
// FrameTx encoder, whose incompatibility is the reason for the activation gate.
func TestHegotaIndependentEncodingGoldens(t *testing.T) {
	data, err := os.ReadFile("../../../specs/hegota/codec-goldens.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Vectors []struct {
			ID      string `json:"id"`
			Payload []any  `json:"payload"`
			Bytes   string `json:"bytes"`
			Hash    string `json:"transaction_hash"`
			SigHash string `json:"signature_hash"`
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&golden); err != nil {
		t.Fatal(err)
	}
	if len(golden.Vectors) != 7 {
		t.Fatal("missing goldens")
	}
	for _, v := range golden.Vectors {
		t.Run(v.ID, func(t *testing.T) {
			p := goldenRLPValue(t, v.Payload)
			encoded, err := gethrlp.EncodeToBytes(p)
			if err != nil {
				t.Fatal(err)
			}
			wire := append([]byte{6}, encoded...)
			if "0x"+hex.EncodeToString(wire) != v.Bytes || gethcrypto.Keccak256Hash(wire).Hex() != v.Hash {
				t.Fatal("independent RLP/hash mismatch")
			}
			signatures := v.Payload[4].([]any)
			for _, entry := range signatures {
				s := entry.([]any)
				if s[2].(string) == "0x" {
					sig, err := hex.DecodeString(s[3].(string)[2:])
					if err != nil {
						t.Fatal(err)
					}
					s[3] = json.Number(strconv.FormatUint(uint64(len(sig)), 10))
				}
			}
			elided, err := gethrlp.EncodeToBytes(goldenRLPValue(t, v.Payload))
			if err != nil {
				t.Fatal(err)
			}
			if gethcrypto.Keccak256Hash(append([]byte{6}, elided...)).Hex() != v.SigHash {
				t.Fatal("signature hash/elision mismatch")
			}
		})
	}
}

func goldenRLPValue(t *testing.T, v any) any {
	t.Helper()
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = goldenRLPValue(t, e)
		}
		return out
	case json.Number:
		n, err := strconv.ParseUint(string(x), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	case string:
		b, err := hex.DecodeString(x[2:])
		if err != nil {
			t.Fatal(err)
		}
		return b
	default:
		t.Fatalf("unsupported golden value %T", v)
		return nil
	}
}
