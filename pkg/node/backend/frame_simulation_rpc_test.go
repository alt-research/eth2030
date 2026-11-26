package backend

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/rpc/ethapi"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

// EIP-1474 distinguishes invalid method parameters from internal errors. The
// Frame extension retains authenticated execution for valid hypothetical input.
func TestFrameCSimulationOverrideErrorsThroughRPC(t *testing.T) {
	backend, db, _, options := frameSimulationFixture(t)
	root := db.GetRoot()
	api := ethapi.NewEthAPI(backend, nil)
	invoke := func(api *ethapi.EthAPI, options rpctypes.FrameSimulationOptions, contexts ...context.Context) *ethapi.Response {
		t.Helper()
		raw, err := json.Marshal(options)
		if err != nil {
			t.Fatal(err)
		}
		request := &ethapi.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "eth_simulateFrameV1", Params: []json.RawMessage{raw, json.RawMessage(`"latest"`)}}
		if len(contexts) > 0 {
			request = request.WithContext(contexts[0])
		}
		return api.HandleRequest(request)
	}
	valid := invoke(api, options)
	if !t.Run("valid_replay", func(t *testing.T) {
		if valid.Error != nil || !valid.Result.(*rpctypes.FrameSimulationResult).Valid {
			t.Fatalf("valid signed replay: %+v", valid)
		}
	}) {
		t.FailNow()
	}
	sender := *valid.Result.(*rpctypes.FrameSimulationResult).Payer
	zero, odd, short := "0x00", "0x0", "0x01"
	oversized := "0x10000000000000000000000000000000000000000000000000000000000000000"
	slot := types.Hash{}.Hex()
	for _, test := range []struct {
		name      string
		overrides rpctypes.StateOverride
	}{
		{"address", rpctypes.StateOverride{short: {Nonce: &zero}}},
		{"nonce", rpctypes.StateOverride{sender: {Nonce: &zero}}},
		{"balance", rpctypes.StateOverride{sender: {Balance: &zero}}},
		{"balance_overflow", rpctypes.StateOverride{sender: {Balance: &oversized}}},
		{"code", rpctypes.StateOverride{sender: {Code: &odd}}},
		{"conflicting_storage", rpctypes.StateOverride{sender: {State: map[string]string{slot: slot}, StateDiff: map[string]string{slot: slot}}}},
		{"storage_key", rpctypes.StateOverride{sender: {StateDiff: map[string]string{short: slot}}}},
		{"storage_value", rpctypes.StateOverride{sender: {StateDiff: map[string]string{slot: short}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := options
			candidate.StateOverrides = test.overrides
			response := invoke(api, candidate)
			if response.Error == nil {
				t.Fatalf("invalid override must be -32602, not an internal failure: %+v", response)
			}
			if response.Error.Code != rpctypes.ErrCodeInvalidParams || response.Error.Message == "" || response.Result != nil {
				t.Fatalf("invalid override error code=%d message=%q, want -32602", response.Error.Code, response.Error.Message)
			}
			if db.GetRoot() != root {
				t.Fatal("invalid override changed canonical state")
			}
		})
	}
	// A well-encoded nonce that makes the transaction invalid is an execution
	// result, not malformed method input. It must not be reclassified by the fix.
	t.Run("valid_override", func(t *testing.T) {
		nonce := "0x0"
		candidate := options
		candidate.StateOverrides = rpctypes.StateOverride{sender: {Nonce: &nonce}}
		response := invoke(api, candidate)
		if response.Error != nil || !response.Result.(*rpctypes.FrameSimulationResult).Valid || db.GetRoot() != root {
			t.Fatalf("valid override lost authenticated replay/state isolation: %+v", response)
		}
	})
	t.Run("nonce_mismatch", func(t *testing.T) {
		nonce := "0x1"
		candidate := options
		candidate.StateOverrides = rpctypes.StateOverride{sender: {Nonce: &nonce}}
		response := invoke(api, candidate)
		if response.Error != nil {
			t.Fatalf("valid hypothetical override rejected: %+v", response.Error)
		}
		result := response.Result.(*rpctypes.FrameSimulationResult)
		if result.Valid || result.Error == "" || db.GetRoot() != root {
			t.Fatal("nonce mismatch lost execution failure/state isolation")
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		response := invoke(api, options, ctx)
		if response.Error == nil || response.Error.Code != -32016 || response.Result != nil || db.GetRoot() != root {
			t.Fatalf("cancellation lost its execution error/state isolation: %+v", response)
		}
	})
}
