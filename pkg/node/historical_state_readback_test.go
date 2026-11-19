package node

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	nodebackend "github.com/eth2030/eth2030/node/backend"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
)

func TestRetainedHistoricalHeaderReadsColdStateAndProofs(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		t.Run(map[bool]string{false: "Osaka", true: "Amsterdam"}[amsterdam], func(t *testing.T) {
			const token = "retained-historical-state"
			raw, _, _, _ := simulationV1Genesis(t, 0)
			var genesis map[string]any
			if err := json.Unmarshal(raw, &genesis); err != nil {
				t.Fatal(err)
			}
			if !amsterdam {
				delete(genesis["config"].(map[string]any), "amsterdamTime")
			}
			reader := types.Address{19: 0x71}
			code := "0x475f5260205ff3"
			slot := types.Hash{}
			genesis["alloc"].(map[string]any)[reader.Hex()] = map[string]any{"balance": "10", "nonce": "0x1", "code": code, "storage": map[string]string{slot.Hex(): (types.Hash{31: 7}).Hex()}}
			raw, err := json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}
			cfg := makeTestConfig(t)
			cfg.EngineAuthToken = token
			cfg.CacheBlockSize, cfg.CacheStateSize = 1, 1
			cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
			if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			n := newTestNode(t, &cfg)
			rpc := track4NodeRPC(t, n, token)
			fcMethod, getMethod, newMethod := "engine_forkchoiceUpdatedV3", "engine_getPayloadV5", "engine_newPayloadV4"
			if amsterdam {
				fcMethod, getMethod, newMethod = "engine_forkchoiceUpdatedV4", "engine_getPayloadV6", "engine_newPayloadV5"
			}
			build := func(parent types.Hash, timestamp, slot, amount uint64) types.Hash {
				t.Helper()
				beacon := types.Hash{31: byte(slot)}
				attrs := map[string]any{"timestamp": hexutil.EncodeUint64(timestamp), "slotNumber": hexutil.EncodeUint64(slot), "targetGasLimit": "0x1c9c380", "withdrawals": []any{map[string]any{"index": "0x1", "validatorIndex": "0x1", "address": reader.Hex(), "amount": hexutil.EncodeUint64(amount)}}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon}
				var fc payload.ForkchoiceUpdatedResult
				rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
				if fc.PayloadID == nil {
					t.Fatal("payload missing")
				}
				var built struct {
					ExecutionPayload  payload.ExecutionPayloadV5
					ExecutionRequests []string
				}
				rpc.call(t, getMethod, &built, fc.PayloadID)
				var status payload.PayloadStatusV1
				if amsterdam {
					rpc.call(t, newMethod, &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests)
				} else {
					rpc.call(t, newMethod, &status, built.ExecutionPayload.ExecutionPayloadV4.ExecutionPayloadV3, []types.Hash{}, beacon, built.ExecutionRequests)
				}
				if status.Status != "VALID" {
					t.Fatal(status)
				}
				rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
				if fc.PayloadStatus.Status != "VALID" {
					t.Fatal(fc)
				}
				return built.ExecutionPayload.BlockHash
			}
			genesisHash := n.blockchain.Genesis().Hash()
			first := build(genesisHash, 12, 1, 1)
			second := build(first, 24, 2, 2)
			orphan := build(genesisHash, 18, 1, 10)
			var fc payload.ForkchoiceUpdatedResult
			rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: second}, nil)
			if fc.PayloadStatus.Status != "VALID" {
				t.Fatal(fc)
			}
			head := build(second, 36, 3, 3)
			for block := uint64(4); block <= 6; block++ {
				head = build(head, block*12, block, block)
			}
			headRoot := n.blockchain.CurrentBlock().Root()
			read := func(label string) {
				t.Helper()
				var call string
				rpc.call(t, "eth_call", &call, map[string]any{"to": reader.Hex(), "gas": "0x493e0"}, "0x1")
				expected := big.NewInt(1000000010)
				if call != types.BytesToHash(expected.Bytes()).Hex() {
					t.Fatalf("%s read current state instead of block one: %s", label, call)
				}
				for _, query := range []struct {
					method   string
					params   []any
					expected string
				}{
					{"eth_getBalance", []any{reader.Hex(), "0x1"}, hexutil.EncodeBig(expected)},
					{"eth_getCode", []any{reader.Hex(), "0x1"}, code},
					{"eth_getTransactionCount", []any{reader.Hex(), "0x1"}, "0x1"},
					{"eth_getStorageAt", []any{reader.Hex(), "0x0", "0x1"}, (types.Hash{31: 7}).Hex()},
				} {
					var output string
					rpc.call(t, query.method, &output, query.params...)
					if output != query.expected {
						t.Fatalf("%s %s=%s", label, query.method, output)
					}
				}
				var proof struct {
					Balance      *hexutil.Big
					CodeHash     gethcommon.Hash
					StorageHash  gethcommon.Hash
					AccountProof []hexutil.Bytes
					StorageProof []struct {
						Value *hexutil.Big
						Proof []hexutil.Bytes
					}
				}
				rpc.call(t, "eth_getProof", &proof, reader.Hex(), []string{"0x0"}, "0x1")
				header := n.blockchain.GetBlock(first).Header()
				encoded, err := header.EncodeRLP()
				if err != nil {
					t.Fatal(err)
				}
				var independent gethtypes.Header
				if err := gethrlp.DecodeBytes(encoded, &independent); err != nil {
					t.Fatal(err)
				}
				if types.Hash(independent.Hash()) != first {
					t.Fatal("independent complete header hash differs")
				}
				database := memorydb.New()
				for _, node := range proof.AccountProof {
					if err := database.Put(gethcrypto.Keccak256(node), node); err != nil {
						t.Fatal(err)
					}
				}
				value, err := gethtrie.VerifyProof(independent.Root, gethcrypto.Keccak256(reader[:]), database)
				if err != nil || len(value) == 0 {
					t.Fatalf("%s historical account proof: %v", label, err)
				}
				var account gethtypes.StateAccount
				if err := gethrlp.DecodeBytes(value, &account); err != nil {
					t.Fatal(err)
				}
				if account.Balance.ToBig().Cmp(expected) != 0 || proof.Balance == nil || (*big.Int)(proof.Balance).Cmp(expected) != 0 || account.Root != proof.StorageHash || gethcommon.BytesToHash(account.CodeHash) != proof.CodeHash {
					t.Fatal("historical account proof/value mismatch")
				}
				if len(proof.StorageProof) != 1 {
					t.Fatal("historical storage proof missing")
				}
				database = memorydb.New()
				for _, node := range proof.StorageProof[0].Proof {
					if err := database.Put(gethcrypto.Keccak256(node), node); err != nil {
						t.Fatal(err)
					}
				}
				value, err = gethtrie.VerifyProof(account.Root, gethcrypto.Keccak256(slot[:]), database)
				if err != nil {
					t.Fatal(err)
				}
				var integer []byte
				if err := gethrlp.DecodeBytes(value, &integer); err != nil {
					t.Fatal(err)
				}
				if new(big.Int).SetBytes(integer).Uint64() != 7 || proof.StorageProof[0].Value == nil || (*big.Int)(proof.StorageProof[0].Value).Uint64() != 7 {
					t.Fatal("historical storage proof does not bind value")
				}
				backend := nodebackend.NewRPCBackend(toNodeDeps(n), n.engBackend).(*nodebackend.RPCBackend)
				orphanHeader := n.blockchain.GetBlock(orphan).Header()
				state, err := backend.StateAtHeader(orphanHeader)
				if err != nil {
					t.Fatal(err)
				}
				if state.GetRoot() != orphanHeader.Root || state.GetBalance(reader).Cmp(big.NewInt(10000000010)) != 0 {
					t.Fatal("retained orphan state substituted a canonical state")
				}
				state.AddBalance(reader, big.NewInt(1))
				again, err := backend.StateAtHeader(orphanHeader)
				if err != nil || again.GetRoot() != orphanHeader.Root {
					t.Fatalf("historical overlay leaked: %v", err)
				}
				forged := *n.blockchain.CurrentBlock().Header()
				forged.Time++
				if _, err := backend.StateAtHeader(&forged); err == nil {
					t.Fatal("unknown header accepted merely because its state root matched current state")
				}
				if _, err := backend.StateAtHeader(nil); err == nil {
					t.Fatal("nil header accepted")
				}
				if n.blockchain.CurrentBlock().Hash() != head || n.blockchain.CurrentBlock().Root() != headRoot {
					t.Fatal("historical queries changed canonical state")
				}
			}
			read("cache evicted")
			if err := n.Stop(); err != nil {
				t.Fatal(err)
			}
			n = newTestNode(t, &cfg)
			rpc = track4NodeRPC(t, n, token)
			read("restarted")
		})
	}
}
