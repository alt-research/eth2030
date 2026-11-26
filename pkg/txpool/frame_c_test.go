package txpool

import (
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	"math"
	"math/big"
	"testing"
)

func TestFrameCPublicRecentRootSlotAndReorg(t *testing.T) {
	for _, head := range []uint64{1, 8191, 8192, math.MaxUint64} {
		t.Run(new(big.Int).SetUint64(head).String(), func(t *testing.T) {
			pool, db, frame := framePoolFixture(t)
			slot := head
			pool.Reset(db, &types.Header{Time: 100, Number: big.NewInt(1), SlotNumber: &slot, GasLimit: 30000000, BaseFee: big.NewInt(1)})
			ref := types.RecentRootReference{Slot: 1, Root: types.Hash{31: 7}}
			db.SetCode(types.RecentRootAddress, types.RecentRootCode)
			db.SetState(types.RecentRootAddress, ref.StorageKey(), ref.EntryHash())
			frame.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: ref.AppendTo(ref.AppendTo(nil)), Limits: types.FrameGasLimits{Execution: 20000}}}, frame.Frames...)
			tx := signPoolFrame(t, frame, 1)
			err := pool.AddRemote(tx)
			want := head == 1 || head == 8191
			if (err == nil) != want {
				t.Fatalf("admission=%v want=%v", err, want)
			}
			if !want {
				return
			}
			validation := pool.lookup.frameValidations[tx.Hash()]
			key := execution.FrameDependency{Address: types.RecentRootAddress, Kind: execution.FrameStorageDependency, Slot: ref.StorageKey()}
			if len(pool.lookup.frameDependencies[key]) != 1 {
				t.Fatal("duplicate tuple duplicated dependency")
			}
			pool.Reset(db)
			if pool.lookup.frameValidations[tx.Hash()] != validation {
				t.Fatal("unchanged dependencies re-executed validation")
			}
			// Removed-branch root vanishes without any write in the added branch's BAL.
			db.SetState(types.RecentRootAddress, ref.StorageKey(), types.Hash{})
			pool.Reset(db)
			if pool.Get(tx.Hash()) != nil || len(pool.lookup.frameDependencies) != 0 || len(pool.lookup.framePayers) != 0 {
				t.Fatal("orphaned root or dependency metadata survived rollback")
			}
		})
	}
}

func TestFrameCValidationDependencyCache(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	frame.Frames[0].Limits.Execution = 30000
	db.SetCode(frame.Sender, []byte{0x5f, 0x54, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa})
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	before := pool.lookup.frameValidations[tx.Hash()]
	db.SetState(types.Address{19: 99}, types.Hash{}, types.Hash{31: 1})
	pool.Reset(db)
	if pool.lookup.frameValidations[tx.Hash()] != before {
		t.Fatal("unrelated write invoked VM")
	}
	db.SetState(frame.Sender, types.Hash{}, types.Hash{31: 1})
	pool.Reset(db)
	if pool.Get(tx.Hash()) == nil || pool.lookup.frameValidations[tx.Hash()] == before {
		t.Fatal("sender storage did not invalidate cache")
	}
	db.SetCode(frame.Sender, []byte{0x43, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa})
	pool.Reset(db)
	if pool.Get(tx.Hash()) != nil || len(pool.lookup.frameDependencies) != 0 {
		t.Fatal("banned code change survived")
	}
}

func TestFrameCFailedAdmissionAndByteReplacementAreAtomic(t *testing.T) {
	pool, _, frame := framePoolFixture(t)
	old := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(old); err != nil {
		t.Fatal(err)
	}
	pool.config.MaxBytes = transactionPoolBytes(old)
	next := *frame
	next.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(1100000), MaxPriorityFeePerGas: big.NewInt(3)}
	next.Frames = append([]types.Frame8141(nil), frame.Frames...)
	next.Frames = append(next.Frames, types.Frame8141{Mode: types.ModeSender, Data: make([]byte, 1024)})
	replacement := signPoolFrame(t, &next, 1)
	if err := pool.AddRemote(replacement); err != ErrTxPoolFull {
		t.Fatalf("oversized replacement: %v", err)
	}
	if pool.Get(old.Hash()) == nil || len(pool.lookup.frameValidations) != 1 || len(pool.lookup.framePayers) != 1 {
		t.Fatal("failed replacement changed resources")
	}
	pool.Remove(old.Hash())
	if len(pool.lookup.frameDependencies) != 0 {
		t.Fatal("removal leaked reverse index")
	}
}

func TestFrameCLocalPaymasterRequiresExplicitProfile(t *testing.T) {
	pool, _, frame, _ := paymasterPoolFixture(t)
	pool.config.ExperimentalFramePaymasterV1 = false
	if err := pool.AddRemote(signPoolFrame(t, frame, 1, 2)); err == nil {
		t.Fatal("local paymaster silently granted canonical privileges")
	}
}
