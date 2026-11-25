package node

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// The CL owns attestation authentication and FCR. This test exercises its
// standard Engine output through real HTTP and the native safe-tag consumer.
func TestFastConfirmationEngineSafeCheckpointConsumer(t *testing.T) {
	const token = "fcr-native-consumer"
	raw, _, _, _ := simulationV1Genesis(t, 0)
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	wire := track4NodeRPC(t, n, token)
	genesis := n.blockchain.Genesis().Hash()
	blocks := []types.Hash{genesis}
	for slot := uint64(1); slot <= 4; slot++ {
		beacon := types.Hash{31: byte(slot)}
		attrs := map[string]any{"timestamp": hexutil.EncodeUint64(slot * 12), "slotNumber": hexutil.EncodeUint64(slot), "targetGasLimit": "0x1c9c380", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon}
		var fc payload.ForkchoiceUpdatedResult
		wire.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: blocks[len(blocks)-1], SafeBlockHash: genesis, FinalizedBlockHash: genesis}, attrs)
		if fc.PayloadID == nil {
			t.Fatalf("slot %d: no build", slot)
		}
		var built struct {
			ExecutionPayload  payload.ExecutionPayloadV5
			ExecutionRequests []string
		}
		wire.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
		var status payload.PayloadStatusV1
		wire.call(t, "engine_newPayloadV5", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatal(status)
		}
		blocks = append(blocks, built.ExecutionPayload.BlockHash)
	}
	assertSafe := func(want types.Hash) {
		t.Helper()
		var block struct{ Hash types.Hash }
		wire.call(t, "eth_getBlockByNumber", &block, "safe", false)
		if block.Hash != want || n.blockchain.CurrentSafeBlock().Hash() != want {
			t.Fatalf("safe consumer = %s, want %s", block.Hash, want)
		}
		if n.blockchain.CurrentBlock().Hash() != blocks[4] || n.blockchain.CurrentFinalBlock().Hash() != genesis {
			t.Fatal("safe update changed head or finality")
		}
	}
	// FCR can advance and retract confidence without a new head/checkpoint.
	for _, index := range []int{1, 3, 2, 0, 2} {
		var result payload.ForkchoiceUpdatedResult
		wire.call(t, "engine_forkchoiceUpdatedV4", &result, payload.ForkchoiceStateV1{HeadBlockHash: blocks[4], SafeBlockHash: blocks[index], FinalizedBlockHash: genesis}, nil)
		if result.PayloadStatus.Status != "VALID" {
			t.Fatal(result)
		}
		assertSafe(blocks[index])
	}
	// An unknown alleged confirmation must not be published or change state.
	var result payload.ForkchoiceUpdatedResult
	err := wire.invoke("engine_forkchoiceUpdatedV4", &result, payload.ForkchoiceStateV1{HeadBlockHash: blocks[4], SafeBlockHash: types.Hash{31: 0xee}, FinalizedBlockHash: genesis}, nil)
	if detail, ok := err.(*track4RPCError); !ok || detail.Code != -38002 {
		t.Fatalf("unknown safe block: %v", err)
	}
	assertSafe(blocks[2])
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	n = newTestNode(t, &cfg)
	wire = track4NodeRPC(t, n, token)
	// A restart conservatively resets confidence to finalized. Only a fresh CL
	// update restores a newer safe block; the EL never invents validator votes.
	assertSafe(genesis)
	wire.call(t, "engine_forkchoiceUpdatedV4", &result, payload.ForkchoiceStateV1{HeadBlockHash: blocks[4], SafeBlockHash: blocks[2], FinalizedBlockHash: genesis}, nil)
	assertSafe(blocks[2])
}
