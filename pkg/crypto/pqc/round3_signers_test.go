//go:build cgo

package pqc

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestHistoricalPQCleanReferenceSignatures(t *testing.T) {
	file, err := os.Open("testdata/round3_pqclean.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var cases []struct{ Name, Public, Private, Signature string }
	if err := json.NewDecoder(reader).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 {
		t.Fatalf("want both historical formats, got %d", len(cases))
	}
	for _, vector := range cases {
		t.Run(vector.Name, func(t *testing.T) {
			type signer interface {
				GenerateKey() ([]byte, []byte, error)
				Sign([]byte, []byte) ([]byte, error)
				Verify([]byte, []byte, []byte) bool
			}
			var historical signer
			var current PQSigner
			switch vector.Name {
			case "falcon":
				historical, current = Falcon512Round3{}, &FalconSigner{}
			case "sphincs":
				historical, current = SPHINCSPlusSHA2128fRound3{}, NewSPHINCSSigner()
			default:
				t.Fatalf("unknown format %q", vector.Name)
			}
			decode := func(value string) []byte {
				decoded, err := hex.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				return decoded
			}
			message := []byte("eth2030 historical signature fixture")
			public, private, signature := decode(vector.Public), decode(vector.Private), decode(vector.Signature)
			if !historical.Verify(public, message, signature) {
				t.Fatal("rejected reference signature from the standalone C executable")
			}
			if current.Verify(public, message, signature) {
				t.Fatal("historical signature was accepted by the transaction provider")
			}
			if historical.Verify(public, []byte("different message"), signature) ||
				historical.Verify(public[:len(public)-1], message, signature) ||
				historical.Verify(public, message, signature[:len(signature)-1]) {
				t.Fatal("accepted the wrong message or a truncated wire field")
			}
			signature[1] ^= 1
			if historical.Verify(public, message, signature) {
				t.Fatal("accepted an altered signature")
			}
			if _, err := historical.Sign(private[:len(private)-1], message); err == nil {
				t.Fatal("signed with a truncated private key")
			}
			generatedSignature, err := historical.Sign(private, message)
			if err != nil || !historical.Verify(public, message, generatedSignature) {
				t.Fatalf("signing with the independently serialized private key: %v", err)
			}
			newPublic, newPrivate, err := historical.GenerateKey()
			if err != nil {
				t.Fatal(err)
			}
			generatedSignature, err = historical.Sign(newPrivate, nil)
			if err != nil || !historical.Verify(newPublic, nil, generatedSignature) ||
				historical.Verify(public, nil, generatedSignature) {
				t.Fatalf("key generation and signing an empty message: %v", err)
			}
		})
	}
}

func TestFalcon512Round3EIP8052Vector(t *testing.T) {
	data, err := os.ReadFile("testdata/eip8052_falcon_case0.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct{ Public, Message, Signature string }
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	decode := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	public, message, signature := decode(vector.Public), decode(vector.Message), decode(vector.Signature)
	signer := Falcon512Round3{}
	if !signer.Verify(public, message, signature) {
		t.Fatal("rejected Falcon signature from the pinned EIP-8052 vectors")
	}
	signature[len(signature)-1] = 1
	if signer.Verify(public, message, signature) {
		t.Fatal("accepted nonzero signature padding")
	}
}
