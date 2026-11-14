package vm

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// newVerkleEVM creates an EVM with the Verkle jump table and a WitnessGasTracker.
func newVerkleEVM(sender, to types.Address) (*EVM, *state.MemoryStateDB) {
	stateDB := state.NewMemoryStateDB()
	evm := NewEVMWithState(
		BlockContext{BlockNumber: big.NewInt(1), GasLimit: 30000000},
		TxContext{Origin: sender},
		Config{},
		stateDB,
	)
	evm.SetJumpTable(NewVerkleJumpTable())
	evm.SetWitnessGasTracker(NewWitnessGasTracker())
	return evm, stateDB
}

// --- WitnessGasTracker unit tests ---

func TestWitnessGasTrackerAccessEvent(t *testing.T) {
	tracker := NewWitnessGasTracker()
	addr := types.BytesToAddress([]byte{0x01})

	// First access: branch + chunk.
	gas := tracker.TouchAccessEvent(addr, 0, BasicDataLeafKey)
	expected := WitnessBranchCost + WitnessChunkCost
	if gas != expected {
		t.Errorf("first access gas = %d, want %d", gas, expected)
	}

	// Second access to same (addr, subKey, leafKey): no charge.
	gas = tracker.TouchAccessEvent(addr, 0, BasicDataLeafKey)
	if gas != 0 {
		t.Errorf("repeat access gas = %d, want 0", gas)
	}

	// Same subtree, different leaf: only chunk cost.
	gas = tracker.TouchAccessEvent(addr, 0, CodeHashLeafKey)
	if gas != WitnessChunkCost {
		t.Errorf("same subtree different leaf gas = %d, want %d", gas, WitnessChunkCost)
	}

	// Different subtree: branch + chunk.
	gas = tracker.TouchAccessEvent(addr, 1, 0)
	expected = WitnessBranchCost + WitnessChunkCost
	if gas != expected {
		t.Errorf("new subtree access gas = %d, want %d", gas, expected)
	}
}

func TestWitnessGasTrackerWriteEvent(t *testing.T) {
	tracker := NewWitnessGasTracker()
	addr := types.BytesToAddress([]byte{0x02})

	// First write to a subtree (no fill).
	gas := tracker.TouchWriteEvent(addr, 0, BasicDataLeafKey, false)
	expected := SubtreeEditCost + ChunkEditCost
	if gas != expected {
		t.Errorf("first write gas = %d, want %d", gas, expected)
	}

	// Second write to same leaf: no charge.
	gas = tracker.TouchWriteEvent(addr, 0, BasicDataLeafKey, false)
	if gas != 0 {
		t.Errorf("repeat write gas = %d, want 0", gas)
	}

	// Write to same subtree, different leaf: only chunk edit.
	gas = tracker.TouchWriteEvent(addr, 0, CodeHashLeafKey, false)
	if gas != ChunkEditCost {
		t.Errorf("same subtree different leaf write gas = %d, want %d", gas, ChunkEditCost)
	}

	// Write with fill: chunk edit + fill cost.
	addr2 := types.BytesToAddress([]byte{0x03})
	gas = tracker.TouchWriteEvent(addr2, 0, 0, true)
	expected = SubtreeEditCost + ChunkEditCost + ChunkFillCost
	if gas != expected {
		t.Errorf("fill write gas = %d, want %d", gas, expected)
	}
}

// --- Storage slot tree key tests ---

func TestGetStorageSlotTreeKeys(t *testing.T) {
	// Slot 0: header storage area, pos = 64 + 0 = 64.
	treeKey, subKey := GetStorageSlotTreeKeys(0)
	if treeKey != 0 || subKey != 64 {
		t.Errorf("slot 0: treeKey=%d, subKey=%d, want 0, 64", treeKey, subKey)
	}

	// Slot 63: pos = 64 + 63 = 127, still in subtree 0.
	treeKey, subKey = GetStorageSlotTreeKeys(63)
	if treeKey != 0 || subKey != 127 {
		t.Errorf("slot 63: treeKey=%d, subKey=%d, want 0, 127", treeKey, subKey)
	}

	// Slot 64: pos = CodeOffset - HeaderStorageOffset = 128 - 64 = 64, so
	// slot 64 >= 64, goes to MainStorageOffset. pos = 16384 + 64 = 16448.
	treeKey, subKey = GetStorageSlotTreeKeys(64)
	expectedTree := uint64(16448) / VerkleNodeWidth
	expectedSub := uint8(16448 % VerkleNodeWidth)
	if treeKey != expectedTree || subKey != expectedSub {
		t.Errorf("slot 64: treeKey=%d, subKey=%d, want %d, %d", treeKey, subKey, expectedTree, expectedSub)
	}
}

