package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestCanonicalFrame8141ThroughEngineAndRestart(t *testing.T) {
	const token = "canonical-frame-integration"
	var genesis map[string]any
	if err := json.Unmarshal(minimalGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	genesis["gasLimit"] = "0x186a0"
	spec := genesis["config"].(map[string]any)
	spec["terminalTotalDifficulty"], spec["amsterdamTime"], spec["bogotaTime"] = 0, 0, 0
	alloc := genesis["alloc"].(map[string]any)
	keys := []string{fmt.Sprintf("%064x", 1), fmt.Sprintf("%064x", 2)}
	addresses := make([]types.Address, 2)
	for i, text := range keys {
		key, err := gethcrypto.HexToECDSA(text)
		if err != nil {
			t.Fatal(err)
		}
		addresses[i] = types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
		alloc[addresses[i].Hex()] = map[string]string{"balance": "1000000000000000000"}
	}
	target := types.Address{18: 0x89}
	alloc[target.Hex()] = map[string]string{"code": "0x60015f555f5fa000", "balance": "0", "nonce": "1"}
	rawGenesis, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	config := func() *Config {
		cfg := makeTestConfig(t)
		cfg.GenesisPath, cfg.EngineAuthToken = filepath.Join(cfg.DataDir, "genesis.json"), token
		if err := os.WriteFile(cfg.GenesisPath, rawGenesis, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	producer := newTestNode(t, config())
	receiverConfig := config()
	receiver := newTestNode(t, receiverConfig)
	source, destination := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: addresses[0],
		Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100_000_000_000_000_000), MaxPriorityFeePerGas: big.NewInt(1_000_000_000)},
		Frames: []types.Frame8141{
			{Mode: types.ModeVerify, Flags: 2, Limits: types.FrameGasLimits{Execution: 1000}},
			{Mode: types.ModeVerify, Flags: 1, Target: &addresses[1], Limits: types.FrameGasLimits{Execution: 4000}},
			{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 30000, State: 97920}},
		}, Signatures: make([]types.FrameSignature, 2)}
	for i := range frame.Signatures {
		frame.Signatures[i] = types.FrameSignature{Scheme: 1, Signature: make([]byte, 65)}
	}
	frame.Signatures[1].Signer = &addresses[1]
	hash, err := frame.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	for i, text := range keys {
		key, _ := gethcrypto.HexToECDSA(text)
		signature, err := gethcrypto.Sign(hash[:], key)
		if err != nil {
			t.Fatal(err)
		}
		frame.Signatures[i].Signature = append([]byte{signature[64]}, signature[:64]...)
	}
	raw, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	rawHex := "0x" + hex.EncodeToString(raw)
	var txHash types.Hash
	beacon := types.Hash{31: 1}
	var emptyFC payload.ForkchoiceUpdatedResult
	source.call(t, "engine_forkchoiceUpdatedV5", &emptyFC, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, map[string]any{
		"timestamp": "0x6", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x186a0", "inclusionListTransactions": []string{}})
	if emptyFC.PayloadID == nil {
		t.Fatal("could not build empty gas-accounting alternative")
	}
	var empty bogotaIntegrationPayload
	source.call(t, "engine_getPayloadV6", &empty, emptyFC.PayloadID)
	var omitted bogotaIntegrationStatus
	destination.call(t, "engine_newPayloadV6", &omitted, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, []string{rawHex})
	if omitted.Status != "VALID" || omitted.InclusionListSatisfied == nil || *omitted.InclusionListSatisfied {
		t.Fatalf("omitted Frame fits both dimensions and must constrain the block: %+v", omitted)
	}
	destination.call(t, "engine_forkchoiceUpdatedV5", &emptyFC, payload.ForkchoiceStateV1{HeadBlockHash: receiver.Blockchain().Genesis().Hash()}, nil)
	source.call(t, "eth_sendRawTransaction", &txHash, rawHex)
	if txHash != types.NewTransaction(frame).Hash() {
		t.Fatal("pool changed Frame identity")
	}
	attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{},
		"suggestedFeeRecipient": types.Address{19: 0xfe}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x186a0", "inclusionListTransactions": []string{rawHex}}
	var fc payload.ForkchoiceUpdatedResult
	source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, attrs)
	if fc.PayloadStatus.Status != "VALID" || fc.PayloadID == nil {
		t.Fatalf("Frame build FCU: %+v", fc)
	}
	var envelope bogotaIntegrationPayload
	source.call(t, "engine_getPayloadV6", &envelope, fc.PayloadID)
	if len(envelope.ExecutionPayload.Transactions) != 1 {
		t.Fatal("Frame was not built from the inclusion list")
	}
	for _, rpc := range []track4RPC{source, destination} {
		var status bogotaIntegrationStatus
		rpc.call(t, "engine_newPayloadV6", &status, envelope.ExecutionPayload, []types.Hash{}, beacon, envelope.ExecutionRequests, []string{rawHex})
		if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
			t.Fatalf("Frame payload/list verdict: %+v", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: envelope.ExecutionPayload.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" {
			t.Fatalf("Frame FCU: %+v", fc)
		}
	}
	read := func(rpc track4RPC) map[string]any {
		t.Helper()
		var receipt map[string]any
		rpc.call(t, "eth_getTransactionReceipt", &receipt, txHash.Hex())
		if receipt["type"] != "0x6" || receipt["payer"] != addresses[1].Hex() {
			t.Fatalf("missing Frame RPC receipt: %+v", receipt)
		}
		frames, ok := receipt["frameResults"].([]any)
		if !ok || len(frames) != 3 || frames[2].(map[string]any)["stateGasUsed"] != "0x17e80" {
			t.Fatalf("split Frame receipt: %+v", receipt)
		}
		var value types.Hash
		rpc.call(t, "eth_getStorageAt", &value, target.Hex(), "0x0", receipt["blockNumber"])
		if value != (types.Hash{31: 1}) {
			t.Fatal("Frame application state missing")
		}
		return receipt
	}
	expected := read(source)
	if !reflect.DeepEqual(expected, read(destination)) {
		t.Fatal("Engine peers returned different receipts")
	}
	if err := receiver.Stop(); err != nil {
		t.Fatal(err)
	}
	reopened := newTestNode(t, receiverConfig)
	if !reflect.DeepEqual(expected, read(track4NodeRPC(t, reopened, token))) {
		t.Fatal("Frame receipt changed across a node restart")
	}
}
