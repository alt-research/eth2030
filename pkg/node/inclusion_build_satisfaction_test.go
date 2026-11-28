package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestGroupEBuilderRejectsUnsatisfiedViewAndRetries(t *testing.T) {
	const token = "heze-build-must-satisfy"
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address{18: 0x78, 19: 6}
	// Nonempty ordinary calldata clears slot zero. Empty VERIFY calldata
	// approves only after that state transition; otherwise it reverts.
	code := []byte{0x36, 0x15, 0x60, 9, 0x57, 0x5f, 0x5f, 0x55, 0x00, 0x5b, 0x5f, 0x54, 0x15, 0x60, 19, 0x57, 0x5f, 0x5f, 0xfd, 0x5b, 0x60, 3, 0x5f, 0x5f, 0xaa}
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	fields := genesis["config"].(map[string]any)
	fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"] = 0, 0, 0
	alloc := genesis["alloc"].(map[string]any)
	alloc[sender.Hex()] = map[string]any{"balance": "1000000000000000000", "code": "0x" + hex.EncodeToString(code), "storage": map[string]string{(types.Hash{}).Hex(): (types.Hash{31: 1}).Hex()}}
	alloc[gethcrypto.PubkeyToAddress(key.PublicKey).Hex()] = map[string]string{"balance": "10000000000000000000"}
	wire, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken, cfg.GenesisPath = token, filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100000000000000000), MaxPriorityFeePerGas: big.NewInt(1)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 20000}}}}
	raw, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	list := []string{"0x" + hex.EncodeToString(raw)}
	beacon := types.Hash{31: 1}
	attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": list}
	head := payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, head, attrs)
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	firstID := *fc.PayloadID
	var built bogotaIntegrationPayload
	if err := rpc.invoke("engine_getPayloadV6", &built, fc.PayloadID); err == nil || !strings.Contains(err.Error(), "does not satisfy inclusion list") {
		t.Fatal("builder published an unsatisfied requested IL instead of failing", err)
	}
	to := gethcommon.Address(sender)
	setter, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Gas: 1000000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1), To: &to, Data: []byte{1}}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := setter.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var submitted types.Hash
	rpc.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(encoded))
	rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, head, attrs)
	if fc.PayloadID == nil || *fc.PayloadID != firstID {
		t.Fatal("failed build did not retry the same immutable IL request", fc)
	}
	rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	if len(built.ExecutionPayload.Transactions) != 2 || !bytes.Equal(built.ExecutionPayload.Transactions[0], encoded) || !bytes.Equal(built.ExecutionPayload.Transactions[1], raw) {
		t.Fatal("retry did not execute the state transition before the requested Frame")
	}
	var verdict bogotaIntegrationStatus
	rpc.call(t, "engine_newPayloadV6", &verdict, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests, list)
	if verdict.Status != "VALID" || verdict.InclusionListSatisfied == nil || !*verdict.InclusionListSatisfied {
		t.Fatal("successful retry did not satisfy its requested view", verdict)
	}
}
