package node

import (
	"encoding/json"
	"github.com/eth2030/eth2030/engine/payload"
	sdkparams "github.com/ethereum/go-ethereum/params"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestSimulateV1ActualHTTP(t *testing.T) {
	const token = "simulate-v1-http"
	raw, sender, writer, threshold := simulationV1Genesis(t, 0)
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	root := n.blockchain.CurrentBlock().Root()
	initial := new(big.Int).Set(n.blockchain.State().GetBalance(sender))
	type callResult struct {
		ReturnData string
		GasUsed    string
		Status     string
		Logs       []json.RawMessage
		Error      *track4RPCError
	}
	type blockResult struct {
		Hash                types.Hash
		ParentHash          types.Hash
		Number              string
		GasUsed             string
		StateRoot           types.Hash
		BlockAccessListHash *types.Hash
		Calls               []callResult
		Transactions        []json.RawMessage
	}
	t.Run("sequential-calls-and-blocks", func(t *testing.T) {
		var results []blockResult
		write := map[string]any{"from": sender.Hex(), "to": writer.Hex(), "input": "0x07"}
		read := map[string]any{"from": sender.Hex(), "to": writer.Hex()}
		rpc.call(t, "eth_simulateV1", &results, map[string]any{"blockStateCalls": []any{map[string]any{"calls": []any{write, read}}, map[string]any{"calls": []any{read}}}}, "latest")
		if len(results) != 2 || len(results[0].Calls) != 2 || len(results[1].Calls) != 1 {
			t.Fatalf("incomplete sequential result: %+v", results)
		}
		// CALLDATALOAD pads the single input byte on the right.
		want := types.Hash{0: 7}.Hex()
		for _, block := range results {
			if block.BlockAccessListHash == nil || block.StateRoot == root {
				t.Fatal("simulated block omitted actual state/BAL commitment")
			}
			for _, call := range block.Calls {
				if call.Status != "0x1" || call.ReturnData != want {
					t.Fatalf("state did not carry across calls/blocks: %+v", call)
				}
			}
		}
		if results[1].ParentHash != results[0].Hash || results[0].ParentHash != n.blockchain.Genesis().Hash() {
			t.Fatal("simulated ancestry is not chained")
		}
	})
	t.Run("validation-selects-transaction-gas", func(t *testing.T) {
		for _, validation := range []bool{false, true} {
			var result []blockResult
			args := map[string]any{"validation": validation, "blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"baseFeePerGas": "0x0"}, "calls": []any{map[string]any{"from": sender.Hex(), "to": threshold.Hex(), "gas": "0x1c9c380"}}}}}
			rpc.call(t, "eth_simulateV1", &result, args, "latest")
			if len(result) != 1 || len(result[0].Calls) != 1 {
				t.Fatal("missing gas-mode result")
			}
			want := "0x1"
			if validation {
				want = "0x0"
			}
			if result[0].Calls[0].Status != want {
				t.Fatalf("validation=%t result=%+v", validation, result)
			}
		}
	})
	t.Run("explicit-nonce-validation", func(t *testing.T) {
		var result []blockResult
		err := rpc.invoke("eth_simulateV1", &result, map[string]any{"validation": true, "blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"baseFeePerGas": "0x0"}, "calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "nonce": "0x1"}}}}}, "latest")
		failure, ok := err.(*track4RPCError)
		if !ok || failure.Code != -38011 {
			t.Fatalf("wrong nonce error: %v", err)
		}
	})

	t.Run("default-reservations-and-full-transactions", func(t *testing.T) {
		var out []struct {
			GasUsed      string
			Transactions []struct {
				Gas   string
				Nonce string
				From  string
			}
			Calls []callResult
		}
		call := map[string]any{"from": sender.Hex(), "to": sender.Hex()}
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"returnFullTransactions": true, "blockStateCalls": []any{map[string]any{"calls": []any{call, call}}}}, "latest")
		if len(out) != 1 || len(out[0].Transactions) != 2 || len(out[0].Calls) != 2 || out[0].GasUsed != "0x5dc0" {
			t.Fatalf("default consecutive result: %+v", out)
		}
		for index, tx := range out[0].Transactions {
			gas, err := hexutil.DecodeUint64(tx.Gas)
			want := uint64(30_000_000) - uint64(index)*12000
			if err != nil || gas != want || tx.Nonce != hexutil.EncodeUint64(uint64(index)) || !strings.EqualFold(tx.From, sender.Hex()) || out[0].Calls[index].Status != "0x1" {
				t.Fatalf("call %d reservation/from/nonce: %+v", index, out[0])
			}
		}
	})
	t.Run("global-gas-budget-and-runtime-failures", func(t *testing.T) {
		var out []blockResult
		block := map[string]any{"stateOverrides": map[string]any{writer.Hex(): map[string]any{"code": "0xfe"}}, "calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x1c9c380"}}}
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{block, block}}, "latest")
		if len(out) != 2 {
			t.Fatal("missing budget results")
		}
		for i, block := range out {
			want := uint64(30_000_000)
			if i == 1 {
				want = 20_000_000
			}
			if len(block.Calls) != 1 || block.Calls[0].Status != "0x0" || block.Calls[0].Error == nil || block.Calls[0].Error.Code != -32015 || block.Calls[0].GasUsed != hexutil.EncodeUint64(want) || block.GasUsed != hexutil.EncodeUint64(want) {
				t.Fatalf("global budget call %d: %+v", i, block)
			}
		}
	})
	t.Run("precompile-code-and-relocation", func(t *testing.T) {
		original := types.HexToAddress("0x02")
		destination := types.HexToAddress("0x4200000000000000000000000000000000000042")
		var out []blockResult
		overrides := map[string]any{original.Hex(): map[string]any{"code": "0x602a5f5260205ff3", "movePrecompileToAddress": destination.Hex()}}
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": overrides, "calls": []any{map[string]any{"to": original.Hex()}, map[string]any{"to": destination.Hex()}}}}}, "latest")
		if len(out) != 1 || len(out[0].Calls) != 2 || out[0].Calls[0].ReturnData != (types.Hash{31: 42}).Hex() || out[0].Calls[1].ReturnData != "0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
			t.Fatalf("precompile override/relocation: %+v", out)
		}
	})

	t.Run("relocation-preserves-original-warming", func(t *testing.T) {
		original := types.HexToAddress("0x02")
		destination := types.HexToAddress("0x4200000000000000000000000000000000000042")
		var costs []uint64
		for _, moved := range []bool{false, true} {
			target := original
			if moved {
				target = destination
			}
			code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
			code = append(code, target[:]...)
			code = append(code, 0x5a, 0xfa, 0x50, 0)
			overrides := map[string]any{writer.Hex(): map[string]any{"code": hexutil.Encode(code)}}
			if moved {
				overrides[original.Hex()] = map[string]any{"movePrecompileToAddress": destination.Hex()}
			}
			var out []blockResult
			rpc.call(t, "eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": overrides, "calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x7a120"}}}}}, "latest")
			if len(out) != 1 || len(out[0].Calls) != 1 || out[0].Calls[0].Status != "0x1" {
				t.Fatal("warm/cold precompile calls failed")
			}
			gas, err := hexutil.DecodeUint64(out[0].Calls[0].GasUsed)
			if err != nil {
				t.Fatal(err)
			}
			costs = append(costs, gas)
		}
		if costs[0] != 15175 || costs[1] != 18075 {
			t.Fatalf("relocation changed original warming: %v", costs)
		}
		var output string
		rpc.call(t, "eth_call", &output, map[string]any{"to": original.Hex()}, "latest")
		if output != "0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
			t.Fatal("request mutated the global precompile map")
		}
	})

	t.Run("uncapped-call-calldata-floor", func(t *testing.T) {
		for _, validation := range []bool{false, true} {
			var out []blockResult
			call := map[string]any{"from": sender.Hex(), "to": sender.Hex(), "input": "0x" + strings.Repeat("00", 262145), "gas": "0x1c9c380"}
			options := map[string]any{"validation": validation, "blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"baseFeePerGas": "0x0"}, "calls": []any{call}}}}
			err := rpc.invoke("eth_simulateV1", &out, options, "latest")
			if validation {
				failure, ok := err.(*track4RPCError)
				if !ok || failure.Code != -38013 {
					t.Fatalf("strict floor cap: %v", err)
				}
			} else {
				if err != nil || len(out) != 1 || len(out[0].Calls) != 1 || out[0].Calls[0].Status != "0x1" || out[0].Calls[0].GasUsed != hexutil.EncodeUint64(12000+262145*64) {
					t.Fatalf("uncapped floor result: %+v %v", out, err)
				}
			}
		}
	})

	t.Run("uint256-balance-overrides", func(t *testing.T) {
		selfdestruct := append([]byte{0x73}, sender[:]...)
		selfdestruct = append(selfdestruct, 0xff)
		query := append([]byte{0x73}, writer[:]...)
		query = append(query, 0x31, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3)
		overrides := map[string]any{sender.Hex(): map[string]any{"balance": "0x2"}, writer.Hex(): map[string]any{"balance": "0x" + strings.Repeat("ff", 32), "code": hexutil.Encode(selfdestruct)}, threshold.Hex(): map[string]any{"code": hexutil.Encode(query)}}
		var out []blockResult
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": overrides, "calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "value": "0x1", "gas": "0x7a120"}, map[string]any{"from": sender.Hex(), "to": threshold.Hex(), "gas": "0x7a120"}}}}}, "latest")
		if len(out) != 1 || len(out[0].Calls) != 2 || out[0].Calls[0].Status != "0x1" || out[0].Calls[1].ReturnData != (types.Hash{}).Hex() {
			t.Fatalf("balance arithmetic did not match uint256: %+v", out)
		}
	})
	t.Run("client-call-count-boundaries", func(t *testing.T) {
		for _, count := range []int{5000, 5001} {
			calls := make([]any, count)
			for i := range calls {
				calls[i] = map[string]any{"from": sender.Hex(), "to": sender.Hex(), "gas": "0x0"}
			}
			var out []blockResult
			err := rpc.invoke("eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"calls": calls}}}, "latest")
			want := -38013
			if count == 5001 {
				want = -38026
			}
			failure, ok := err.(*track4RPCError)
			if !ok || failure.Code != want {
				t.Fatalf("call count %d: %v", count, err)
			}
		}
	})
	t.Run("block-gaps-and-basefee-default", func(t *testing.T) {
		var out []struct {
			Number              string
			Timestamp           string
			BaseFee             string `json:"baseFeePerGas"`
			ParentHash          types.Hash
			Hash                types.Hash
			Calls               []callResult
			BlockAccessListHash *types.Hash
		}
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"baseFeePerGas": "0x64"}}, map[string]any{"blockOverrides": map[string]any{"number": "0x4"}}}}, "latest")
		if len(out) != 4 {
			t.Fatalf("gap blocks omitted: %+v", out)
		}
		for i, block := range out {
			if block.Number != hexutil.EncodeUint64(uint64(i+1)) || block.Timestamp != hexutil.EncodeUint64(uint64(i+1)*12) || block.BlockAccessListHash == nil || (i > 0 && block.ParentHash != out[i-1].Hash) {
				t.Fatalf("gap context %d: %+v", i, block)
			}
			want := "0x0"
			if i == 0 {
				want = "0x64"
			}
			if block.BaseFee != want {
				t.Fatal("non-validation base fee inherited instead of defaulting to zero")
			}
		}
	})

	t.Run("simulated-blockhash-context", func(t *testing.T) {
		var out []blockResult
		call := func(number byte) map[string]any {
			return map[string]any{"from": sender.Hex(), "to": writer.Hex(), "input": (types.Hash{31: number}).Hex(), "gas": "0x7a120"}
		}
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": map[string]any{writer.Hex(): map[string]any{"code": "0x5f35405f5260205ff3"}}, "calls": []any{call(0)}}, map[string]any{"calls": []any{call(1), call(0), call(2)}}}}, "latest")
		genesisHash := n.blockchain.Genesis().Hash().Hex()
		if len(out) != 2 || out[0].Calls[0].ReturnData != genesisHash || out[1].Calls[0].ReturnData != out[0].Hash.Hex() || out[1].Calls[1].ReturnData != genesisHash || out[1].Calls[2].ReturnData != (types.Hash{}).Hex() {
			t.Fatalf("simulated BLOCKHASH ancestry: %+v", out)
		}
	})
	t.Run("contract-sender-with-validation", func(t *testing.T) {
		var out []blockResult
		rpc.call(t, "eth_simulateV1", &out, map[string]any{"validation": true, "blockStateCalls": []any{map[string]any{"blockOverrides": map[string]any{"baseFeePerGas": "0x0"}, "calls": []any{map[string]any{"from": writer.Hex(), "to": sender.Hex(), "gas": "0xc350"}}}}}, "latest")
		if len(out) != 1 || len(out[0].Calls) != 1 || out[0].Calls[0].Status != "0x1" {
			t.Fatalf("contract sender rejected: %+v", out)
		}
	})
	t.Run("invalid-input-boundaries", func(t *testing.T) {
		good := func() map[string]any {
			return map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x7a120"}
		}
		cases := []struct {
			name       string
			code       int
			validation bool
			block      map[string]any
		}{
			{"number", -38020, false, map[string]any{"blockOverrides": map[string]any{"number": "0x0"}}},
			{"timestamp", -38021, false, map[string]any{"blockOverrides": map[string]any{"time": "0x0"}}},
			{"gap-span", -38026, false, map[string]any{"blockOverrides": map[string]any{"number": "0x101"}}},
			{"unknown-block-key", -32602, false, map[string]any{"blockOverrides": map[string]any{"unsupported": "0x1"}}},
			{"post-merge-difficulty", -32602, false, map[string]any{"blockOverrides": map[string]any{"difficulty": "0x0"}}},
			{"unknown-state-key", -32602, false, map[string]any{"stateOverrides": map[string]any{sender.Hex(): map[string]any{"unsupported": "0x1"}}}},
			{"state-key-aliases", -32602, false, map[string]any{"stateOverrides": map[string]any{sender.Hex(): map[string]any{"code": "0x", "Code": "0x"}}}},
			{"nonce-low", -38010, true, map[string]any{"blockOverrides": map[string]any{"baseFeePerGas": "0x0"}, "stateOverrides": map[string]any{sender.Hex(): map[string]any{"nonce": "0x1"}}, "calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "nonce": "0x0"}}}},
			{"intrinsic", -38013, false, map[string]any{"calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x3a97"}}}},
			{"fee", -32602, true, map[string]any{"calls": []any{good()}}},
			{"value", -38014, false, map[string]any{"calls": []any{map[string]any{"to": writer.Hex(), "value": "0x1"}}}},
			{"block-gas", -38015, false, map[string]any{"calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x1c9c381"}}}},
			{"nonce-max", -32603, true, map[string]any{"stateOverrides": map[string]any{sender.Hex(): map[string]any{"nonce": "0xffffffffffffffff"}}, "calls": []any{good()}}},
			{"move-self", -38022, false, map[string]any{"stateOverrides": map[string]any{types.HexToAddress("0x02").Hex(): map[string]any{"movePrecompileToAddress": types.HexToAddress("0x02").Hex()}}}},
			{"initcode-limit", -38025, false, map[string]any{"calls": []any{map[string]any{"from": sender.Hex(), "input": "0x" + strings.Repeat("00", 131073)}}}},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				var out []blockResult
				err := rpc.invoke("eth_simulateV1", &out, map[string]any{"validation": test.validation, "blockStateCalls": []any{test.block}}, "latest")
				failure, ok := err.(*track4RPCError)
				if !ok || failure.Code != test.code {
					t.Fatalf("want error %d got %v", test.code, err)
				}
			})
		}
	})
	t.Run("independent-complete-block", func(t *testing.T) { verifySimulationAgainstSDK(t, rpc, raw, sender, writer) })
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetBalance(sender).Cmp(initial) != 0 || n.blockchain.State().GetNonce(sender) != 0 || n.blockchain.State().GetState(writer, types.Hash{}) != (types.Hash{}) {
		t.Fatal("stateful simulation changed canonical state")
	}
}

