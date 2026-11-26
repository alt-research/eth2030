package txpool

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func proposalPaymasterPool(t *testing.T) (*TxPool, *state.MemoryStateDB, *types.FrameTx8141, types.Address) {
	t.Helper()
	pool, db, frame := framePoolFixture(t)
	pool.config.ExperimentalFramePaymaster12041 = true
	payer := types.Address{19: 0x93}
	owner := framePoolAddress(t, 2)
	db.SetCode(payer, framepay.Runtime12041())
	db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
	db.AddBalance(payer, big.NewInt(100000000))
	frame.Frames[0].Flags = 2
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 15000}})
	return pool, db, frame, payer
}

func TestPR12041PoolExactSelectionAndUsage(t *testing.T) {
	for _, kind := range []string{"enabled", "disabled", "both", "mutated", "delegated", "local_v1", "gas_over", "admin_data"} {
		t.Run(kind, func(t *testing.T) {
			pool, db, frame, payer := proposalPaymasterPool(t)
			switch kind {
			case "disabled":
				pool.config.ExperimentalFramePaymaster12041 = false
			case "both":
				pool.config.ExperimentalFramePaymasterV1 = true
			case "mutated":
				db.SetCode(payer, append(framepay.Runtime12041(), 0))
			case "delegated":
				target := types.Address{19: 0x94}
				db.SetCode(target, framepay.Runtime12041())
				db.SetCode(payer, types.AddressToDelegation(target))
			case "local_v1":
				pool.config.ExperimentalFramePaymaster12041 = false
				pool.config.ExperimentalFramePaymasterV1 = true
			case "gas_over":
				frame.Frames[1].Limits.Execution = 15001
			case "admin_data":
				frame.Frames[1].Data = []byte{3}
			}
			tx := signPoolFrame(t, frame, 1, 2)
			root := db.GetRoot()
			err := pool.AddRemote(tx)
			if (err == nil) != (kind == "enabled") {
				t.Fatalf("%s: %v", kind, err)
			}
			if db.GetRoot() != root {
				t.Fatal("admission leaked state")
			}
			if kind == "enabled" {
				other := *frame
				other.Sender = framePoolAddress(t, 3)
				if err := pool.AddRemote(signPoolFrame(t, &other, 3, 2)); err != nil {
					t.Fatal("proposal incorrectly limited to generic PAY cap", err)
				}
			}
		})
	}
}

func TestPR12041PoolWithdrawalReservationAndLifecycle(t *testing.T) {
	pool, db, frame, payer := proposalPaymasterPool(t)
	db.SetState(payer, types.Hash{31: 1}, types.BytesToHash(big.NewInt(98000000).Bytes()))
	first := signPoolFrame(t, frame, 1, 2)
	other := *frame
	other.Sender = framePoolAddress(t, 3)
	second := signPoolFrame(t, &other, 3, 2)
	for _, tx := range []*types.Transaction{first, second} {
		if err := pool.AddRemote(tx); err != nil {
			t.Fatal("withdrawal was subtracted more than once", err)
		}
	}
	thirdFrame := *frame
	thirdFrame.Sender = framePoolAddress(t, 4)
	third := signPoolFrame(t, &thirdFrame, 4, 2)
	if err := pool.AddRemote(third); err != ErrInsufficientFunds {
		t.Fatalf("oversubscribed raw slot-1 reservation: %v", err)
	}
	for _, clock := range []byte{0, 1, 255} {
		db.SetState(payer, types.Hash{31: 2}, types.Hash{31: clock})
		pool.Reset(db)
		if pool.Count() != 2 {
			t.Fatal("maturity released or duplicated withdrawal reservation")
		}
	}
	pool.Remove(first.Hash())
	if err := pool.AddRemote(third); err != nil {
		t.Fatal("removal did not release max_fee", err)
	}
	db.SetState(payer, types.Hash{31: 1}, types.BytesToHash(big.NewInt(99000000).Bytes()))
	pool.Reset(db)
	if pool.Count() != 1 {
		t.Fatal("head withdrawal change did not arbitrate aggregate exposure")
	}
	// A reverted/cancelled withdrawal restores available funds after head reset.
	db.SetState(payer, types.Hash{31: 1}, types.Hash{})
	db.SetState(payer, types.Hash{31: 2}, types.Hash{})
	pool.Reset(db)
	for _, tx := range []*types.Transaction{first, second, third} {
		if pool.Get(tx.Hash()) == nil {
			if err := pool.AddRemote(tx); err != nil {
				t.Fatal("rollback/re-admission", err)
			}
		}
	}
	for _, tx := range []*types.Transaction{first, second, third} {
		db.SetNonce(tx.FrameSender(), 1)
	}
	pool.Reset(db, &types.Header{Number: big.NewInt(2), Time: 101, GasLimit: pool.config.BlockGasLimit, BaseFee: big.NewInt(1)})
	if pool.Count() != 0 || len(pool.lookup.framePayers) != 0 || len(pool.lookup.frameDependencies) != 0 {
		t.Fatal("inclusion retained exposure or indexes")
	}
}

