package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine"
)

func TestGetBlobsV4ActualSyncReadiness(t *testing.T) {
	const token = "blob-readiness"
	cfg := makeTestConfig(t)
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	cfg.EngineAuthToken = token
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	chainConfig := genesis["config"].(map[string]any)
	chainConfig["osakaTime"], chainConfig["amsterdamTime"] = 0, 0
	input, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(cfg.GenesisPath, input, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	hashes := []types.Hash{{0: 1, 31: 99}}
	mask := "0x01000000000000000000000000000000"
	var result json.RawMessage
	rpc.call(t, "engine_getBlobsV4", &result, hashes, mask)
	if string(result) != "[null]" {
		t.Fatalf("healthy empty pool returned %s", result)
	}
	var fc engine.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, engine.ForkchoiceStateV1{HeadBlockHash: types.Hash{31: 99}}, nil)
	if fc.PayloadStatus.Status != engine.StatusSyncing {
		t.Fatalf("unknown target status: %s", fc.PayloadStatus.Status)
	}
	rpc.call(t, "engine_getBlobsV4", &result, hashes, mask)
	if string(result) != "null" {
		t.Fatalf("actual sync target returned %s, want null", result)
	}
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, engine.ForkchoiceStateV1{HeadBlockHash: n.blockchain.Genesis().Hash()}, nil)
	if fc.PayloadStatus.Status != engine.StatusValid {
		t.Fatalf("known target status: %s", fc.PayloadStatus.Status)
	}
	rpc.call(t, "engine_getBlobsV4", &result, hashes, mask)
	if string(result) != "[null]" {
		t.Fatalf("recovered pool returned %s", result)
	}
}
