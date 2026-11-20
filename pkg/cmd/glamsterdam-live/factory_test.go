package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFactoryPreflightRequiresExactGenesisRuntimeAndNonce(t *testing.T) {
	for _, scenario := range []struct {
		name, code, nonce string
		accepted          bool
	}{{"canonical", canonicalFactoryCode, "0x1", true}, {"wrong runtime", "0x00", "0x1", false}, {"missing genesis nonce", canonicalFactoryCode, "0x0", false}} {
		t.Run(scenario.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     uint64
					Method string
					Params []json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				if len(request.Params) != 2 || string(request.Params[1]) != `"0x0"` {
					t.Error("factory preflight did not bind original genesis state")
				}
				result := scenario.code
				if request.Method == "eth_getTransactionCount" {
					result = scenario.nonce
				} else if request.Method != "eth_getCode" {
					t.Error("unexpected preflight method")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
			}))
			defer server.Close()
			c := client{url: server.URL, http: server.Client(), record: &recorder{writer: io.Discard}}
			if err := factoryGenesis(context.Background(), []client{c}); (err == nil) != scenario.accepted {
				t.Fatalf("canonical genesis acceptance=%v: %v", scenario.accepted, err)
			}
		})
	}
}
