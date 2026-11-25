package execution

import (
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

func TestFrame8141WarmnessFollowsFrameAndBatchRollback(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	reverted, reader, skipped := types.Address{19: 0x90}, types.Address{19: 0x91}, types.Address{19: 0x92}
	db.SetCode(reverted, []byte{0x5f, 0x5f, 0xfd})
	db.SetCode(reader, []byte{0x5f, 0x54, 0x50, 0})
	db.SetCode(skipped, []byte{0})
	frame := func(target *types.Address, flags uint8) types.Frame8141 {
		return types.Frame8141{Mode: types.ModeSender, Target: target, Flags: flags, Limits: types.FrameGasLimits{Execution: 10000}}
	}
	tx.Frames = append(tx.Frames, frame(&reverted, 0), frame(&reverted, 0), frame(&reader, 4), frame(&reverted, 4), frame(&skipped, 0), frame(&reader, 0), frame(&reader, 0))
	receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	// Entry access: cold 3000/warm 100. PUSH0+SLOAD+POP costs
	// 2104 cold/104 warm. A failed frame/batch must undo its warming.
	for index, want := range []uint64{100, 3004, 3004, 5104, 3004, 0, 5104, 204} {
		if got := receipt.FrameResults8141[index].GasUsed.Execution; got != want {
			t.Fatalf("frame %d execution gas %d want %d", index, got, want)
		}
	}
	if db.AddressInAccessList(reverted) || db.AddressInAccessList(skipped) || !db.AddressInAccessList(reader) {
		t.Fatal("frame/batch warm-account journal differs")
	}
	if _, warm := db.SlotInAccessList(reader, types.Hash{}); !warm {
		t.Fatal("successful storage read did not persist warming")
	}
}
