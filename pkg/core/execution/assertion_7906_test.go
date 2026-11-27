package execution

import (
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

func TestAssertion7906FailureWithoutPayerInvalidatesTransaction(t *testing.T) {
	tx, db, cfg, h := frameExecutionFixture(t)
	zero := uint64(0)
	cfg.EIP7906Time = &zero
	body, assertion := types.Address{19: 0xa1}, types.Address{19: 0xa2}
	db.SetCode(tx.Sender, []byte{0})
	db.SetCode(body, []byte{0x60, 1, 0x5f, 0x55, 0})
	db.SetCode(assertion, []byte{0x5f, 0x5f, 0xfd})
	tx.Frames = []types.Frame8141{{Mode: types.ModeDefault, Target: &body, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}}, {Mode: types.ModeAssert7906, Target: &assertion, Limits: types.FrameGasLimits{Execution: 10000}}}
	transaction := types.NewTransaction(tx)
	before := db.GetRoot()
	gp := new(gaspool.GasPool).AddGas(h.GasLimit)
	r, _, err := ApplyTransaction(cfg, db, h, transaction, gp)
	if err == nil || !strings.Contains(err.Error(), "ASSERT failed before payment approval") || r != nil || before != db.GetRoot() || gp.Gas() != h.GasLimit || gp.StateGas() != h.GasLimit {
		t.Fatal("unpaid assertion failure did not invalidate/revert transaction", r, err)
	}
}

func TestAssertion7906WholeBodyRollbackAndPrefixSettlement(t *testing.T) {
	for _, assertion := range [][]byte{{0}, {0x5f, 0x5f, 0xfd}, {0xfe}, {0x5f, 0x5f, 0x60, 3, 0xaa}} {
		tx, db, cfg, h := frameExecutionFixture(t)
		zero := uint64(0)
		cfg.EIP7906Time = &zero
		body, assert := types.Address{19: 0x90}, types.Address{19: 0x91}
		db.SetCode(body, []byte{0x60, 1, 0x5f, 0x55, 0x5f, 0x5f, 0xa0, 0})
		db.SetCode(assert, assertion)
		tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &body, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}}, types.Frame8141{Mode: types.ModeAssert7906, Target: &assert, Limits: types.FrameGasLimits{Execution: 100000}}, types.Frame8141{Mode: types.ModeAssert7906, Target: &body, Limits: types.FrameGasLimits{Execution: 10000}})
		transaction := signFrameExecution(t, tx, false)
		db.SetTxContext(transaction.Hash(), 0)
		before := new(big.Int).Set(db.GetBalance(tx.Sender))
		tracker := bal.NewTracker()
		gp := new(gaspool.GasPool).AddGas(h.GasLimit)
		r, used, err := ApplyTransactionWithBAL(cfg, db, h, transaction, gp, tracker)
		if err != nil {
			t.Fatal(err)
		}
		// Even a successful first assertion is followed by a static SSTORE, so
		// every case reverts the whole body and retains nonce/payment.
		if !db.GetState(body, types.Hash{}).IsZero() || db.GetNonce(tx.Sender) != 1 || len(r.Logs) != 0 || r.FrameResults8141[1].Status != 1 || r.FrameResults8141[1].GasUsed.State != 0 || r.FrameResults8141[1].GasUsed.Execution == 0 {
			t.Fatal("rollback or prefix", r)
		}
		if assertion[0] != 0 && r.FrameResults8141[3].Status != types.FrameStatusSkipped {
			t.Fatal("remaining assertion executed")
		}
		charge := new(big.Int).Sub(before, db.GetBalance(tx.Sender))
		if charge.Cmp(new(big.Int).Mul(new(big.Int).SetUint64(used), big.NewInt(3))) != 0 {
			t.Fatal("gas not charged", charge, used)
		}
		if gp.StateGas() != h.GasLimit {
			t.Fatal("body state gas not refunded")
		}
		list := tracker.Build(1)
		found := false
		for _, entry := range list.Entries {
			if entry.Address == body {
				found = true
				if len(entry.StorageChanges) != 0 || len(entry.StorageReads) != 1 {
					t.Fatal("rollback lost BAL read", entry)
				}
			}
		}
		if !found {
			t.Fatal("missing body access")
		}
	}
}

func TestAssertion7906PrestateNetDiffSortedAndLazy(t *testing.T) {
	db := state.NewMemoryStateDB()
	a, b := types.Address{19: 1}, types.Address{19: 2}
	k1, k2 := types.Hash{31: 1}, types.Hash{31: 2}
	db.AddBalance(a, big.NewInt(10))
	db.SetNonce(a, 4)
	db.SetState(a, k1, types.Hash{31: 7})
	s := newAssertionState7906(db, types.Hash{})
	s.AddBalance(b, big.NewInt(2))
	s.SubBalance(a, big.NewInt(3))
	s.SetNonce(a, 5)
	s.SetCode(b, []byte{0})
	s.SetState(a, k2, types.Hash{31: 9})
	s.SetState(a, k1, types.Hash{31: 8})
	id := s.Snapshot()
	s.SetState(b, k1, types.Hash{31: 5})
	s.RevertToSnapshot(id)
	s.SetState(a, k1, types.Hash{31: 7}) // No net change despite two writes.
	if s.trace != nil {
		t.Fatal("eager trace index")
	}
	trace := s.index()
	if len(trace.Balances) != 2 || trace.Balances[0].Address != a || len(trace.Slots) != 1 || trace.Slots[0].Key != k2 || len(trace.Nonces) != 1 || trace.Nonces[0].Before.Uint64() != 4 || len(trace.Codes) != 1 || trace.Codes[0].Before.Cmp(new(big.Int).SetBytes(types.EmptyCodeHash[:])) != 0 {
		t.Fatal("net/prestate/order", trace)
	}
	if s.index() != trace {
		t.Fatal("index not cached")
	}
}

func TestAssertion7906TraceOpcodesExecuteAgainstPaidPrefix(t *testing.T) {
	tx, db, cfg, h := frameExecutionFixture(t)
	zero := uint64(0)
	cfg.EIP7906Time = &zero
	assert := types.Address{19: 0x92}
	// Assert the observed nonce table count is one; use TXTRACE operand order.
	db.SetCode(assert, []byte{0x5f, 0x60, 3, byte(vm.TXTRACE), 0x60, 1, 0x14, 0x60, 14, 0x57, 0x5f, 0x5f, 0xfd, 0x00, 0x5b, 0x00})
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeAssert7906, Target: &assert, Limits: types.FrameGasLimits{Execution: 10000}})
	r, _, err := ApplyTransaction(cfg, db, h, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(h.GasLimit))
	if err != nil || r.FrameResults8141[1].Status != 1 {
		t.Fatal("paid prefix trace", r, err)
	}
}
