package engine

import (
	"bytes"
	"encoding/json"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestLegacyBuilderMethodsAreRetired(t *testing.T) {
	api := NewEngineAPI(nil)
	secret := [32]byte{1, 2, 3}
	api.SetJWTSecret(secret)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"iat": time.Now().Unix()}).SignedString(secret[:])
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.Handler())
	defer server.Close()
	for _, method := range []string{"engine_registerBuilderV1", "engine_submitBuilderBidV1", "engine_getBuilderBidsV1", "engine_getBestBuilderBidV1", "engine_getPayloadHeaderV1", "engine_submitBlindedBlockV1"} {
		t.Run(method, func(t *testing.T) {
			// Nontrivial formerly accepted content must not enter a local auction.
			body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 17, "method": method, "params": []any{map[string]any{"message": map[string]any{"slot": 10, "builderIndex": 0, "value": 100, "blockHash": "0x11", "parentBlockHash": "0x22"}, "signature": make([]byte, 96)}}})
			req, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			response, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var reply struct {
				ID     int             `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&reply); err != nil {
				t.Fatal(err)
			}
			if reply.ID != 17 || reply.Error == nil || reply.Error.Code != -32601 || len(reply.Result) != 0 {
				t.Fatalf("retired method accepted or misclassified: %+v", reply)
			}
		})
	}
}

func TestCapabilitiesDoNotAdvertiseLegacyBuilderExtensions(t *testing.T) {
	api := NewEngineAPI(nil)
	caps := api.ExchangeCapabilities(nil)
	for _, required := range []string{"engine_newPayloadV5", "engine_forkchoiceUpdatedV4", "engine_getPayloadBodiesByHashV2"} {
		found := false
		for _, method := range caps {
			if method == required {
				found = true
			}
		}
		if !found {
			t.Errorf("standard capability removed: %s", required)
		}
	}
	for _, method := range caps {
		switch method {
		case "engine_submitBuilderBidV1", "engine_getBuilderBidsV1", "engine_getPayloadHeaderV1", "engine_submitBlindedBlockV1":
			t.Errorf("retired capability advertised: %s", method)
		}
	}
}

func TestRecordedQualifiedEngineTrafficRemainsAdvertised(t *testing.T) {
	raw, err := os.ReadFile("../../docs/reviews/builder-authority/QUALIFIED_ENGINE_METHODS.json")
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Methods map[string]int `json:"methods"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	supported := map[string]bool{}
	for _, method := range NewEngineAPI(&bogotaTestBackend{}).ExchangeCapabilities(nil) {
		supported[method] = true
	}
	if len(record.Methods) < 4 {
		t.Fatal("qualified traffic record is empty")
	}
	for method, count := range record.Methods {
		// Negotiation is the bootstrap operation, not an advertised payload
		// capability. Exercise its actual handler instead of inventing a label.
		if method == "engine_exchangeCapabilities" {
			response := NewEngineAPI(nil).HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"engine_exchangeCapabilities","params":[[]]}`))
			var reply struct {
				Error  *jsonrpcError `json:"error"`
				Result []string      `json:"result"`
			}
			if err := json.Unmarshal(response, &reply); err != nil || reply.Error != nil || len(reply.Result) == 0 {
				t.Fatalf("bootstrap negotiation unavailable: %s", response)
			}
			continue
		}
		if count <= 0 || !supported[method] {
			t.Errorf("qualified method removed: %s", method)
		}
	}
}
