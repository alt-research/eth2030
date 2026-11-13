package proofs

import (
	"bytes"
	"errors"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

// This test provider deliberately models a statement verifier, not a STARK.
type testAggregateProvider struct{ proveCalls, verifyCalls int }

func (p *testAggregateProvider) Prove(proofs []ExecutionProof, root types.Hash) ([]byte, error) {
	p.proveCalls++
	return append([]byte("test-artifact:"), root[:]...), nil
}
func (p *testAggregateProvider) Verify(proofs []ExecutionProof, root types.Hash, artifact []byte) (bool, error) {
	p.verifyCalls++
	for _, proof := range proofs {
		if !bytes.Equal(proof.ProofData, []byte("valid")) {
			return false, nil
		}
	}
	return bytes.Equal(artifact, append([]byte("test-artifact:"), root[:]...)), nil
}

func TestSTARKAggregatorRequiresProvider(t *testing.T) {
	agg := NewSTARKAggregator()
	proofs := []ExecutionProof{{ProofData: []byte("invalid")}}
	if _, err := agg.Aggregate(proofs); !errors.Is(err, ErrSTARKAggregateNoProvider) {
		t.Fatalf("got %v", err)
	}
	if ok, err := agg.Verify(&AggregatedProof{Proofs: proofs, AggregateProof: []byte("x")}); ok || !errors.Is(err, ErrSTARKAggregateNoProvider) {
		t.Fatalf("got %v, %v", ok, err)
	}
}

func TestSTARKAggregatorAuthenticatesArtifactAndInputs(t *testing.T) {
	provider := &testAggregateProvider{}
	agg := NewSTARKAggregatorWithProvider(provider)
	input := []ExecutionProof{{StateRoot: types.Hash{1}, BlockHash: types.Hash{2}, Type: ZKSNARK, ProofData: []byte("valid"), ProverID: "p"}}
	result, err := agg.Aggregate(input)
	if err != nil {
		t.Fatal(err)
	}
	if provider.proveCalls != 1 || provider.verifyCalls != 1 || len(result.AggregateProof) == 0 {
		t.Fatal("provider was not called or proof was discarded")
	}
	ok, err := agg.Verify(result)
	if err != nil || !ok {
		t.Fatalf("verify: %v, %v", ok, err)
	}
	input[0].ProofData[0] ^= 1
	if ok, err := agg.Verify(result); err != nil || !ok {
		t.Fatalf("input alias changed artifact: %v, %v", ok, err)
	}
	result.Proofs[0].ProofData[0] ^= 1
	if ok, _ := agg.Verify(result); ok {
		t.Fatal("accepted altered inner proof")
	}
	result.Proofs[0].ProofData[0] ^= 1
	result.AggregateProof[0] ^= 1
	if ok, _ := agg.Verify(result); ok {
		t.Fatal("accepted altered aggregate proof")
	}
	result.AggregateProof[0] ^= 1
	result.AggregateRoot[0] ^= 1
	if ok, _ := agg.Verify(result); ok {
		t.Fatal("accepted altered aggregate root")
	}
}

func TestSTARKAggregatorRejectsInvalidInputEvenWithArtifact(t *testing.T) {
	provider := &testAggregateProvider{}
	agg := NewSTARKAggregatorWithProvider(provider)
	if _, err := agg.Aggregate([]ExecutionProof{{ProofData: []byte("invalid")}}); !errors.Is(err, ErrProofInvalid) {
		t.Fatalf("invalid proof accepted: %v", err)
	}
}

func TestSTARKAggregateRootLengthDelimitsFields(t *testing.T) {
	a := []ExecutionProof{{ProofData: []byte("ab"), ProverID: "c"}}
	b := []ExecutionProof{{ProofData: []byte("a"), ProverID: "bc"}}
	if starkAggregateRoot(a) == starkAggregateRoot(b) {
		t.Fatal("ambiguous field encoding")
	}
}
