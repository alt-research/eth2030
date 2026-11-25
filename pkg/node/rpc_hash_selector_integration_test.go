package node

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
)

func TestHashSelectedExecutionRPCUsesExactBranchStateAndAncestry(t *testing.T) {
	const token = "hash-selected-execution"
	raw, sender, _, _ := simulationV1Genesis(t, 0)
	var genesis map[string]any
	if err := json.Unmarshal(raw, &genesis); err != nil {
		t.Fatal(err)
	}
	reader := types.Address{19: 0x61}
	guard := types.Address{19: 0x62}
	alloc := genesis["alloc"].(map[string]any)
	alloc[reader.Hex()] = map[string]string{"balance": "1", "nonce": "0x1", "code": "0x5f35405f5260205ff3"}
	alloc[guard.Hex()] = map[string]string{"balance": "1", "nonce": "0x1", "code": "0x6001405f3514600c575f5ffd5b5f545000"}
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
	build := func(parent types.Hash, timestamp, slot, amount uint64) payload.ExecutionPayloadV5 {
		t.Helper()
		beacon := types.Hash{31: 1}
		withdrawal := map[string]any{"index": "0x1", "validatorIndex": "0x1", "address": sender.Hex(), "amount": hexutil.EncodeUint64(amount)}
		attrs := map[string]any{"timestamp": hexutil.EncodeUint64(timestamp), "slotNumber": hexutil.EncodeUint64(slot), "withdrawals": []any{withdrawal}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380"}
		var fc payload.ForkchoiceUpdatedResult
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
		if fc.PayloadID == nil {
			t.Fatal("branch payload missing")
		}
		var result struct {
			ExecutionPayload  payload.ExecutionPayloadV5
			ExecutionRequests []string
		}
		rpc.call(t, "engine_getPayloadV6", &result, fc.PayloadID)
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV5", &status, result.ExecutionPayload, []types.Hash{}, beacon, result.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatal(status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: result.ExecutionPayload.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" {
			t.Fatal("head selection failed")
		}
		header := n.blockchain.GetBlock(result.ExecutionPayload.BlockHash).Header()
		encoded, err := header.EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		var independent gethtypes.Header
		if err := gethrlp.DecodeBytes(encoded, &independent); err != nil {
			t.Fatal(err)
		}
		if types.Hash(independent.Hash()) != result.ExecutionPayload.BlockHash {
			t.Fatal("independent complete header commitment differs")
		}
		return result.ExecutionPayload
	}
	parent := n.blockchain.Genesis().Hash()
	a1 := build(parent, 12, 1, 1)
	a2 := build(a1.BlockHash, 24, 2, 1)
	b1 := build(parent, 18, 1, 2)
	b2 := build(b1.BlockHash, 30, 2, 2)
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: a2.BlockHash}, nil)
	if fc.PayloadStatus.Status != "VALID" {
		t.Fatal("restore canonical branch failed")
	}
	frozenRoot := n.blockchain.CurrentBlock().Root()
	args := map[string]any{"from": sender.Hex(), "to": reader.Hex(), "data": (types.Hash{31: 1}).Hex(), "gas": "0x7a120"}
	for _, tc := range []struct {
		name     string
		selector any
		hash     types.Hash
	}{{"canonical hash", map[string]any{"blockHash": a2.BlockHash, "requireCanonical": true}, a1.BlockHash}, {"orphan hash", map[string]any{"blockHash": b2.BlockHash, "requireCanonical": false}, b1.BlockHash}, {"bare orphan hash", b2.BlockHash.Hex(), b1.BlockHash}, {"number object", map[string]any{"blockNumber": "0x2"}, a1.BlockHash}} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			rpc.call(t, "eth_call", &got, args, tc.selector)
			if got != tc.hash.Hex() {
				t.Fatalf("BLOCKHASH selected %s, expected branch %s", got, tc.hash)
			}
		})
	}
	orphan := map[string]any{"blockHash": b2.BlockHash, "requireCanonical": false}
	guarded := map[string]any{"from": sender.Hex(), "to": guard.Hex(), "data": b1.BlockHash.Hex(), "gas": "0x7a120"}
	var estimate string
	rpc.call(t, "eth_estimateGas", &estimate, guarded, orphan)
	if gas, err := hexutil.DecodeUint64(estimate); err != nil || gas == 0 {
		t.Fatalf("orphan estimate=%s/%v", estimate, err)
	}
	var access struct {
		GasUsed string
		Error   string
	}
	rpc.call(t, "eth_createAccessList", &access, guarded, orphan)
	if access.Error != "" || access.GasUsed == "" {
		t.Fatalf("orphan access discovery used wrong ancestry: %+v", access)
	}
	var trace struct {
		Failed      bool
		ReturnValue string
	}
	rpc.call(t, "debug_traceCall", &trace, args, orphan, map[string]any{})
	if trace.Failed || trace.ReturnValue != b1.BlockHash.Hex() {
		t.Fatalf("orphan trace %+v", trace)
	}
	var aBalance, bBalance string
	rpc.call(t, "eth_getBalance", &aBalance, sender.Hex(), map[string]any{"blockHash": a2.BlockHash})
	rpc.call(t, "eth_getBalance", &bBalance, sender.Hex(), orphan)
	av, _ := hexutil.DecodeBig(aBalance)
	bv, _ := hexutil.DecodeBig(bBalance)
	if new(big.Int).Sub(bv, av).Cmp(big.NewInt(2000000000)) != 0 {
		t.Fatalf("branch balances not frozen independently: %s/%s", aBalance, bBalance)
	}
	for _, selected := range []struct {
		name    string
		header  types.Hash
		balance string
	}{{"canonical", a2.BlockHash, aBalance}, {"orphan", b2.BlockHash, bBalance}} {
		t.Run("independent proof "+selected.name, func(t *testing.T) {
			var proof struct {
				Balance      string
				Nonce        string
				StorageHash  gethcommon.Hash
				CodeHash     gethcommon.Hash
				AccountProof []hexutil.Bytes
			}
			rpc.call(t, "eth_getProof", &proof, sender.Hex(), []string{}, map[string]any{"blockHash": selected.header})
			proofDB := memorydb.New()
			for _, node := range proof.AccountProof {
				if err := proofDB.Put(gethcrypto.Keccak256(node), node); err != nil {
					t.Fatal(err)
				}
			}
			body, err := gethtrie.VerifyProof(gethcommon.Hash(n.blockchain.GetBlock(selected.header).Root()), gethcrypto.Keccak256(sender[:]), proofDB)
			if err != nil || len(body) == 0 {
				t.Fatalf("selected account proof: %v", err)
			}
			var account gethtypes.StateAccount
			if err := gethrlp.DecodeBytes(body, &account); err != nil {
				t.Fatal(err)
			}
			balance, err := hexutil.DecodeBig(proof.Balance)
			if err != nil || proof.Balance != selected.balance || account.Balance.ToBig().Cmp(balance) != 0 || account.Root != proof.StorageHash || gethcommon.BytesToHash(account.CodeHash) != proof.CodeHash {
				t.Fatalf("proof fields do not bind the selected header/account: %+v", proof)
			}
		})
	}
	var output string
	overrides := map[string]any{reader.Hex(): map[string]any{"code": "0x5f545f5260205ff3", "stateDiff": map[string]string{(types.Hash{}).Hex(): (types.Hash{31: 123}).Hex()}}}
	rpc.call(t, "eth_call", &output, args, orphan, overrides)
	if output != (types.Hash{31: 123}).Hex() {
		t.Fatalf("hash-selected state/code overrides ignored: %s", output)
	}
	rpc.call(t, "eth_call", &output, args, orphan)
	if output != b1.BlockHash.Hex() {
		t.Fatalf("override leaked into frozen branch: %s", output)
	}
	for _, query := range []struct {
		method   string
		params   []any
		expected string
	}{
		{"eth_getCode", []any{reader.Hex(), orphan}, "0x5f35405f5260205ff3"},
		{"eth_getTransactionCount", []any{reader.Hex(), orphan}, "0x1"},
		{"eth_getStorageAt", []any{reader.Hex(), "0x0", orphan}, (types.Hash{}).Hex()},
	} {
		rpc.call(t, query.method, &output, query.params...)
		if output != query.expected {
			t.Fatalf("%s branch getter changed: %s", query.method, output)
		}
	}
	for _, method := range []string{"eth_call", "eth_estimateGas", "eth_createAccessList", "debug_traceCall", "eth_getBalance", "eth_getCode", "eth_getTransactionCount", "eth_getStorageAt", "eth_getProof"} {
		t.Run("selector errors "+method, func(t *testing.T) {
			invoke := func(selector any) error {
				var out any
				switch method {
				case "eth_getBalance", "eth_getCode", "eth_getTransactionCount":
					return rpc.invoke(method, &out, sender.Hex(), selector)
				case "eth_getStorageAt":
					return rpc.invoke(method, &out, sender.Hex(), "0x0", selector)
				case "eth_getProof":
					return rpc.invoke(method, &out, sender.Hex(), []string{}, selector)
				case "debug_traceCall":
					return rpc.invoke(method, &out, args, selector, map[string]any{})
				default:
					return rpc.invoke(method, &out, args, selector)
				}
			}
			for _, failure := range []struct {
				name     string
				selector any
				code     int
			}{{"orphan canonical", map[string]any{"blockHash": b2.BlockHash, "requireCanonical": true}, -32000}, {"unknown before canonical", map[string]any{"blockHash": types.Hash{31: 0xff}, "requireCanonical": true}, -32001}, {"ambiguous", map[string]any{"blockHash": a2.BlockHash, "blockNumber": "0x2"}, -32602}, {"malformed", map[string]any{"blockHash": "0x01"}, -32602}} {
				err := invoke(failure.selector)
				response, ok := err.(*track4RPCError)
				if !ok || response.Code != failure.code {
					t.Fatalf("%s %s: %v", method, failure.name, err)
				}
			}
		})
	}
	if n.blockchain.CurrentBlock().Hash() != a2.BlockHash || n.blockchain.CurrentBlock().Root() != frozenRoot {
		t.Fatal("hash-selected simulation changed canonical state")
	}
}
