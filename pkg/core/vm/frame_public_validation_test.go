package vm

import (
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
