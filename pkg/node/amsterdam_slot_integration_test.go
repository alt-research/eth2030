package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestAmsterdamSlotCommitmentAcrossEngineAndRestart(t *testing.T) {
	const token = "amsterdam-slot-integration"
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	cfg := genesis["config"].(map[string]any)
	cfg["terminalTotalDifficulty"], cfg["amsterdamTime"] = 0, 0
	codeAddress := types.Address{19: 0x42}
	genesis["alloc"].(map[string]any)[codeAddress.Hex()] = map[string]string{
		"balance": "0", "nonce": "0x1", "code": "0x4b60005260206000f3",
	}
	wire, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	var reference gethcore.Genesis
	if err := json.Unmarshal(wire, &reference); err != nil {
		t.Fatal(err)
	}
	wantGenesis := types.Hash(reference.ToBlock().Hash())
	newConfig := func() *Config {
		t.Helper()
		cfg := makeTestConfig(t)
		cfg.GenesisPath, cfg.EngineAuthToken = filepath.Join(cfg.DataDir, "genesis.json"), token
		if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}

	proposerConfig, verifierConfig := newConfig(), newConfig()
	proposerNode := newTestNode(t, proposerConfig)
	verifierNode := newTestNode(t, verifierConfig)
	proposer, verifier := track4NodeRPC(t, proposerNode, token), track4NodeRPC(t, verifierNode, token)
	for _, node := range []*Node{proposerNode, verifierNode} {
		if got := node.Blockchain().Genesis().Hash(); got != wantGenesis {
			t.Fatalf("genesis hash %s, want Geth %s", got, wantGenesis)
		}
	}
	beaconRoot := types.Hash{31: 1}
	attrs := map[string]any{
		"timestamp": "0x60", "slotNumber": "0x8", "withdrawals": []any{},
		"prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{19: 1},
		"parentBeaconBlockRoot": beaconRoot,
	}
	var fc payload.ForkchoiceUpdatedResult
	proposer.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: wantGenesis}, attrs)
	if fc.PayloadID == nil || fc.PayloadStatus.Status != "VALID" {
		t.Fatalf("Amsterdam forkchoice %+v", fc)
	}
	var built struct {
		ExecutionPayload  payload.ExecutionPayloadV5
		ExecutionRequests []string
	}
	proposer.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	if built.ExecutionPayload.SlotNumber != "0x8" {
		t.Fatalf("built slot %s", built.ExecutionPayload.SlotNumber)
	}
	wrongSlot := built.ExecutionPayload
	wrongSlot.SlotNumber = "0x9"
	var status payload.PayloadStatusV1
	verifier.call(t, "engine_newPayloadV5", &status, wrongSlot, []types.Hash{}, beaconRoot, built.ExecutionRequests)
	if status.Status != "INVALID" {
		t.Fatalf("slot modification returned %+v", status)
	}
	p := built.ExecutionPayload
	verifier.call(t, "engine_newPayloadV5", &status, p, []types.Hash{}, beaconRoot, built.ExecutionRequests)
	if status.Status != "VALID" || status.LatestValidHash == nil || *status.LatestValidHash != p.BlockHash {
		t.Fatalf("verifier status %+v", status)
	}
	verifier.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: p.BlockHash}, nil)
	if fc.PayloadStatus.Status != "VALID" {
		t.Fatalf("select Amsterdam block %+v", fc)
	}

	assertCanonicalSlot := func(node *Node) {
		t.Helper()
		header := node.Blockchain().CurrentBlock().Header()
		encoded, err := header.EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		var reference gethtypes.Header
		if err := gethrlp.DecodeBytes(encoded, &reference); err != nil {
			t.Fatal(err)
		}
		if reference.SlotNumber == nil || *reference.SlotNumber != 8 || reference.BlockAccessListHash == nil || types.Hash(reference.Hash()) != p.BlockHash {
			t.Fatalf("independent header slot %v hash %s, want slot 8 hash %s", reference.SlotNumber, reference.Hash(), p.BlockHash)
		}
		rpc := track4NodeRPC(t, node, token)
		for _, full := range []bool{false, true} {
			var block struct {
				Hash                types.Hash
				SlotNumber          string
				BlockAccessListHash types.Hash
			}
			rpc.call(t, "eth_getBlockByNumber", &block, "latest", full)
			if block.Hash != p.BlockHash || block.SlotNumber != "0x8" || block.BlockAccessListHash != types.Hash(*reference.BlockAccessListHash) {
				t.Fatalf("canonical block via RPC %+v", block)
			}
		}
		var value string
		rpc.call(t, "eth_call", &value, map[string]any{"to": codeAddress}, "latest")
		if value != "0x0000000000000000000000000000000000000000000000000000000000000008" {
			t.Fatalf("SLOTNUM returned %s", value)
		}
	}
	assertCanonicalSlot(verifierNode)
	if err := verifierNode.Stop(); err != nil {
		t.Fatal(err)
	}
	assertCanonicalSlot(newTestNode(t, verifierConfig))
}
