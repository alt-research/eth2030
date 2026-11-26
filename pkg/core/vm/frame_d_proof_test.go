package vm

import (
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"testing"
)

func TestFrameDProofExecutionHasNoCallOrAccountAccess(t *testing.T) {
	db := state.NewMemoryStateDB()
	tracker := new(mockBALTracker)
	dependency := types.Dependency8504{Scheme: types.LeanSPHINCSScheme8504, VerificationKeyHash: types.Hash{31: 1}}
	frame := types.Frame8141{Mode: types.ModeProof8504, Data: dependency.Bytes(), Limits: types.FrameGasLimits{Execution: 3000}}
	tx := &types.FrameTx8141{Sender: types.Address{19: 0x90}, Frames: []types.Frame8141{frame}}
	if err := tx.ValidateWithExtensions(types.FrameExtensions{Proof8504: true}); err != nil {
		t.Fatal(err)
	}
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
	evm.SetForkRules(ForkRules{IsEIP8504: true})
	evm.SetBALTracker(tracker, 0)
	ev := evm.SetFrameTransaction(tx, nil)
	root := db.GetRoot()
	left, err := evm.RunFrame8141(0)
	if err != nil || left != 0 || ev.Results[0].GasUsed.State != 0 || len(tracker.addressTouches) != 0 || db.AddressInAccessList(tx.Sender) || db.GetRoot() != root {
		t.Fatal("proof frame executed a call or state/access effect", left, err)
	}
	evm.SetForkRules(ForkRules{})
	if _, err := evm.RunFrame8141(0); err == nil {
		t.Fatal("proof frame executed before activation")
	}
}
