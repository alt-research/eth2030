package txpool

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/p2p/ethproto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestCanonicalFrame8141BlobProofsAndWire(t *testing.T) {
	pool, _, frame := framePoolFixture(t)
	blob := make([]byte, types.FrameBytesPerBlob)
	blob[31] = 1
	kzg := bls.DefaultKZGBackend()
	commitment, err := kzg.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	_, proofs, err := kzg.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	frame.BlobVersionedHashes = []types.Hash{types.Hash(hash)}
	sidecar := &types.BlobSidecarData{Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
	for _, proof := range proofs {
		sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(proof[:]))
	}
	tx := signPoolFrame(t, frame, 1)
	tx.SetBlobSidecar(sidecar)
	original := tx.Hash()
	packet := ethproto.PooledTransactionsPacket{RequestID: 17, Transactions: []*types.Transaction{tx}}
	encoded, err := packet.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	var decoded ethproto.PooledTransactionsPacket
	if err := decoded.DecodeRLP(encoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RequestID != 17 || len(decoded.Transactions) != 1 {
		t.Fatal("lost request identity or Frame")
	}
	actual := decoded.Transactions[0]
	if actual.Hash() != original || actual.BlobSidecar() == nil || len(actual.BlobSidecar().Proofs) != 128 {
		t.Fatal("pooled wrapper changed transaction identity or lost proofs")
	}
	if err := pool.AddRemote(actual); err != nil {
		t.Fatalf("valid cell proofs rejected: %v", err)
	}
	retrieved := pool.GetBlobsByVersionedHashes(frame.BlobVersionedHashes)
	if len(retrieved) != 1 || retrieved[0] == nil || !bytes.Equal(retrieved[0].Blob, blob) {
		t.Fatal("Frame blob unavailable through Engine lookup")
	}
	if valid, err := kzg.VerifyBlobProof(retrieved[0].Blob, retrieved[0].Commitment, retrieved[0].Proof); err != nil || !valid {
		t.Fatalf("Engine blob proof is not a blob proof: %v", err)
	}
	plain, err := actual.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	body, err := types.DecodeTxRLP(plain)
	if err != nil || body.BlobSidecar() != nil || body.Hash() != original {
		t.Fatal("block body contains the pooled wrapper")
	}
	wrapper, err := types.EncodePooledTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := types.DecodeTxRLP(wrapper); err == nil {
		t.Fatal("block body accepted the network wrapper")
	}
	var fields []gethrlp.RawValue
	if err := gethrlp.DecodeBytes(wrapper[1:], &fields); err != nil {
		t.Fatal(err)
	}
	fields[1] = gethrlp.RawValue{2}
	badVersion, err := gethrlp.EncodeToBytes(fields)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := types.DecodePooledTransaction(append([]byte{6}, badVersion...)); err == nil {
		t.Fatal("unknown wrapper version accepted")
	}
	pool.Remove(actual.Hash())
	for _, test := range []struct {
		name   string
		change func(*types.BlobSidecarData)
	}{
		{"missing sidecar", func(sc *types.BlobSidecarData) { sc.Blobs = nil }},
		{"missing cell proof", func(sc *types.BlobSidecarData) { sc.Proofs = sc.Proofs[:127] }},
		{"wrong commitment", func(sc *types.BlobSidecarData) { sc.Commitments[0][0] ^= 1 }},
		{"non field element", func(sc *types.BlobSidecarData) { sc.Blobs[0][0] = 255 }},
		{"wrong final cell proof", func(sc *types.BlobSidecarData) { sc.Proofs[127][47] ^= 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			sc := &types.BlobSidecarData{Blobs: [][]byte{bytes.Clone(blob)}, Commitments: [][]byte{bytes.Clone(commitment[:])}}
			for _, proof := range proofs {
				sc.Proofs = append(sc.Proofs, bytes.Clone(proof[:]))
			}
			test.change(sc)
			candidate := signPoolFrame(t, frame, 1)
			candidate.SetBlobSidecar(sc)
			if err := pool.AddRemote(candidate); err == nil {
				t.Fatal("invalid sidecar admitted")
			}
			if len(pool.lookup.framePayers) != 0 {
				t.Fatal("rejected blob retained payer exposure")
			}
		})
	}
}
