package bls

import (
	"bytes"
	"testing"
)

func TestKZGBlobCellProofBatchAuthenticatesEveryIndex(t *testing.T) {
	backend, err := NewGoEthKZGBackend()
	if err != nil {
		t.Fatal(err)
	}
	blob := make([]byte, KZGBytesPerBlob)
	blob[31] = 1
	commitment, err := backend.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	_, generated, err := backend.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	proofs := make([][]byte, len(generated))
	for i := range generated {
		proofs[i] = bytes.Clone(generated[i][:])
	}
	if valid, err := backend.VerifyBlobCellProofs(blob, commitment[:], proofs); err != nil || !valid {
		t.Fatalf("valid proof batch: %v", err)
	}
	// Reuse a valid curve point at the wrong index, rather than a malformed point.
	proofs[len(proofs)-1] = bytes.Clone(proofs[0])
	if valid, err := backend.VerifyBlobCellProofs(blob, commitment[:], proofs); err == nil && valid {
		t.Fatal("last cell was not authenticated against its own index")
	}
	if valid, err := backend.VerifyBlobCellProofs(blob, commitment[:], proofs[:127]); err == nil || valid {
		t.Fatal("truncated proof batch accepted")
	}
}
