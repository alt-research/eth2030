package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGroupEBogotaStrictAttributesOpaqueILAndBackendCapabilities(t *testing.T) {
	base := map[string]any{"timestamp": "0x12c", "prevRandao": "0x" + strings.Repeat("00", 32), "suggestedFeeRecipient": "0x" + strings.Repeat("00", 20), "withdrawals": []any{}, "parentBeaconBlockRoot": "0x" + strings.Repeat("00", 32), "slotNumber": "0x1", "targetGasLimit": "0x1c9c380", "inclusionListTransactions": []string{"0xff", "0x", "0x00"}}
	// Arbitrary or invalid transaction envelopes, including empty DATA, are
	// tolerated by Engine IL input; authenticated signed IL gossip forbids empties.
	for _, value := range []any{nil, []any{1}, []string{"0x0"}, []string{"ff"}, []string{"0xgg"}} {
		base["inclusionListTransactions"] = value
		raw, _ := json.Marshal(base)
		if validateBogotaAttributes(raw) == nil {
			t.Fatal("malformed IL wire accepted", string(raw))
		}
	}
	base["inclusionListTransactions"] = []string{"0xff", "0x", "0x00"}
	raw, _ := json.Marshal(base)
	if err := validateBogotaAttributes(raw); err != nil {
		t.Fatal("opaque Engine IL rejected", err)
	}
	delete(base, "inclusionListTransactions")
	raw, _ = json.Marshal(base)
	if validateBogotaAttributes(raw) == nil {
		t.Fatal("missing required IL accepted")
	}
	for _, backend := range []Backend{nil, &bogotaTestBackend{}} {
		caps := NewEngineAPI(backend).ExchangeCapabilities(nil)
		has := map[string]bool{}
		for _, name := range caps {
			has[name] = true
		}
		want := backend != nil
		if has["engine_newPayloadV6"] != want || has["engine_forkchoiceUpdatedV5"] != want || has["engine_newInclusionListV1"] {
			t.Fatal("capabilities disagree with actual backend", caps)
		}
	}
	api := NewEngineAPI(&mockBackend{})
	response := api.HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"engine_forkchoiceUpdatedV5","params":[{"headBlockHash":"0x` + strings.Repeat("00", 32) + `","safeBlockHash":"0x` + strings.Repeat("00", 32) + `","finalizedBlockHash":"0x` + strings.Repeat("00", 32) + `"},null]}`))
	var reply struct{ Error *jsonrpcError }
	if err := json.Unmarshal(response, &reply); err != nil || reply.Error == nil || reply.Error.Code != UnsupportedForkCode {
		t.Fatalf("alternate backend silently accepted Bogotá: %s", response)
	}
}

func TestGroupEBogotaSatisfactionStatusNullability(t *testing.T) {
	retired := NewEngineAPI(&bogotaTestBackend{}).HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"engine_newInclusionListV1","params":[]}`))
	var reply struct{ Error *jsonrpcError }
	if err := json.Unmarshal(retired, &reply); err != nil || reply.Error == nil || reply.Error.Code != -32601 {
		t.Fatalf("nonstandard legacy update method remained reachable: %s", retired)
	}
	for _, status := range []string{StatusValid, StatusInvalid, StatusSyncing, StatusAccepted, StatusInvalidBlockHash} {
		result := payloadStatusV2(PayloadStatusV1{Status: status}, true)
		raw, _ := json.Marshal(result)
		var fields map[string]json.RawMessage
		json.Unmarshal(raw, &fields)
		if _, ok := fields["inclusionListSatisfied"]; !ok {
			t.Fatal("required nullable field omitted")
		}
		if status == StatusValid {
			if string(fields["inclusionListSatisfied"]) != "true" {
				t.Fatal(string(raw))
			}
		} else if string(fields["inclusionListSatisfied"]) != "null" {
			t.Fatal(string(raw))
		}
	}
}

func TestGroupELighthouseProgressiveSSZRejectsInvalidOffsets(t *testing.T) {
	for _, encoding := range []string{"0x00", "0x040000", "0x08000000", "0x0900000008000000ff", "0x04000000"} {
		raw := strings.Replace(lighthouseAttrsJSON, "0x0800000009000000ff0102", encoding, 1)
		// An offset-only list contains an empty transaction and is legal Engine
		// DATA, even though the same list would be rejected as a signed Heze IL.
		_, err := parsePayloadAttributes(json.RawMessage(raw))
		if encoding == "0x04000000" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatal("invalid SSZ accepted", encoding)
		}
	}
}
