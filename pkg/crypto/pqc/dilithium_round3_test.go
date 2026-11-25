package pqc

import (
	"bytes"
	"testing"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
)

func TestDilithium3Round3WireInteroperability(t *testing.T) {
	signer := Dilithium3Round3{}
	pk, sk, err := signer.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	message := []byte("round-3 Dilithium3 wire format")
	sig, err := signer.Sign(sk, message)
	if err != nil {
		t.Fatal(err)
	}
	var providerKey mode3.PublicKey
	if err := providerKey.UnmarshalBinary(pk); err != nil || !mode3.Verify(&providerKey, message, sig) {
		t.Fatalf("provider rejected the serialized wrapper signature: %v", err)
	}
	var seed [mode3.SeedSize]byte
	seed[0] = 1
	providerPK, providerSK := mode3.NewKeyFromSeed(&seed)
	external := make([]byte, mode3.SignatureSize)
	mode3.SignTo(providerSK, message, external)
	if !signer.Verify(providerPK.Bytes(), message, external) {
		t.Fatal("wrapper rejected provider signature")
	}
	wrapperSig, err := signer.Sign(providerSK.Bytes(), message)
	if err != nil || !bytes.Equal(wrapperSig, external) {
		t.Fatalf("deterministic signing disagrees with provider: %v", err)
	}
	if signer.Verify(pk, message, external) || signer.Verify(providerPK.Bytes(), []byte("wrong message"), external) {
		t.Fatal("accepted the wrong key or message")
	}
	if (&DilithiumSigner{}).Verify(providerPK.Bytes(), message, external) {
		t.Fatal("ML-DSA-65 transaction provider accepted a round-3 signature")
	}
	external[0] ^= 1
	if signer.Verify(providerPK.Bytes(), message, external) || signer.Verify(providerPK.Bytes(), message, external[:len(external)-1]) {
		t.Fatal("accepted an altered or truncated signature")
	}
	if _, err := signer.Sign(sk[:len(sk)-1], message); err == nil {
		t.Fatal("accepted a truncated private key")
	}
}
