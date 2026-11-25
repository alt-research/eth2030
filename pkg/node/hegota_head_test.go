package node

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/forkchoice"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/eth"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/rlp"
)

// Exercise the installed P2P callback and real reorg listener, rather than only
// calling TxPool.ResetAtHead directly with an artificial timestamp.
func TestNativeP2PAndReorgRefreshFrameActivation(t *testing.T) {
	cfg := makeTestConfig(t)
	var genesis map[string]any
	if err := json.Unmarshal(minimalGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	chain := genesis["config"].(map[string]any)
	chain["terminalTotalDifficulty"] = 0
	chain["bogotaTime"] = 1
	chain["experimentalLegacyFrames"] = true
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
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1337), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(100), MaxPriorityFeePerGas: big.NewInt(1)})
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
	if n.blockchain.CurrentBlock().Hash() != child.Hash() {
		t.Fatal("P2P block was not imported")
	}
	if err := n.txPool.AddLocal(tx); err != nil {
		t.Fatalf("P2P import failed to activate pool: %v", err)
	}
	n.fcStateManager.AddBlock(&forkchoice.BlockInfo{Hash: child.Hash(), ParentHash: parent.Hash(), Number: child.NumberU64(), Slot: 1})
	if err := n.fcStateManager.ProcessForkchoiceUpdate(payload.ForkchoiceStateV1{HeadBlockHash: child.Hash()}); err != nil {
		t.Fatal(err)
	}
	if err := n.blockchain.SetHead(0); err != nil {
		t.Fatal(err)
	}
	if err := n.fcStateManager.ProcessForkchoiceUpdate(payload.ForkchoiceStateV1{HeadBlockHash: parent.Hash()}); err != nil {
		t.Fatal(err)
	}
	if n.txPool.Get(tx.Hash()) != nil {
		t.Fatal("reorg listener retained post-fork prototype")
	}
	if err := n.txPool.AddLocal(tx); err == nil {
		t.Fatal("reorg listener left stale activation time")
	}
}
