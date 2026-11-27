package txpool

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	"github.com/eth2030/eth2030/p2p/ethproto"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestDependency8504BlobSubmissionWrapperAdmissionAndLookup(t *testing.T) {
	pool, _, _ := dependencyPool8504()
	// The independent explicit-message witness isolates blob/witness transport;
	// canonical zero-message wallet authorization is qualified separately.
	tx := proof8504.Transaction(0)
	blob := make([]byte, types.FrameBytesPerBlob)
	blob[31] = 1
	kzg := bls.DefaultKZGBackend()
	commitment, err := kzg.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	_, cells, err := kzg.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	tx.Inner().(*types.FrameTx8141).BlobVersionedHashes = []types.Hash{types.Hash(hash)}
	sc := &types.BlobSidecarData{Blobs: [][]byte{bytes.Clone(blob)}, Commitments: [][]byte{bytes.Clone(commitment[:])}}
	for _, proof := range cells {
		sc.Proofs = append(sc.Proofs, bytes.Clone(proof[:]))
	}
	tx.SetBlobSidecar(sc)
	candidate := tx.WithDependencyWitnesses8504(tx.DependencyWitnesses8504())
	sc.Blobs[0][0] ^= 1
	sc.Proofs[0][0] ^= 1
	canonical, err := candidate.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := types.EncodePooledTransaction(candidate)
	if err != nil {
		t.Fatal(err)
	}
	var fields []rlp.RawValue
	if err := rlp.DecodeBytes(raw[1:], &fields); err != nil || len(fields) != 6 || !bytes.Equal(fields[0], canonical[1:]) || !bytes.Equal(fields[1], []byte{1}) {
		t.Fatal("blob submission is not the specified six-field form", err)
	}
	decoded, err := types.DecodePooledTransaction(raw)
	if err != nil || decoded.Hash() != candidate.Hash() || decoded.BlobSidecar() == nil || len(decoded.DependencyWitnesses8504()) != 1 {
		t.Fatal("raw blob/witness composition lost identity", err)
	}
	wrapped, err := types.EncodeDependencyWrapper8504(decoded)
	if err != nil {
		t.Fatal(err)
	}
	var wrapper struct {
		Transactions [][]byte
		Mode         uint64
		Content      struct{ Dependencies, Witnesses [][]byte }
	}
	if err := rlp.DecodeBytes(wrapped, &wrapper); err != nil || len(wrapper.Transactions) != 1 || len(wrapper.Content.Witnesses) != 1 {
		t.Fatal("blob wrapper shape", err)
	}
	var inner []rlp.RawValue
	if err := rlp.DecodeBytes(wrapper.Transactions[0][1:], &inner); err != nil || len(inner) != 5 {
		t.Fatal("wrapper nested raw-submission witnesses inside its pooled blob form", err)
	}
	decoded, err = types.DecodeDependencyWrapper8504(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&ethproto.PooledTransactionsPacket{Transactions: []*types.Transaction{decoded}}).EncodeRLP(); err == nil {
		t.Fatal("ordinary eth pooled message accepted dependency blob")
	}
	if err := pool.AddRemote(decoded); err != nil {
		t.Fatal("real blob and dependency witnesses rejected", err)
	}
	if pool.DependencyBytes8504() != uint64(len(wrapped)) || transactionPoolBytes(decoded) != uint64(len(wrapped)) {
		t.Fatal("wrapper budget omitted blobs or witnesses")
	}
	got := pool.GetBlobsByVersionedHashes(candidate.BlobHashes())
	if len(got) != 1 || got[0] == nil || !bytes.Equal(got[0].Blob, blob) {
		t.Fatal("dependency blob missing from Engine blob lookup")
	}
	if ok, err := kzg.VerifyBlobProof(got[0].Blob, got[0].Commitment, got[0].Proof); err != nil || !ok {
		t.Fatal("retrieved blob proof invalid", err)
	}
	pool.Remove(decoded.Hash())
	for _, kind := range []string{"version", "missing-witness", "missing-blob", "bad-cell", "bad-signature", "extra-field"} {
		err = nil
		bad := append([]rlp.RawValue(nil), fields...)
		switch kind {
		case "version":
			bad[1] = rlp.RawValue{2}
		case "missing-witness":
			bad = bad[:5]
		case "missing-blob":
			bad[2] = rlp.RawValue{0xc0}
		case "bad-cell":
			var proofs [][]byte
			if err := rlp.DecodeBytes(bad[4], &proofs); err != nil {
				t.Fatal(err)
			}
			proofs[127][0] ^= 1
			bad[4], err = rlp.EncodeToBytes(proofs)
		case "bad-signature":
			w := candidate.DependencyWitnesses8504()
			w[0][0] ^= 1
			bad[5], err = rlp.EncodeToBytes(w)
		case "extra-field":
			bad = append(bad, rlp.RawValue{0x80})
		}
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := rlp.EncodeToBytes(bad)
		if err != nil {
			t.Fatal(err)
		}
		invalid, err := types.DecodePooledTransaction(append([]byte{types.FrameTxType}, encoded...))
		if err == nil {
			err = pool.AddRemote(invalid)
		}
		if err == nil || pool.Count() != 0 || pool.DependencyBytes8504() != 0 {
			t.Fatal(kind, "invalid blob/witness composition admitted or leaked metadata", err)
		}
	}
}
