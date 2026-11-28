package heze

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/leansphincs"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

// These are measurements of the direct EIP-8504 research profile. Repeated
// occurrences measure byte capacity, not independently spendable transactions.
func TestGroupEPQCanonicalCapacityVerifierAndAuthenticatedPipeline(t *testing.T) {
	tx, _ := proof8504.CanonicalTransaction(false)
	raw, err := tx.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	deps, err := tx.Inner().(*types.FrameTx8141).Dependencies8504()
	if err != nil {
		t.Fatal(err)
	}
	witness := tx.DependencyWitnesses8504()[0]
	start := time.Now()
	for i := 0; i < 16; i++ {
		if !leansphincs.Verify(deps[0].VerificationKeyHash[:], deps[0].DataHash[:], witness) {
			t.Fatal("real canonical PQ signature failed")
		}
	}
	verifyNS := time.Since(start).Nanoseconds() / 16
	bad := append([]byte(nil), witness...)
	bad[len(bad)/2] ^= 1
	if leansphincs.Verify(deps[0].VerificationKeyHash[:], deps[0].DataHash[:], bad) {
		t.Fatal("tampered PQ signature accepted")
	}
	capacity := MaxTransactionBytes / len(raw)
	if capacity < 1 || len(witness) != 7740 {
		t.Fatal("canonical fixture or provisional profile changed", len(raw), len(witness))
	}
	occurrences := make([][]byte, capacity)
	for i := range occurrences {
		occurrences[i] = raw
	}
	if CheckTransactions(occurrences) != nil || CheckTransactions(append(occurrences, raw)) == nil {
		t.Fatal("8192-byte capacity boundary")
	}
	v, s := fixture(t)
	s.Message.Transactions = occurrences
	start = time.Now()
	store := NewStore()
	var serialized int
	for i := uint64(0); i < 16; i++ {
		s.Message.ValidatorIndex = i
		sign(&s, v)
		wire := encoded(t, s)
		serialized += len(wire)
		if err := store.Gossip(wire, v); err != nil {
			t.Fatal(err)
		}
	}
	stored := store.Transactions(Key{s.Message.Slot, s.Message.DependentRoot}, false)
	if len(stored) != 1 {
		t.Fatal("exact-byte union did not deduplicate", len(stored))
	}
	pipelineNS := time.Since(start).Nanoseconds()
	report := map[string]any{"profile": "direct8504 research; omitted witness transport unspecified", "go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU(), "transaction_bytes": len(raw), "external_witness_bytes": len(witness), "byte_capacity_occurrences": capacity, "committee_size": CommitteeSize, "max_transaction_bytes": MaxTransactionBytes, "committee_signed_ssz_bytes": serialized, "verify_uncached_mean_ns": verifyNS, "local_sign_encode_BLS_auth_store_union_ns": pipelineNS, "network_latency_measured": false, "public_conformance": false}
	b, _ := json.Marshal(report)
	t.Log(string(b))
}
