package proofs

import (
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
)

// The test verifier is a stub; these tests exercise batch control flow only.
func newTestBatchAggregator(config AggregationConfig, inner ProofAggregator) *BatchAggregator {
	ba := NewBatchAggregator(config, inner)
	ba.SetExecutionProofVerifier(func(ExecutionProof) (bool, error) { return true, nil })
	return ba
}

func TestDefaultAggregationConfig(t *testing.T) {
	cfg := DefaultAggregationConfig()

	if cfg.MaxProofs != 64 {
		t.Errorf("MaxProofs: got %d, want 64", cfg.MaxProofs)
	}
	if cfg.VerificationTimeout != 5*time.Second {
		t.Errorf("VerificationTimeout: got %v, want 5s", cfg.VerificationTimeout)
	}
	if !cfg.ParallelVerify {
		t.Error("ParallelVerify should default to true")
	}
}

func TestBatchAggregatorAddProof(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), nil)

	p := ExecutionProof{
		StateRoot: types.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		BlockHash: types.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		ProofData: []byte{0x01, 0x02},
		ProverID:  "test",
		Type:      ZKSNARK,
	}

	err := ba.AddProof(p)
	if err != nil {
		t.Fatalf("AddProof error: %v", err)
	}
}

func TestBatchAggregatorAddProofFull(t *testing.T) {
	cfg := DefaultAggregationConfig()
	cfg.MaxProofs = 2
	ba := newTestBatchAggregator(cfg, nil)

	p := ExecutionProof{
		StateRoot: types.Hash{0x01},
		BlockHash: types.Hash{0x02},
		ProofData: []byte{0x03},
		ProverID:  "test",
		Type:      ZKSNARK,
	}

	ba.AddProof(p)
	ba.AddProof(p)

	err := ba.AddProof(p)
	if err != ErrBatchFull {
		t.Errorf("expected ErrBatchFull, got %v", err)
	}
}

func TestBatchAggregatorAggregateBatch(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), nil)

	proofs := makeTestProofs(3)
	for _, p := range proofs {
		ba.AddProof(p)
	}

	batch, err := ba.AggregateBatch()
	if err != nil {
		t.Fatalf("AggregateBatch error: %v", err)
	}
	if batch == nil {
		t.Fatal("batch is nil")
	}
	if len(batch.Proofs) != 3 {
		t.Errorf("expected 3 proofs, got %d", len(batch.Proofs))
	}
	if batch.AggregateHash.IsZero() {
		t.Error("expected non-zero AggregateHash")
	}
	if batch.Verified {
		t.Error("batch should not be verified before VerifyBatch call")
	}
}

func TestBatchAggregatorAggregateBatchEmpty(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), nil)

	_, err := ba.AggregateBatch()
	if err != ErrBatchEmpty {
		t.Errorf("expected ErrBatchEmpty, got %v", err)
	}
}

func TestBatchAggregatorVerifyBatch(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())

	proofs := makeTestProofs(4)
	for _, p := range proofs {
		ba.AddProof(p)
	}

	batch, _ := ba.AggregateBatch()

	valid, err := ba.VerifyBatch(batch)
	if err != nil {
		t.Fatalf("VerifyBatch error: %v", err)
	}
	if !valid {
		t.Error("expected valid batch")
	}
	if !batch.Verified {
		t.Error("expected Verified=true after successful VerifyBatch")
	}
	if batch.VerifiedAt.IsZero() {
		t.Error("expected non-zero VerifiedAt")
	}
}

func TestBatchAggregatorVerifyBatchTampered(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())

	proofs := makeTestProofs(3)
	for _, p := range proofs {
		ba.AddProof(p)
	}

	batch, _ := ba.AggregateBatch()

	// Tamper with the aggregate hash.
	batch.AggregateHash[0] ^= 0xff

	valid, err := ba.VerifyBatch(batch)
	if err != nil {
		t.Fatalf("VerifyBatch error: %v", err)
	}
	if valid {
		t.Error("expected tampered batch to fail verification")
	}
}

func TestBatchAggregatorVerifyBatchNil(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), nil)

	_, err := ba.VerifyBatch(nil)
	if err != ErrBatchEmpty {
		t.Errorf("expected ErrBatchEmpty for nil batch, got %v", err)
	}
}

