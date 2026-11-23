package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
)

type queryBlobBackend struct {
	handlerMockBackend
	calls int
}

func (b *queryBlobBackend) GetBlobsV2ByVersionedHashes(h []types.Hash) []*payload.BlobAndProofV2 {
	b.calls++
	return make([]*payload.BlobAndProofV2, len(h))
}
func (b *queryBlobBackend) GetBlobsV3ByVersionedHashes(h []types.Hash) []*payload.BlobAndProofV2 {
	b.calls++
	return make([]*payload.BlobAndProofV2, len(h))
}
func (b *queryBlobBackend) GetBlobsV4ByVersionedHashes(h []types.Hash, _ types.CellMask) []*payload.BlobCellsAndProofsV1 {
	b.calls++
	return make([]*payload.BlobCellsAndProofsV1, len(h))
}
func blobQueryRequest(t *testing.T, version int, hashes any) []byte {
	t.Helper()
	params := []any{hashes}
	if version == 4 {
		params = append(params, "0x01000000000000000000000000000000")
	}
	raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": fmt.Sprintf("engine_getBlobsV%d", version), "params": params})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func TestBlobQueriesV2V3V4ValidateBeforeBackend(t *testing.T) {
	for _, version := range []int{2, 3, 4} {
		for _, hashes := range []any{nil, []string{"0x01"}, []string{"0x" + strings.Repeat("gg", 32)}} {
			b := new(queryBlobBackend)
			api := NewEngineAPI(b)
			var reply jsonrpcResponse
			raw := api.HandleRequest(blobQueryRequest(t, version, hashes))
			if err := json.Unmarshal(raw, &reply); err != nil {
				t.Fatal(err)
			}
			if reply.Error == nil || reply.Error.Code != InvalidParamsCode || b.calls != 0 {
				t.Fatalf("V%d malformed query reached backend: %s, calls %d", version, raw, b.calls)
			}
		}
		b := new(queryBlobBackend)
		api := NewEngineAPI(b)
		hashes := make([]string, 129)
		for i := range hashes {
			hashes[i] = "0x01" + strings.Repeat("00", 31)
		}
		var reply jsonrpcResponse
		raw := api.HandleRequest(blobQueryRequest(t, version, hashes))
		if err := json.Unmarshal(raw, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Error == nil || reply.Error.Code != TooLargeRequestCode || b.calls != 0 {
			t.Fatalf("V%d oversized query reached backend: %s", version, raw)
		}
		// Retrieval accepts unknown 32-byte versioned hashes and reports them as
		// missing; it must not invent a KZG-prefix transaction-validity check.
		raw = api.HandleRequest(blobQueryRequest(t, version, []string{"0x02" + strings.Repeat("00", 31)}))
		reply = jsonrpcResponse{}
		if err := json.Unmarshal(raw, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Error != nil || b.calls != 1 {
			t.Fatalf("V%d rejected well-formed unknown hash: %s", version, raw)
		}
	}
}
