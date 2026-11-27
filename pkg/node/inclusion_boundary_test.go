package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// EIP-7805 defines S after the last transaction. EIP-4895 withdrawals are
// applied later and cannot fund an appended transaction in this same block.
func TestGroupEWithdrawalDoesNotFundSameBlockOmission(t *testing.T) {
	const token = "il-transaction-boundary"
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	to := gethcrypto.PubkeyToAddress(key.PublicKey)
	to[19] ^= 1
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Gas: 50000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1), To: &to, Value: big.NewInt(1)}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	list := []string{"0x" + hex.EncodeToString(raw)}
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	spec := genesis["config"].(map[string]any)
	spec["terminalTotalDifficulty"], spec["amsterdamTime"], spec["bogotaTime"] = 0, 0, 0
	delete(genesis["alloc"].(map[string]any), sender.Hex())
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
	producer := newTestNode(t, makeConfig())
	receiverConfig := makeConfig()
	receiver := newTestNode(t, receiverConfig)
	source, destination := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	beacon := types.Hash{31: 1}
	attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{map[string]any{"index": "0x0", "validatorIndex": "0x0", "address": sender, "amount": "0x3b9aca00"}}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": list}
	var fc payload.ForkchoiceUpdatedResult
	source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, attrs)
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	var block bogotaIntegrationPayload
	source.call(t, "engine_getPayloadV6", &block, fc.PayloadID)
	if len(block.ExecutionPayload.Transactions) != 0 {
		t.Fatal("withdrawal spent before it was processed")
	}
	var status bogotaIntegrationStatus
	destination.call(t, "engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatalf("post-transaction funding boundary: %+v", status)
	}
	state, err := receiver.Blockchain().StateAtBlock(receiver.Blockchain().GetBlock(block.ExecutionPayload.BlockHash))
	if err != nil || state.GetBalance(sender).Cmp(big.NewInt(1000000000000000000)) != 0 {
		t.Fatal("normal withdrawal processing changed", err)
	}
	// Pruned metadata must trigger authenticated prefix reconstruction. A
	// corrupt commitment is an error, never an excuse for ignoring the IL.
	balBytes, err := receiver.Blockchain().BlockAccessList(block.ExecutionPayload.BlockHash)
	if err != nil || len(balBytes) == 0 {
		t.Fatal("missing fixture BAL", err)
	}
	balKey := append([]byte("BAL/"), block.ExecutionPayload.BlockHash[:]...)
	if err := receiver.db.Delete(balKey); err != nil {
		t.Fatal(err)
	}
	destination.call(t, "engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal("pruned BAL changed cutoff", status)
	}
	if err := receiver.db.Put(balKey, []byte{0xc0}); err != nil {
		t.Fatal(err)
	}
	if err := destination.invoke("engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list); err == nil {
		t.Fatal("corrupt BAL became an omission verdict")
	}
	if err := receiver.Blockchain().StoreBlockAccessListRLP(block.ExecutionPayload.BlockHash, balBytes); err != nil {
		t.Fatal(err)
	}
	destination.call(t, "engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal("restored BAL changed cutoff", status)
	}

	// This withdrawal is available to the next block's transaction phase.
	source.call(t, "engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list)
	attrs["timestamp"], attrs["slotNumber"], attrs["withdrawals"] = "0x18", "0x2", []any{}
	source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, attrs)
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	var next bogotaIntegrationPayload
	source.call(t, "engine_getPayloadV6", &next, fc.PayloadID)
	if len(next.ExecutionPayload.Transactions) != 1 {
		t.Fatal("previous-block withdrawal did not fund IL", next.ExecutionPayload.Transactions)
	}
	destination.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	destination.call(t, "engine_newPayloadV6", &status, next.ExecutionPayload, []types.Hash{}, beacon, next.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal("next-block funded IL rejected", status)
	}
	// Persist the previous implementation's wrong final-block-state verdict.
	// Upgrade/restart must recalculate it from its exact retained CL view.
	if err := receiver.Blockchain().WriteBogotaPayload(block.ExecutionPayload.BlockHash, &chain.BogotaPayload{Number: block.ExecutionPayload.BlockNumber, InclusionList: [][]byte{raw}, Satisfied: false}); err != nil {
		t.Fatal(err)
	}
	if err := receiver.Stop(); err != nil {
		t.Fatal(err)
	}
	receiver = newTestNode(t, receiverConfig)
	var restored struct{ PayloadStatus bogotaIntegrationStatus }
	rpc := track4NodeRPC(t, receiver, token)
	rpc.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	if restored.PayloadStatus.Status != "VALID" || restored.PayloadStatus.InclusionListSatisfied == nil || !*restored.PayloadStatus.InclusionListSatisfied {
		t.Fatal("boundary verdict lost on restart", restored)
	}
	// A legacy compact record has lost the CL view needed for correction.
	// Requiring resubmission is safer than inventing a satisfied Boolean.
	if err := receiver.Blockchain().WriteBogotaPayload(block.ExecutionPayload.BlockHash, &chain.BogotaPayload{Number: block.ExecutionPayload.BlockNumber, InclusionList: [][]byte{raw}, Satisfied: false}); err != nil {
		t.Fatal(err)
	}
	if err := receiver.Blockchain().PruneBogotaPayloads(block.ExecutionPayload.BlockNumber + 1); err != nil {
		t.Fatal(err)
	}
	rpc.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	if restored.PayloadStatus.Status != "SYNCING" || restored.PayloadStatus.InclusionListSatisfied != nil {
		t.Fatal("legacy compact verdict reused without original view", restored)
	}
	rpc.call(t, "engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal("resubmitted exact view did not recover verdict", status)
	}
	if err := receiver.Blockchain().PruneBogotaPayloads(block.ExecutionPayload.BlockNumber + 1); err != nil {
		t.Fatal(err)
	}
	rpc.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	if restored.PayloadStatus.Status != "VALID" || restored.PayloadStatus.InclusionListSatisfied == nil || !*restored.PayloadStatus.InclusionListSatisfied {
		t.Fatal("current compact verdict was not retained", restored)
	}
}

