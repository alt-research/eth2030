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

func TestFrameCBuildContextAndUnpaidRetryBound(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	slot := uint64(2)
	pool.Reset(db, &types.Header{Time: 100, Number: big.NewInt(1), SlotNumber: &slot, GasLimit: 30000000, BaseFee: big.NewInt(1)})
	ref := types.RecentRootReference{Slot: 1, Root: types.Hash{31: 7}}
	db.SetCode(types.RecentRootAddress, types.RecentRootCode)
	db.SetState(types.RecentRootAddress, ref.StorageKey(), ref.EntryHash())
	frame.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: ref.AppendTo(nil), Limits: types.FrameGasLimits{Execution: 20000}}}, frame.Frames...)
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	root := db.GetRoot()
	candidate := &types.Header{Time: 101, Number: big.NewInt(2), SlotNumber: &slot, GasLimit: 30000000, BaseFee: big.NewInt(1)}
	if err := pool.ValidateFrameForBuild(tx, db, candidate); err != nil {
		t.Fatal(err)
	}
	skipped := uint64(8193)
	candidate.SlotNumber = &skipped
	if err := pool.ValidateFrameForBuild(tx, db, candidate); err == nil {
		t.Fatal("builder used admission slot instead of exact skipped payload slot")
	}
	pool.FrameBuildResult(tx.Hash(), false)
	pool.FrameBuildResult(tx.Hash(), false)
	if pool.Get(tx.Hash()) == nil {
		t.Fatal("transient validation evicted before multiple attempts")
	}
	pool.FrameBuildResult(tx.Hash(), true)
	if len(pool.lookup.frameBuildFailures) != 0 {
		t.Fatal("approval did not reset failures")
	}
	for i := 0; i < 3; i++ {
		pool.FrameBuildResult(tx.Hash(), false)
	}
	if pool.Get(tx.Hash()) != nil || len(pool.lookup.frameDependencies) != 0 || len(pool.lookup.frameBuildFailures) != 0 {
		t.Fatal("bounded unpaid work retained transaction or metadata")
	}
	pool.FrameBuildResult(tx.Hash(), false)
	if len(pool.lookup.frameBuildFailures) != 0 || db.GetRoot() != root {
		t.Fatal("removed attempt resurrected metadata or mutated state")
	}
}

func TestFrameCConfiguredReplacementAndSidecarBytes(t *testing.T) {
	pool, _, frame := framePoolFixture(t)
	pool.config.FramePriceBump = 25
	old := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(old); err != nil {
		t.Fatal(err)
	}
	next := *frame
	next.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(1100000), MaxPriorityFeePerGas: big.NewInt(3)}
	if err := pool.AddRemote(signPoolFrame(t, &next, 1)); err != ErrReplacementUnderpriced {
		t.Fatalf("configured bump ignored: %v", err)
	}
	next.Fees.MaxFee = big.NewInt(1250000)
	replacement := signPoolFrame(t, &next, 1)
	if err := pool.AddRemote(replacement); err != nil {
		t.Fatal(err)
	}
	plain := transactionPoolBytes(replacement)
	replacement.SetBlobSidecar(&types.BlobSidecarData{Blobs: [][]byte{make([]byte, 131072)}, Commitments: [][]byte{make([]byte, 48)}, Proofs: [][]byte{make([]byte, 48)}})
	if transactionPoolBytes(replacement) <= plain+131072 {
		t.Fatal("pool bytes omitted sidecar metadata")
	}
	if !frameFeeBumpedPercent(big.NewInt(1), big.NewInt(2), 25) || frameFeeBumpedPercent(big.NewInt(0), big.NewInt(0), 25) {
		t.Fatal("rounding/zero fee increment differs")
	}
}

func TestFrameCZeroFeeBudgetDoesNotPanic(t *testing.T) {
	pool, _, frame := framePoolFixture(t)
	pool.config.MinGasPrice = new(big.Int)
	pool.SetBaseFee(new(big.Int))
	frame.Fees = types.FrameFeeBudget{}
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	pool.Reset(pool.state)
	if pool.Get(tx.Hash()) == nil {
		t.Fatal("zero fee Frame lost on unchanged head")
	}
}

func TestFrameCCaughtInvalidOpcodeCannotApprovePublicPrefix(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	helper := types.Address{19: 0x90}
	db.SetCode(helper, []byte{0xfe})
	code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
	code = append(code, helper[:]...)
	code = append(code, 0x61, 0x27, 0x10, 0xfa, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa)
	db.SetCode(frame.Sender, code)
	frame.Frames[0].Limits.Execution = 30000
	if err := pool.AddRemote(signPoolFrame(t, frame, 1)); err == nil {
		t.Fatal("caught INVALID lost its public validation trace error")
	}
	if len(pool.lookup.frameValidations) != 0 || len(pool.lookup.framePayers) != 0 {
		t.Fatal("invalid child prefix retained metadata")
	}
}