func TestPR12041PoolTracksAllSlotsAndRotation(t *testing.T) {
	pool, db, frame, payer := proposalPaymasterPool(t)
	enableRecovery8151(pool)
	tx := signPoolFrame(t, frame, 1, 2)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	validation := pool.lookup.frameValidations[tx.Hash()]
	if len(validation.SignerRecoveries) != 0 {
		t.Fatal("protocol signature invented recovery shares")
	}
	owner := framePoolAddress(t, 2)
	db.SetCode(owner, []byte{0})
	pool.Reset(db)
	if pool.lookup.frameValidations[tx.Hash()] != validation {
		t.Fatal("SIGPARAM authority acquired a false stateful ecRecover dependency")
	}
	for _, slot := range []byte{1, 2, 3} {
		dep := execution.FrameDependency{Address: payer, Kind: execution.FrameStorageDependency, Slot: types.Hash{31: slot}}
		if _, ok := validation.Dependencies[dep]; !ok {
			t.Fatalf("missing slot %d", slot)
		}
		db.SetState(payer, dep.Slot, types.Hash{31: 1})
		pool.Reset(db)
		next := pool.lookup.frameValidations[tx.Hash()]
		if next == nil || next == validation {
			t.Fatalf("slot %d ignored", slot)
		}
		validation = next
	}
	newOwner := framePoolAddress(t, 3)
	db.SetState(payer, types.Hash{}, types.BytesToHash(newOwner[:]))
	pool.Reset(db)
	if pool.Get(tx.Hash()) != nil || len(pool.lookup.framePayers) != 0 || len(pool.lookup.frameDependencies) != 0 {
		t.Fatal("rotation retained old authorization")
	}
	if err := pool.AddRemote(signPoolFrame(t, frame, 1, 3)); err != nil {
		t.Fatal("new authority rejected", err)
	}
	pool.config.ExperimentalFramePaymaster12041 = false
	pool.Reset(db)
	if pool.Count() != 0 {
		t.Fatal("profile change retained stale canonical privilege")
	}
	pool.config.ExperimentalFramePaymaster12041 = true
	if err := pool.AddRemote(signPoolFrame(t, frame, 1, 3)); err != nil {
		t.Fatal(err)
	}
	db.SetCode(payer, append(framepay.Runtime12041(), 0))
	pool.Reset(db)
	if pool.Count() != 0 {
		t.Fatal("code mutation retained privilege")
	}
}

func TestPR12041PoolReplacementMovesExposure(t *testing.T) {
	pool, db, frame, payer := proposalPaymasterPool(t)
	db.SetState(payer, types.Hash{31: 1}, types.BytesToHash(big.NewInt(99000000).Bytes()))
	first := signPoolFrame(t, frame, 1, 2)
	if err := pool.AddRemote(first); err != nil {
		t.Fatal(err)
	}
	alternate := types.Address{19: 0x95}
	owner := framePoolAddress(t, 2)
	db.SetCode(alternate, framepay.Runtime12041())
	db.SetState(alternate, types.Hash{}, types.BytesToHash(owner[:]))
	db.AddBalance(alternate, big.NewInt(2050000))
	db.SetState(alternate, types.Hash{31: 1}, types.BytesToHash(big.NewInt(1000000).Bytes()))
	replacement := *frame
	replacement.Frames = append([]types.Frame8141(nil), frame.Frames...)
	replacement.Frames[1].Target = &alternate
	replacement.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(1100000), MaxPriorityFeePerGas: big.NewInt(3)}
	next := signPoolFrame(t, &replacement, 1, 2)
	if err := pool.AddRemote(next); err != ErrInsufficientFunds || pool.Get(first.Hash()) == nil {
		t.Fatalf("failed replacement lost old exposure: %v", err)
	}
	db.AddBalance(alternate, big.NewInt(100000))
	if err := pool.AddRemote(next); err != nil {
		t.Fatal(err)
	}
	if pool.Get(first.Hash()) != nil || pool.lookup.framePayers[next.Hash()] != alternate {
		t.Fatal("replacement did not move exposure")
	}
	third := *frame
	third.Sender = framePoolAddress(t, 3)
	if err := pool.AddRemote(signPoolFrame(t, &third, 3, 2)); err != nil {
		t.Fatal("old payer reservation was not released", err)
	}
	if err := pool.ValidateFrameForBuild(next, db, &types.Header{Number: big.NewInt(1), Time: 100, GasLimit: pool.config.BlockGasLimit, BaseFee: big.NewInt(1)}); err != nil {
		t.Fatal("builder lost selected profile", err)
	}
}

func TestPR12041BuildRechecksWithdrawalAvailability(t *testing.T) {
	pool, db, frame, payer := proposalPaymasterPool(t)
	tx := signPoolFrame(t, frame, 1, 2)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	candidate := db.Dup()
	candidate.SetState(payer, types.Hash{31: 1}, types.BytesToHash(big.NewInt(99000001).Bytes()))
	header := &types.Header{Number: big.NewInt(1), Time: 100, GasLimit: pool.config.BlockGasLimit, BaseFee: big.NewInt(1)}
	if err := pool.ValidateFrameForBuild(tx, candidate, header); err != ErrInsufficientFunds {
		t.Fatalf("candidate withdrawal reservation ignored: %v", err)
	}
	candidate.SetState(payer, types.Hash{31: 1}, types.Hash{})
	if err := pool.ValidateFrameForBuild(tx, candidate, header); err != nil {
		t.Fatal("cancelled candidate withdrawal retained exposure", err)
	}
	if db.GetState(payer, types.Hash{31: 1}) != (types.Hash{}) {
		t.Fatal("build validation mutated original state")
	}
}
