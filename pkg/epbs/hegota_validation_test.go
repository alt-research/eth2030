package epbs

import (
	"errors"
	"testing"
)

func TestBuilderBidNeverAcceptsUnauthenticatedInput(t *testing.T) {
	if !errors.Is(ValidateBuilderBid(nil), ErrBLSSignatureInvalid) {
		t.Fatal("nil bid accepted")
	}
	b := makeValidBid()
	if !errors.Is(ValidateBuilderBid(b), ErrBLSSignatureInvalid) {
		t.Fatal("unsigned bid accepted")
	}
	b.Signature[0] = 0x80
	if !errors.Is(ValidateBuilderBid(b), ErrBLSSignatureInvalid) {
		t.Fatal("zero key accepted")
	}
	b.Message.BuilderPubkey[0] = 0x80
	if !errors.Is(ValidateBuilderBid(b), ErrBidAuthenticationUnavailable) {
		t.Fatal("self-supplied key/hash accepted without canonical domain")
	}
}
