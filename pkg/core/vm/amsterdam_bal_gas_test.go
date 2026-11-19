package vm

import (
	"errors"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestAmsterdamSstoreBALPreAndPostStateGasBoundary(t *testing.T) {
	for _, test := range []struct {
		budget uint64
		reads  int
	}{{2305, 0}, {2306, 1}} {
		db := state.NewMemoryStateDB()
		sender, contract := types.Address{19: 0x99}, types.Address{0: 0x10, 19: 1}
		db.AddBalance(sender, big.NewInt(1_000_000))
		db.SetNonce(contract, 1)
		db.SetCode(contract, []byte{0x60, 1, 0x5f, 0x55, 0})
		tracker := new(mockBALTracker)
		evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
		rules := ForkRules{IsAmsterdam: true, IsPrague: true, IsBerlin: true, IsShanghai: true}
		evm.SetForkRules(rules)
		evm.SetJumpTable(SelectJumpTable(rules))
		evm.SetBALTracker(tracker, 0)
		if _, _, err := evm.Call(sender, contract, nil, test.budget, new(big.Int)); !errors.Is(err, ErrOutOfGas) {
			t.Fatal("expected opcode OOG")
		}
		if len(tracker.storageReads) != test.reads || len(tracker.storageChanges) != 0 || db.GetState(contract, types.Hash{}) != (types.Hash{}) {
			t.Fatalf("budget %d reads=%d changes=%d", test.budget, len(tracker.storageReads), len(tracker.storageChanges))
		}
	}
}
func TestAmsterdamDelegationBALBeforeBalanceFailure(t *testing.T) {
	for _, budget := range []uint64{16000, 100000} {
		db := state.NewMemoryStateDB()
		sender, parent, authority, delegate := types.Address{19: 0x99}, types.Address{0: 0x10, 19: 1}, types.Address{0: 0x20, 19: 2}, types.Address{0: 0x30, 19: 3}
		db.AddBalance(sender, big.NewInt(1_000_000))
		db.AddBalance(parent, big.NewInt(5))
		db.SetNonce(parent, 1)
		db.SetCode(authority, types.AddressToDelegation(delegate))
		db.SetNonce(authority, 1)
		db.SetCode(delegate, []byte{0})
		db.SetNonce(delegate, 1)
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x60, 7, 0x73}
		code = append(code, authority[:]...)
		code = append(code, 0x5a, 0xf1, 0x50, 0)
		db.SetCode(parent, code)
		tracker := new(mockBALTracker)
		evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
		rules := ForkRules{IsAmsterdam: true, IsPrague: true, IsBerlin: true, IsShanghai: true}
		evm.SetForkRules(rules)
		evm.SetJumpTable(SelectJumpTable(rules))
		evm.SetBALTracker(tracker, 0)
		_, _, err := evm.Call(sender, parent, nil, budget, new(big.Int))
		if budget == 16000 && !errors.Is(err, ErrOutOfGas) || budget == 100000 && err != nil {
			t.Fatalf("budget %d outcome: %v", budget, err)
		}
		seen := make(map[types.Address]bool)
		for _, address := range tracker.addressTouches {
			seen[address] = true
		}
		if !seen[authority] || seen[delegate] != (budget == 100000) {
			t.Fatalf("budget %d authority=%t delegate=%t", budget, seen[authority], seen[delegate])
		}
		if db.GetBalance(parent).Int64() != 5 {
			t.Fatal("insufficient-balance child transferred value")
		}
	}
}
