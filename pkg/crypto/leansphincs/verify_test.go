package leansphincs

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestEIP8504IndependentEthLibVerificationVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/ethlib-verify.json")
	if err != nil {
		t.Fatal(err)
	}
	// Preserve the source's snake_case fields and null (decode rejection).
	var source struct {
		Cases []struct {
			Name      string `json:"name"`
			PublicKey string `json:"public_key"`
			Message   string `json:"message"`
			Signature string `json:"signature"`
			Expected  *bool  `json:"expected"`
		}
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Cases) != 26 {
		t.Fatal("independent corpus incomplete")
	}
	decode := func(s string) []byte {
		b, e := hex.DecodeString(strings.TrimPrefix(s, "0x"))
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	for _, v := range source.Cases {
		t.Run(v.Name, func(t *testing.T) {
			pk, msg, sig := decode(v.PublicKey), decode(v.Message), decode(v.Signature)
			got, work := verify(pk, msg, sig)
			want := v.Expected != nil && *v.Expected
			if got != want {
				t.Fatalf("verification=%t want=%t", got, want)
			}
			if work > MaxVerifyPermutations || (got && work != MaxVerifyPermutations) {
				t.Fatalf("unbounded/incorrect permutation work %d", work)
			}
			if v.Expected == nil && work != 0 {
				t.Fatal("malformed input hashed before length rejection")
			}
		})
	}
}

func TestEIP8504VerifierRejectsEveryTruncationBeforeHashing(t *testing.T) {
	for n := 0; n < SignatureSize; n++ {
		if ok, work := verify(make([]byte, 32), make([]byte, 32), make([]byte, n)); ok || work != 0 {
			t.Fatalf("length %d performed work", n)
		}
	}
}
