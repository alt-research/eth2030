//go:build !cgo

package pqc

import (
	"errors"
	"testing"
)

func TestHistoricalProvidersReportUnavailableWithoutCGO(t *testing.T) {
	if _, _, err := (Falcon512Round3{}).GenerateKey(); !errors.Is(err, ErrRound3ProviderUnavailable) {
		t.Fatalf("Falcon provider availability: %v", err)
	}
	if _, err := (SPHINCSPlusSHA2128fRound3{}).Sign(make([]byte, 64), nil); !errors.Is(err, ErrRound3ProviderUnavailable) {
		t.Fatalf("SPHINCS+ provider availability: %v", err)
	}
	if (Falcon512Round3{}).Verify(make([]byte, 897), nil, make([]byte, 666)) {
		t.Fatal("unavailable verifier accepted a signature")
	}
}
