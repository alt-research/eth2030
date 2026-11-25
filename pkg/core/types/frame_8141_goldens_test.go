package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestFrame8141IndependentWireGoldens(t *testing.T) {
	raw, err := os.ReadFile("../../../specs/hegota/codec-goldens.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Vectors []struct {
			ID          string `json:"id"`
			Bytes       string `json:"bytes"`
			Hash        string `json:"transaction_hash"`
			SigningHash string `json:"signature_hash"`
		} `json:"vectors"`
	}
	if err = json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Vectors) != 7 {
		t.Fatal("missing independent base goldens")
	}
	for _, v := range corpus.Vectors {
		t.Run(v.ID, func(t *testing.T) {
			wire, err := hex.DecodeString(v.Bytes[2:])
			if err != nil {
				t.Fatal(err)
			}
			transaction, err := DecodeTxRLP(wire)
			if err != nil {
				t.Fatal(err)
			}
			frame, ok := transaction.Inner().(*FrameTx8141)
			if !ok {
				t.Fatalf("wrong schema %T", transaction.Inner())
			}
			if err := frame.Validate(); err != nil {
				t.Fatal(err)
			}
			encoded, err := transaction.EncodeRLP()
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatal("native wire differs", err)
			}
			if transaction.Hash() != HexToHash(v.Hash) || transaction.SigningHash() != HexToHash(v.SigningHash) {
				t.Fatal("native hash/signing hash differs from independent oracle")
			}
		})
	}
}
