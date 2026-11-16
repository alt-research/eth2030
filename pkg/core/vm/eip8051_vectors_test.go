package vm

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
)

// The pinned EIP assets label these vectors Dilithium2. They contain compact
// keys and signed variable-length messages, not the draft's expanded-key ABI.
func TestEIP8051PinnedVectorsIdentifyAlgorithm(t *testing.T) {
	data, err := os.ReadFile("testdata/eip8051_dilithium2_case0.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors map[string]struct {
		PublicKey string `json:"public_key"`
		Message   string `json:"message"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nist", "evm"} {
		t.Run(name, func(t *testing.T) {
			vector, ok := vectors[name]
			if !ok {
				t.Fatal("missing vector")
			}
			decode := func(s string) []byte {
				b, err := hex.DecodeString(s)
				if err != nil {
					t.Fatal(err)
				}
				return b
			}
			pk, msg, sig := decode(vector.PublicKey), decode(vector.Message), decode(vector.Signature)
			if len(pk) != 1312 || len(msg) != 33 || len(sig) != 2420 {
				t.Fatal("unexpected vector encoding")
			}
			var round3 mode2.PublicKey
			if err := round3.UnmarshalBinary(pk); err != nil {
				t.Fatal(err)
			}
			var fips mldsa44.PublicKey
			if err := fips.UnmarshalBinary(pk); err != nil {
				t.Fatal(err)
			}
			round3Valid := mode2.Verify(&round3, msg, sig)
			fipsValid := mldsa44.Verify(&fips, msg, nil, sig)
			if round3Valid {
				t.Fatal("vector unexpectedly verified as round-3 Dilithium2")
			}
			if fipsValid != (name == "nist") {
				t.Fatalf("FIPS verification = %v", fipsValid)
			}
			if fipsValid {
				sig[0] ^= 1
				if mldsa44.Verify(&fips, msg, nil, sig) {
					t.Fatal("accepted modified signature")
				}
			}
		})
	}
}
