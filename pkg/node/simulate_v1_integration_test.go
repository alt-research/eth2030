package node

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestSimulateV1ActualHTTP(t *testing.T) {
	const token = "simulate-v1-http"
	sender := types.HexToAddress("0x1000000000000000000000000000000000000011")
	writer := types.HexToAddress("0x2000000000000000000000000000000000000022")
	threshold := types.HexToAddress("0x3000000000000000000000000000000000000033")
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["amsterdamTime"] = 0, 0
	alloc := genesis["alloc"].(map[string]any)
	alloc[sender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil).String()}
	alloc[writer.Hex()] = map[string]string{"nonce": "0x1", "code": "0x36156009575f355f555b5f545f5260205ff3"}
	alloc[threshold.Hex()] = map[string]string{"nonce": "0x1", "code": "0x5a6301312d0081106011575f5260205ff35b5f5ffd"}
	for address, code := range map[types.Address][]byte{eips.HistoryStorageAddress: eips.HistoryStorageRuntime(), eips.BeaconRootAddress: eips.BeaconRootsRuntime()} {
		alloc[address.Hex()] = map[string]string{"nonce": "0x1", "code": hexutil.Encode(code)}
	}
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
		err := rpc.invoke("eth_simulateV1", &result, map[string]any{"validation": true, "blockStateCalls": []any{map[string]any{"calls": []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "nonce": "0x1"}}}}}, "latest")
		failure, ok := err.(*track4RPCError)
		if !ok || failure.Code != -38011 {
			t.Fatalf("wrong nonce error: %v", err)
		}
	})
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetBalance(sender).Cmp(initial) != 0 || n.blockchain.State().GetNonce(sender) != 0 || n.blockchain.State().GetState(writer, types.Hash{}) != (types.Hash{}) {
		t.Fatal("stateful simulation changed canonical state")
	}
}