func simulationV1Genesis(t *testing.T, activation uint64) ([]byte, types.Address, types.Address, types.Address) {
	t.Helper()
	sender := types.HexToAddress("0x1000000000000000000000000000000000000011")
	writer := types.HexToAddress("0x2000000000000000000000000000000000000022")
	threshold := types.HexToAddress("0x3000000000000000000000000000000000000033")
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["amsterdamTime"], forks["terminalTotalDifficulty"] = 0, activation, 0
	alloc := genesis["alloc"].(map[string]any)
	alloc[sender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil).String()}
	alloc[writer.Hex()] = map[string]string{"balance": "0", "nonce": "0x1", "code": "0x36156009575f355f555b5f545f5260205ff3"}
	alloc[threshold.Hex()] = map[string]string{"balance": "0", "nonce": "0x1", "code": "0x5a6301312d0081106011575f5260205ff35b5f5ffd"}
	for address, code := range map[types.Address][]byte{eips.HistoryStorageAddress: eips.HistoryStorageRuntime(), eips.BeaconRootAddress: eips.BeaconRootsRuntime()} {
		alloc[address.Hex()] = map[string]string{"balance": "0", "nonce": "0x1", "code": hexutil.Encode(code)}
	}
	inhibitor := "0x" + strings.Repeat("ff", 32)
	for address, code := range map[types.Address][]byte{types.WithdrawalRequestAddress: sdkparams.WithdrawalQueueCode, types.ConsolidationRequestAddress: sdkparams.ConsolidationQueueCode} {
		alloc[address.Hex()] = map[string]any{"balance": "0", "nonce": "0x1", "code": hexutil.Encode(code), "storage": map[string]string{(types.Hash{}).Hex(): inhibitor}}
	}
	raw, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	return raw, sender, writer, threshold
}

