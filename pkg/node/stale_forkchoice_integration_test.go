package node

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestRecoveringCLStaleCanonicalForkchoicePreservesDurableFinality(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		t.Run(map[bool]string{false: "Osaka", true: "Amsterdam"}[amsterdam], func(t *testing.T) {
			const token = "recovering-cl-forkchoice"
			raw, sender, _, _ := simulationV1Genesis(t, 0)
			if !amsterdam {
				var genesis map[string]any
				if err := json.Unmarshal(raw, &genesis); err != nil {
					t.Fatal(err)
				}
				delete(genesis["config"].(map[string]any), "amsterdamTime")
				var err error
				raw, err = json.Marshal(genesis)
				if err != nil {
					t.Fatal(err)
				}
			}
			cfg := makeTestConfig(t)
			cfg.EngineAuthToken = token
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
			build := func(parent types.Hash, timestamp, slot uint64) types.Hash {
				t.Helper()
				beacon := types.Hash{31: byte(slot)}
				attrs := map[string]any{"timestamp": hexutil.EncodeUint64(timestamp), "slotNumber": hexutil.EncodeUint64(slot), "targetGasLimit": "0x1c9c380", "withdrawals": []any{map[string]any{"index": "0x1", "validatorIndex": "0x1", "address": sender.Hex(), "amount": hexutil.EncodeUint64(timestamp)}}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon}
				var fc payload.ForkchoiceUpdatedResult
				rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
				if fc.PayloadID == nil {
					t.Fatal("build missing")
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
				header := n.blockchain.GetBlock(built.ExecutionPayload.BlockHash).Header()
				encoded, err := header.EncodeRLP()
				if err != nil {
					t.Fatal(err)
				}
				var independent gethtypes.Header
				if err := gethrlp.DecodeBytes(encoded, &independent); err != nil {
					t.Fatal(err)
				}
				if types.Hash(independent.Hash()) != built.ExecutionPayload.BlockHash {
					t.Fatal("independent full header hash differs")
				}
				return built.ExecutionPayload.BlockHash
			}
			genesis := n.blockchain.Genesis().Hash()
			a1 := build(genesis, 12, 1)
			a2 := build(a1, 24, 2)
			a3 := build(a2, 36, 3)
			a4 := build(a3, 48, 4)
			orphan := build(genesis, 18, 1)
			var fc payload.ForkchoiceUpdatedResult
			rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: a4, SafeBlockHash: a3, FinalizedBlockHash: a3}, nil)
			if fc.PayloadStatus.Status != "VALID" {
				t.Fatal(fc)
			}
			root := n.blockchain.CurrentBlock().Root()
			if err := n.Stop(); err != nil {
				t.Fatal(err)
			}
			n = newTestNode(t, &cfg)
			rpc = track4NodeRPC(t, n, token)
			unchanged := func() {
				t.Helper()
				stored := rawdb.ReadFinalizedBlockHash(n.db)
				if stored != a3 || n.blockchain.CurrentBlock().Hash() != a4 || n.blockchain.CurrentBlock().Root() != root || n.blockchain.CurrentFinalBlock().Hash() != a3 {
					t.Fatal("stale CL notification changed durable head/finality")
				}
			}
			unchanged()
			for _, state := range []payload.ForkchoiceStateV1{{HeadBlockHash: a1, SafeBlockHash: a1, FinalizedBlockHash: a1}, {HeadBlockHash: a3, SafeBlockHash: a2, FinalizedBlockHash: a1}} {
				rpc.call(t, fcMethod, &fc, state, nil)
				if fc.PayloadStatus.Status != "VALID" || fc.PayloadStatus.LatestValidHash == nil || *fc.PayloadStatus.LatestValidHash != state.HeadBlockHash || fc.PayloadID != nil {
					t.Fatalf("stale canonical response %+v", fc)
				}
				unchanged()
			}
			for _, tc := range []struct {
				name  string
				state payload.ForkchoiceStateV1
			}{
				{"unknown safe", payload.ForkchoiceStateV1{HeadBlockHash: a1, SafeBlockHash: types.Hash{31: 0xee}, FinalizedBlockHash: a1}},
				{"foreign safe", payload.ForkchoiceStateV1{HeadBlockHash: a1, SafeBlockHash: orphan, FinalizedBlockHash: a1}},
				{"foreign finalized", payload.ForkchoiceStateV1{HeadBlockHash: a1, SafeBlockHash: a1, FinalizedBlockHash: orphan}},
				{"checkpoint above nominated head", payload.ForkchoiceStateV1{HeadBlockHash: a1, SafeBlockHash: a2, FinalizedBlockHash: a1}},
				{"safe before final", payload.ForkchoiceStateV1{HeadBlockHash: a3, SafeBlockHash: a1, FinalizedBlockHash: a2}},
				{"noncanonical head below final", payload.ForkchoiceStateV1{HeadBlockHash: orphan, FinalizedBlockHash: genesis}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var response payload.ForkchoiceUpdatedResult
					err := rpc.invoke(fcMethod, &response, tc.state, nil)
					detail, ok := err.(*track4RPCError)
					if !ok || detail.Code != -38002 {
						t.Fatalf("invalid checkpoint accepted: %+v/%v", response, err)
					}
					unchanged()
				})
			}
			// A normal new payload still advances head while preserving prior finality.
			a5 := build(a4, 60, 5)
			if a5 == a4 || n.blockchain.CurrentBlock().Hash() != a5 || n.blockchain.CurrentFinalBlock().Hash() != a3 || n.blockchain.State().GetBalance(sender).Cmp(new(big.Int)) <= 0 {
				t.Fatal("normal forward publication regressed")
			}
		})
	}
}
