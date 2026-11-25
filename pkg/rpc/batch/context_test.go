package rpcbatch

import (
	"context"
	"encoding/json"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"sync/atomic"
	"testing"
	"time"
)

type cancelBatchHandler struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (h *cancelBatchHandler) HandleRequest(request *rpctypes.Request) *rpctypes.Response {
	h.calls.Add(1)
	select {
	case h.entered <- struct{}{}:
	default:
	}
	select {
	case <-request.Context().Done():
	case <-h.release:
	}
	return rpctypes.NewErrorResponse(request.ID, -32000, "cancelled")
}
func TestCancelledBatchStopsQueuedWorkAndRetainsResponseIDs(t *testing.T) {
	h := &cancelBatchHandler{entered: make(chan struct{}, 5), release: make(chan struct{})}
	defer close(h.release)
	batch := NewBatchHandler(h)
	batch.SetParallelism(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan []BatchResponse, 1)
	go func() {
		result, err := batch.HandleBatchContext(ctx, []byte(`[{"jsonrpc":"2.0","id":1,"method":"eth_call"},{"jsonrpc":"2.0","id":2,"method":"eth_call"},{"jsonrpc":"2.0","id":3,"method":"eth_call"}]`))
		if err != nil {
			done <- nil
		} else {
			done <- result
		}
	}()
	select {
	case <-h.entered:
	case <-time.After(time.Second):
		t.Fatal("first batch item not admitted")
	}
	cancel()
	select {
	case result := <-done:
		if len(result) != 3 || h.calls.Load() != 1 {
			t.Fatalf("queued requests executed after cancellation: calls=%d responses=%d", h.calls.Load(), len(result))
		}
		for i, response := range result {
			var id int
			if json.Unmarshal(response.ID, &id) != nil || id != i+1 || response.Error == nil {
				t.Fatalf("cancelled response %d: %+v", i, response)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled batch did not release its active and queued work")
	}
}
