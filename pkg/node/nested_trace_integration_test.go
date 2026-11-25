package node

import (
	"encoding/json"
	"fmt"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"os"
	"path/filepath"
	"strings"
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
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	signer := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	genesis["gasLimit"] = "0x3938700"
	genesis["alloc"].(map[string]any)[signer.Hex()] = map[string]any{"balance": new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil).String()}
	logParent := types.HexToAddress("0x5000000000000000000000000000000000000055")
	logChild := types.HexToAddress("0x6000000000000000000000000000000000000066")
	logCode := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
	logCode = append(logCode, logChild[:]...)
	logCode = append(logCode, 0x5a, 0xf1, 0x50, 0x5f, 0x5f, 0xa0, 0)
	for address, code := range map[types.Address][]byte{logParent: logCode, logChild: {0x60, 1, 0x60, 0, 0x55, 0x5f, 0x5f, 0xa0, 0}} {
		genesis["alloc"].(map[string]any)[address.Hex()] = map[string]any{"balance": "0", "nonce": "0x1", "code": hexutil.Encode(code)}
	}
	raw, err = json.Marshal(genesis)
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

	t.Run("creation-frames-and-failure-address-omission", func(t *testing.T) {
		for _, revert := range []bool{false, true} {
			input := "0x60015ff3"
			if revert {
				input = "0x5f5ffd"
			}
			var out map[string]json.RawMessage
			rpc.call(t, "debug_traceCall", &out, map[string]any{"from": sender.Hex(), "gas": "0x7a120", "input": input}, "latest", map[string]any{"tracer": "callTracer"})
			if string(out["type"]) != `"CREATE"` || out["input"] == nil {
				t.Fatal("root creation was not traced")
			}
			if revert {
				if out["to"] != nil || string(out["error"]) != `"execution reverted"` {
					t.Fatalf("failed creation address leaked: %+v", out)
				}
			} else if out["to"] == nil || string(out["output"]) != `"0x00"` {
				t.Fatalf("creation output/address: %+v", out)
			}
		}
		// The initcode extends memory: tracing must not read past the old memory.
		one := []byte{0x5f, 0x60, 1, 0x5f, 0x5f, 0xf5, 0x50}
		twice := append(append(append([]byte(nil), one...), one...), 0)
		var out struct{ Calls []map[string]json.RawMessage }
		config := map[string]any{"tracer": "callTracer", "stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(twice)}}}
		rpc.call(t, "debug_traceCall", &out, args, "latest", config)
		if len(out.Calls) != 2 || string(out.Calls[0]["type"]) != `"CREATE2"` || out.Calls[0]["to"] == nil || out.Calls[1]["to"] != nil || out.Calls[1]["error"] == nil || string(out.Calls[1]["gasUsed"]) != string(out.Calls[1]["gas"]) {
			t.Fatalf("CREATE2/collision frame semantics: %+v", out)
		}
		var steps struct {
			StructLogs []struct {
				Op           string
				StateGasCost *int64
			}
		}
		rpc.call(t, "debug_traceCall", &steps, args, "latest", map[string]any{"stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(append(one, 0))}}})
		found := false
		for _, step := range steps.StructLogs {
			if step.Op == "CREATE2" {
				found = true
				if step.StateGasCost == nil || *step.StateGasCost != 183600 {
					t.Fatal("CREATE2 account state charge omitted")
				}
			}
		}
		if !found {
			t.Fatal("CREATE2 opcode omitted")
		}
	})
	t.Run("callcode-delegate-and-selfdestruct-frames", func(t *testing.T) {
		makeCall := func(kind byte, value byte) []byte {
			body := []byte{0x5f, 0x5f, 0x5f, 0x5f}
			if kind == 0xf2 {
				body = append(body, 0x60, value)
			}
			body = append(body, 0x73)
			body = append(body, child[:]...)
			return append(body, 0x5a, kind, 0x50, 0)
		}
		var missing struct{ Calls []map[string]json.RawMessage }
		rpc.call(t, "debug_traceCall", &missing, args, "latest", map[string]any{"tracer": "callTracer", "stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(makeCall(0xf2, 1))}}})
		if len(missing.Calls) != 1 || string(missing.Calls[0]["type"]) != `"CALLCODE"` || string(missing.Calls[0]["gasUsed"]) != `"0x0"` || missing.Calls[0]["error"] == nil {
			t.Fatal("CALLCODE balance failure frame missing")
		}
		for _, kind := range []byte{0xf2, 0xf4} {
			callArgs := map[string]any{"from": sender.Hex(), "to": parent.Hex(), "gas": "0x7a120", "value": "0x7"}
			var out struct {
				Calls []struct {
					Type  string
					From  string
					To    string
					Value string
					Logs  []struct{ Address string }
				}
			}
			override := map[string]any{parent.Hex(): map[string]any{"balance": "0xa", "code": hexutil.Encode(makeCall(kind, 1))}, child.Hex(): map[string]any{"code": "0x5f5fa000"}}
			rpc.call(t, "debug_traceCall", &out, callArgs, "latest", map[string]any{"tracer": "callTracer", "tracerConfig": map[string]any{"withLog": true}, "stateOverrides": override})
			want := "0x1"
			if kind == 0xf4 {
				want = "0x7"
			}
			if len(out.Calls) != 1 || out.Calls[0].From != parent.Hex() || out.Calls[0].To != child.Hex() || out.Calls[0].Value != want || len(out.Calls[0].Logs) != 1 || out.Calls[0].Logs[0].Address != parent.Hex() {
				t.Fatalf("execution-context trace kind=%x: %+v", kind, out)
			}
		}
		destruction := append([]byte{0x73}, sender[:]...)
		destruction = append(destruction, 0xff)
		var out struct {
			Calls []struct{ Calls []map[string]json.RawMessage }
		}
		rpc.call(t, "debug_traceCall", &out, args, "latest", map[string]any{"tracer": "callTracer", "stateOverrides": map[string]any{child.Hex(): map[string]any{"code": hexutil.Encode(destruction), "balance": "0x1"}}})
		if len(out.Calls) != 1 || len(out.Calls[0].Calls) != 1 || string(out.Calls[0].Calls[0]["type"]) != `"SELFDESTRUCT"` || string(out.Calls[0].Calls[0]["gas"]) != `"0x0"` || string(out.Calls[0].Calls[0]["gasUsed"]) != `"0x0"` {
			t.Fatal("SELFDESTRUCT frame missing or charged")
		}
	})

	t.Run("call-opcode-execution-cost-excludes-account-state", func(t *testing.T) {
		fresh := types.HexToAddress("0x7100000000000000000000000000000000000071")
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x60, 1, 0x73}
		code = append(code, fresh[:]...)
		code = append(code, 0x5a, 0xf1, 0x50, 0)
		var out struct {
			StructLogs []struct {
				Op           string
				Gas          uint64
				GasCost      uint64
				StateGasCost *int64
			}
		}
		rpc.call(t, "debug_traceCall", &out, args, "latest", map[string]any{"stateOverrides": map[string]any{parent.Hex(): map[string]any{"balance": "0x2", "code": hexutil.Encode(code)}}})
		found := false
		for _, step := range out.StructLogs {
			if step.Op == "CALL" {
				found = true
				if step.StateGasCost == nil || *step.StateGasCost != 183600 || step.GasCost+uint64(*step.StateGasCost) > step.Gas {
					t.Fatalf("account state was counted twice in opcode cost: %+v", step)
				}
			}
		}
		if !found {
			t.Fatal("value call opcode missing")
		}
	})
	t.Run("return-data-is-local-to-each-frame", func(t *testing.T) {
		grand := types.HexToAddress("0x4000000000000000000000000000000000000044")
		call := func(target types.Address, output bool) []byte {
			code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f}
			if output {
				code = []byte{0x60, 32, 0x5f, 0x5f, 0x5f, 0x5f}
			}
			code = append(code, 0x73)
			code = append(code, target[:]...)
			return append(code, 0x5a, 0xf1, 0x50)
		}
		code := append(call(child, false), call(grand, true)...)
		code = append(code, 0x60, 32, 0x5f, 0xf3)
		override := map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(code)}, child.Hex(): map[string]any{"code": "0x602a5f5260205ff3"}, grand.Hex(): map[string]any{"nonce": "0x1", "code": "0x3d5f5260205ff3"}}
		var output string
		rpc.call(t, "eth_call", &output, args, "latest", override)
		if output != (types.Hash{}).Hex() {
			t.Fatalf("new child inherited its parent's return-data buffer: %s", output)
		}
	})

	t.Run("log-order-options-and-child-rollback", func(t *testing.T) {
		call := func(target types.Address) []byte {
			code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
			code = append(code, target[:]...)
			return append(code, 0x5a, 0xf1, 0x50)
		}
		parentBody := append(call(child), 0x5f, 0x5f, 0xa0, 0)
		for _, revert := range []bool{false, true} {
			for _, onlyTop := range []bool{false, true} {
				childBody := []byte{0x5f, 0x5f, 0xa0}
				if revert {
					childBody = append(childBody, 0x5f, 0x5f, 0xfd)
				} else {
					childBody = append(childBody, 0)
				}
				var out struct {
					Logs []struct {
						Index    string
						Position string
					}
					Calls []struct {
						Error string
						Logs  []json.RawMessage
					}
				}
				config := map[string]any{"tracer": "callTracer", "tracerConfig": map[string]any{"withLog": true, "onlyTopCall": onlyTop}, "stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(parentBody)}, child.Hex(): map[string]any{"code": hexutil.Encode(childBody)}}}
				rpc.call(t, "debug_traceCall", &out, args, "latest", config)
				wantIndex, wantPosition := "0x1", "0x1"
				if revert {
					wantIndex = "0x0"
				}
				if onlyTop {
					wantPosition = "0x0"
				}
				if len(out.Logs) != 1 || out.Logs[0].Index != wantIndex || out.Logs[0].Position != wantPosition {
					t.Fatalf("onlyTop=%t revert=%t global log position: %+v", onlyTop, revert, out)
				}
				if onlyTop {
					if len(out.Calls) != 0 {
						t.Fatal("onlyTop collected child frames")
					}
				} else {
					if len(out.Calls) != 1 {
						t.Fatal("child frame missing")
					}
					if revert {
						if out.Calls[0].Error != "execution reverted" || len(out.Calls[0].Logs) != 0 {
							t.Fatal("reverted child logs survived")
						}
					} else if len(out.Calls[0].Logs) != 1 {
						t.Fatal("successful child log omitted")
					}
				}
			}
		}
	})
	t.Run("opcode-options-and-post-store-snapshot", func(t *testing.T) {
		var disabled map[string]json.RawMessage
		rpc.call(t, "debug_traceCall", &disabled, args, "latest", map[string]any{"disableStack": true, "disableStorage": true, "enableMemory": true, "enableReturnData": true, "limit": 3})
		var steps []map[string]json.RawMessage
		_ = json.Unmarshal(disabled["structLogs"], &steps)
		if len(steps) != 3 {
			t.Fatalf("step limit=%d", len(steps))
		}
		for _, step := range steps {
			if step["stack"] != nil || step["storage"] != nil || step["memory"] == nil || string(step["returnData"]) != `"0x"` {
				t.Fatalf("trace option presence: %+v", step)
			}
		}
		var out struct {
			StructLogs []struct {
				Op      string
				Storage map[string]string
			}
		}
		rpc.call(t, "debug_traceCall", &out, args, "latest")
		seen := false
		for _, step := range out.StructLogs {
			if step.Op == "SSTORE" {
				seen = true
				if step.Storage[(types.Hash{}).Hex()] != (types.Hash{31: 1}).Hex() {
					t.Fatalf("SSTORE snapshot showed pre-state: %+v", step)
				}
			} else if step.Storage != nil {
				t.Fatal("storage leaked onto a non-storage opcode")
			}
		}
		if !seen {
			t.Fatal("storage opcode not observed")
		}
	})

	t.Run("static-opcode-fails-before-gas-charge", func(t *testing.T) {
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
		code = append(code, child[:]...)
		code = append(code, 0x5a, 0xfa, 0x50, 0)
		var out struct {
			StructLogs []struct {
				Op           string
				GasCost      uint64
				Error        string
				StateGasCost *int64
			}
		}
		rpc.call(t, "debug_traceCall", &out, args, "latest", map[string]any{"stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(code)}}})
		seen := false
		for _, step := range out.StructLogs {
			if step.Op == "SSTORE" {
				seen = true
				if step.GasCost != 0 || step.Error == "" || step.StateGasCost != nil {
					t.Fatalf("write-protected opcode was charged: %+v", step)
				}
			}
		}
		if !seen {
			t.Fatal("failing step was omitted")
		}
	})
	t.Run("static-failure-has-a-frame-and-no-value", func(t *testing.T) {
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
		code = append(code, child[:]...)
		code = append(code, 0x5a, 0xfa, 0x50, 0)
		var out struct{ Calls []map[string]json.RawMessage }
		rpc.call(t, "debug_traceCall", &out, args, "latest", map[string]any{"tracer": "callTracer", "stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(code)}}})
		if len(out.Calls) != 1 || out.Calls[0]["value"] != nil || out.Calls[0]["error"] == nil || string(out.Calls[0]["gas"]) != string(out.Calls[0]["gasUsed"]) {
			t.Fatalf("static failure frame: %+v", out)
		}
	})

	t.Run("capture-capacity-and-timeout-return-no-partial-result", func(t *testing.T) {
		// Expansion to one MiB is legal; repeatedly copying it into the trace is
		// bounded independently from the program's gas and explicit step limit.
		code := "0x5f62100000525b600656"
		var out map[string]json.RawMessage
		wideArgs := map[string]any{"from": sender.Hex(), "to": parent.Hex(), "gas": "0x2faf080"}
		err := rpc.invoke("debug_traceCall", &out, wideArgs, "latest", map[string]any{"enableMemory": true, "stateOverrides": map[string]any{parent.Hex(): map[string]any{"code": code}}})
		failure, ok := err.(*track4RPCError)
		if !ok || !strings.Contains(failure.Message, "memory limit") {
			t.Fatalf("trace collection was not bounded: %v", err)
		}
		err = rpc.invoke("debug_traceCall", &out, args, "latest", map[string]any{"tracer": "callTracer", "timeout": "0s"})
		failure, ok = err.(*track4RPCError)
		if !ok || failure.Code != -32016 {
			t.Fatalf("trace timeout produced a partial tree: %v", err)
		}
		err = rpc.invoke("debug_traceBlockByNumber", &out, "0x0")
		if err == nil {
			t.Fatal("genesis trace was fabricated")
		}
	})
	t.Run("independent-sdk-call-trees", func(t *testing.T) { verifyNestedTraceSDK(t, rpc, raw, sender, parent, child, parentCode) })
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().GetState(child, types.Hash{}) != (types.Hash{}) || n.blockchain.State().GetNonce(sender) != 0 {
		t.Fatal("trace adopted its execution")
	}
	t.Run("signed-funded-replay-and-block-log-indices", func(t *testing.T) {
		var hashes []types.Hash
		to := gethcrypto.PubkeyToAddress(key.PublicKey)
		copy(to[:], logParent[:])
		for nonce := uint64(0); nonce < 2; nonce++ {
			signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: nonce, Gas: 30_000_000, GasFeeCap: big.NewInt(2_000_000_000), GasTipCap: big.NewInt(1_000_000_000), To: &to, Value: new(big.Int)}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := signed.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var hash types.Hash
			rpc.call(t, "eth_sendRawTransaction", &hash, hexutil.Encode(encoded))
			hashes = append(hashes, hash)
		}
		beacon := types.Hash{31: 1}
		attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": sender, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x3938700"}
		var fc payload.ForkchoiceUpdatedResult
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.blockchain.Genesis().Hash()}, attrs)
		if fc.PayloadID == nil {
			t.Fatal("signed trace payload ID missing")
		}
		var built struct {
			ExecutionPayload  payload.ExecutionPayloadV5
			ExecutionRequests []string
		}
		rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
		if len(built.ExecutionPayload.Transactions) != 2 {
			t.Fatal("both signed funded transactions were not included")
		}
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV5", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatalf("signed trace import: %+v", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
		beforeHead, beforeRoot := n.blockchain.CurrentBlock().Hash(), n.blockchain.CurrentBlock().Root()
		beforeBalance := new(big.Int).Set(n.blockchain.State().GetBalance(signer))
		type frame struct {
			Type              string
			Gas               string
			GasUsed           string
			RegularGasUsed    string
			StateGasUsed      string
			GasRefund         string
			StateGasReservoir string
			Logs              []struct{ Index string }
			Calls             []struct {
				GasUsed           string
				StateGasReservoir string
				Logs              []struct{ Index string }
			}
		}
		var traces []struct {
			TxHash string
			Result frame
		}
		options := map[string]any{"tracer": "callTracer", "tracerConfig": map[string]any{"withLog": true}}
		rpc.call(t, "debug_traceBlockByHash", &traces, built.ExecutionPayload.BlockHash, options)
		if len(traces) != 2 {
			t.Fatal("missing real block call trees")
		}
		var regular, stateGas uint64
		for i, entry := range traces {
			var receipt struct {
				GasUsed string
				Logs    []struct{ LogIndex string }
			}
			rpc.call(t, "eth_getTransactionReceipt", &receipt, hashes[i])
			f := entry.Result
			if f.GasUsed != receipt.GasUsed || len(f.Calls) != 1 || len(f.Logs) != 1 || len(f.Calls[0].Logs) != 1 || len(receipt.Logs) != 2 || f.Logs[0].Index != receipt.Logs[1].LogIndex || f.Calls[0].Logs[0].Index != receipt.Logs[0].LogIndex {
				t.Fatalf("trace/receipt %d: frame=%+v receipt=%+v", i, f, receipt)
			}
			r, _ := hexutil.DecodeUint64(f.RegularGasUsed)
			st, _ := hexutil.DecodeUint64(f.StateGasUsed)
			regular += r
			stateGas += st
			paid, _ := hexutil.DecodeUint64(f.GasUsed)
			refund, _ := hexutil.DecodeUint64(f.GasRefund)
			if r+st != paid+refund {
				t.Fatal("root gas dimensions were double counted")
			}
			if f.StateGasReservoir != hexutil.EncodeUint64(30_000_000-(1<<24)) || f.Calls[0].StateGasReservoir != f.StateGasReservoir {
				t.Fatalf("full reservoir did not enter child: %+v", f)
			}
		}
		if traces[0].Result.Calls[0].GasUsed != hexutil.EncodeUint64(12106+379) || traces[0].Result.StateGasUsed != hexutil.EncodeUint64(97920) || traces[1].Result.Logs[0].Index != "0x3" || max(regular, stateGas) != built.ExecutionPayload.GasUsed {
			t.Fatalf("funded/nested/block accounting: %+v header=%d", traces, built.ExecutionPayload.GasUsed)
		}
		var selected frame
		rpc.call(t, "debug_traceTransaction", &selected, hashes[1], options)
		if selected.GasUsed != traces[1].Result.GasUsed || selected.Logs[0].Index != "0x3" {
			t.Fatal("single transaction did not replay prefix log indices")
		}
		var only frame
		rpc.call(t, "debug_traceTransaction", &only, hashes[1], map[string]any{"tracer": "callTracer", "tracerConfig": map[string]any{"withLog": true, "onlyTopCall": true}})
		if len(only.Calls) != 0 || len(only.Logs) != 1 || only.Logs[0].Index != "0x3" {
			t.Fatal("onlyTop changed block-global log numbering")
		}
		if n.blockchain.CurrentBlock().Hash() != beforeHead || n.blockchain.CurrentBlock().Root() != beforeRoot || n.blockchain.State().GetBalance(signer).Cmp(beforeBalance) != 0 {
			t.Fatal("real trace replay adopted its result")
		}
	})

}

func TestDimensionalTraceForkPresenceAndNonAdditiveFloor(t *testing.T) {
	const token = "trace-fork-presence"
	raw, sender, parent, child := simulationV1Genesis(t, 24)
	var genesis map[string]any
	if err := json.Unmarshal(raw, &genesis); err != nil {
		t.Fatal(err)
	}
	call := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
	call = append(call, child[:]...)
	call = append(call, 0x5a, 0xf1, 0x50, 0)
	genesis["alloc"].(map[string]any)[parent.Hex()].(map[string]any)["code"] = hexutil.Encode(call)
	genesis["alloc"].(map[string]any)[child.Hex()].(map[string]any)["code"] = "0x00"
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
	args := map[string]any{"from": sender.Hex(), "to": parent.Hex(), "gas": "0x7a120"}
	var before map[string]json.RawMessage
	rpc.call(t, "debug_traceCall", &before, args, "latest", map[string]any{"tracer": "callTracer"})
	var children []map[string]json.RawMessage
	_ = json.Unmarshal(before["calls"], &children)
	for _, frame := range append([]map[string]json.RawMessage{before}, children...) {
		for _, key := range []string{"regularGasUsed", "stateGasUsed", "gasRefund", "stateGasReservoir"} {
			if frame[key] != nil {
				t.Fatalf("pre-fork field %s leaked", key)
			}
		}
	}
	fresh := types.HexToAddress("0x7000000000000000000000000000000000000077")
	var after struct {
		GasUsed           string
		RegularGasUsed    string
		StateGasUsed      string
		GasRefund         string
		StateGasReservoir *string
	}
	rpc.call(t, "debug_traceCall", &after, map[string]any{"from": sender.Hex(), "to": fresh.Hex(), "value": "0x1", "input": "0x" + strings.Repeat("00", 128), "gas": "0x7a120"}, "latest", map[string]any{"tracer": "callTracer", "blockOverrides": map[string]any{"time": "0x18", "number": "0x1"}})
	if after.RegularGasUsed != hexutil.EncodeUint64(29192) || after.StateGasUsed != hexutil.EncodeUint64(183600) || after.GasUsed != hexutil.EncodeUint64(205112) || after.GasRefund != "0x0" || after.StateGasReservoir == nil || *after.StateGasReservoir != "0x0" {
		t.Fatalf("floor-bound dimensions incorrectly additive: %+v", after)
	}
	if n.blockchain.CurrentBlock().Root() != root || n.blockchain.State().Exist(fresh) {
		t.Fatal("fork-context trace adopted state")
	}
}
