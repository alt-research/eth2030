package rpctypes

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRequestContextIsLocalAndRequestScoped(t *testing.T) {
	var request Request
	raw := []byte(`{"jsonrpc":"2.0","id":7,"method":"eth_call","params":[]}`)
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(&request)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	bound := request.WithContext(ctx)
	cancel()
	if bound.Context().Err() != context.Canceled || request.Context().Err() != nil {
		t.Fatal("request context was not isolated")
	}
	encoded, err := json.Marshal(bound)
	if err != nil || string(encoded) != string(before) {
		t.Fatalf("transport metadata changed JSON-RPC wire: %s %v", encoded, err)
	}
}
