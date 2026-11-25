package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

func TestRecentRoot8272NativeWriteVerifyAndReorg(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	db.SetNonce(types.RecentRootAddress, 1)
	db.SetCode(types.RecentRootAddress, types.RecentRootCode)
	slot := uint64(37)
	header.SlotNumber = &slot
	salt, root := types.Hash{31: 3}, types.Hash{31: 4}
	ref := types.RecentRootReference{Source: types.RecentRootSource(tx.Sender, salt), Slot: slot, Root: root}
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &types.RecentRootAddress,
		Data: append(append([]byte{}, salt[:]...), root[:]...), Limits: types.FrameGasLimits{Execution: 30000, State: 97920}})
	orphan := db.Dup()
	receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.FrameResults8141[1].Status != 1 || db.GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() || receipt.FrameResults8141[1].GasUsed.State != 97920 {
		t.Fatalf("write differs from normative domains: %+v", receipt.FrameResults8141)
	}
	// The timestamp is intentionally unrelated to the consensus slot.
	header.Time = 9_000_000
	slot++
	check := *tx
	check.Nonce = 1
	check.Frames = []types.Frame8141{{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: ref.AppendTo(nil), Limits: types.FrameGasLimits{Execution: 12000}}, tx.Frames[0]}
	receipt, _, err = ApplyTransaction(cfg, db, header, signFrameExecution(t, &check, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil || receipt.FrameResults8141[0].Status != 1 || len(receipt.Logs) != 0 {
		t.Fatalf("verify: %+v %v", receipt, err)
	}
	check.Nonce = 0
	if _, _, err = ApplyTransaction(cfg, orphan, header, signFrameExecution(t, &check, false), new(gaspool.GasPool).AddGas(header.GasLimit)); err == nil {
		t.Fatal("accepted orphaned root")
	}
}

func TestRecentRoot8272NormativeRuntimeBoundaries(t *testing.T) {
	source := types.HexToHash("0xb9382d35273c75a50631a3e84d3c75ec9266e2b18c35a627e16cdbf26a18ca85")
	reference := types.RecentRootReference{Source: source, Slot: 1, Root: types.Hash{31: 2}}
	wrongRoot := reference
	wrongRoot.Root[31] = 3
	wrongSource := reference
	wrongSource.Source[31] ^= 1
	wrongSlot := reference
	wrongSlot.Slot = 0
	for _, tc := range []struct {
		name       string
		slot       uint64
		data       []byte
		gas        uint64
		undeployed bool
		want       bool
	}{
		{"wrong_root", 2, wrongRoot.AppendTo(nil), 12000, false, false},
		{"wrong_source", 2, wrongSource.AppendTo(nil), 12000, false, false},
		{"wrong_slot", 2, wrongSlot.AppendTo(nil), 12000, false, false},
		{"previous", 2, reference.AppendTo(nil), 12000, false, true},
		{"last_usable", 8192, reference.AppendTo(nil), 12000, false, true},
		{"expired", 8193, reference.AppendTo(nil), 12000, false, false},
		{"current", 1, reference.AppendTo(nil), 12000, false, false},
		{"future", 0, reference.AppendTo(nil), 12000, false, false},
		{"duplicates", 2, reference.AppendTo(reference.AppendTo(nil)), 12000, false, true},
		{"empty", 2, nil, 12000, false, false},
		{"71_bytes", 2, make([]byte, 71), 12000, false, false},
		{"73_bytes", 2, make([]byte, 73), 12000, false, false},
		{"17_tuples", 2, bytes.Repeat(reference.AppendTo(nil), 17), 100000, false, false},
		{"no_entry_gas", 2, reference.AppendTo(nil), 2999, false, false},
		{"undeployed", 2, reference.AppendTo(nil), 12000, true, false},
		{"write_in_verify", 2, make([]byte, 64), 30000, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			header.SlotNumber = &tc.slot
			if !tc.undeployed {
				db.SetNonce(types.RecentRootAddress, 1)
				db.SetCode(types.RecentRootAddress, types.RecentRootCode)
			}
			db.SetState(types.RecentRootAddress, reference.StorageKey(), reference.EntryHash())
			tx.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: tc.data, Limits: types.FrameGasLimits{Execution: tc.gas}}}, tx.Frames...)
			before := new(big.Int).Set(db.GetBalance(tx.Sender))
			receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
			if (err == nil) != tc.want {
				t.Fatalf("success=%v want=%v: %v", err == nil, tc.want, err)
			}
			if !tc.want && (db.GetNonce(tx.Sender) != 0 || db.GetBalance(tx.Sender).Cmp(before) != 0) {
				t.Fatal("invalid VERIFY retained effects")
			}
			if tc.want && (receipt.FrameResults8141[0].GasUsed.State != 0 || len(receipt.Logs) != 0) {
				t.Fatal("verifier wrote state/logs")
			}
		})
	}
}

