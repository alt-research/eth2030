package backend

import (
	"encoding/json"
	"fmt"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/engine/payload"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine"
)

type cellServingNode struct {
	*mockNodeDeps
	syncing bool
	handler EthHandlerDeps
}

func (n *cellServingNode) SyncProgress() (bool, uint64, uint64, uint64) { return n.syncing, 0, 0, 1 }
func (n *cellServingNode) EthHandler() EthHandlerDeps                   { return n.handler }

type cellServingHandler struct {
	rows        map[types.Hash]types.BlobCells
	calls       int
	unavailable bool
}

func (h *cellServingHandler) BroadcastTransactions([]*types.Transaction) {}
func (h *cellServingHandler) GetBlobCells(hashes []types.Hash, mask types.CellMask) []types.BlobCells {
	h.calls++
	if h.unavailable {
		return nil
	}
	rows := make([]types.BlobCells, len(hashes))
	for i, hash := range hashes {
		rows[i] = h.rows[hash]
	}
	return rows
}
func TestGetBlobsV4GeneralUnavailabilityIsNull(t *testing.T) {
	for _, kind := range []string{"syncing", "no-handler", "unavailable", "closed"} {
		t.Run(kind, func(t *testing.T) {
			h := &cellServingHandler{}
			node := &cellServingNode{mockNodeDeps: newTestBackendNode(t), handler: h, syncing: kind == "syncing"}
			if kind == "no-handler" {
				node.handler = nil
			}
			h.unavailable = kind == "unavailable"
			b := NewEngineBackend(node)
			t.Cleanup(b.Close)
			if kind == "closed" {
				b.Close()
			}
			hash := types.Hash{0: 1}
			params, _ := json.Marshal([]any{[]types.Hash{hash}, "0x01000000000000000000000000000000"})
			raw := engine.NewEngineAPI(b).HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"engine_getBlobsV4","params":` + string(params) + `}`))
			var reply struct {
				Result json.RawMessage `json:"result"`
				Error  any             `json:"error"`
			}
			if err := json.Unmarshal(raw, &reply); err != nil {
				t.Fatal(err)
			}
			if reply.Error != nil || string(reply.Result) != "null" {
				t.Fatalf("want null during %s, got %s", kind, raw)
			}
			if (kind == "syncing" || kind == "closed") && h.calls != 0 {
				t.Fatal("unavailable node accessed blob pool")
			}
		})
	}
}
func TestGetBlobsV4AvailablePoolPreservesMissingPositionsAndPairs(t *testing.T) {
	known, missing := types.Hash{0: 1, 31: 1}, types.Hash{0: 1, 31: 2}
	h := &cellServingHandler{rows: map[types.Hash]types.BlobCells{known: {Cells: [][]byte{make([]byte, 2048), nil}, Proofs: [][]byte{make([]byte, 48), make([]byte, 48)}}}}
	n := &cellServingNode{mockNodeDeps: newTestBackendNode(t), handler: h}
	b := NewEngineBackend(n)
	t.Cleanup(b.Close)
	mask := types.CellMask{3}
	hashes := make([]types.Hash, 128)
	for i := range hashes {
		hashes[i] = known
	}
	hashes[1] = missing
	got := b.GetBlobsV4ByVersionedHashes(hashes, mask)
	if len(got) != 128 || got[1] != nil {
		t.Fatal("available pool lost requested order or missing-blob position")
	}
	for _, i := range []int{0, 127} {
		if got[i] == nil || len(got[i].BlobCells) != 2 || got[i].BlobCells[0] == nil || got[i].Proofs[0] == nil {
			t.Fatal("known cell/proof missing")
		}
		if got[i].BlobCells[1] != nil || got[i].Proofs[1] != nil {
			t.Fatal("missing cell leaked a non-null proof")
		}
	}
	n.syncing = true
	if got := b.GetBlobsV4ByVersionedHashes(hashes, mask); got != nil {
		t.Fatal("sync transition still served pool")
	}
	n.syncing = false
	if got := b.GetBlobsV4ByVersionedHashes([]types.Hash{missing}, mask); len(got) != 1 || got[0] != nil {
		t.Fatal("recovered serving must report a missing position, not general unavailability")
	}
}

func TestGetBlobsV2V3LifecycleBlocksCachedDataDuringSyncAndClose(t *testing.T) {
	for _, version := range []int{2, 3} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			n := &cellServingNode{mockNodeDeps: newTestBackendNode(t), handler: &cellServingHandler{}}
			b := NewEngineBackend(n)
			t.Cleanup(b.Close)
			hash := types.Hash{0: 1}
			blob := make([]byte, 131072)
			_, cells, err := bls.DefaultKZGBackend().ComputeCellsAndProofs(blob)
			if err != nil {
				t.Fatal(err)
			}
			proofs := make([][]byte, len(cells))
			for i, proof := range cells {
				proofs[i] = append([]byte(nil), proof[:]...)
			}
			b.blobCache[hash] = copyBlobCellProofs(blob, proofs)
			read := func() []*payload.BlobAndProofV2 {
				if version == 2 {
					return b.GetBlobsV2ByVersionedHashes([]types.Hash{hash})
				}
				return b.GetBlobsV3ByVersionedHashes([]types.Hash{hash})
			}
			if got := read(); len(got) != 1 || got[0] == nil {
				t.Fatal("healthy cached blob unavailable")
			}
			n.syncing = true
			if got := read(); got != nil {
				t.Fatal("syncing served cached blob")
			}
			n.syncing = false
			if got := read(); len(got) != 1 || got[0] == nil {
				t.Fatal("recovery lost cached blob")
			}
			b.Close()
			if got := read(); got != nil {
				t.Fatal("closed backend served cached blob")
			}
		})
	}
}
