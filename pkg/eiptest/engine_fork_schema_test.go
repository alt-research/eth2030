package eiptest_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/eiptest"
	"github.com/eth2030/eth2030/engine"
)

func TestEngineCanonicalForkSchemaThroughActualHTTP(t *testing.T) {
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkAmsterdam, 24))
	p := s.Engine()
	var capabilities []string
	if err := p.Call("engine_exchangeCapabilities", &capabilities, []string{}); err != nil {
		t.Fatal(err)
	}
	for _, method := range capabilities {
		if method == "engine_exchangeCapabilities" {
			t.Fatal("capability exchange must not advertise itself")
		}
	}
	assertCode := func(method string, code int, args ...any) {
		t.Helper()
		var result json.RawMessage
		err := p.Call(method, &result, args...)
		if err == nil || !strings.Contains(err.Error(), fmt.Sprint(code)) {
			t.Fatalf("%s: expected %d, got %v", method, code, err)
		}
	}
	for _, timestamp := range []uint64{12, 24} {
		attrs := p.Attributes()
		attrs.Timestamp, attrs.SlotNumber = timestamp, 0
		// Presence, not nonzero contents, is the Engine requirement.
		attrs.ParentBeaconBlockRoot = types.Hash{}
		id, err := p.FCUWith(attrs)
		if err != nil {
			t.Fatal(err)
		}
		built, err := p.GetPayload(id)
		if err != nil {
			t.Fatal(err)
		}
		getMethod := "engine_getPayloadV5"
		if timestamp == 24 {
			getMethod = "engine_getPayloadV6"
		}
		assertCode(getMethod, engine.InvalidParamsCode, strings.TrimPrefix(id.String(), "0x"))
		var wire struct {
			ExecutionPayload map[string]json.RawMessage `json:"executionPayload"`
			BlobsBundle      map[string]json.RawMessage `json:"blobsBundle"`
		}
		if err := p.Call(getMethod, &wire, id); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"blobs", "commitments", "proofs"} {
			if string(wire.BlobsBundle[field]) != "[]" {
				t.Fatalf("empty Osaka bundle %s must be []", field)
			}
		}
		for _, version := range []int{3, 4, 5, 6} {
			method := fmt.Sprintf("engine_getPayloadV%d", version)
			if method != getMethod {
				assertCode(method, engine.UnsupportedForkCode, id)
			}
		}
		if timestamp == 12 {
			for _, field := range []string{"blockAccessList", "slotNumber", "executionRequests"} {
				if _, exists := wire.ExecutionPayload[field]; exists {
					t.Fatalf("Osaka leaked Amsterdam/outer field %s", field)
				}
			}
			candidate := make(map[string]json.RawMessage)
			for field, value := range wire.ExecutionPayload {
				candidate[field] = value
			}
			candidate["blockAccessList"], candidate["slotNumber"] = json.RawMessage(`"0xc0"`), json.RawMessage(`"0x0"`)
			assertCode("engine_newPayloadV5", engine.UnsupportedForkCode, candidate, []types.Hash{}, types.Hash{}, []string{})
		} else {
			if string(wire.ExecutionPayload["slotNumber"]) != `"0x0"` {
				t.Fatal("present zero slot was lost")
			}
			assertCode("engine_newPayloadV3", engine.UnsupportedForkCode, built.ExecutionPayload.ExecutionPayloadV3, []types.Hash{}, types.Hash{})
			assertCode("engine_newPayloadV4", engine.UnsupportedForkCode, built.ExecutionPayload.ExecutionPayloadV3, []types.Hash{}, types.Hash{}, []string{})
			// Every required field is independently absent/null, using an otherwise
			// valid built payload so unrelated invalid input cannot mask the check.
			fields := []string{"parentHash", "feeRecipient", "stateRoot", "receiptsRoot", "logsBloom", "prevRandao", "blockNumber", "gasLimit", "gasUsed", "timestamp", "extraData", "baseFeePerGas", "blockHash", "transactions", "withdrawals", "blobGasUsed", "excessBlobGas", "blockAccessList", "slotNumber"}
			for _, field := range fields {
				original := wire.ExecutionPayload[field]
				delete(wire.ExecutionPayload, field)
				assertCode("engine_newPayloadV5", engine.InvalidParamsCode, wire.ExecutionPayload, []types.Hash{}, types.Hash{}, []string{})
				wire.ExecutionPayload[field] = json.RawMessage("null")
				assertCode("engine_newPayloadV5", engine.InvalidParamsCode, wire.ExecutionPayload, []types.Hash{}, types.Hash{}, []string{})
				wire.ExecutionPayload[field] = original
			}
			for _, malformed := range []struct{ field, value string }{
				{"slotNumber", `"0x10000000000000000"`}, {"slotNumber", `"0x00"`}, {"timestamp", `24`},
				{"timestamp", `"0x10000000000000018"`}, {"gasLimit", `"0x10000000001c9c380"`},
				{"gasLimit", `"-0x1"`}, {"gasUsed", `"0x+1"`}, {"baseFeePerGas", `"0x1` + strings.Repeat("0", 64) + `"`},
				{"parentHash", `"0x01"`}, {"feeRecipient", `"0x00"`}, {"logsBloom", `"0x"`},
				{"transactions", `["0x0"]`}, {"withdrawals", `[null]`},
			} {
				original := wire.ExecutionPayload[malformed.field]
				wire.ExecutionPayload[malformed.field] = json.RawMessage(malformed.value)
				assertCode("engine_newPayloadV5", engine.InvalidParamsCode, wire.ExecutionPayload, []types.Hash{}, types.Hash{}, []string{})
				wire.ExecutionPayload[malformed.field] = original
			}
			assertCode("engine_newPayloadV5", engine.InvalidParamsCode, wire.ExecutionPayload, nil, types.Hash{}, []string{})
			assertCode("engine_newPayloadV5", engine.InvalidParamsCode, wire.ExecutionPayload, []string{"0x01"}, types.Hash{}, []string{})
			assertCode("engine_newPayloadV5", engine.InvalidParamsCode, wire.ExecutionPayload, []types.Hash{}, nil, []string{})
			originalBAL := wire.ExecutionPayload["blockAccessList"]
			wire.ExecutionPayload["blockAccessList"] = json.RawMessage(`"0x00"`)
			var invalid engine.PayloadStatusV1
			if err := p.Call("engine_newPayloadV5", &invalid, wire.ExecutionPayload, []types.Hash{}, types.Hash{}, []string{}); err != nil || invalid.Status != engine.StatusInvalid || invalid.LatestValidHash != nil {
				t.Fatalf("malformed BAL RLP must be INVALID with null ancestor: %v %+v", err, invalid)
			}
			wire.ExecutionPayload["blockAccessList"] = originalBAL
		}
		status, err := p.Process(built.ExecutionPayload)
		if err != nil || status.Status != engine.StatusValid {
			t.Fatalf("valid %s import: %v %+v", getMethod, err, status)
		}
	}
	// Invalid build attributes must not roll back a valid forkchoice update.
	head := p.HeadHash()
	state := engine.ForkchoiceStateV1{HeadBlockHash: head, SafeBlockHash: head, FinalizedBlockHash: head}
	attrs := p.Attributes()
	raw, _ := json.Marshal(attrs)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"timestamp", "prevRandao", "suggestedFeeRecipient", "withdrawals", "parentBeaconBlockRoot", "slotNumber", "targetGasLimit"} {
		original := fields[field]
		delete(fields, field)
		assertCode("engine_forkchoiceUpdatedV4", -38003, state, fields, nil)
		if p.Backend().GetFinalizedHash() != head || p.Backend().GetSafeHash() != head {
			t.Fatal("attribute failure rolled back valid forkchoice")
		}
		fields[field] = json.RawMessage("null")
		assertCode("engine_forkchoiceUpdatedV4", -38003, state, fields, nil)
		fields[field] = original
	}
	delete(fields, "slotNumber")
	var syncing engine.ForkchoiceUpdatedResult
	if err := p.Call("engine_forkchoiceUpdatedV4", &syncing, engine.ForkchoiceStateV1{HeadBlockHash: types.Hash{99}}, fields, nil); err != nil || syncing.PayloadStatus.Status != engine.StatusSyncing {
		t.Fatalf("unknown head must take precedence over attributes: %v %+v", err, syncing)
	}
}
