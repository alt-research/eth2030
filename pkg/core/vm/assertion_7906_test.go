package vm

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func TestAssertion7906AllSelectorsAndFullWidthBounds(t *testing.T) {
	a := types.Address{19: 0x41}
	entry := AssertionChange7906{Address: a, Before: big.NewInt(7), After: big.NewInt(9), Key: types.Hash{31: 11}}
	trace := &AssertionTrace7906{Balances: []AssertionChange7906{entry}, Slots: []AssertionChange7906{entry}, Codes: []AssertionChange7906{entry}, Nonces: []AssertionChange7906{entry}, Events: []*types.Log{{Address: a, Topics: []types.Hash{{31: 12}, {31: 13}, {31: 14}, {31: 15}}, Data: []byte{16, 17}}}}
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{FrameAssertionTrace: func() *AssertionTrace7906 { return trace }}, state.NewMemoryStateDB())
	evm.SetForkRules(ForkRules{IsEIP7906: true})
	tx := &types.FrameTx8141{Frames: []types.Frame8141{{Mode: types.ModeAssert7906}}}
	ctx := evm.SetFrameTransaction(tx, nil)
	ctx.Payer = &a
	wants := []uint64{1, 1, 1, 1, 0x41, 7, 9, 0x41, 7, 9, 11, 0x41, 7, 9, 0x41, 7, 9, 1, 0x41, 4, 12, 13, 14, 15, 2, 0, 0x41}
	for p, want := range wants {
		s := NewStack()
		s.Push(new(big.Int))
		s.Push(big.NewInt(int64(p)))
		if _, err := opTxTrace7906(nil, evm, nil, nil, s); err != nil || s.Len() != 1 || s.Peek().Uint64() != want {
			t.Fatal("selector", p, s.Peek(), want, err)
		}
	}
	wide := new(big.Int).Lsh(big.NewInt(1), 200)
	for p := uint64(0); p <= 0x1a; p++ {
		for _, index := range []*big.Int{big.NewInt(1), wide} {
			s := NewStack()
			s.Push(new(big.Int).Set(index))
			s.Push(new(big.Int).SetUint64(p))
			if _, err := opTxTrace7906(nil, evm, nil, nil, s); err == nil {
				t.Fatal("index/reserved truncation", p, index)
			}
		}
	}
	for _, param := range []*big.Int{big.NewInt(0x1b), wide} {
		s := NewStack()
		s.Push(new(big.Int))
		s.Push(param)
		if _, err := opTxTrace7906(nil, evm, nil, nil, s); err == nil {
			t.Fatal("selector truncation")
		}
	}
	trace.Events[0].Topics = nil
	s := NewStack()
	s.Push(new(big.Int))
	s.Push(big.NewInt(0x14))
	if _, err := opTxTrace7906(nil, evm, nil, nil, s); err == nil {
		t.Fatal("missing topic accepted")
	}
	for _, mode := range []uint8{types.ModeDefault, types.ModeVerify, types.ModeSender, types.ModeProof8504} {
		tx.Frames[0].Mode = mode
		s := NewStack()
		s.Push(new(big.Int))
		s.Push(new(big.Int))
		if _, err := opTxTrace7906(nil, evm, nil, nil, s); err == nil {
			t.Fatal("trace outside ASSERT", mode)
		}
	}
}

func TestAssertion7906EventCopyBoundsAndGas(t *testing.T) {
	db := state.NewMemoryStateDB()
	tracker := new(mockBALTracker)
	trace := &AssertionTrace7906{Events: []*types.Log{{Data: []byte{1, 2, 3}}}}
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{FrameAssertionTrace: func() *AssertionTrace7906 { return trace }}, db)
	evm.SetForkRules(ForkRules{IsEIP7906: true})
	evm.SetBALTracker(tracker, 0)
	evm.SetFrameTransaction(&types.FrameTx8141{Frames: []types.Frame8141{{Mode: types.ModeAssert7906}}}, nil)
	wide := new(big.Int).Lsh(big.NewInt(1), 256)
	wide.Sub(wide, big.NewInt(1))
	for _, test := range []struct {
		index, offset, length *big.Int
		valid                 bool
	}{
		{big.NewInt(0), big.NewInt(1), big.NewInt(2), true}, {big.NewInt(0), big.NewInt(3), big.NewInt(0), true},
		{big.NewInt(0), big.NewInt(4), big.NewInt(0), false}, {big.NewInt(0), wide, big.NewInt(2), false},
		{big.NewInt(1), big.NewInt(0), big.NewInt(0), false}, {wide, big.NewInt(0), big.NewInt(0), false},
	} {
		s := NewStack()
		s.Push(test.length)
		s.Push(test.offset)
		s.Push(big.NewInt(0))
		s.Push(test.index)
		mem := NewMemory()
		c := NewContract(types.Address{}, types.Address{}, nil, 100000)
		size, overflow := memoryEventData7906(s)
		if overflow {
			t.Fatal("unexpected memory overflow")
		}
		cost, err := gasEventData7906(evm, c, s, mem, size)
		if (err == nil) != test.valid {
			t.Fatal("source bounds", test, err)
		}
		if test.valid {
			if test.length.Sign() != 0 {
				if cost != 6 {
					t.Fatal("copy+memory gas", cost)
				}
				mem.Resize(32)
			}
			if _, err := opEventDataCopy7906(nil, evm, c, mem, s); err != nil {
				t.Fatal(err)
			}
			if test.length.Sign() != 0 && !bytes.Equal(mem.Get(0, 2), []byte{2, 3}) {
				t.Fatal("event bytes")
			}
		}
	}
	if len(tracker.addressTouches) != 0 || db.AddressInAccessList(types.Address{}) {
		t.Fatal("trace caused access/BAL effects")
	}
}