// The latest merged EIP-7805 never runs an omitted Frame's VERIFY code.
// Runtime storage validity is therefore not an omission excuse; actual payload
// building still performs full execution. This does not activate Profile 2.
func TestGroupELatest7805FrameRuntimeStorageDoesNotExcuseOmission(t *testing.T) {
	for _, before := range []byte{0, 1} {
		t.Run(fmt.Sprintf("before_%d", before), func(t *testing.T) {
			const token = "il-post-system-boundary"
			post := before ^ 1
			queue := types.WithdrawalRequestAddress
			code := []byte{0x33, 0x73}
			system := types.SystemAddress
			code = append(code, system[:]...)
			code = append(code, 0x14)
			systemJump := len(code) + 1
			code = append(code, 0x60, 0, 0x57, 0x60, 7, 0x54, 0x15)
			approveJump := len(code) + 1
			code = append(code, 0x60, 0, 0x57, 0x5f, 0x5f, 0xfd)
			code[approveJump] = byte(len(code))
			code = append(code, 0x5b, 0x60, 3, 0x5f, 0x5f, 0xaa)
			code[systemJump] = byte(len(code))
			code = append(code, 0x5b, 0x60, post, 0x60, 7, 0x55, 0)
			var genesis map[string]any
			if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
				t.Fatal(err)
			}
			spec := genesis["config"].(map[string]any)
			spec["terminalTotalDifficulty"], spec["amsterdamTime"], spec["bogotaTime"] = 0, 0, 0
			genesis["alloc"].(map[string]any)[queue.Hex()] = map[string]any{"code": "0x" + hex.EncodeToString(code), "balance": "1000000000000000000", "nonce": "0x0", "storage": map[string]string{(types.Hash{31: 7}).Hex(): (types.Hash{31: before}).Hex()}}
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
			frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: queue, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100000000000000000), MaxPriorityFeePerGas: big.NewInt(1)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 20000}}}}
			raw, err := types.EncodeFrameTx8141(frame)
			if err != nil {
				t.Fatal(err)
			}
			list := []string{"0x" + hex.EncodeToString(raw)}
			beacon := types.Hash{31: 1}
			build := func(ts string, il []string) bogotaIntegrationPayload {
				attrs := map[string]any{"timestamp": ts, "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": il}
				var fc payload.ForkchoiceUpdatedResult
				source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, attrs)
				if fc.PayloadID == nil {
					t.Fatal(fc)
				}
				var envelope bogotaIntegrationPayload
				source.call(t, "engine_getPayloadV6", &envelope, fc.PayloadID)
				return envelope
			}
			empty := build("0xc", []string{})
			var verdict bogotaIntegrationStatus
			destination.call(t, "engine_newPayloadV6", &verdict, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, list)
			want := false
			if verdict.Status != "VALID" || verdict.InclusionListSatisfied == nil || *verdict.InclusionListSatisfied != want {
				t.Fatalf("post-system storage changed omission: before=%d status=%+v want=%v", before, verdict, want)
			}
			state, err := receiver.Blockchain().StateAtBlock(receiver.Blockchain().GetBlock(empty.ExecutionPayload.BlockHash))
			if err != nil || state.GetState(queue, types.Hash{31: 7}) != (types.Hash{31: post}) || state.GetNonce(queue) != 0 {
				t.Fatal("omission probe mutated final block state", err)
			}
			built := build("0xd", list)
			count := 0
			if before == 0 {
				count = 1
			}
			if len(built.ExecutionPayload.Transactions) != count {
				t.Fatalf("build/import boundary disagreement: %d transactions, want %d", len(built.ExecutionPayload.Transactions), count)
			}
			destination.call(t, "engine_newPayloadV6", &verdict, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests, list)
			if verdict.Status != "VALID" || verdict.InclusionListSatisfied == nil || *verdict.InclusionListSatisfied != (before == 0) {
				t.Fatal("pre-execution omission check was replaced by runtime validation", verdict)
			}
		})
	}
}
