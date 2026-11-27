package node

import (
	"bytes"
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

// A legacy envelope can use the independent state pool before an IL Frame.
// Header gas_used alone and Frame-only receipt sums cannot recover both pools.
func TestGroupEFrameCapacityIncludesOrdinaryStateGas(t *testing.T) {
	const token = "heze-ordinary-state-capacity"
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 3))
	if err != nil {
		t.Fatal(err)
	}
	from := gethcrypto.PubkeyToAddress(key.PublicKey)
	target := from
	target[0] ^= 1
	sender := types.Address{19: 0x90}
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	fields := genesis["config"].(map[string]any)
	fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"] = 0, 0, 0
	genesis["gasLimit"] = "0xf4240"
	alloc := genesis["alloc"].(map[string]any)
	alloc[from.Hex()] = map[string]string{"balance": "10000000000000000000"}
	alloc[target.Hex()] = map[string]string{"balance": "0", "code": "0x600160005500", "nonce": "0x1"}
	alloc[sender.Hex()] = map[string]string{"balance": "10000000000000000000", "code": "0x60035f5faa", "nonce": "0x0"}
	wire, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	beacon := types.Hash{31: 1}
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Gas: 900000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1), To: &target}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	build := func(ts uint64, list []string) bogotaIntegrationPayload {
		var fc payload.ForkchoiceUpdatedResult
		rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": fmt.Sprintf("0x%x", ts), "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0xf4240", "inclusionListTransactions": list})
		if fc.PayloadID == nil {
			t.Fatal(fc)
		}
		var block bogotaIntegrationPayload
		rpc.call(t, "engine_getPayloadV6", &block, fc.PayloadID)
		return block
	}
	first := build(6, []string{"0x" + hex.EncodeToString(ordinary)})
	if len(first.ExecutionPayload.Transactions) != 1 {
		t.Fatal("ordinary storage transaction missing")
	}
	rpc.call(t, "engine_newPayloadV6", new(bogotaIntegrationStatus), first.ExecutionPayload, []types.Hash{}, beacon, first.ExecutionRequests, []string{})
	receipts := n.Blockchain().GetReceipts(first.ExecutionPayload.BlockHash)
	if len(receipts) != 1 || receipts[0].StateGasUsed == 0 || receipts[0].FrameResults8141 != nil {
		t.Fatal("fixture requires actual ordinary state gas", receipts)
	}
	used := receipts[0].StateGasUsed
	if first.ExecutionPayload.GasUsed != max(receipts[0].BlockGasUsed, used) {
		t.Fatal("fixture header does not reflect both dimensions")
	}
	// At the exact execution boundary, the ordinary transaction's state cost
	// must not also be subtracted from the execution pool. Amsterdam gross
	// metadata is required even when the older Glamsterdam flag is absent.
	executionFrame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000000000000000), MaxPriorityFeePerGas: big.NewInt(1)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 10000}}}}
	initialBudget, err := executionFrame.GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	executionFrame.Frames[0].Limits.Execution = first.ExecutionPayload.GasLimit - receipts[0].BlockGasUsed - initialBudget.Intrinsic
	executionRaw, err := types.EncodeFrameTx8141(executionFrame)
	if err != nil {
		t.Fatal(err)
	}
	var exact bogotaIntegrationStatus
	rpc.call(t, "engine_newPayloadV6", &exact, first.ExecutionPayload, []types.Hash{}, beacon, first.ExecutionRequests, []string{"0x" + hex.EncodeToString(executionRaw)})
	if exact.Status != "VALID" || exact.InclusionListSatisfied == nil || *exact.InclusionListSatisfied {
		t.Error("ordinary state gas was charged again as execution gas", exact)
	}
	frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000000000000000), MaxPriorityFeePerGas: big.NewInt(1)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 10000, State: first.ExecutionPayload.GasLimit - used + 1}}}}
	raw, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	list := []string{"0x" + hex.EncodeToString(raw)}
	block := build(7, []string{"0x" + hex.EncodeToString(ordinary), list[0]})
	if len(block.ExecutionPayload.Transactions) != 1 || !bytes.Equal(block.ExecutionPayload.Transactions[0], ordinary) {
		t.Fatal("builder ignored ordinary state reservation")
	}
	check := func(rpc track4RPC) {
		t.Helper()
		var status bogotaIntegrationStatus
		rpc.call(t, "engine_newPayloadV6", &status, block.ExecutionPayload, []types.Hash{}, beacon, block.ExecutionRequests, list)
		if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
			t.Error("ordinary state usage did not constrain omitted Frame", status)
		}
	}
	check(rpc)
	// Force the previously published predicate's wrong Boolean into a retained
	// record. A version-2 upgrade must reevaluate its exact list after restart.
	if err := n.Blockchain().WriteBogotaPayload(block.ExecutionPayload.BlockHash, &chain.BogotaPayload{Number: block.ExecutionPayload.BlockNumber, InclusionList: [][]byte{raw}, Satisfied: false, SatisfactionVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	n = newTestNode(t, &cfg)
	rpc = track4NodeRPC(t, n, token)
	var restored struct{ PayloadStatus bogotaIntegrationStatus }
	rpc.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	if restored.PayloadStatus.Status != "VALID" || restored.PayloadStatus.InclusionListSatisfied == nil || !*restored.PayloadStatus.InclusionListSatisfied {
		t.Error("version-1 wrong verdict survived upgrade", restored)
	}
	check(rpc)
	if err := n.Blockchain().WriteBogotaPayload(block.ExecutionPayload.BlockHash, &chain.BogotaPayload{Number: block.ExecutionPayload.BlockNumber, InclusionList: [][]byte{raw}, Satisfied: false, SatisfactionVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if err := n.Blockchain().PruneBogotaPayloads(block.ExecutionPayload.BlockNumber + 1); err != nil {
		t.Fatal(err)
	}
	rpc.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	if restored.PayloadStatus.Status != "SYNCING" || restored.PayloadStatus.InclusionListSatisfied != nil {
		t.Fatal("version-1 compact record bypassed migration", restored)
	}
	check(rpc)
	if err := n.Blockchain().PruneBogotaPayloads(block.ExecutionPayload.BlockNumber + 1); err != nil {
		t.Fatal(err)
	}
	rpc.call(t, "engine_forkchoiceUpdatedV5", &restored, payload.ForkchoiceStateV1{HeadBlockHash: block.ExecutionPayload.BlockHash}, nil)
	if restored.PayloadStatus.Status != "VALID" || restored.PayloadStatus.InclusionListSatisfied == nil || !*restored.PayloadStatus.InclusionListSatisfied {
		t.Fatal("current compact verdict lost", restored)
	}
	// With only the Frame in an empty block, its reservation fits and omission
	// must remain unsatisfied. The extra ordinary state charge is decisive.
	empty := build(8, []string{})
	var omitted bogotaIntegrationStatus
	rpc.call(t, "engine_newPayloadV6", &omitted, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, list)
	if omitted.Status != "VALID" || omitted.InclusionListSatisfied == nil || *omitted.InclusionListSatisfied {
		t.Fatal("empty-state positive control lost enforcement", omitted)
	}
}
