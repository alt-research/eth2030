package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// 7906 observes a created/destroyed adopter before 6780/8246 deletion. A
// failed assertion instead restores the body, including the factory nonce,
// while preserving the payment prefix. The transient child is never a valid
// SETCODEFROM source, even though its adopted code is immediately visible.
func TestAssertion7906AdoptionCreationDestructionAndBodyRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		frame, db, cfg, h := frameExecutionFixture(t)
		zero := uint64(0)
		cfg.AmsterdamTime, cfg.GlamsterdanTime, cfg.EIP8298Time, cfg.EIP7906Time = &zero, &zero, &zero, &zero
		factory, template, assertion := types.Address{19: 0xd1}, types.Address{19: 0xd2}, types.Address{19: 0xd3}
		child := types.Address(gethcrypto.CreateAddress(gethcommon.Address(factory), 0))
		runtime := []byte{byte(vm.STOP)}
		db.SetCode(template, runtime)
		init := append([]byte{byte(vm.PUSH20)}, template[:]...)
		init = append(init, byte(vm.SETCODEFROM), byte(vm.POP), byte(vm.PUSH20))
		init = append(init, factory[:]...)
		init = append(init, byte(vm.SELFDESTRUCT))
		code := []byte{0x60, byte(len(init)), 0x61, 0, 0, 0x5f, 0x39, 0x60, byte(len(init)), 0x5f, 0x5f, 0xf0, 0x80, byte(vm.SETCODEFROM), 0x15, 0x61, 0, 0}
		jump := len(code) - 2
		code = append(code, 0x57, 0x5f, 0x5f, 0xfd)
		code[jump], code[jump+1] = byte(len(code)>>8), byte(len(code))
		code = append(code, 0x5b, 0x50, 0)
		code[3], code[4] = byte(len(code)>>8), byte(len(code))
		code = append(code, init...)
		db.SetCode(factory, code)
		var checks []byte
		for i, selector := range []byte{2, 0xb, 0xc, 0xd, 3} {
			checks = append(checks, 0x5f, 0x60, selector, byte(vm.TXTRACE), 0x60, byte(i*32), 0x52)
		}
		checks = append(checks, 0x60, 160, 0x5f)
		if fail {
			checks = append(checks, byte(vm.REVERT))
		} else {
			checks = append(checks, byte(vm.RETURN))
		}
		db.SetCode(assertion, checks)
		frame.Frames = append(frame.Frames,
			types.Frame8141{Mode: types.ModeSender, Target: &factory, Limits: types.FrameGasLimits{Execution: 200000, State: 1000000}},
			types.Frame8141{Mode: types.ModeAssert7906, Target: &assertion, Limits: types.FrameGasLimits{Execution: 10000}},
		)
		outputs := make(assertionOutputs7906)
		tracker := bal.NewTracker()
		tx := signFrameExecution(t, frame, false)
		db.SetTxContext(tx.Hash(), 0)
		r, _, err := applyTransactionInternal(cfg, nil, db, h, tx, new(gaspool.GasPool).AddGas(h.GasLimit), tracker, nil, vm.Config{FrameObserver: outputs})
		if err != nil || r.FrameResults8141[1].Status != 1 || len(outputs[2]) != 160 {
			t.Fatal("creation/adoption/trace failed", fail, r, err, outputs)
		}
		codeHash := crypto.Keccak256Hash(runtime)
		want := []*big.Int{big.NewInt(1), new(big.Int).SetBytes(child[:]), new(big.Int).SetBytes(types.EmptyCodeHash[:]), new(big.Int).SetBytes(codeHash[:]), big.NewInt(3)}
		for i, expected := range want {
			if new(big.Int).SetBytes(outputs[2][i*32:(i+1)*32]).Cmp(expected) != 0 {
				t.Fatal("pre-finalization trace omitted created/destroyed adopter", fail, i, outputs[2])
			}
		}
		wantNonce, wantStatus := uint64(1), uint64(1)
		if fail {
			wantNonce, wantStatus = 0, 0
		}
		if db.Exist(child) || db.GetNonce(factory) != wantNonce || db.GetNonce(frame.Sender) != 1 || r.FrameResults8141[2].Status != wantStatus || !bytes.Equal(db.GetCode(factory), code) || !bytes.Equal(db.GetCode(template), runtime) {
			t.Fatal("deletion/body rollback/prefix or created-source rejection", fail, r)
		}
		if fail && r.FrameResults8141[1].GasUsed.State != 0 {
			t.Fatal("reverted creation retained state gas")
		}
		for _, entry := range tracker.Build(1).Entries {
			if entry.Address == child && entry.CodeChange != nil {
				t.Fatal("deleted/rolled-back adoption leaked a BAL code change", fail, entry)
			}
		}
	}
}
