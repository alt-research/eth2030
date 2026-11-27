package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestAssertion7906PaymasterTokenFalseReturnRetryAndAtomic(t *testing.T) {
	for _, atomic := range []bool{false, true} {
		for _, returnsFalse := range []bool{false, true} {
			frame, db, cfg, h := frameExecutionFixture(t)
			zero := uint64(0)
			cfg.EIP7906Time, cfg.AmsterdamTime, cfg.GlamsterdanTime, cfg.EIP8151Time = &zero, &zero, &zero, &zero
			payer, token, assert, aux, bad := types.Address{19: 0x93}, types.Address{19: 0x94}, types.Address{19: 0x95}, types.Address{19: 0x96}, types.Address{19: 0x97}
			ownerKey := paymaster12041Key(t, 2)
			owner := types.Address(gethcrypto.PubkeyToAddress(ownerKey.PublicKey))
			db.SetCode(payer, framepay.Runtime12041())
			db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
			db.AddBalance(payer, big.NewInt(100_000_000))
			// A token-style transfer updates two balance slots and emits an event.
			// A false return does not undo its writes and is not a protocol revert.
			code := []byte{0x60, 5, 0x5f, 0x54, 0x03, 0x5f, 0x55, 0x60, 5, 0x60, 1, 0x54, 0x01, 0x60, 1, 0x55, 0x5f, 0x5f, 0xa0, 0x60, 1, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3}
			if returnsFalse {
				code[len(code)-7] = 0
			}
			db.SetCode(token, code)
			db.SetState(token, types.Hash{}, types.Hash{31: 100})
			db.SetCode(assert, []byte{0x5f, 0x5f, 0xfd})
			frame.Frames[0].Flags = types.FrameApproveExecution
			frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: types.FrameApprovePayment, Target: &payer, Limits: types.FrameGasLimits{Execution: 15000}}, types.Frame8141{Mode: types.ModeSender, Target: &token, Limits: types.FrameGasLimits{Execution: 100000, State: 97920}})
			if atomic {
				db.SetCode(aux, []byte{0x60, 1, 0x5f, 0x55, 0})
				db.SetCode(bad, []byte{0x5f, 0x5f, 0xfd})
				frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeSender, Target: &aux, Flags: types.FrameAtomicBatch, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}}, types.Frame8141{Mode: types.ModeSender, Target: &bad, Limits: types.FrameGasLimits{Execution: 5000}})
			}
			frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeAssert7906, Target: &assert, Limits: types.FrameGasLimits{Execution: 10000}})
			before := new(big.Int).Set(db.GetBalance(payer))
			senderBefore := new(big.Int).Set(db.GetBalance(frame.Sender))
			apply := func() *types.Receipt {
				t.Helper()
				tx := paymaster12041Sign(t, frame, 1, ownerKey, nil)
				db.SetTxContext(tx.Hash(), 0)
				outputs := make(assertionOutputs7906)
				r, _, err := ApplyTransactionWithVMConfig(cfg, nil, db, h, tx, new(gaspool.GasPool).AddGas(h.GasLimit), vm.Config{FrameObserver: outputs})
				if len(outputs[2]) != 32 || (new(big.Int).SetBytes(outputs[2]).Sign() == 0) != returnsFalse {
					t.Fatal("token fixture did not return its claimed boolean", outputs[2])
				}
				if err != nil {
					t.Fatal(err)
				}
				return r
			}
			r := apply()
			if r.Payer != payer || r.FrameResults8141[len(frame.Frames)-1].Status != 0 || db.GetNonce(frame.Sender) != 1 || db.GetState(token, types.Hash{}) != (types.Hash{31: 100}) || !db.GetState(token, types.Hash{31: 1}).IsZero() || len(r.Logs) != 0 || db.GetBalance(payer).Cmp(before) >= 0 || db.GetBalance(frame.Sender).Cmp(senderBefore) != 0 {
				t.Fatal("ASSERT failed to preserve payment/prefix or roll back token", atomic, returnsFalse, r)
			}
			if r.FrameResults8141[2].GasUsed.State != 0 || r.FrameResults8141[2].Status != 1 {
				t.Fatal("body receipt state/status")
			}
			// A separately authorized nonce-one retry can commit. No automatic
			// token compensation is invented for the first attempt's paid ETH gas.
			paidAfterFailure := new(big.Int).Set(db.GetBalance(payer))
			frame.Nonce = 1
			db.SetCode(assert, []byte{0})
			r = apply()
			if db.GetState(token, types.Hash{}) != (types.Hash{31: 95}) || db.GetState(token, types.Hash{31: 1}) != (types.Hash{31: 5}) || db.GetNonce(frame.Sender) != 2 || db.GetBalance(payer).Cmp(paidAfterFailure) >= 0 || r.FrameResults8141[2].Status != 1 {
				t.Fatal("retry/false-return semantics", atomic, returnsFalse, r)
			}
		}
	}
}

