package prover

import (
	"errors"
	"testing"
)

func TestValidationFrameProverRejectsUnprovenStatements(t *testing.T) {
	p := NewSTARKValidationFrameProver()
	if proof, err := p.ProveValidationFrame([]byte("data"), []byte{1}); proof != nil || !errors.Is(err, ErrFrameProofUnavailable) {
		t.Fatalf("unavailable frame proof: %v", err)
	}
	if proof, err := p.ProveAllValidationFrames([][]byte{[]byte("data")}); proof != nil || !errors.Is(err, ErrFrameProofUnavailable) {
		t.Fatalf("unavailable batch proof: %v", err)
	}
	if p.Verify(&STARKProofData{}) {
		t.Fatal("accepted proof without a statement-bound provider")
	}
	if _, err := p.ProveValidationFrame(nil, nil); !errors.Is(err, ErrFrameNilOutput) {
		t.Fatalf("nil output: %v", err)
	}
	if _, err := p.ProveValidationFrame(nil, []byte{0}); !errors.Is(err, ErrFrameReverted) {
		t.Fatalf("reverted frame: %v", err)
	}
	if _, err := p.ProveAllValidationFrames(nil); !errors.Is(err, ErrFrameEmptyBatch) {
		t.Fatalf("empty batch: %v", err)
	}
}

func TestStubProverIsOnlyATestDouble(t *testing.T) {
	var _ ValidationFrameProver = &StubValidationFrameProver{}
	stub := &StubValidationFrameProver{}
	proof, err := stub.ProveValidationFrame([]byte("data"), []byte{1})
	if err != nil || !stub.Verify(proof) {
		t.Fatalf("test stub: %v", err)
	}
}
