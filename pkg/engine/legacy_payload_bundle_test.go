package engine

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	enginepayload "github.com/eth2030/eth2030/engine/payload"
)

// Exercise both return paths involved in the merge: the legacy response must
// retain real cell proofs and must refuse publication when blob data is missing.
func TestLegacyPayloadV5RetainsBlobBundle(t *testing.T) {
	kzg, err := bls.NewGoEthKZGBackend()
	if err != nil {
		t.Fatal(err)
	}
	blob := make([]byte, 131072)
	blob[31] = 1
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
	for _, mode := range []string{"stored", "async"} {
		for _, missing := range []bool{false, true} {
			name := mode + "/valid"
			if missing {
				name = mode + "/missing_sidecar"
			}
			t.Run(name, func(t *testing.T) {
				tx := types.NewTransaction(&types.BlobTx{BlobHashes: []types.Hash{types.Hash(hash)}})
				if !missing {
					sidecar := &types.BlobSidecarData{Version: 1, Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
					for _, proof := range proofs {
						sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(proof[:]))
					}
					tx.SetBlobSidecar(sidecar)
				}
				slot := uint64(42)
				block := types.NewBlock(&types.Header{Number: big.NewInt(1), Time: 12, Difficulty: new(big.Int), BaseFee: big.NewInt(1), SlotNumber: &slot}, &types.Body{Transactions: []*types.Transaction{tx}})
				id := PayloadID{1}
				b := &EngineBackend{}
				if mode == "async" {
					pending := enginepayload.NewPendingPayload(id, types.Hash{}, 12, types.Address{}, types.Hash{}, nil)
					pending.SlotNumber = slot
					pending.SetCompleted(block, nil, bal.NewBlockAccessList(), big.NewInt(7))
					b.asyncPayloads = map[PayloadID]*enginepayload.PendingPayload{id: pending}
				} else {
					b.payloads = map[PayloadID]*pendingPayload{id: {block: block, bal: bal.NewBlockAccessList(), blockValue: big.NewInt(7), slotNumber: slot}}
				}
				response, err := b.GetPayloadV5(id)
				if missing {
					if err == nil || !strings.Contains(err.Error(), "incomplete sidecar") || response != nil {
						t.Fatalf("missing blob data was published: response=%+v error=%v", response, err)
					}
					return
				}
				if err != nil || response == nil || response.ExecutionPayload == nil || response.ExecutionPayload.BlockHash != block.Hash() || response.ExecutionPayload.SlotNumber != "0x2a" {
					t.Fatalf("legacy payload lost block/slot: response=%+v error=%v", response, err)
				}
				bundle := response.BlobsBundle
				if bundle == nil || len(bundle.Blobs) != 1 || len(bundle.Commitments) != 1 || len(bundle.Proofs) != len(proofs) || !bytes.Equal(bundle.Blobs[0], blob) || !bytes.Equal(bundle.Commitments[0], commitment[:]) {
					t.Fatalf("legacy response lost blob data: %+v", bundle)
				}
				for i, proof := range proofs {
					if !bytes.Equal(bundle.Proofs[i], proof[:]) {
						t.Fatalf("cell proof %d changed", i)
					}
				}
			})
		}
	}
}
