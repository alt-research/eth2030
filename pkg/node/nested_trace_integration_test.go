package node

import (
	"encoding/json"
	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNestedDimensionalTraceThroughActualHTTP(t *testing.T) {
	const token = "nested-dimensional-trace"
	raw, sender, parent, child := simulationV1Genesis(t, 0)
	var genesis map[string]any
	if err := json.Unmarshal(raw, &genesis); err != nil {
		t.Fatal(err)
	}
	parentCode := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
	parentCode = append(parentCode, child[:]...)
	parentCode = append(parentCode, 0x5a, 0xf1, 0x50, 0)
	for address, code := range map[types.Address][]byte{parent: parentCode, child: {0x60, 1, 0x60, 0, 0x55, 0}} {
		genesis["alloc"].(map[string]any)[address.Hex()].(map[string]any)["code"] = hexutil.Encode(code)
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
	args := map[string]any{"from": sender.Hex(), "to": parent.Hex(), "gas": "0x7a120"}
	t.Run("nested-call-and-state-gas-spill", func(t *testing.T) {
		var out map[string]json.RawMessage
		rpc.call(t, "debug_traceCall", &out, args, "latest", map[string]any{"tracer": "callTracer"})
		var typ string
		_ = json.Unmarshal(out["type"], &typ)
		if typ != "CALL" {
			t.Fatalf("named tracer did not produce an actual root frame: %s", out["type"])
		}
		var frames []map[string]json.RawMessage
		if err := json.Unmarshal(out["calls"], &frames); err != nil {
			t.Fatal(err)
		}
		if len(frames) != 1 {
			t.Fatalf("missing nested frame: %+v", out)
		}
		var gas string
		_ = json.Unmarshal(frames[0]["gasUsed"], &gas)
		if gas != "0x1adca" {
			t.Fatalf("child gas excludes spilled state: %s", gas)
		}
		for _, field := range []string{"regularGasUsed", "stateGasUsed", "gasRefund"} {
			if out[field] == nil {
				t.Fatalf("missing root gas dimension %s", field)
			}
			if frames[0][field] != nil {
				t.Fatalf("root-only field %s leaked into child", field)
			}
		}
		for _, frame := range []map[string]json.RawMessage{out, frames[0]} {
			if string(frame["stateGasReservoir"]) != `"0x0"` {
				t.Fatal("zero reservoir was omitted")
			}
			if string(frame["value"]) != `"0x0"` {
				t.Fatal("zero value was omitted")
			}
		}
	})
	t.Run("opcode-state-gas-is-separated", func(t *testing.T) {
		var out struct {
			StateGasUsed   *uint64
			RegularGasUsed *uint64
			GasRefund      *uint64
			StructLogs     []struct {
				Op                string
				GasCost           uint64
				StateGasCost      *int64
				StateGasReservoir *uint64
			}
		}
		rpc.call(t, "debug_traceCall", &out, args, "latest")
		if out.StateGasUsed == nil || *out.StateGasUsed != 97920 || out.RegularGasUsed == nil || out.GasRefund == nil {
			t.Fatalf("missing transaction dimensions: %+v", out)
		}
		found := false
		for _, step := range out.StructLogs {
			if step.Op == "SSTORE" {
				found = true
				if step.GasCost != 12100 || step.StateGasCost == nil || *step.StateGasCost != 97920 || step.StateGasReservoir != nil {
					t.Fatalf("opcode dimensions: %+v", step)
				}
			}
		}
		if !found {
			t.Fatal("actual storage opcode omitted")
		}
	})
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetState(child, types.Hash{}) != (types.Hash{}) || n.blockchain.State().GetNonce(sender) != 0 {
		t.Fatal("trace adopted its execution")
	}
}
