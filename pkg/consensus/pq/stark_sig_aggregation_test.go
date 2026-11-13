package pq

import (
	"bytes"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/pqc"
	"github.com/eth2030/eth2030/proofs"
)

func signTestAttestations(t *testing.T, count int) []PQAttestation {
	t.Helper()
	signer := pqc.NewMLDSASigner()
	result := make([]PQAttestation, count)
	for i := range result {
		key, err := signer.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		a := PQAttestation{
			Slot: 42, CommitteeIndex: 2, BeaconBlockRoot: types.Hash{1},
			SourceEpoch: 3, TargetEpoch: 4, ValidatorIndex: uint64(i),
			PQPublicKey: key.PublicKey,
		}
		a.PQSignature, err = signer.Sign(key, attestationMessage(a.Slot, a.CommitteeIndex, a.BeaconBlockRoot, a.SourceEpoch, a.TargetEpoch))
		if err != nil {
			t.Fatal(err)
		}
		result[i] = a
	}
	return result
}

func TestPQAggregationVerifiesEverySignatureAndMetadata(t *testing.T) {
	input := signTestAttestations(t, 6)
	aggregator := NewSTARKSignatureAggregator()
	verifier := NewPQAttestationVerifier(testPQAttestationConfig(input...))
	result, err := verifier.STARKAggregateVerify(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.NumValidators != 6 || result.AggregateProof != nil || result.GasSavings() != 0 {
		t.Fatalf("unexpected batch metadata: count %d, proof %v, estimated savings %d", result.NumValidators, result.AggregateProof != nil, result.GasSavings())
	}
	if valid, err := aggregator.Verify(result); err != nil || !valid {
		t.Fatalf("valid signed batch: %v", err)
	}
	input[5].PQSignature[0] ^= 1
	if valid, err := aggregator.Verify(result); err != nil || !valid {
		t.Fatalf("source mutation changed the result: %v", err)
	}
	if _, err := verifier.STARKAggregateVerify(input); err != ErrSTARKAggVerifyFailed {
		t.Fatalf("accepted invalid signature beyond the old four-entry threshold: %v", err)
	}
	verified, failed := verifier.Stats()
	if verified != 6 || failed != 6 {
		t.Fatalf("batch counters = %d, %d", verified, failed)
	}

	for _, test := range []struct {
		name   string
		change func(*STARKSignatureAggregation)
	}{
		{"proof-only", func(a *STARKSignatureAggregation) { a.AggregateProof = &proofs.STARKProofData{} }},
		{"mixed message", func(a *STARKSignatureAggregation) { a.Signatures[5].BeaconBlockRoot[0] ^= 1 }},
		{"mixed committee", func(a *STARKSignatureAggregation) { a.Signatures[5].CommitteeIndex++ }},
		{"duplicate validator", func(a *STARKSignatureAggregation) { a.Signatures[5].ValidatorIndex = a.Signatures[0].ValidatorIndex }},
		{"duplicate public key", func(a *STARKSignatureAggregation) {
			a.Signatures[5].PQPublicKey = a.Signatures[0].PQPublicKey
			a.Signatures[5].PQSignature = a.Signatures[0].PQSignature
		}},
		{"forged count", func(a *STARKSignatureAggregation) { a.NumValidators++ }},
		{"forged epoch", func(a *STARKSignatureAggregation) { a.TargetEpoch++ }},
		{"forged committee root", func(a *STARKSignatureAggregation) { a.CommitteeRoot[0] ^= 1 }},
		{"forged signature", func(a *STARKSignatureAggregation) {
			a.Signatures[5].PQSignature = bytes.Clone(a.Signatures[5].PQSignature)
			a.Signatures[5].PQSignature[0] ^= 1
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := *result
			bad.Signatures = append([]PQAttestation{}, result.Signatures...)
			test.change(&bad)
			if ok, err := aggregator.Verify(&bad); ok || err == nil {
				t.Fatal("accepted altered batch")
			}
		})
	}
	if _, err := BatchVerifyPQAttestations(nil); err != ErrSTARKAggNoSigs {
		t.Fatalf("empty batch: %v", err)
	}
}

func TestComputeCommitteeRoot(t *testing.T) {
	pubkeys := [][]byte{
		[]byte("pubkey-1"),
		[]byte("pubkey-2"),
		[]byte("pubkey-3"),
	}

	root := computeCommitteeRoot(pubkeys)
	var zeroHash types.Hash
	if root == zeroHash {
		t.Error("committee root should not be zero")
	}

	// Same keys should produce same root.
	root2 := computeCommitteeRoot(pubkeys)
	if root != root2 {
		t.Error("committee root should be deterministic")
	}

	// Different keys should produce different root.
	pubkeys2 := [][]byte{
		[]byte("pubkey-a"),
		[]byte("pubkey-b"),
	}
	root3 := computeCommitteeRoot(pubkeys2)
	if root == root3 {
		t.Error("different keys should produce different root")
	}
}

func TestComputeCommitteeRootEmpty(t *testing.T) {
	root := computeCommitteeRoot(nil)
	var zeroHash types.Hash
	if root != zeroHash {
		t.Error("empty pubkeys should produce zero root")
	}
}

func TestAggregateCount(t *testing.T) {
	// Nil aggregation.
	var nilAgg *STARKSignatureAggregation
	if nilAgg.AggregateCount() != 0 {
		t.Error("nil aggregation should have count 0")
	}

	agg := &STARKSignatureAggregation{NumValidators: 42}
	if agg.AggregateCount() != 42 {
		t.Errorf("expected count 42, got %d", agg.AggregateCount())
	}
}

func TestMerklePQAggregationChecksSignedInputs(t *testing.T) {
	input := signTestAttestations(t, 2)
	result, err := MerkleAggregatePQAttestations(input)
	if err != nil || result.NumValidators != 2 || result.Slot != input[0].Slot {
		t.Fatalf("valid signed batch: %v", err)
	}
	input[1].PQSignature[0] ^= 1
	if _, err := MerkleAggregatePQAttestations(input); err != ErrSTARKAggVerifyFailed {
		t.Fatalf("invalid signed batch: %v", err)
	}
}
