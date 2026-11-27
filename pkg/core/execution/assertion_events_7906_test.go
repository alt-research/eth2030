package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

func TestAssertion7906NativeLogTopicsDataCopyAndRollback(t *testing.T) {
	for _, badCopy := range []bool{false, true} {
		frame, db, cfg, h := frameExecutionFixture(t)
		zero := uint64(0)
		cfg.EIP7906Time = &zero
		body, assertion := types.Address{19: 0xe1}, types.Address{19: 0xe2}
		// LOG3(data=0xabcd, topics=[1,2,3]).
		db.SetCode(body, []byte{0x61, 0xab, 0xcd, 0x5f, 0x52, 0x60, 3, 0x60, 2, 0x60, 1, 0x60, 2, 0x60, 30, 0xa3, 0})
		var code []byte
		for i, selector := range []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x18} {
			code = append(code, 0x5f, 0x60, selector, byte(vm.TXTRACE), 0x60, byte(i*32), 0x52)
		}
		if badCopy {
			// A zero-length source range beginning past event data is invalid.
			code = append(code, 0x5f, 0x60, 3)
		} else {
			code = append(code, 0x60, 2, 0x5f)
		}
		code = append(code, 0x60, 224, 0x5f, byte(vm.EVENTDATACOPY), 0x60, 226, 0x5f, 0xf3)
		db.SetCode(assertion, code)
		frame.Frames = append(frame.Frames,
			types.Frame8141{Mode: types.ModeSender, Target: &body, Limits: types.FrameGasLimits{Execution: 20000}},
			types.Frame8141{Mode: types.ModeAssert7906, Target: &assertion, Limits: types.FrameGasLimits{Execution: 10000}},
		)
		tx := signFrameExecution(t, frame, false)
		db.SetTxContext(tx.Hash(), 0)
		outputs := make(assertionOutputs7906)
		r, _, err := ApplyTransactionWithVMConfig(cfg, nil, db, h, tx, new(gaspool.GasPool).AddGas(h.GasLimit), vm.Config{FrameObserver: outputs})
		if err != nil || r.FrameResults8141[1].Status != 1 || db.GetNonce(frame.Sender) != 1 {
			t.Fatal("native event trace transaction", badCopy, r, err)
		}
		if badCopy {
			if r.FrameResults8141[2].Status != 0 || len(r.Logs) != 0 || len(r.FrameResults8141[1].Logs) != 0 {
				t.Fatal("invalid event copy did not roll back body logs", r)
			}
			continue
		}
		if r.FrameResults8141[2].Status != 1 || len(outputs[2]) != 226 || len(r.Logs) != 1 || !bytes.Equal(outputs[2][224:], []byte{0xab, 0xcd}) {
			t.Fatal("actual event bytes not exposed", r, outputs)
		}
		want := []*big.Int{big.NewInt(1), new(big.Int).SetBytes(body[:]), big.NewInt(3), big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(2)}
		for i, expected := range want {
			if new(big.Int).SetBytes(outputs[2][i*32:(i+1)*32]).Cmp(expected) != 0 {
				t.Fatal("actual event selector", badCopy, i, outputs[2])
			}
		}
	}
}
