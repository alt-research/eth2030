package vm

import (
	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestFrameCPublicVerifierPermissionsAreExactAndTopLevel(t *testing.T) {
	db := state.NewMemoryStateDB()
	ref := types.RecentRootReference{Slot: 1, Root: types.Hash{31: 1}}
	db.SetCode(types.RecentRootAddress, types.RecentRootCode)
	frame := types.Frame8141{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: ref.AppendTo(nil)}
	tx := &types.FrameTx8141{Sender: types.Address{19: 42}, Frames: []types.Frame8141{frame, {Mode: types.ModeVerify, Flags: 3}}}
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
	ctx := evm.SetFrameTransaction(tx, nil)
	ctx.PublicValidation = true
	evm.depth = 1
	contract := NewContract(types.EntryPointAddress, types.RecentRootAddress, new(big.Int), 100000)
	contract.Code = types.RecentRootCode
	key := ref.StorageKey()
	stack := NewStack()
	stack.Push(new(big.Int).SetBytes(key[:]))
	if err := evm.checkFrameValidation(SLOTNUM, contract, stack, 0); err != nil {
		t.Fatal(err)
	}
	if err := evm.checkFrameValidation(SLOAD, contract, stack, 0); err != nil {
		t.Fatal(err)
	}
	stack.Pop()
	stack.Push(big.NewInt(9))
	if err := evm.checkFrameValidation(SLOAD, contract, stack, 0); err == nil {
		t.Fatal("undeclared root key allowed")
	}
	evm.depth = 2
	stack.Pop()
	stack.Push(new(big.Int).SetBytes(key[:]))
	for _, op := range []OpCode{SLOTNUM, SLOAD} {
		if err := evm.checkFrameValidation(op, contract, stack, 0); err == nil {
			t.Fatalf("nested %s inherited root permission", op)
		}
	}
	evm.depth = 1
	db.SetCode(types.RecentRootAddress, []byte{0})
	if err := evm.checkFrameValidation(SLOTNUM, contract, stack, 0); err == nil {
		t.Fatal("incorrect deployed runtime allowed")
	}
	// Matching expiry bytecode used as a helper is not an expiry frame.
	tx.Frames[0] = types.Frame8141{Mode: types.ModeVerify, Flags: 3}
	contract.Address = types.FrameExpiryVerifier
	contract.Code = types.FrameExpiryCode
	if err := evm.checkFrameValidation(TIMESTAMP, contract, stack, 0); err == nil {
		t.Fatal("helper inherited expiry timestamp permission")
	}
	tx.Frames[0] = types.Frame8141{Mode: types.ModeVerify, Target: &types.FrameExpiryVerifier, Data: make([]byte, 8)}
	if err := evm.checkFrameValidation(TIMESTAMP, contract, stack, 0); err != nil {
		t.Fatal(err)
	}
	evm.depth = 2
	if err := evm.checkFrameValidation(TIMESTAMP, contract, stack, 0); err == nil {
		t.Fatal("nested expiry permission")
	}
}
func TestFrameCSenderDefaultCodeCallException(t *testing.T) {
	db := state.NewMemoryStateDB()
	tx := &types.FrameTx8141{Sender: types.Address{19: 42}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3}}}
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
	ctx := evm.SetFrameTransaction(tx, nil)
	ctx.PublicValidation = true
	stack := NewStack()
	stack.Push(new(big.Int).SetBytes(tx.Sender[:]))
	contract := NewContract(types.EntryPointAddress, tx.Sender, new(big.Int), 10000)
	if err := evm.checkFrameValidation(EXTCODESIZE, contract, stack, 0); err != nil {
		t.Fatalf("sender default-code exception rejected: %v", err)
	}
	stack.Pop()
	stack.Push(big.NewInt(99))
	if err := evm.checkFrameValidation(EXTCODESIZE, contract, stack, 0); err == nil {
		t.Fatal("empty third party accepted")
	}
}

type frameRecognitionReads struct {
	StateDB
	codeReads int
}

func (s *frameRecognitionReads) GetCode(address types.Address) []byte {
	s.codeReads++
	return s.StateDB.GetCode(address)
}

func TestPR12041PaymasterRecognitionIsFrameBounded(t *testing.T) {
	db := state.NewMemoryStateDB()
	payer := types.Address{19: 0x93}
	db.SetCode(payer, make([]byte, 24576))
	reads := &frameRecognitionReads{StateDB: db}
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, reads)
	pay := types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer}
	ctx := evm.SetFrameTransaction(&types.FrameTx8141{Sender: types.Address{19: 42}, Frames: []types.Frame8141{pay, pay}}, nil)
	ctx.PublicValidation = true
	contract := NewContract(types.EntryPointAddress, payer, new(big.Int), 100000)
	stack := NewStack()
	for _, profile := range []framepay.Profile{framepay.NoPaymaster, framepay.LocalV1, framepay.PR12041} {
		evm.Config.FramePaymasterProfile = profile
		reads.codeReads = 0
		for i := 0; i < 100; i++ {
			if err := evm.checkFrameValidation(PUSH0, contract, stack, 0); err != nil {
				t.Fatal(err)
			}
		}
		want := 1
		if profile == framepay.NoPaymaster {
			want = 0
		}
		if reads.codeReads > want {
			t.Fatalf("profile %d copied code %d times, want at most %d per immutable frame", profile, reads.codeReads, want)
		}
	}
	// A later frame observes its own runtime, and a profile change must not
	// reuse another version's permission. Cache state follows frame snapshots.
	db.SetCode(payer, framepay.Runtime12041())
	contract.Code = framepay.Runtime12041()
	ctx.Index = 1
	reads.codeReads = 0
	for i := 0; i < 100; i++ {
		if err := evm.checkFrameValidation(SLOAD, contract, stack, 0); err != nil {
			t.Fatal("exact proposal lost exemption", err)
		}
	}
	if reads.codeReads > 1 {
		t.Fatal("exact runtime recognition repeated per opcode")
	}
	id := evm.StateDB.Snapshot()
	ctx.Index = 0
	reads.codeReads = 0
	if err := evm.checkFrameValidation(SLOAD, contract, stack, 0); err != nil || reads.codeReads > 1 {
		t.Fatal("frame identity did not refresh recognition", err)
	}
	evm.StateDB.RevertToSnapshot(id)
	reads.codeReads = 0
	if ctx.Index != 1 {
		t.Fatal("frame snapshot not restored")
	}
	if err := evm.checkFrameValidation(SLOAD, contract, stack, 0); err != nil || reads.codeReads != 0 {
		t.Fatal("restored immutable frame lost cached recognition", err)
	}
	for _, profile := range []framepay.Profile{framepay.NoPaymaster, framepay.LocalV1} {
		evm.Config.FramePaymasterProfile = profile
		reads.codeReads = 0
		if err := evm.checkFrameValidation(SLOAD, contract, stack, 0); err == nil {
			t.Fatal("stale proposal permission crossed profile boundary")
		}
		want := 1
		if profile == framepay.NoPaymaster {
			want = 0
		}
		if reads.codeReads > want {
			t.Fatal("unexpected recognition read count")
		}
	}
}
