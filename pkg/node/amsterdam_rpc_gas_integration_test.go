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
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

func TestAmsterdamRPCGasModesThroughActualHTTP(t *testing.T) {
	const token = "amsterdam-rpc-gas"
	sender := types.HexToAddress("0x1000000000000000000000000000000000000001")
	gasReader := types.HexToAddress("0x1000000000000000000000000000000000000002")
	writer := types.HexToAddress("0x1000000000000000000000000000000000000003")
	historyReader := types.HexToAddress("0x1000000000000000000000000000000000000004")
	thresholdReader := types.HexToAddress("0x1000000000000000000000000000000000000005")
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	genesis["gasLimit"] = "0x8f0d180"
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["amsterdamTime"], forks["terminalTotalDifficulty"] = 0, 0, 0
	alloc := genesis["alloc"].(map[string]any)
	for address, code := range map[types.Address][]byte{eips.HistoryStorageAddress: eips.HistoryStorageRuntime(), eips.BeaconRootAddress: eips.BeaconRootsRuntime()} {
		alloc[address.Hex()] = map[string]string{"nonce": "0x1", "code": hexutil.Encode(code)}
	}
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	signedSender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	alloc[signedSender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil).String()}
	alloc[sender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil).String()}
	historyCode := []byte{0x60, 1, 0x5f, 0x52, 0x60, 32, 0x5f, 0x60, 32, 0x5f, 0x73}
	historyCode = append(historyCode, eips.HistoryStorageAddress[:]...)
	historyCode = append(historyCode, 0x5a, 0xfa, 0x50, 0x60, 32, 0x5f, 0xf3)
	alloc[historyReader.Hex()] = map[string]string{"nonce": "0x1", "code": hexutil.Encode(historyCode)}
	alloc[thresholdReader.Hex()] = map[string]string{"nonce": "0x1", "code": "0x5a6301312d0081106011575f5260205ff35b5f5ffd"}
	alloc[gasReader.Hex()] = map[string]string{"nonce": "0x1", "code": "0x5a5f5260205ff3"}
	alloc[writer.Hex()] = map[string]any{"nonce": "0x1", "code": "0x60015f5500", "storage": map[string]string{types.HexToHash("0x01").Hex(): types.HexToHash("0x07").Hex()}}
	raw, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	// This test exercises debug_trace* through the real HTTP path.
	cfg.HTTPModules = append(cfg.HTTPModules, "debug")
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

	t.Run("actual-access-discovery-uses-transaction-policy", func(t *testing.T) {
		var result struct {
			AccessList []struct {
				Address     string
				StorageKeys []string
			}
			GasUsed string
			Error   string
		}
		rpc.call(t, "eth_createAccessList", &result, map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x7a120"}, "latest")
		gas, err := hexutil.DecodeUint64(result.GasUsed)
		if err != nil || gas != 131253 || result.Error != "" || len(result.AccessList) != 1 || !strings.EqualFold(result.AccessList[0].Address, writer.Hex()) || len(result.AccessList[0].StorageKeys) != 1 || result.AccessList[0].StorageKeys[0] != (types.Hash{}).Hex() {
			t.Fatalf("actual converged list/gas: %+v %v", result, err)
		}
		rpc.call(t, "eth_createAccessList", &result, map[string]any{"from": sender.Hex(), "to": thresholdReader.Hex(), "gas": "0x2000000"}, "latest")
		if result.Error == "" {
			t.Fatalf("access discovery promised impossible >2^24 execution: %+v", result)
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

	t.Run("caught-inner-failure-is-observable", func(t *testing.T) {
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
		code = append(code, writer[:]...)
		code = append(code, 0x61, 0x0b, 0xb8, 0xf1, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3)
		overrides := map[string]any{gasReader.Hex(): map[string]any{"code": hexutil.Encode(code)}}
		var output string
		rpc.call(t, "eth_call", &output, args, "latest", overrides)
		if output != (types.Hash{}).Hex() {
			t.Fatalf("inner failure was hidden: %s", output)
		}
		var trace struct {
			Failed      bool
			ReturnValue string
			StructLogs  []json.RawMessage
		}
		rpc.call(t, "debug_traceCall", &trace, args, "latest", map[string]any{"stateOverrides": overrides})
		if trace.Failed || trace.ReturnValue != (types.Hash{}).Hex() || len(trace.StructLogs) == 0 {
			t.Fatal("outer success was confused with inner failure")
		}
	})

	t.Run("calldata-floor-and-flat-access-list", func(t *testing.T) {
		for _, data := range []string{"0x" + strings.Repeat("00", 128), "0x" + strings.Repeat("01", 128)} {
			var estimate string
			rpc.call(t, "eth_estimateGas", &estimate, map[string]any{"from": sender.Hex(), "to": gasReader.Hex(), "input": data}, "latest")
			gas, err := hexutil.DecodeUint64(estimate)
			if err != nil || gas != 23192 {
				t.Fatalf("uniform calldata floor=%s %v", estimate, err)
			}
		}
		var estimate string
		rpc.call(t, "eth_estimateGas", &estimate, map[string]any{"from": sender.Hex(), "to": gasReader.Hex(), "accessList": []any{map[string]any{"address": writer.Hex(), "storageKeys": []string{(types.Hash{}).Hex()}}}}, "latest")
		gas, err := hexutil.DecodeUint64(estimate)
		if err != nil || gas != 23243 {
			t.Fatalf("flat access-list intrinsic=%s %v", estimate, err)
		}
	})
	t.Run("signed-authorization-input", func(t *testing.T) {
		authorityKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
		if err != nil {
			t.Fatal(err)
		}
		target := gethcrypto.PubkeyToAddress(authorityKey.PublicKey)
		copy(target[:], gasReader[:])
		auth, err := gethtypes.SignSetCode(authorityKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: target, Nonce: 0})
		if err != nil {
			t.Fatal(err)
		}
		fields := map[string]string{"chainId": "0x539", "address": gasReader.Hex(), "nonce": "0x0", "yParity": hexutil.EncodeUint64(uint64(auth.V)), "r": hexutil.EncodeBig(auth.R.ToBig()), "s": hexutil.EncodeBig(auth.S.ToBig())}
		var estimate string
		rpc.call(t, "eth_estimateGas", &estimate, map[string]any{"from": sender.Hex(), "to": gasReader.Hex(), "authorizationList": []any{fields}}, "latest")
		gas, err := hexutil.DecodeUint64(estimate)
		if err != nil || gas != 250621 {
			t.Fatalf("runtime authorization estimate=%s %v", estimate, err)
		}
		authority := types.Address(gethcrypto.PubkeyToAddress(authorityKey.PublicKey))
		if n.blockchain.State().Exist(authority) {
			t.Fatal("estimated authorization escaped into canonical state")
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

		for _, method := range []string{"debug_traceBlockByHash", "debug_traceBlockByNumber"} {
			selector := beforeHead.Hex()
			if method == "debug_traceBlockByNumber" {
				selector = "0x1"
			}
			var blockTraces []struct {
				TxHash string
				Result struct {
					Gas        uint64
					Failed     bool
					StructLogs []json.RawMessage
				}
			}
			rpc.call(t, method, &blockTraces, selector)
			if len(blockTraces) != 2 || blockTraces[0].Result.Gas != 125025 || blockTraces[1].Result.Gas != 17105 || len(blockTraces[1].Result.StructLogs) != 4 || blockTraces[1].TxHash != hashes[1].Hex() {
				t.Fatalf("actual %s: %+v", method, blockTraces)
			}
		}
		if n.blockchain.CurrentBlock().Hash() != beforeHead || n.blockchain.State().GetBalance(signedSender).Cmp(beforeBalance) != 0 || n.blockchain.State().GetNonce(signedSender) != 2 {
			t.Fatal("trace replay changed canonical chain/fees/nonce")
		}
		t.Run("preblock-system-replay", func(t *testing.T) {
			copy(to[:], historyReader[:])
			signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: 2, Gas: 500000, GasFeeCap: big.NewInt(2_000_000_000), GasTipCap: big.NewInt(1_000_000_000), To: &to, Value: new(big.Int)}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := signed.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var hash types.Hash
			rpc.call(t, "eth_sendRawTransaction", &hash, hexutil.Encode(raw))
			attrs["timestamp"], attrs["slotNumber"] = "0x18", "0x2"
			rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: beforeHead}, attrs)
			if fc.PayloadID == nil {
				t.Fatal("missing second payload ID")
			}
			var next struct {
				ExecutionPayload  payload.ExecutionPayloadV5
				ExecutionRequests []string
			}
			rpc.call(t, "engine_getPayloadV6", &next, fc.PayloadID)
			if len(next.ExecutionPayload.Transactions) != 1 {
				t.Fatal("history consumer not included")
			}
			rpc.call(t, "engine_newPayloadV5", &status, next.ExecutionPayload, []types.Hash{}, beacon, next.ExecutionRequests)
			if status.Status != "VALID" {
				t.Fatalf("system consumer import: %+v", status)
			}
			rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: next.ExecutionPayload.BlockHash}, nil)
			var trace struct {
				Failed      bool
				ReturnValue string
				StructLogs  []json.RawMessage
			}
			rpc.call(t, "debug_traceTransaction", &trace, hash)
			if trace.Failed || trace.ReturnValue != beforeHead.Hex() || len(trace.StructLogs) == 0 {
				t.Fatalf("trace omitted current pre-block history write: %+v", trace)
			}
			if n.blockchain.CurrentBlock().Hash() != next.ExecutionPayload.BlockHash || n.blockchain.State().GetNonce(signedSender) != 3 {
				t.Fatal("history replay adopted its temporary state")
			}
		})

	})

}

