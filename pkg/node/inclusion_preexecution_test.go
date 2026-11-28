package node

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
)

// The merged EIP-7805 rebase checks pre-execution validity, never candidate
// execution. This VERIFY fails if replayed, but passes pre-execution checks.
func TestGroupELatest7805FramePreExecutionAndVersionMigration(t *testing.T) {
	const token = "latest-7805-pre-execution"
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	spec := genesis["config"].(map[string]any)
	spec["terminalTotalDifficulty"], spec["amsterdamTime"], spec["bogotaTime"] = 0, 0, 0
	sender := types.Address{18: 0x78, 19: 5}
	genesis["alloc"].(map[string]any)[sender.Hex()] = map[string]any{"balance": "1000000000000000000", "nonce": "0x0", "code": "0x5f5ffd"}
	wire, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	makeConfig := func() *Config {
		cfg := makeTestConfig(t)
		cfg.EngineAuthToken, cfg.GenesisPath = token, filepath.Join(cfg.DataDir, "genesis.json")
		if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	producer, receiver := newTestNode(t, makeConfig()), newTestNode(t, makeConfig())
	source, destination := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	beacon := types.Hash{31: 1}
	attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": []string{}}
	var fc payload.ForkchoiceUpdatedResult
	source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, attrs)
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	var envelope bogotaIntegrationPayload
	source.call(t, "engine_getPayloadV6", &envelope, fc.PayloadID)
	frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100000000000000000), MaxPriorityFeePerGas: big.NewInt(1)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 20000}}}}
	raw, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	list := []string{"0x" + hex.EncodeToString(raw)}
	var verdict bogotaIntegrationStatus
	destination.call(t, "engine_newPayloadV6", &verdict, envelope.ExecutionPayload, []types.Hash{}, beacon, envelope.ExecutionRequests, list)
	if verdict.Status != "VALID" || verdict.InclusionListSatisfied == nil || *verdict.InclusionListSatisfied {
		t.Fatalf("candidate EVM failure excused a pre-execution-valid omission: %+v", verdict)
	}
	hash := envelope.ExecutionPayload.BlockHash
	if err := receiver.Blockchain().WriteBogotaPayload(hash, &chain.BogotaPayload{Number: envelope.ExecutionPayload.BlockNumber, InclusionList: [][]byte{raw}, Satisfied: true, SatisfactionVersion: 2}); err != nil {
		t.Fatal(err)
	}
	var restored struct{ PayloadStatus bogotaIntegrationStatus }
	destination.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: hash}, nil)
	if restored.PayloadStatus.Status != "VALID" || restored.PayloadStatus.InclusionListSatisfied == nil || *restored.PayloadStatus.InclusionListSatisfied {
		t.Fatalf("pre-rebase cache survived the changed predicate: %+v", restored)
	}
	if err := receiver.Blockchain().WriteBogotaPayload(hash, &chain.BogotaPayload{Number: envelope.ExecutionPayload.BlockNumber, InclusionList: [][]byte{raw}, Satisfied: true, SatisfactionVersion: 2}); err != nil {
		t.Fatal(err)
	}
	if err := receiver.Blockchain().PruneBogotaPayloads(envelope.ExecutionPayload.BlockNumber + 1); err != nil {
		t.Fatal(err)
	}
	destination.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: hash}, nil)
	if restored.PayloadStatus.Status != "SYNCING" || restored.PayloadStatus.InclusionListSatisfied != nil {
		t.Fatal("compact pre-rebase verdict bypassed view resubmission", restored)
	}
	destination.call(t, "engine_newPayloadV6", &verdict, envelope.ExecutionPayload, []types.Hash{}, beacon, envelope.ExecutionRequests, list)
	if verdict.Status != "VALID" || verdict.InclusionListSatisfied == nil || *verdict.InclusionListSatisfied {
		t.Fatal(verdict)
	}
	if err := receiver.Blockchain().PruneBogotaPayloads(envelope.ExecutionPayload.BlockNumber + 1); err != nil {
		t.Fatal(err)
	}
	destination.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: hash}, nil)
	if restored.PayloadStatus.Status != "VALID" || restored.PayloadStatus.InclusionListSatisfied == nil || *restored.PayloadStatus.InclusionListSatisfied {
		t.Fatal("current compact pre-execution verdict lost", restored)
	}
	db, err := receiver.Blockchain().StateAtBlock(receiver.Blockchain().GetBlock(hash))
	if err != nil {
		t.Fatal(err)
	}
	if db.GetNonce(sender) != 0 || db.GetBalance(sender).Cmp(big.NewInt(1000000000000000000)) != 0 {
		t.Fatal("omission checks changed payload state")
	}
}
