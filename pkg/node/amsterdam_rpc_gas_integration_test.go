package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eth2030/eth2030/engine/payload"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestAmsterdamRPCGasModesThroughActualHTTP(t *testing.T) {
	const token = "amsterdam-rpc-gas"
	sender := types.HexToAddress("0x1000000000000000000000000000000000000001")
	gasReader := types.HexToAddress("0x1000000000000000000000000000000000000002")
	writer := types.HexToAddress("0x1000000000000000000000000000000000000003")
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	genesis["gasLimit"] = "0x8f0d180"
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["amsterdamTime"], forks["terminalTotalDifficulty"] = 0, 0, 0
	alloc := genesis["alloc"].(map[string]any)
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	signedSender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	alloc[signedSender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil).String()}
	alloc[sender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil).String()}
	alloc[gasReader.Hex()] = map[string]string{"nonce": "0x1", "code": "0x5a5f5260205ff3"}
	alloc[writer.Hex()] = map[string]any{"nonce": "0x1", "code": "0x60015f5500", "storage": map[string]string{types.HexToHash("0x01").Hex(): types.HexToHash("0x07").Hex()}}
	raw, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	root := n.blockchain.CurrentBlock().Root()
	balance := new(big.Int).Set(n.blockchain.State().GetBalance(sender))
	args := map[string]any{"from": sender.Hex(), "to": gasReader.Hex(), "gas": "0x2000000"}
	t.Run("uncapped-call-pays-intrinsic", func(t *testing.T) {
		var output string
		rpc.call(t, "eth_call", &output, args, "latest")
		data, err := hexutil.Decode(output)
		if err != nil {
			t.Fatal(err)
		}
		if new(big.Int).SetBytes(data).Uint64() != 33554432-15000-2 {
			t.Fatalf("GAS observation=%s want uncapped limit minus current intrinsic and GAS", output)
		}
	})
	t.Run("estimate-funds-state-creation", func(t *testing.T) {
		var estimate string
		rpc.call(t, "eth_estimateGas", &estimate, map[string]any{"from": sender.Hex(), "to": writer.Hex()}, "latest")
		gas, err := hexutil.DecodeUint64(estimate)
		if err != nil {
			t.Fatal(err)
		}
		if gas != 125025 {
			t.Fatalf("slot estimate=%d want 15000+3+2+2100+10000+97920", gas)
		}
	})
	t.Run("trace-call-uses-actual-steps", func(t *testing.T) {
		var trace struct {
			Gas         uint64
			Failed      bool
			ReturnValue string
			StructLogs  []json.RawMessage
		}
		rpc.call(t, "debug_traceCall", &trace, args, "latest")
		if trace.Failed || trace.Gas != 15015 || len(trace.StructLogs) != 6 {
			t.Fatalf("actual trace: %+v", trace)
		}
	})
	t.Run("creation-returns-initcode-result", func(t *testing.T) {
		var output string
		rpc.call(t, "eth_call", &output, map[string]any{"from": sender.Hex(), "gas": "0x68e7780", "input": "0x620100005ff3"}, "latest")
		code, err := hex.DecodeString(output[2:])
		if err != nil || len(code) != 65536 {
			t.Fatalf("creation result bytes=%d error=%v", len(code), err)
		}
	})

	t.Run("method-specific-cap-and-overrides", func(t *testing.T) {
		code := "0x5a6301312d0081106011575f5260205ff35b5f5ffd"
		overrides := map[string]any{gasReader.Hex(): map[string]any{"code": code}}
		var output string
		rpc.call(t, "eth_call", &output, args, "latest", overrides)
		data, err := hexutil.Decode(output)
		if err != nil || new(big.Int).SetBytes(data).Uint64() < 20_000_000 {
			t.Fatalf("uncapped high GAS call failed: %s %v", output, err)
		}
		var estimate string
		if err := rpc.invoke("eth_estimateGas", &estimate, args, "latest", overrides); err == nil {
			t.Fatalf("estimate promised impossible >2^24 execution: %s", estimate)
		}
		var trace struct {
			Failed     bool
			StructLogs []json.RawMessage
		}
		rpc.call(t, "debug_traceCall", &trace, args, "latest", map[string]any{"stateOverrides": overrides})
		if trace.Failed || len(trace.StructLogs) == 0 {
			t.Fatal("trace call did not apply uncapped policy/override")
		}
	})
	t.Run("complete-state-replacement", func(t *testing.T) {
		code := "0x6001545f5260205ff3"
		query := map[string]any{"from": sender.Hex(), "to": writer.Hex()}
		for _, field := range []string{"stateDiff", "state"} {
			var output string
			rpc.call(t, "eth_call", &output, query, "latest", map[string]any{writer.Hex(): map[string]any{"code": code, field: map[string]string{}}})
			raw, err := hexutil.Decode(output)
			if err != nil {
				t.Fatal(err)
			}
			want := uint64(7)
			if field == "state" {
				want = 0
			}
			if new(big.Int).SetBytes(raw).Uint64() != want {
				t.Fatalf("%s retained/replaced wrong storage: %s", field, output)
			}
		}
	})
	t.Run("low-budget-and-new-account-estimate", func(t *testing.T) {
		var output string
		low := map[string]any{"from": sender.Hex(), "to": gasReader.Hex(), "gas": "0x3a97"}
		if err := rpc.invoke("eth_call", &output, low, "latest"); err == nil {
			t.Fatal("call below intrinsic accepted")
		}
		newAccount := types.HexToAddress("0x2000000000000000000000000000000000000001")
		var estimate string
		rpc.call(t, "eth_estimateGas", &estimate, map[string]any{"from": sender.Hex(), "to": newAccount.Hex(), "value": "0x1"}, "latest")
		gas, err := hexutil.DecodeUint64(estimate)
		if err != nil || gas != 204600 {
			t.Fatalf("new account estimate=%s %v", estimate, err)
		}
	})
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetBalance(sender).Cmp(balance) != 0 || n.blockchain.State().GetNonce(sender) != 0 || n.blockchain.State().GetState(writer, types.Hash{}) != (types.Hash{}) {
		t.Fatal("RPC simulation mutated canonical state")
	}
	t.Run("historical-transaction-trace-replays-prefix", func(t *testing.T) {
		var hashes []types.Hash
		to := gethcrypto.PubkeyToAddress(key.PublicKey)
		copy(to[:], writer[:])
		for nonce := uint64(0); nonce < 2; nonce++ {
			signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: nonce, Gas: 500000, GasFeeCap: big.NewInt(2_000_000_000), GasTipCap: big.NewInt(1_000_000_000), To: &to, Value: new(big.Int)}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := signed.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var hash types.Hash
			rpc.call(t, "eth_sendRawTransaction", &hash, hexutil.Encode(raw))
			hashes = append(hashes, hash)
		}
		beacon := types.Hash{31: 1}
		attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x8f0d180"}
		var fc payload.ForkchoiceUpdatedResult
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.blockchain.Genesis().Hash()}, attrs)
		if fc.PayloadID == nil {
			t.Fatal("missing payload ID")
		}
		var built struct {
			ExecutionPayload  payload.ExecutionPayloadV5
			ExecutionRequests []string
		}
		rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
		if len(built.ExecutionPayload.Transactions) != 2 {
			t.Fatal("signed prefix/target not included")
		}
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV5", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatalf("import %+v", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
		beforeHead := n.blockchain.CurrentBlock().Hash()
		beforeBalance := new(big.Int).Set(n.blockchain.State().GetBalance(signedSender))
		var receipt struct {
			GasUsed string
			Status  string
		}
		rpc.call(t, "eth_getTransactionReceipt", &receipt, hashes[1])
		if receipt.GasUsed != "0x42d1" || receipt.Status != "0x1" {
			t.Fatalf("second actual slot-write receipt=%+v", receipt)
		}
		var trace struct {
			Gas        uint64
			Failed     bool
			StructLogs []struct {
				Op  string
				Gas uint64
			}
		}
		rpc.call(t, "debug_traceTransaction", &trace, hashes[1])
		if trace.Failed || trace.Gas != 17105 || len(trace.StructLogs) != 4 || trace.StructLogs[0].Gas != 485000 {
			t.Fatalf("historical prefix/target trace: %+v", trace)
		}
		if n.blockchain.CurrentBlock().Hash() != beforeHead || n.blockchain.State().GetBalance(signedSender).Cmp(beforeBalance) != 0 || n.blockchain.State().GetNonce(signedSender) != 2 {
			t.Fatal("trace replay changed canonical chain/fees/nonce")
		}
	})

}