func TestRPCGasModesKeepHistoricalForkContext(t *testing.T) {
	const token = "rpc-gas-fork-context"
	address := types.HexToAddress("0x1000000000000000000000000000000000000042")
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["amsterdamTime"] = 0, 12
	genesis["alloc"].(map[string]any)[address.Hex()] = map[string]string{"nonce": "0x1", "code": "0x5a5f5260205ff3"}
	raw, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	// This test exercises debug_trace* through the real HTTP path.
	cfg.HTTPModules = append(cfg.HTTPModules, "debug")
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	check := func(at string, intrinsic uint64) {
		t.Helper()
		var output string
		rpc.call(t, "eth_call", &output, map[string]any{"to": address.Hex(), "gas": "0x186a0"}, at)
		data, err := hexutil.Decode(output)
		if err != nil || new(big.Int).SetBytes(data).Uint64() != 100000-intrinsic-2 {
			t.Fatalf("historical %s GAS=%s %v", at, output, err)
		}
		var estimate string
		rpc.call(t, "eth_estimateGas", &estimate, map[string]any{"to": address.Hex()}, at)
		gas, err := hexutil.DecodeUint64(estimate)
		if err != nil || gas < intrinsic+15 || (intrinsic != 21000 && gas != intrinsic+15) || gas > (intrinsic+15)*1015/1000 {
			t.Fatalf("historical %s estimate=%s %v", at, estimate, err)
		}
		// Classic estimation allows a small verified upper-bound margin;
		// the returned allowance still has to execute on that exact header.
		rpc.call(t, "eth_call", &output, map[string]any{"to": address.Hex(), "gas": estimate}, at)
	}
	check("earliest", 21000)
	beacon := types.Hash{31: 1}
	attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380"}
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.blockchain.Genesis().Hash()}, attrs)
	if fc.PayloadID == nil {
		t.Fatal("missing fork-boundary payload ID")
	}
	var built struct {
		ExecutionPayload  payload.ExecutionPayloadV5
		ExecutionRequests []string
	}
	rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	var status payload.PayloadStatusV1
	rpc.call(t, "engine_newPayloadV5", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests)
	if status.Status != "VALID" {
		t.Fatalf("boundary import: %+v", status)
	}
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
	check("latest", 15000)
	check("0x0", 21000)
}
