package pqc

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/cloudflare/circl/sign/slhdsa"
	"github.com/eth2030/eth2030/core/types"
	"github.com/pornin/go-fn-dsa/fndsa"
)

func TestProviderInteroperabilityAndHistoricalNames(t *testing.T) {
	message := []byte("provider interoperability")
	t.Run("ML-DSA-65", func(t *testing.T) {
		public, private, err := mldsa65.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer := &DilithiumSigner{}
		signature, err := signer.Sign(private.Bytes(), message)
		if err != nil {
			t.Fatal(err)
		}
		if !mldsa65.Verify(public, message, nil, signature) {
			t.Fatal("provider rejected wrapper signature")
		}
		external := make([]byte, mldsa65.SignatureSize)
		if err := mldsa65.SignTo(private, message, nil, false, external); err != nil {
			t.Fatal(err)
		}
		if !signer.Verify(public.Bytes(), message, external) {
			t.Fatal("wrapper rejected provider signature")
		}
		external[0] ^= 1
		if signer.Verify(public.Bytes(), message, external) {
			t.Fatal("accepted tampered signature")
		}
		oldPublic, oldPrivate, err := mode3.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		oldSig := make([]byte, mode3.SignatureSize)
		mode3.SignTo(oldPrivate, message, oldSig)
		if !mode3.Verify(oldPublic, message, oldSig) {
			t.Fatal("round-3 reference signature invalid")
		}
		if signer.Verify(oldPublic.Bytes(), message, oldSig) {
			t.Fatal("historical algorithm name accepted incompatible round-3 signature")
		}
	})
	t.Run("FN-DSA-512", func(t *testing.T) {
		secret, public, err := fndsa.KeyGen(9, rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer := &FalconSigner{}
		signature, err := signer.Sign(secret, message)
		if err != nil {
			t.Fatal(err)
		}
		if !fndsa.Verify(public, nil, 0, message, signature) {
			t.Fatal("provider rejected wrapper signature")
		}
		external, err := fndsa.Sign(rand.Reader, secret, nil, 0, message)
		if err != nil {
			t.Fatal(err)
		}
		if !signer.Verify(public, message, external) {
			t.Fatal("wrapper rejected provider signature")
		}
		external[0] ^= 1
		if signer.Verify(public, message, external) {
			t.Fatal("accepted tampered signature")
		}
	})
}

// Public NIST ACVP case carried by CIRCL v1.6.3. The secret key is test data.
func TestSLHDSADeterministicNISTVector(t *testing.T) {
	data, err := os.ReadFile("testdata/slh_dsa_sha2_128f_acvp.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		SecretKey string `json:"secret_key"`
		Message   string `json:"message"`
		Signature string `json:"signature"`
	}
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
	sk, msg, expected := decode(vector.SecretKey), decode(vector.Message), decode(vector.Signature)
	private := slhdsa.PrivateKey{ID: slhdsa.SHA2_128f}
	if err := private.UnmarshalBinary(sk); err != nil {
		t.Fatal(err)
	}
	pk := private.PublicKey()
	public, err := pk.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewSPHINCSSigner()
	actual, err := signer.Sign(sk, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("signature differs from NIST ACVP vector")
	}
	if !signer.Verify(public, msg, expected) {
		t.Fatal("wrapper rejected NIST ACVP vector")
	}
	expected[0] ^= 1
	if signer.Verify(public, msg, expected) {
		t.Fatal("wrapper accepted tampered NIST signature")
	}
}

func TestPQSignersRejectNegativeUnsignedFields(t *testing.T) {
	pipeline := NewPQSigningPipeline()
	pk, sk, err := pipeline.GenerateKey(PipelineAlgMLDSA65)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewPQTxSigner(SigDilithium3)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"chain", "value", "gas_price"} {
		t.Run(field, func(t *testing.T) {
			tx := pipelineTestTx()
			switch field {
			case "chain":
				tx.ChainID = big.NewInt(-1)
			case "value":
				tx.Value = big.NewInt(-1)
			case "gas_price":
				tx.GasPrice = big.NewInt(-1)
			}
			if _, err := tx.EncodePQ(); err != types.ErrPQTxNegativeInteger {
				t.Fatalf("encoded negative unsigned integer: %v", err)
			}
			if _, _, err := pipeline.SignTransaction(tx, PipelineAlgMLDSA65, sk, pk); err == nil {
				t.Fatal("pipeline signed a negative unsigned integer")
			}
			if _, err := signer.SignTransaction(types.NewTransaction(tx), &PQPrivateKey{Scheme: SigDilithium3, Raw: sk}); err != ErrInvalidSigningPayload {
				t.Fatalf("signer: %v", err)
			}
			tx.SignWithPQ(types.PQSigDilithium, pk, make([]byte, types.DilithiumSigSize))
			if err := types.ValidatePQTransactionSignature(tx); err == nil {
				t.Fatal("native validation accepted a negative unsigned integer")
			}
		})
	}
}
