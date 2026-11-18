package node

import (
	"encoding/hex"
	"encoding/json"
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
	alloc[sender.Hex()] = map[string]string{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil).String()}
	alloc[gasReader.Hex()] = map[string]string{"nonce": "0x1", "code": "0x5a5f5260205ff3"}
	alloc[writer.Hex()] = map[string]string{"nonce": "0x1", "code": "0x60015f5500"}
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
		if trace.Failed || trace.Gas != 15017 || len(trace.StructLogs) != 6 {
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
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetBalance(sender).Cmp(balance) != 0 || n.blockchain.State().GetNonce(sender) != 0 || n.blockchain.State().GetState(writer, types.Hash{}) != (types.Hash{}) {
		t.Fatal("RPC simulation mutated canonical state")
	}
}