func TestGetCodeChunkTreeKeys(t *testing.T) {
	// Chunk 0: pos = 128 + 0 = 128.
	treeKey, subKey := GetCodeChunkTreeKeys(0)
	if treeKey != 0 || subKey != 128 {
		t.Errorf("chunk 0: treeKey=%d, subKey=%d, want 0, 128", treeKey, subKey)
	}

	// Chunk 127: pos = 128 + 127 = 255, still in subtree 0.
	treeKey, subKey = GetCodeChunkTreeKeys(127)
	if treeKey != 0 || subKey != 255 {
		t.Errorf("chunk 127: treeKey=%d, subKey=%d, want 0, 255", treeKey, subKey)
	}

	// Chunk 128: pos = 128 + 128 = 256, subtree 1.
	treeKey, subKey = GetCodeChunkTreeKeys(128)
	if treeKey != 1 || subKey != 0 {
		t.Errorf("chunk 128: treeKey=%d, subKey=%d, want 1, 0", treeKey, subKey)
	}
}

// --- EVM opcode-level gas tests ---

func TestEIP4762SloadFirstAccess(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(to)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	// PUSH1 0x00, SLOAD, STOP
	contract.Code = []byte{
		byte(PUSH1), 0x00,
		byte(SLOAD),
		byte(STOP),
	}

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// PUSH1 (3) + SLOAD witness (branch 1900 + chunk 200) = 2103
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost
	if used != expectedGas {
		t.Errorf("EIP-4762 SLOAD gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762SloadSecondAccessSameSubtree(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(to)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	// Two SLOADs to slot 0 and slot 1 (both in same subtree, different leaves).
	// PUSH1 0x00, SLOAD, POP, PUSH1 0x01, SLOAD, STOP
	contract.Code = []byte{
		byte(PUSH1), 0x00,
		byte(SLOAD),
		byte(POP),
		byte(PUSH1), 0x01,
		byte(SLOAD),
		byte(STOP),
	}

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// First SLOAD slot 0: PUSH1 (3) + branch(1900) + chunk(200) = 2103
	// POP: 2
	// Second SLOAD slot 1: PUSH1 (3) + chunk(200) = 203 (same subtree, no branch cost)
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost + GasPop + GasPush + WitnessChunkCost
	if used != expectedGas {
		t.Errorf("EIP-4762 adjacent SLOAD gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762SloadRepeatSameSlot(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(to)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	// Two SLOADs to the same slot: second should be free.
	// PUSH1 0x00, SLOAD, POP, PUSH1 0x00, SLOAD, STOP
	contract.Code = []byte{
		byte(PUSH1), 0x00,
		byte(SLOAD),
		byte(POP),
		byte(PUSH1), 0x00,
		byte(SLOAD),
		byte(STOP),
	}

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// First SLOAD: PUSH1 (3) + branch(1900) + chunk(200) = 2103
	// POP: 2
	// Second SLOAD: PUSH1 (3) + 0 (already accessed) = 3
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost + GasPop + GasPush
	if used != expectedGas {
		t.Errorf("EIP-4762 repeat SLOAD gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762SstoreFirstWrite(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(to)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	// PUSH1 0x42, PUSH1 0x00, SSTORE, STOP
	contract.Code = []byte{
		byte(PUSH1), 0x42, // value
		byte(PUSH1), 0x00, // key
		byte(SSTORE),
		byte(STOP),
	}

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// PUSH1 (3) + PUSH1 (3) + SSTORE:
	//   base (100) + access branch(1900) + access chunk(200) +
	//   write subtree(3000) + write chunk(500) + fill(6200) = 11900
	expectedGas := GasPush + GasPush + WarmStorageReadCost +
		WitnessBranchCost + WitnessChunkCost +
		SubtreeEditCost + ChunkEditCost + ChunkFillCost
	if used != expectedGas {
		t.Errorf("EIP-4762 SSTORE gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762SstoreOverwrite(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(to)
	// Pre-set slot 0 to a non-zero value so it's not a fill.
	slot := types.Hash{}
	val := types.Hash{31: 0x01}
	stateDB.SetState(to, slot, val)
	// Commit so GetCommittedState returns the non-zero value.
	stateDB.Commit()

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	// PUSH1 0x42, PUSH1 0x00, SSTORE, STOP
	contract.Code = []byte{
		byte(PUSH1), 0x42,
		byte(PUSH1), 0x00,
		byte(SSTORE),
		byte(STOP),
	}

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// No fill cost since slot already has a value.
	expectedGas := GasPush + GasPush + WarmStorageReadCost +
		WitnessBranchCost + WitnessChunkCost +
		SubtreeEditCost + ChunkEditCost
	if used != expectedGas {
		t.Errorf("EIP-4762 SSTORE overwrite gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762BalanceColdAccess(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})
	target := types.BytesToAddress([]byte{0xcc})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(target)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	code := []byte{byte(PUSH20)}
	code = append(code, target[:]...)
	code = append(code, byte(BALANCE), byte(STOP))
	contract.Code = code

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// PUSH20 (3) + BALANCE witness (branch 1900 + chunk 200) = 2103
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost
	if used != expectedGas {
		t.Errorf("EIP-4762 BALANCE gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762BalanceSecondAccessFree(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})
	target := types.BytesToAddress([]byte{0xcc})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(target)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	// Two BALANCEs to the same target.
	code := []byte{byte(PUSH20)}
	code = append(code, target[:]...)
	code = append(code, byte(BALANCE), byte(POP))
	code = append(code, byte(PUSH20))
	code = append(code, target[:]...)
	code = append(code, byte(BALANCE), byte(STOP))
	contract.Code = code

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// First: PUSH20(3) + branch(1900) + chunk(200) = 2103
	// POP: 2
	// Second: PUSH20(3) + 0 (already accessed) = 3
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost + GasPop + GasPush
	if used != expectedGas {
		t.Errorf("EIP-4762 repeat BALANCE gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762ExtCodeHashAccess(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})
	target := types.BytesToAddress([]byte{0xdd})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(target)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	code := []byte{byte(PUSH20)}
	code = append(code, target[:]...)
	code = append(code, byte(EXTCODEHASH), byte(POP), byte(STOP))
	contract.Code = code

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// PUSH20(3) + EXTCODEHASH witness (branch 1900 + chunk 200 for CodeHashLeafKey) + POP(2) = 2105
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost + GasPop
	if used != expectedGas {
		t.Errorf("EIP-4762 EXTCODEHASH gas used = %d, want %d", used, expectedGas)
	}
}

func TestEIP4762ExtCodeSizeAccess(t *testing.T) {
	sender := types.BytesToAddress([]byte{0xaa})
	to := types.BytesToAddress([]byte{0xbb})
	target := types.BytesToAddress([]byte{0xee})

	evm, stateDB := newVerkleEVM(sender, to)
	stateDB.CreateAccount(target)

	gas := uint64(100000)
	contract := NewContract(sender, to, big.NewInt(0), gas)
	code := []byte{byte(PUSH20)}
	code = append(code, target[:]...)
	code = append(code, byte(EXTCODESIZE), byte(POP), byte(STOP))
	contract.Code = code

	_, err := evm.Run(contract, nil)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	used := gas - contract.Gas
	// PUSH20(3) + EXTCODESIZE witness (branch 1900 + chunk 200) + POP(2)
	expectedGas := GasPush + WitnessBranchCost + WitnessChunkCost + GasPop
	if used != expectedGas {
		t.Errorf("EIP-4762 EXTCODESIZE gas used = %d, want %d", used, expectedGas)
	}
}

// TestEIP4762AdjacentSlotsShareSubtree verifies that slots in the header
// storage area (0-63) share subtree 0, so only the first access charges
// branch cost.
func TestEIP4762AdjacentSlotsShareSubtree(t *testing.T) {
	tracker := NewWitnessGasTracker()
	addr := types.BytesToAddress([]byte{0x01})

	// Slot 0 and slot 1 should be in the same subtree.
	treeKey0, subKey0 := GetStorageSlotTreeKeys(0)
	treeKey1, subKey1 := GetStorageSlotTreeKeys(1)

	if treeKey0 != treeKey1 {
		t.Fatalf("slots 0 and 1 should share treeKey, got %d and %d", treeKey0, treeKey1)
	}
	if subKey0 == subKey1 {
		t.Fatalf("slots 0 and 1 should have different subKeys")
	}

	// First access: branch + chunk.
	gas := tracker.TouchAccessEvent(addr, treeKey0, subKey0)
	if gas != WitnessBranchCost+WitnessChunkCost {
		t.Errorf("slot 0 access gas = %d, want %d", gas, WitnessBranchCost+WitnessChunkCost)
	}

	// Second access (same subtree): only chunk.
	gas = tracker.TouchAccessEvent(addr, treeKey1, subKey1)
	if gas != WitnessChunkCost {
		t.Errorf("slot 1 access gas = %d, want %d (only chunk, same subtree)", gas, WitnessChunkCost)
	}
}

// TestEIP4762VerkleJumpTableSelection verifies that SelectJumpTable returns the
// Verkle table when IsVerkle is set.
func TestEIP4762VerkleJumpTableSelection(t *testing.T) {
	rules := ForkRules{IsVerkle: true, IsGlamsterdan: true, IsCancun: true, IsBerlin: true}
	jt := SelectJumpTable(rules)

	// Under Verkle, SLOAD should have constantGas=0 (all dynamic from witness).
	sloadOp := jt[SLOAD]
	if sloadOp == nil {
		t.Fatal("SLOAD operation is nil in Verkle jump table")
	}
	if sloadOp.constantGas != 0 {
		t.Errorf("Verkle SLOAD constantGas = %d, want 0", sloadOp.constantGas)
	}
	if sloadOp.dynamicGas == nil {
		t.Error("Verkle SLOAD dynamicGas is nil, want gasSloadEIP4762")
	}
}

// TestEIP4762TrackerIsolation verifies that different addresses have independent
// tracking in the witness gas tracker.
func TestEIP4762TrackerIsolation(t *testing.T) {
	tracker := NewWitnessGasTracker()
	addr1 := types.BytesToAddress([]byte{0x01})
	addr2 := types.BytesToAddress([]byte{0x02})

	// Access slot 0 on addr1.
	gas1 := tracker.TouchAccessEvent(addr1, 0, BasicDataLeafKey)
	// Access slot 0 on addr2 (should also charge full cost).
	gas2 := tracker.TouchAccessEvent(addr2, 0, BasicDataLeafKey)

	if gas1 != gas2 {
		t.Errorf("addr isolation: gas1=%d, gas2=%d, both should be %d",
			gas1, gas2, WitnessBranchCost+WitnessChunkCost)
	}
}

// TestEIP4762GasConstants verifies the gas constant values from the EIP.
func TestEIP4762GasConstants(t *testing.T) {
	if WitnessBranchCost != 1900 {
		t.Errorf("WitnessBranchCost = %d, want 1900", WitnessBranchCost)
	}
	if WitnessChunkCost != 200 {
		t.Errorf("WitnessChunkCost = %d, want 200", WitnessChunkCost)
	}
	if SubtreeEditCost != 3000 {
		t.Errorf("SubtreeEditCost = %d, want 3000", SubtreeEditCost)
	}
	if ChunkEditCost != 500 {
		t.Errorf("ChunkEditCost = %d, want 500", ChunkEditCost)
	}
	if ChunkFillCost != 6200 {
		t.Errorf("ChunkFillCost = %d, want 6200", ChunkFillCost)
	}
	if GasCreateVerkle != 1000 {
		t.Errorf("GasCreateVerkle = %d, want 1000", GasCreateVerkle)
	}
	if CodeChunkSize != 31 {
		t.Errorf("CodeChunkSize = %d, want 31", CodeChunkSize)
	}
}

func TestEIP4762StorageGasUsesFullSlotWidth(t *testing.T) {
	sender, target := types.Address{19: 0xaa}, types.Address{19: 0xbb}
	cases := []struct {
		name              string
		first, high, last byte
		gas               uint64
	}{
		{"same leaf", 64, 0, 64, 2100},
		{"adjacent leaf", 64, 0, 65, 2300},
		{"distinct high bits", 64, 1, 64, 4200},
		{"distinct inline and main storage", 0, 1, 0, 4200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evm, _ := newVerkleEVM(sender, target)
			const startGas = 100_000
			contract := NewContract(sender, target, new(big.Int), startGas)
			contract.Code = []byte{byte(PUSH1), tc.first, byte(SLOAD), byte(POP), byte(PUSH9), tc.high}
			contract.Code = append(contract.Code, make([]byte, 7)...)
			contract.Code = append(contract.Code, tc.last, byte(SLOAD), byte(STOP))
			if _, err := evm.Run(contract, nil); err != nil {
				t.Fatal(err)
			}
			if used, want := startGas-contract.Gas, tc.gas+2*GasPush+GasQuickStep; used != want {
				t.Fatalf("used %d gas, want %d", used, want)
			}
		})
	}
}

func TestEIP4762UnpaidStorageWriteLeavesNoWitness(t *testing.T) {
	sender, target := types.Address{19: 1}, types.Address{19: 2}
	evm, db := newVerkleEVM(sender, target)
	run := func(gas uint64) (*Contract, error) {
		contract := NewContract(sender, target, new(big.Int), gas)
		contract.Code = []byte{byte(PUSH1), 1, byte(PUSH1), 0, byte(SSTORE), byte(STOP)}
		_, err := evm.Run(contract, nil)
		return contract, err
	}
	const coldWrite = 11906
	failed, err := run(coldWrite - 1)
	if !errors.Is(err, ErrOutOfGas) || failed.Gas != 0 || !db.GetState(target, types.Hash{}).IsZero() {
		t.Fatalf("unpaid write: gas %d, err %v", failed.Gas, err)
	}
	paid, err := run(coldWrite)
	if err != nil || paid.Gas != 0 || db.GetState(target, types.Hash{}) != (types.Hash{31: 1}) {
		t.Fatalf("second write did not pay full witness gas: gas %d, err %v", paid.Gas, err)
	}
}

func TestEIP4762ExplicitZeroIsAlreadyFilled(t *testing.T) {
	sender, target := types.Address{19: 1}, types.Address{19: 2}
	frozen := state.NewMemoryStateDB()
	frozen.SetState(target, types.Hash{31: 1}, types.Hash{31: 3})
	overlay, err := state.NewBinaryOverlayState(frozen)
	if err != nil {
		t.Fatal(err)
	}
	overlay.SetState(target, types.Hash{}, types.Hash{})
	if _, err := overlay.Commit(); err != nil {
		t.Fatal(err)
	}
	evm, _ := newVerkleEVM(sender, target)
	evm.StateDB = overlay
	contract := NewContract(sender, target, new(big.Int), 5706)
	contract.Code = []byte{byte(PUSH1), 7, byte(PUSH1), 0, byte(SSTORE), byte(STOP)}
	if _, err := evm.Run(contract, nil); err != nil || contract.Gas != 0 {
		t.Fatalf("overwrite of explicit zero: gas %d, err %v", contract.Gas, err)
	}
	evm.SetWitnessGasTracker(NewWitnessGasTracker())
	contract = NewContract(sender, target, new(big.Int), 5706)
	contract.Code = []byte{byte(PUSH1), 0, byte(PUSH1), 1, byte(SSTORE), byte(STOP)}
	if _, err := evm.Run(contract, nil); err != nil || overlay.GetRefund() != 0 {
		t.Fatalf("clear inherited a legacy SSTORE refund: refund %d, err %v", overlay.GetRefund(), err)
	}
}

func TestEIP4762CodeChunkRangeClampsWithoutOverflow(t *testing.T) {
	sender, target := types.Address{19: 1}, types.Address{19: 2}
	evm, _ := newVerkleEVM(sender, target)
	if got := gasCodeChunksAccess(evm, target, 31, math.MaxUint64, 64); got != 2300 {
		t.Fatalf("copying from offset 31 into end-of-code: witness gas %d, want 2300", got)
	}
}
