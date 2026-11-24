package zkvm

import (
	"crypto/sha256"
	"errors"
	"testing"
)

func TestProofBackendRequiresExecutableStatement(t *testing.T) {
	if _, err := ProveExecution(nil); !errors.Is(err, ErrProofBackendNilRequest) {
		t.Fatal(err)
	}
	for _, trace := range []*RVWitnessCollector{nil, NewRVWitnessCollector()} {
		if _, err := ProveExecution(&ProofRequest{Trace: trace, PublicInputs: []byte("arbitrary"), ProgramHash: HashProgram([]byte("program"))}); !errors.Is(err, ErrProofBackendStatementRequired) {
			t.Fatal("trace-only request manufactured a proof", err)
		}
	}
	if _, err := ProveExecution(&ProofRequest{Program: []byte{0x73}, ProgramHash: HashProgram([]byte("different"))}); !errors.Is(err, ErrProofBackendInvalidProof) {
		t.Fatal(err)
	}
}

func TestProofBackendRejectsSyntheticAndMalformedResults(t *testing.T) {
	if valid, err := VerifyExecProof(nil, [32]byte{}); valid || !errors.Is(err, ErrProofBackendNilResult) {
		t.Fatal(valid, err)
	}
	for _, data := range [][]byte{nil, make([]byte, 256), append([]byte("mock-proof:"), make([]byte, 32)...), []byte(`{}`), []byte(`{} {}`)} {
		if valid, err := VerifyExecProof(&ProofResult{ProofBytes: data}, [32]byte{}); valid || err == nil {
			t.Fatal("accepted a synthetic proof", valid, err)
		}
	}
}

func TestLegacyTracePipelineCannotClaimExecutionProof(t *testing.T) {
	if _, err := VerifyWithBackend(nil, nil, nil); !errors.Is(err, ErrProofBackendNilTrace) {
		t.Fatal(err)
	}
	trace := NewRVWitnessCollector()
	if _, err := VerifyWithBackend(trace, nil, nil); !errors.Is(err, ErrProofBackendEmptyTrace) {
		t.Fatal(err)
	}
	trace.RecordStep(0, 0x73, [32]uint32{}, [32]uint32{}, nil)
	if _, err := VerifyWithBackend(trace, []byte{0x73}, nil); !errors.Is(err, ErrProofBackendStatementRequired) {
		t.Fatal(err)
	}
}

func TestProofBackendHashProgram(t *testing.T) {
	program := []byte{1, 2, 3}
	if HashProgram(program) != sha256.Sum256(program) {
		t.Fatal("program digest differs")
	}
}