func TestBatchAggregatorStats(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())

	// Initial stats should be zero.
	batched, verified, failed := ba.Stats()
	if batched != 0 || verified != 0 || failed != 0 {
		t.Errorf("initial stats: batched=%d verified=%d failed=%d", batched, verified, failed)
	}

	// Add and aggregate some proofs.
	proofs := makeTestProofs(5)
	for _, p := range proofs {
		ba.AddProof(p)
	}
	batch, _ := ba.AggregateBatch()

	batched, _, _ = ba.Stats()
	if batched != 5 {
		t.Errorf("expected 5 batched, got %d", batched)
	}

	// Verify the batch.
	ba.VerifyBatch(batch)

	_, verified, _ = ba.Stats()
	if verified != 5 {
		t.Errorf("expected 5 verified, got %d", verified)
	}

	// Create a tampered batch to increment failed count.
	proofs2 := makeTestProofs(3)
	for _, p := range proofs2 {
		ba.AddProof(p)
	}
	batch2, _ := ba.AggregateBatch()
	batch2.AggregateHash[0] ^= 0xff
	ba.VerifyBatch(batch2)

	_, _, failed = ba.Stats()
	if failed != 3 {
		t.Errorf("expected 3 failed, got %d", failed)
	}
}

func TestBatchAggregatorDeterministicHash(t *testing.T) {
	ba1 := newTestBatchAggregator(DefaultAggregationConfig(), nil)
	ba2 := newTestBatchAggregator(DefaultAggregationConfig(), nil)

	proofs := makeTestProofs(4)
	for _, p := range proofs {
		ba1.AddProof(p)
		ba2.AddProof(p)
	}

	batch1, _ := ba1.AggregateBatch()
	batch2, _ := ba2.AggregateBatch()

	if batch1.AggregateHash != batch2.AggregateHash {
		t.Error("expected deterministic AggregateHash for same proofs")
	}
}

func TestBatchAggregatorClearsPending(t *testing.T) {
	ba := newTestBatchAggregator(DefaultAggregationConfig(), nil)

	proofs := makeTestProofs(3)
	for _, p := range proofs {
		ba.AddProof(p)
	}

	ba.AggregateBatch()

	// Second aggregate should fail since pending is cleared.
	_, err := ba.AggregateBatch()
	if err != ErrBatchEmpty {
		t.Errorf("expected ErrBatchEmpty after clearing, got %v", err)
	}
}

func TestBatchAggregatorAllProofTypesRoundTrip(t *testing.T) {
	// Round-trip test: aggregate 5 proofs (one of each type + extra ZKSNARK),
	// then verify the batch.
	ba := newTestBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())

	t.Run("aggregate all types", func(t *testing.T) {
		proofTypes := []ProofType{ZKSNARK, ZKSTARK, IPA, KZG, ZKSNARK}
		for i, pt := range proofTypes {
			p := ExecutionProof{
				StateRoot: types.Hash{byte(i + 1)},
				BlockHash: types.Hash{byte(i + 0x10)},
				ProofData: []byte{byte(i), byte(pt), 0xAA},
				ProverID:  "test-prover",
				Type:      pt,
			}
			if err := ba.AddProof(p); err != nil {
				t.Fatalf("AddProof(%d): %v", i, err)
			}
		}

		batch, err := ba.AggregateBatch()
		if err != nil {
			t.Fatalf("AggregateBatch: %v", err)
		}
		if len(batch.Proofs) != 5 {
			t.Fatalf("expected 5 proofs, got %d", len(batch.Proofs))
		}
		if batch.AggregateHash.IsZero() {
			t.Error("AggregateHash should not be zero")
		}

		valid, err := ba.VerifyBatch(batch)
		if err != nil {
			t.Fatalf("VerifyBatch: %v", err)
		}
		if !valid {
			t.Error("expected valid batch with all proof types")
		}
		if !batch.Verified {
			t.Error("expected Verified=true")
		}
	})

	t.Run("tampered single proof in batch", func(t *testing.T) {
		ba2 := newTestBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())
		for i := 0; i < 5; i++ {
			ba2.AddProof(ExecutionProof{
				StateRoot: types.Hash{byte(i + 1)},
				BlockHash: types.Hash{byte(i + 0x20)},
				ProofData: []byte{byte(i), 0xBB},
				ProverID:  "test",
				Type:      ProofType(i % 4),
			})
		}
		batch, _ := ba2.AggregateBatch()
		// Tamper with one proof's data.
		batch.Proofs[2].ProofData[0] ^= 0xFF
		valid, err := ba2.VerifyBatch(batch)
		if err != nil {
			t.Fatalf("VerifyBatch: %v", err)
		}
		if valid {
			t.Error("expected invalid batch after tampering a proof")
		}
	})

	t.Run("empty batch fails", func(t *testing.T) {
		ba3 := newTestBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())
		_, err := ba3.AggregateBatch()
		if err != ErrBatchEmpty {
			t.Errorf("expected ErrBatchEmpty, got %v", err)
		}
	})
}