func TestRecentRoot8272SixteenColdKeysAndExactGas(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	slot := uint64(2)
	header.SlotNumber = &slot
	db.SetNonce(types.RecentRootAddress, 1)
	db.SetCode(types.RecentRootAddress, types.RecentRootCode)
	var data []byte
	for i := byte(1); i <= 16; i++ {
		ref := types.RecentRootReference{Source: types.Hash{31: i}, Slot: 1, Root: types.Hash{31: i}}
		db.SetState(types.RecentRootAddress, ref.StorageKey(), ref.EntryHash())
		data = ref.AppendTo(data)
	}
	tx.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: data, Limits: types.FrameGasLimits{Execution: 80000}}}, tx.Frames...)
	original := db.Dup()
	receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	used := receipt.FrameResults8141[0].GasUsed.Execution
	if used < 3000+16*2100 {
		t.Fatalf("cold target/storage undercharged: %d", used)
	}
	tx.Frames[0].Limits.Execution = used
	if _, _, err = ApplyTransaction(cfg, original.Dup(), header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit)); err != nil {
		t.Fatalf("exact gas: %v", err)
	}
	tx.Frames[0].Limits.Execution--
	if _, _, err = ApplyTransaction(cfg, original.Dup(), header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit)); err == nil {
		t.Fatal("one-gas-short verifier accepted")
	}
}

func TestRecentRoot8272OverwriteAndStorageContext(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	slot := uint64(11)
	header.SlotNumber = &slot
	db.SetNonce(types.RecentRootAddress, 1)
	db.SetCode(types.RecentRootAddress, types.RecentRootCode)
	salt := types.Hash{31: 1}
	write := func(root byte) types.Frame8141 {
		data := append(append([]byte{}, salt[:]...), make([]byte, 32)...)
		data[63] = root
		return types.Frame8141{Mode: types.ModeSender, Target: &types.RecentRootAddress, Data: data, Limits: types.FrameGasLimits{Execution: 30000, State: 97920}}
	}
	tx.Frames = append(tx.Frames, write(1), write(2))
	receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil || receipt.FrameResults8141[2].Status != 1 {
		t.Fatal(receipt, err)
	}
	ref := types.RecentRootReference{Source: types.RecentRootSource(tx.Sender, salt), Slot: slot, Root: types.Hash{31: 2}}
	if db.GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() || receipt.FrameResults8141[2].GasUsed.State != 0 {
		t.Fatal("same-slot overwrite or state gas differs")
	}
	// A delegate call executes the same bytecode in the caller's storage context.
	proxy := types.Address{19: 0x91}
	code := []byte{0x36, 0x5f, 0x5f, 0x37, 0x5f, 0x5f, 0x36, 0x5f, 0x73}
	code = append(code, types.RecentRootAddress[:]...)
	code = append(code, 0x62, 0x01, 0x86, 0xa0, 0xf4, 0x50, 0)
	db.SetCode(proxy, code)
	tx.Nonce = 1
	tx.Frames = tx.Frames[:1]
	frame := write(3)
	frame.Target = &proxy
	frame.Limits.Execution = 50000
	tx.Frames = append(tx.Frames, frame)
	receipt, _, err = ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil || receipt.FrameResults8141[1].Status != 1 {
		t.Fatal(receipt, err)
	}
	if db.GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() {
		t.Fatal("delegate call modified canonical recent-root storage")
	}
	delegated := ref
	delegated.Root = types.Hash{31: 3}
	if db.GetState(proxy, delegated.StorageKey()) != delegated.EntryHash() {
		t.Fatal("delegate call did not use proxy storage")
	}
	// Advancing by the full ring length replaces the key and commits the new slot.
	slot += types.RecentRootLength
	tx.Nonce = 2
	tx.Frames = []types.Frame8141{tx.Frames[0], write(4)}
	receipt, _, err = ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil || receipt.FrameResults8141[1].Status != 1 {
		t.Fatal(receipt, err)
	}
	newer := ref
	newer.Slot = slot
	newer.Root = types.Hash{31: 4}
	if ref.StorageKey() != newer.StorageKey() || db.GetState(types.RecentRootAddress, newer.StorageKey()) != newer.EntryHash() || ref.IsRecent(slot) {
		t.Fatal("ring overwrite retained old root validity")
	}
}