func TestAssertion7906CaughtChildApprovalIsOrdinaryStaticFailure(t *testing.T) {
	frame, db, cfg, h := frameExecutionFixture(t)
	zero := uint64(0)
	cfg.EIP7906Time = &zero
	assert, helper := types.Address{19: 0x98}, types.Address{19: 0x99}
	db.SetCode(helper, []byte{0x60, 3, 0x5f, 0x5f, 0xaa})
	code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
	code = append(code, helper[:]...)
	code = append(code, 0x61, 0xff, 0xff, 0xfa, 0x50, 0x00)
	db.SetCode(assert, code)
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeAssert7906, Target: &assert, Limits: types.FrameGasLimits{Execution: 100000}})
	r, _, err := ApplyTransaction(cfg, db, h, signFrameExecution(t, frame, false), new(gaspool.GasPool).AddGas(h.GasLimit))
	if err != nil || r.FrameResults8141[1].Status != 1 || db.GetNonce(frame.Sender) != 1 {
		t.Fatal("caught child APPROVE latched a transaction failure", r, err)
	}
}

type assertionOutputs7906 map[int][]byte

func (o assertionOutputs7906) BeginFrame(int) {}
func (o assertionOutputs7906) EndFrame(i int, data []byte, _, _ uint64, _ error) {
	o[i] = append([]byte(nil), data...)
}

func TestAssertion7906ActualEscrowAndPrefixPrestateTrace(t *testing.T) {
	frame, db, cfg, h := frameExecutionFixture(t)
	zero := uint64(0)
	cfg.EIP7906Time = &zero
	assert := types.Address{19: 0x9a}
	selectors := []byte{0x19, 0x1a, 0, 4, 5, 6, 3, 0xe, 0xf, 0x10}
	var code []byte
	for i, param := range selectors {
		offset := i * 32
		code = append(code, 0x5f, 0x60, param, byte(vm.TXTRACE), 0x61, byte(offset>>8), byte(offset), 0x52)
	}
	length := len(selectors) * 32
	code = append(code, 0x61, byte(length>>8), byte(length), 0x5f, 0xf3)
	db.SetCode(assert, code)
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeAssert7906, Target: &assert, Limits: types.FrameGasLimits{Execution: 10000}})
	tx := signFrameExecution(t, frame, false)
	payment, err := frame.Payment(h.BaseFee, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	before := new(big.Int).Set(db.GetBalance(frame.Sender))
	outputs := make(assertionOutputs7906)
	r, _, err := ApplyTransactionWithVMConfig(cfg, nil, db, h, tx, new(gaspool.GasPool).AddGas(h.GasLimit), vm.Config{FrameObserver: outputs})
	if err != nil || r.FrameResults8141[1].Status != 1 || len(outputs[1]) != length {
		t.Fatal("trace consumer", err, r)
	}
	want := []*big.Int{payment.Escrow(), new(big.Int).SetBytes(frame.Sender[:]), big.NewInt(1), new(big.Int).SetBytes(frame.Sender[:]), before, new(big.Int).Sub(new(big.Int).Set(before), payment.Escrow()), big.NewInt(1), new(big.Int).SetBytes(frame.Sender[:]), new(big.Int), big.NewInt(1)}
	for i, value := range want {
		if new(big.Int).SetBytes(outputs[1][i*32:(i+1)*32]).Cmp(value) != 0 {
			t.Fatal("trace selector", selectors[i], "does not reflect prestate/actual escrow", value)
		}
	}
	if payment.Escrow().Cmp(frame.Fees.MaxFee) == 0 {
		t.Fatal("fixture did not distinguish actual precharge from signed max_fee")
	}
}