func TestValidateAggregatedProof(t *testing.T) {
	// Nil batch should fail.
	if err := ValidateAggregatedProof(nil); err == nil {
		t.Fatal("expected error for nil batch")
	}

	// Empty proofs should fail.
	if err := ValidateAggregatedProof(&ProofBatch{}); err == nil {
		t.Fatal("expected error for empty proofs")
	}

	// Zero aggregate hash should fail.
	batch := &ProofBatch{
		Proofs: []ExecutionProof{
			{Type: ZKSNARK, ProofData: []byte{0x01}},
		},
	}
	if err := ValidateAggregatedProof(batch); err == nil {
		t.Fatal("expected error for zero aggregate hash")
	}

	// Valid batch.
	batch.AggregateHash = types.Hash{0x01}
	if err := ValidateAggregatedProof(batch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBatchAggregatorRequiresProofValidation(t *testing.T) {
	ba := NewBatchAggregator(DefaultAggregationConfig(), NewSimpleAggregator())
	if err := ba.AddProof(ExecutionProof{ProofData: []byte("invalid")}); err != nil {
		t.Fatal(err)
	}
	batch, err := ba.AggregateBatch()
	if err != nil {
		t.Fatal(err)
	}
	batch.Verified = true
	ok, err := ba.VerifyBatch(batch)
	if ok || err != ErrProofVerifierRequired || batch.Verified {
		t.Fatalf("accepted unverified proof: %v, %v", ok, err)
	}
	ba.SetExecutionProofVerifier(func(ExecutionProof) (bool, error) { return false, nil })
	ok, err = ba.VerifyBatch(batch)
	if ok || err != nil || batch.Verified {
		t.Fatalf("accepted rejected proof: %v, %v", ok, err)
	}
}

func TestExecutionProofHashSeparatesFields(t *testing.T) {
	a := ExecutionProof{ProofData: []byte("ab"), ProverID: "c"}
	b := ExecutionProof{ProofData: []byte("a"), ProverID: "bc"}
	if hashProof(&a) == hashProof(&b) {
		t.Fatal("proof data and prover ID are ambiguously encoded")
	}
}

func TestBatchAggregatorOwnsProofBytesAndClearsFailedStatus(t *testing.T) {
	proof := makeTestProofs(1)[0]
	originalByte := proof.ProofData[0]
	cfg := DefaultAggregationConfig()
	cfg.MaxProofs = 1
	batchAggregator := NewBatchAggregator(cfg, nil)
	batchAggregator.SetExecutionProofVerifier(func(candidate ExecutionProof) (bool, error) {
		return candidate.ProofData[0] == originalByte, nil
	})
	if err := batchAggregator.AddProof(proof); err != nil {
		t.Fatal(err)
	}
	proof.ProofData[0] ^= 1
	batch, err := batchAggregator.AggregateBatch()
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := batchAggregator.VerifyBatch(batch); !ok || err != nil || batch.VerifiedAt.IsZero() {
		t.Fatalf("original submitted proof was lost: %v, %v", ok, err)
	}
	batch.Proofs = append(batch.Proofs, proof)
	if ok, err := batchAggregator.VerifyBatch(batch); ok || err != ErrBatchFull || batch.Verified || !batch.VerifiedAt.IsZero() {
		t.Fatalf("oversized batch kept validity or returned the wrong error: %v, %v", ok, err)
	}
	batch.Proofs = nil
	batch.Verified = true
	batch.VerifiedAt = time.Now()
	if ok, err := batchAggregator.VerifyBatch(batch); ok || err != ErrBatchEmpty || batch.Verified || !batch.VerifiedAt.IsZero() {
		t.Fatalf("empty batch kept validity or returned the wrong error: %v, %v", ok, err)
	}
}

func TestBatchAggregatorTimeoutRetainsProviderSlot(t *testing.T) {
	cfg := DefaultAggregationConfig()
	cfg.ParallelVerify = false
	cfg.VerificationTimeout = 150 * time.Millisecond
	ba := NewBatchAggregator(cfg, nil)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	defer close(release)
	ba.SetExecutionProofVerifier(func(ExecutionProof) (bool, error) {
		started <- struct{}{}
		<-release
		return true, nil
	})
	makeBatch := func() *ProofBatch {
		t.Helper()
		if err := ba.AddProof(makeTestProofs(1)[0]); err != nil {
			t.Fatal(err)
		}
		batch, err := ba.AggregateBatch()
		if err != nil {
			t.Fatal(err)
		}
		return batch
	}
	checkTimeout := func(batch *ProofBatch) {
		t.Helper()
		done := make(chan error, 1)
		go func() {
			_, err := ba.VerifyBatch(batch)
			done <- err
		}()
		select {
		case err := <-done:
			if err != ErrVerifyTimeout || batch.Verified {
				t.Fatalf("blocked verifier result: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("provider held the caller beyond its timeout")
		}
	}
	checkTimeout(makeBatch())
	select {
	case <-started:
	default:
		t.Fatal("test provider did not start")
	}
	checkTimeout(makeBatch())
	select {
	case <-started:
		t.Fatal("timed out provider released its worker slot before returning")
	default:
	}
}
