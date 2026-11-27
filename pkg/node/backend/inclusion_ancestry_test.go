package backend

import (
	"math/big"
	"testing"

	blockpkg "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// A trusted snap pivot retains the BLOCKHASH header window before bodies are
// downloaded. IL execution must use that authenticated ancestry without bodies.
func TestGroupEInclusionAncestryUsesAuthenticatedHeadersWithoutBodies(t *testing.T) {
	cfg := config.TestConfigHogota
	initial := state.NewMemoryStateDB()
	eips.ApplyBuilderRequestGenesis(initial)
	sender := types.Address{19: 0x71}
	// A permissionless fixture verifier approves only when BLOCKHASH(1) exists.
	initial.SetCode(sender, []byte{0x60, 1, 0x40, 0x15, 0x60, 12, 0x57, 0x60, 3, 0x5f, 0x5f, 0xaa, 0x5b, 0x5f, 0x5f, 0xfd})
	initial.AddBalance(sender, big.NewInt(1000000000000000000))
	genesis := types.NewBlock(&types.Header{Number: new(big.Int), GasLimit: 30000000, BaseFee: big.NewInt(1000000000), Root: initial.GetRoot()}, &types.Body{})
	full, err := chain.NewBlockchain(cfg, genesis, initial, rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	blocks := []*types.Block{genesis}
	for n := uint64(1); n <= 3; n++ {
		parent := blocks[len(blocks)-1]
		before, err := full.StateAtBlock(parent)
		if err != nil {
			t.Fatal(err)
		}
		builder := blockpkg.NewBlockBuilder(cfg, nil, nil)
		builder.SetState(before)
		beacon := types.Hash{31: 1}
		slot := n
		block, _, err := builder.BuildBlock(parent.Header(), &blockpkg.BuildBlockAttributes{Timestamp: n * 12, GasLimit: 30000000, SlotNumber: &slot, BeaconRoot: &beacon})
		if err != nil {
			t.Fatal(err)
		}
		if err := full.InsertBlock(block); err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, block)
	}
	pivot := blocks[3]
	current, err := full.StateAtBlock(pivot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := state.ExportStateSnapshot(current)
	if err != nil {
		t.Fatal(err)
	}
	db := rawdb.NewMemoryDB()
	if err := state.StoreStateSnapshot(db, pivot.Root(), snapshot); err != nil {
		t.Fatal(err)
	}
	target, err := chain.NewBlockchain(cfg, genesis, initial, db)
	if err != nil {
		t.Fatal(err)
	}
	if err := target.ImportSnapPivot(pivot, []*types.Header{blocks[2].Header(), blocks[1].Header(), genesis.Header()}); err != nil {
		t.Fatal(err)
	}
	if target.GetBlock(blocks[1].Hash()) != nil || target.GetHeaderByHash(blocks[1].Hash()) == nil {
		t.Fatal("fixture did not retain headers independently of bodies")
	}
	// New payload execution after the pivot has parent state but only headers
	// for older BLOCKHASH ancestors. Its IL probe must use those same headers.
	before, err := target.StateAtBlock(pivot)
	if err != nil {
		t.Fatal(err)
	}
	builder := blockpkg.NewBlockBuilder(cfg, nil, nil)
	builder.SetState(before)
	beacon, slot := types.Hash{31: 1}, uint64(4)
	child, _, err := builder.BuildBlock(pivot.Header(), &blockpkg.BuildBlockAttributes{Timestamp: 48, GasLimit: 30000000, SlotNumber: &slot, BeaconRoot: &beacon})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.InsertSyncedBlock(child); err != nil {
		t.Fatal(err)
	}
	node := newTestBackendNode(t)
	node.blockchain = target
	backend := NewEngineBackend(node)
	t.Cleanup(backend.Close)
	frame := &types.FrameTx8141{ChainID: new(big.Int).Set(cfg.ChainID), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100000000000000000), MaxPriorityFeePerGas: big.NewInt(1)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 20000}}}}
	raw, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	satisfied, err := backend.CheckPayloadInclusionList(child.Hash(), [][]byte{raw})
	if err != nil || satisfied {
		t.Fatalf("header-only ancestry excused a valid omission: satisfied=%v err=%v", satisfied, err)
	}
	boundary, err := newInclusionEndState(target, pivot)
	if err != nil {
		t.Fatal(err)
	}
	if got := boundary.getHash(1); got != blocks[1].Hash() {
		t.Fatalf("header-only BLOCKHASH: got %s want %s", got, blocks[1].Hash())
	}
	if boundary.getHash(3) != (types.Hash{}) || boundary.getHash(4) != (types.Hash{}) {
		t.Fatal("current/future block hash disclosed")
	}
	if boundary.readError(boundary.state) != nil {
		t.Fatal("valid header window produced an availability error")
	}
	if err := rawdb.DeleteHeader(db, blocks[1].NumberU64(), blocks[1].Hash()); err != nil {
		t.Fatal(err)
	}
	if satisfied, err := backend.CheckPayloadInclusionList(child.Hash(), [][]byte{raw}); err == nil {
		t.Fatalf("missing authenticated header became a verdict: %v", satisfied)
	}

}
