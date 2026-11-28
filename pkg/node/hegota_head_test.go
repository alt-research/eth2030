package node

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/eth"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/rlp"
)

// P2P announcements cannot select a PoS head. Execute forkchoice on the actual
// native backend and verify both activation and a reorg across the boundary.
func TestNativeForkchoiceRefreshesFrameActivationAndReorg(t *testing.T) {
	cfg := makeTestConfig(t)
	var genesis map[string]any
	if err := json.Unmarshal(minimalGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	genesisRules := genesis["config"].(map[string]any)
	genesisRules["terminalTotalDifficulty"] = 0
	genesisRules["bogotaTime"] = 1
	genesisRules["experimentalLegacyFrames"] = true
	sender := types.HexToAddress("0x1111111111111111111111111111111111111111")
	genesis["alloc"].(map[string]any)[sender.Hex()] = map[string]any{"balance": "1000000000000000000", "code": "0x00", "nonce": "0x0"}
	raw, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1337), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(2_000_000_000), MaxPriorityFeePerGas: big.NewInt(1)})
	if err := n.txPool.AddLocal(tx); err == nil {
		t.Fatal("prototype admitted before canonical activation")
	}
	parent := n.blockchain.CurrentBlock()
	statedb, err := n.blockchain.StateAtBlock(parent)
	if err != nil {
		t.Fatal(err)
	}
	builder := block.NewBlockBuilder(n.blockchain.Config(), nil, nil)
	builder.SetState(statedb)
	child, _, err := builder.BuildBlock(parent.Header(), &block.BuildBlockAttributes{Timestamp: 1, GasLimit: parent.GasLimit()})
	if err != nil {
		t.Fatal(err)
	}
	blockBytes, err := child.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	encoded := rlp.WrapList(append(blockBytes, 0x80)) // NewBlock wire: [block, totalDifficulty=0]

	peer := eth.NewEthPeer(p2p.NewPeer("hegota-head-test", "127.0.0.1:0", nil), nil)
	if err := n.ethHandler.HandleMsg(peer, p2p.Msg{Code: p2p.NewBlockMsg, Size: uint32(len(encoded)), Payload: encoded}); err != nil {
		t.Fatal(err)
	}
	if n.blockchain.CurrentBlock().Hash() != parent.Hash() {
		t.Fatal("P2P announcement selected an optimistic head")
	}
	if err := n.txPool.AddLocal(tx); err == nil {
		t.Fatal("P2P announcement activated the legacy profile")
	}
	if err := n.blockchain.InsertSyncedBlock(child); err != nil {
		t.Fatal(err)
	}
	// Supply an explicitly evaluated empty CL view in the current cache schema.
	// IL execution semantics are covered by the separate versioned wire tests.
	if err := n.blockchain.WriteBogotaPayload(child.Hash(), &chain.BogotaPayload{Number: child.NumberU64(), Satisfied: true, SatisfactionVersion: chain.BogotaSatisfactionVersion}); err != nil {
		t.Fatal(err)
	}
	selectHead := func(hash types.Hash) {
		t.Helper()
		result, err := n.engBackend.ForkchoiceUpdated(payload.ForkchoiceStateV1{HeadBlockHash: hash}, nil)
		if err != nil || result.PayloadStatus.Status != "VALID" {
			t.Fatalf("forkchoice %s: %+v %v", hash, result, err)
		}
		if n.blockchain.CurrentBlock().Hash() != hash {
			t.Fatal("native FCU did not select the canonical head")
		}
	}
	selectHead(child.Hash())
	if err := n.txPool.AddLocal(tx); err != nil {
		t.Fatalf("FCU failed to activate pool: %v", err)
	}
	selectHead(parent.Hash())
	if n.txPool.Get(tx.Hash()) != nil {
		t.Fatal("reorg listener retained post-fork prototype")
	}
	if err := n.txPool.AddLocal(tx); err == nil {
		t.Fatal("reorg listener left stale activation time")
	}
}
