package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

type assertionCacheReads7906 struct {
	state.StateDB
	active bool
	reads  int
}

func (s *assertionCacheReads7906) GetState(a types.Address, k types.Hash) types.Hash {
	if s.active {
		s.reads++
	}
	return s.StateDB.GetState(a, k)
}
func (s *assertionCacheReads7906) BeginFrame(index int) { s.active = index == 2 }
func (s *assertionCacheReads7906) EndFrame(int, []byte, uint64, uint64, error) {
	s.active = false
}

// EIP-7906's flat-priced reads must reuse the same index while the diff is
// unchanged. A caught STATICCALL revert must not amplify index construction.
func TestAssertion7906TraceCacheSurvivesReadOnlyReverts(t *testing.T) {
	frame, db, cfg, h := frameExecutionFixture(t)
	zero := uint64(0)
	cfg.EIP7906Time = &zero
	body, assertion, helper := types.Address{19: 0xb1}, types.Address{19: 0xb2}, types.Address{19: 0xb3}
	db.SetCode(body, []byte{0x60, 1, 0x5f, 0x55, 0})
	readCount := []byte{0x5f, 0x60, 1, byte(vm.TXTRACE), 0x50}
	db.SetCode(helper, append(append([]byte{}, readCount...), 0x5f, 0x5f, 0xfd))
	code := append([]byte{}, readCount...)
	for i := 0; i < 16; i++ {
		code = append(code, 0x5f, 0x5f, 0x5f, 0x5f, 0x73)
		code = append(code, helper[:]...)
		code = append(code, 0x61, 0x13, 0x88, 0xfa, 0x50)
	}
	code = append(code, readCount...)
	code = append(code, 0)
	db.SetCode(assertion, code)
	frame.Frames = append(frame.Frames,
		types.Frame8141{Mode: types.ModeSender, Target: &body, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}},
		types.Frame8141{Mode: types.ModeAssert7906, Target: &assertion, Limits: types.FrameGasLimits{Execution: 100000}},
	)
	spy := &assertionCacheReads7906{StateDB: db}
	r, _, err := ApplyTransactionWithVMConfig(cfg, nil, spy, h, signFrameExecution(t, frame, false), new(gaspool.GasPool).AddGas(h.GasLimit), vm.Config{FrameObserver: spy})
	if err != nil || r.FrameResults8141[2].Status != 1 || db.GetState(body, types.Hash{}) != (types.Hash{31: 1}) {
		t.Fatal("caught static reverts changed transaction outcome", r, err)
	}
	if spy.reads != 1 {
		t.Fatalf("unchanged diff rebuilt after static child revert: %d slot reads, want one index build", spy.reads)
	}
}

func TestAssertion7906TraceCacheTracksObservableMutations(t *testing.T) {
	db := state.NewMemoryStateDB()
	a, key, txHash := types.Address{19: 0xc1}, types.Hash{31: 1}, types.Hash{31: 2}
	db.SetTxContext(txHash, 0)
	s := newAssertionState7906(db, txHash)
	before := s.Snapshot()
	empty := s.index()
	s.AddBalance(a, big.NewInt(3))
	s.SetNonce(a, 1)
	s.SetCode(a, []byte{0})
	s.SetState(a, key, types.Hash{31: 4})
	s.AddLog(&types.Log{Address: a, Data: []byte{5}})
	changed := s.index()
	if changed == empty || len(changed.Balances) != 1 || len(changed.Nonces) != 1 || len(changed.Codes) != 1 || len(changed.Slots) != 1 || len(changed.Events) != 1 {
		t.Fatal("cached index survived observable mutation", changed)
	}
	readOnly := s.Snapshot()
	s.AddBalance(a, new(big.Int)) // STATICCALL's zero-value account touch.
	s.AddAddressToAccessList(a)
	s.AddSlotToAccessList(a, key)
	s.RevertToSnapshot(readOnly)
	if s.index() != changed {
		t.Fatal("warmness-only snapshot invalidated immutable trace")
	}
	s.RevertToSnapshot(before)
	restored := s.index()
	if len(restored.Balances)+len(restored.Nonces)+len(restored.Codes)+len(restored.Slots)+len(restored.Events) != 0 {
		t.Fatal("mutating rollback retained stale tables", restored)
	}
	// Snapshots can be reused by the underlying journal. A subsequent nested
	// mutation/revert must not alias a discarded cache generation.
	id := s.Snapshot()
	s.SetState(a, key, types.Hash{31: 6})
	if len(s.index().Slots) != 1 {
		t.Fatal("post-revert write did not rebuild")
	}
	s.RevertToSnapshot(id)
	if len(s.index().Slots) != 0 {
		t.Fatal("reused snapshot restored a stale index")
	}
}