func TestSimulateV1TransferLogsAcrossForkAndTimeout(t *testing.T) {
	const token = "simulate-v1-transfer"
	raw, sender, writer, _ := simulationV1Genesis(t, 24)
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	root := n.blockchain.CurrentBlock().Root()
	receiver := types.HexToAddress("0x4000000000000000000000000000000000000044")
	type result struct {
		Hash         types.Hash
		ReceiptsRoot types.Hash
		LogsBloom    string
		SlotNumber   *string
		Calls        []struct {
			Status string
			Logs   []struct {
				Address  string
				LogIndex string
			}
		}
	}
	t.Run("virtual-logs-revert-and-native-fork-logs", func(t *testing.T) {
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x60, 1, 0x73}
		code = append(code, receiver[:]...)
		code = append(code, 0x5a, 0xf1, 0x50, 0x5f, 0x5f, 0xfd)
		transfer := map[string]any{"from": sender.Hex(), "to": receiver.Hex(), "value": "0x1", "gas": "0x7a120"}
		reverting := map[string]any{"from": sender.Hex(), "to": writer.Hex(), "value": "0x1", "gas": "0x7a120"}
		blocks := []any{map[string]any{"stateOverrides": map[string]any{writer.Hex(): map[string]any{"code": hexutil.Encode(code)}}, "calls": []any{transfer, reverting}}, map[string]any{"calls": []any{transfer}}}
		outputs := make([][]result, 2)
		for i, trace := range []bool{false, true} {
			rpc.call(t, "eth_simulateV1", &outputs[i], map[string]any{"traceTransfers": trace, "blockStateCalls": blocks}, "latest")
		}
		for i, out := range outputs {
			if len(out) != 2 || len(out[0].Calls) != 2 || out[0].Calls[0].Status != "0x1" || out[0].Calls[1].Status != "0x0" || len(out[0].Calls[1].Logs) != 0 {
				t.Fatalf("reverted transfer leaked: %+v", out)
			}
			want := 0
			if i == 1 {
				want = 1
			}
			if len(out[0].Calls[0].Logs) != want {
				t.Fatal("pre-fork virtual log presence incorrect")
			}
			if i == 1 && out[0].Calls[0].Logs[0].Address != "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" {
				t.Fatal("wrong synthetic transfer emitter")
			}
			if len(out[1].Calls) != 1 || len(out[1].Calls[0].Logs) != 1 || out[1].Calls[0].Logs[0].Address != "0xfffffffffffffffffffffffffffffffffffffffe" || out[1].Calls[0].Logs[0].LogIndex != "0x0" {
				t.Fatalf("protocol log duplicated or missing: %+v", out[1])
			}
			for _, block := range out {
				if block.SlotNumber != nil {
					t.Fatal("unknown parent slot was invented")
				}
			}
		}
		for i := range outputs[0] {
			if outputs[0][i].Hash != outputs[1][i].Hash || outputs[0][i].ReceiptsRoot != outputs[1][i].ReceiptsRoot || outputs[0][i].LogsBloom != outputs[1][i].LogsBloom {
				t.Fatal("virtual logs changed consensus commitments")
			}
		}
	})
	t.Run("bounded-system-call-work", func(t *testing.T) {
		var out []result
		err := rpc.invoke("eth_simulateV1", &out, map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": map[string]any{eips.BeaconRootAddress.Hex(): map[string]any{"code": "0x5b5f56"}}}, map[string]any{"blockOverrides": map[string]any{"number": "0x100"}}}}, "latest")
		failure, ok := err.(*track4RPCError)
		if !ok || failure.Code != -32016 {
			t.Fatalf("system-call deadline not enforced: %v", err)
		}
	})
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetNonce(sender) != 0 || n.blockchain.State().Exist(receiver) {
		t.Fatal("fork/timeout simulation changed canonical state")
	}
}

