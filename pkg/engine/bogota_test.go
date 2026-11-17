package engine

import (
	"encoding/json"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

type bogotaTestBackend struct {
	mockBackend
	checkedHash types.Hash
	checkedList [][]byte
	satisfied   bool
}

func (*bogotaTestBackend) IsHogota(timestamp uint64) bool { return timestamp >= 300 }
func (b *bogotaTestBackend) CheckPayloadInclusionList(hash types.Hash, list [][]byte) (bool, error) {
	b.checkedHash, b.checkedList = hash, list
	return b.satisfied, nil
}

func (b *bogotaTestBackend) PayloadInclusionListSatisfied(types.Hash) bool { return b.satisfied }

func TestBogotaV6SeparatesPayloadAndListValidity(t *testing.T) {
	hash := types.HexToHash("0x42")
	backend := &bogotaTestBackend{}
	backend.processBlockV5Fn = func(p *ExecutionPayloadV5, _ []types.Hash, _ types.Hash, _ [][]byte) (PayloadStatusV1, error) {
		return PayloadStatusV1{Status: StatusValid, LatestValidHash: &p.BlockHash}, nil
	}
	api := NewEngineAPI(backend)
	p := ExecutionPayloadV5{
		ExecutionPayloadV4: ExecutionPayloadV4{ExecutionPayloadV3: makeTestPayloadV3(hash, 300)},
		BlockAccessList:    json.RawMessage(`[]`),
	}
	pBytes, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := json.Marshal(types.HexToHash("0x99"))
	params := []json.RawMessage{pBytes, json.RawMessage(`[]`), root, json.RawMessage(`[]`), json.RawMessage(`["0xff"]`)}
	for _, satisfied := range []bool{false, true} {
		backend.satisfied = satisfied
		result, err := api.handleNewPayloadV6(params)
		if err != nil {
			t.Fatal(err)
		}
		status := result.(bogotaPayloadStatus)
		if status.Status != StatusValid || status.InclusionListSatisfied == nil || *status.InclusionListSatisfied != satisfied {
			t.Fatalf("unexpected status %+v, want VALID with satisfaction %v", status, satisfied)
		}
		if backend.checkedHash != hash || len(backend.checkedList) != 1 || string(backend.checkedList[0]) != "\xff" {
			t.Fatalf("inclusion input changed: %s %x", backend.checkedHash, backend.checkedList)
		}
	}
	if _, err := api.handleNewPayloadV5(params[:4]); err == nil {
		t.Fatal("Amsterdam method accepted a Bogotá payload")
	}
	params[4] = json.RawMessage(`null`)
	if _, err := api.handleNewPayloadV6(params); err == nil || err.Code != InvalidParamsCode {
		t.Fatalf("expected invalid params for null list, got %v", err)
	}
}