func TestSimulateV1HashSelectedBranchAncestry(t *testing.T) {
	const token = "simulate-v1-branch"
	raw, sender, writer, _ := simulationV1Genesis(t, 0)
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	build := func(parent types.Hash, timestamp, slot, amount uint64) payload.ExecutionPayloadV5 {
		t.Helper()
		beacon := types.Hash{31: 1}
		withdrawal := map[string]any{"index": "0x1", "validatorIndex": "0x1", "address": sender.Hex(), "amount": hexutil.EncodeUint64(amount)}
		attrs := map[string]any{"timestamp": hexutil.EncodeUint64(timestamp), "slotNumber": hexutil.EncodeUint64(slot), "withdrawals": []any{withdrawal}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380"}
		var fc payload.ForkchoiceUpdatedResult
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
		if fc.PayloadID == nil {
			t.Fatal("branch payload ID missing")
		}
		var out struct {
			ExecutionPayload  payload.ExecutionPayloadV5
			ExecutionRequests []string
		}
		rpc.call(t, "engine_getPayloadV6", &out, fc.PayloadID)
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV5", &status, out.ExecutionPayload, []types.Hash{}, beacon, out.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatalf("branch import=%+v", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: out.ExecutionPayload.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" {
			t.Fatal("branch selection failed")
		}
		return out.ExecutionPayload
	}
	genesis := n.blockchain.Genesis().Hash()
	a1 := build(genesis, 12, 1, 1)
	a2 := build(a1.BlockHash, 24, 2, 1)
	b1 := build(genesis, 18, 1, 2)
	b2 := build(b1.BlockHash, 30, 2, 2)
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: a2.BlockHash}, nil)
	if fc.PayloadStatus.Status != "VALID" || n.blockchain.CurrentBlock().Hash() != a2.BlockHash {
		t.Fatal("canonical control branch not restored")
	}
	var out []struct {
		Calls []struct {
			ReturnData string
			Status     string
		}
	}
	options := map[string]any{"blockStateCalls": []any{map[string]any{"stateOverrides": map[string]any{writer.Hex(): map[string]any{"code": "0x5f35405f5260205ff3"}}, "calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "input": (types.Hash{31: 1}).Hex()}}}}}
	rpc.call(t, "eth_simulateV1", &out, options, b2.BlockHash.Hex())
	if len(out) != 1 || len(out[0].Calls) != 1 || out[0].Calls[0].Status != "0x1" || out[0].Calls[0].ReturnData != b1.BlockHash.Hex() || b1.BlockHash == a1.BlockHash {
		t.Fatalf("hash-selected simulation used canonical ancestry: %+v", out)
	}
	if n.blockchain.CurrentBlock().Hash() != a2.BlockHash || n.blockchain.CurrentBlock().Root() != a2.StateRoot {
		t.Fatal("side-branch simulation changed canonical head/state")
	}
}
