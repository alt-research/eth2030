package execution

import (
	"bytes"
	"fmt"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

func amsterdam8246() *config.ChainConfig {
	c := *config.TestConfig
	zero := uint64(0)
	c.AmsterdamTime, c.GlamsterdanTime = &zero, nil
	return &c
}

func assertCleared8246(t *testing.T, db state.StateDB, address types.Address, balance *big.Int) {
	t.Helper()
	if db.GetBalance(address).Cmp(balance) != 0 || db.GetNonce(address) != 0 || len(db.GetCode(address)) != 0 || db.GetState(address, types.Hash{}) != (types.Hash{}) {
		t.Fatalf("final account %s: balance=%s nonce=%d code=%x storage=%s; want balance=%s and cleared account", address, db.GetBalance(address), db.GetNonce(address), db.GetCode(address), db.GetState(address, types.Hash{}), balance)
	}
}

func TestEIP8246NativeCreationFinalization(t *testing.T) {
	for _, tc := range []struct {
		name           string
		self           bool
		prefund, value int64
		lateTip        bool
	}{
		{"self", true, 0, 7, false}, {"prefunded_self", true, 5, 7, false},
		{"other", false, 0, 7, false}, {"prefunded_other", false, 5, 7, false},
		{"prefunded_zero_endowment_other", false, 5, 0, false},
		{"zero", true, 0, 0, false}, {"late_miner_tip", false, 0, 7, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := state.NewMemoryStateDB()
			sender, beneficiary := types.HexToAddress("0x123456"), types.HexToAddress("0xabcdef")
			db.AddBalance(sender, big.NewInt(10000000))
			address := types.Address(gethcrypto.CreateAddress(gethcommon.Address(sender), 0))
			db.AddBalance(address, big.NewInt(tc.prefund))
			// Write storage before destruction; it must survive the opcode but not the transaction.
			init := []byte{0x60, 1, 0x60, 0, 0x55}
			if tc.self {
				init = append(init, 0x30)
			} else {
				init = append(init, 0x73)
				init = append(init, beneficiary[:]...)
			}
			init = append(init, 0xff)
			tx := types.NewTransaction(&types.LegacyTx{Gas: 300000, GasPrice: big.NewInt(2), Value: big.NewInt(tc.value), Data: init})
			tx.SetSender(sender)
			// Match the transaction log context installed by ProcessWithBAL.
			db.SetTxContext(tx.Hash(), 0)
			header := newTestHeader()
			header.BaseFee = big.NewInt(1)
			if tc.lateTip {
				header.Coinbase = address
			}
			receipt, used, err := ApplyTransaction(amsterdam8246(), db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil || receipt.Status != types.ReceiptStatusSuccessful {
				t.Fatalf("creation failed: %v receipt=%+v", err, receipt)
			}
			// EIP-7708 and EIP-8246 must agree in the same native receipt:
			// log real endowment/beneficiary transfers, never self or fee burns.
			type transfer struct {
				from, to types.Address
				amount   int64
			}
			var transfers []transfer
			if tc.value > 0 {
				transfers = append(transfers, transfer{sender, address, tc.value})
			}
			if !tc.self && tc.value+tc.prefund > 0 {
				transfers = append(transfers, transfer{address, beneficiary, tc.value + tc.prefund})
			}
			if len(receipt.Logs) != len(transfers) {
				t.Fatalf("transfer logs=%d, want %d", len(receipt.Logs), len(transfers))
			}
			for i, transfer := range transfers {
				log := receipt.Logs[i]
				var fromTopic, toTopic types.Hash
				copy(fromTopic[12:], transfer.from[:])
				copy(toTopic[12:], transfer.to[:])
				if log.Address != vm.SystemAddress || len(log.Topics) != 3 || log.Topics[0] != vm.TransferEventTopic || log.Topics[1] != fromTopic || log.Topics[2] != toTopic || len(log.Data) != 32 || new(big.Int).SetBytes(log.Data).Cmp(big.NewInt(transfer.amount)) != 0 {
					t.Fatalf("transfer log %d: got %+v, want %s -> %s amount %d", i, log, transfer.from, transfer.to, transfer.amount)
				}
			}
			want := new(big.Int)
			if tc.self {
				want.SetInt64(tc.value + tc.prefund)
			}
			if tc.lateTip {
				want.Add(want, new(big.Int).SetUint64(used))
			}
			assertCleared8246(t, db, address, want)
			if want.Sign() == 0 && db.Exist(address) {
				t.Fatal("empty selfdestructed account survived finalization")
			}
			if !tc.self && db.GetBalance(beneficiary).Cmp(big.NewInt(tc.value+tc.prefund)) != 0 {
				t.Fatal("beneficiary did not receive original balance")
			}
			root, err := db.Commit()
			if err != nil {
				t.Fatal(err)
			}
			if root == (types.Hash{}) {
				t.Fatal("missing committed state")
			}
			assertCleared8246(t, db, address, want)
		})
	}
}

func TestEIP8246CodeVisibleUntilTransactionEnd(t *testing.T) {
	factories := map[string]func() state.StateDB{
		"memory":    func() state.StateDB { return state.NewMemoryStateDB() },
		"trie":      func() state.StateDB { return state.NewTrieStateDB(rawdb.NewMemoryDB()) },
		"stateless": func() state.StateDB { return state.NewStatelessStateDB(state.NewStatelessWitness(types.Hash{})) },
		"binary": func() state.StateDB {
			db, err := state.NewBinaryOverlayState(state.NewMemoryStateDB())
			if err != nil {
				t.Fatal(err)
			}
			return db
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			db := factory()
			sender := types.HexToAddress("0x123456")
			db.AddBalance(sender, big.NewInt(1000000))
			evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, db)
			evm.SetForkRules(vm.ForkRules{IsAmsterdam: true, IsEIP158: true})
			runtime := []byte{0x60, 1, 0x60, 0, 0x55, 0x30, 0xff}
			_, address, _, err := evm.Create(sender, append([]byte{0x60, byte(len(runtime)), 0x60, 12, 0x60, 0, 0x39, 0x60, byte(len(runtime)), 0x60, 0, 0xf3}, runtime...), 300000, big.NewInt(7))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = evm.Call(sender, address, nil, 200000, big.NewInt(0)); err != nil {
				t.Fatal(err)
			}
			if !db.HasSelfDestructed(address) || !bytes.Equal(db.GetCode(address), runtime) || db.GetState(address, types.Hash{}) != (types.Hash{31: 1}) || db.GetBalance(address).Int64() != 7 {
				t.Fatalf("opcode cleared state too early or failed to mark account: destructed=%t code=%x slot=%s balance=%s", db.HasSelfDestructed(address), db.GetCode(address), db.GetState(address, types.Hash{}), db.GetBalance(address))
			}
			// A later CALL in the same transaction can still execute that code.
			if _, _, err = evm.Call(sender, address, nil, 200000, big.NewInt(3)); err != nil {
				t.Fatal(err)
			}
			evm.FinalizeSelfDestructs()
			if f, ok := db.(interface{ FinalizeTransaction() }); ok {
				f.FinalizeTransaction()
			}
			assertCleared8246(t, db, address, big.NewInt(10))
			if _, err := db.Commit(); err != nil {
				t.Fatal(err)
			}
			assertCleared8246(t, db, address, big.NewInt(10))
		})
	}
}

// The parent creates a child, then funds it after the child has selfdestructed.
// It returns the actual CREATE/CREATE2 result for comparison with an independent
// address implementation. The child records storage before destruction.
func lateCreditParent8246(init []byte, create2, revert bool) []byte {
	code := []byte{0x60, byte(len(init)), 0x60, 0, 0x60, 0, 0x39}
	if create2 {
		code = append(code, 0x60, 0x44)
	}
	code = append(code, 0x60, byte(len(init)), 0x60, 0, 0x60, 7)
	if create2 {
		code = append(code, 0xf5)
	} else {
		code = append(code, 0xf0)
	}
	// Keep the returned address below the seven CALL arguments.
	code = append(code, 0x60, 0, 0x60, 0, 0x60, 0, 0x60, 0, 0x60, 3, 0x85, 0x62, 1, 0x86, 0xa0, 0xf1, 0x50)
	code = append(code, 0x60, 0, 0x52, 0x60, 32, 0x60, 0)
	if revert {
		code = append(code, 0xfd)
	} else {
		code = append(code, 0xf3)
	}
	code[3] = byte(len(code))
	return append(code, init...)
}

func TestEIP8246NativeLateCreditsAndCreate2Reuse(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, self := range []bool{false, true} {
			t.Run(fmt.Sprintf("create2=%t/self=%t", create2, self), func(t *testing.T) {
				db := state.NewMemoryStateDB()
				sender := types.HexToAddress("0x123456")
				parent := types.HexToAddress("0xc0ffee")
				beneficiary := types.HexToAddress("0xabcdef")
				db.AddBalance(sender, big.NewInt(10000000))
				db.AddBalance(parent, big.NewInt(30))
				db.SetNonce(parent, 1)
				init := []byte{0x60, 1, 0x60, 0, 0x55}
				if self {
					init = append(init, 0x30)
				} else {
					init = append(init, 0x73)
					init = append(init, beneficiary[:]...)
				}
				init = append(init, 0xff)
				db.SetCode(parent, lateCreditParent8246(init, create2, false))
				for i := uint64(0); i < 2; i++ {
					tx := types.NewTransaction(&types.LegacyTx{Nonce: i, To: &parent, Gas: 800000, GasPrice: big.NewInt(1), Value: new(big.Int)})
					tx.SetSender(sender)
					header := newTestHeader()
					receipt, _, err := ApplyTransaction(amsterdam8246(), db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
					if err != nil || receipt.Status != types.ReceiptStatusSuccessful {
						t.Fatalf("transaction %d: err=%v receipt=%+v", i, err, receipt)
					}
					address := types.Address(gethcrypto.CreateAddress(gethcommon.Address(parent), i+1))
					if create2 {
						salt := [32]byte{31: 0x44}
						address = types.Address(gethcrypto.CreateAddress2(gethcommon.Address(parent), salt, gethcrypto.Keccak256(init)))
					}
					want := int64(3)
					if self {
						want = 10
						if create2 {
							want *= int64(i + 1)
						}
					}
					assertCleared8246(t, db, address, big.NewInt(want))
					if db.GetNonce(parent) != i+2 {
						t.Fatalf("creator nonce=%d want=%d", db.GetNonce(parent), i+2)
					}
				}
				if !self {
					want := int64(14)
					if create2 {
						want = 17
					}
					if db.GetBalance(beneficiary).Int64() != want {
						t.Fatalf("beneficiary=%s want=%d", db.GetBalance(beneficiary), want)
					}
				}
			})
		}
	}
}

func TestEIP8246RevertedCreationAndExistingContract(t *testing.T) {
	for _, revert := range []bool{false, true} {
		t.Run(fmt.Sprintf("revert=%t", revert), func(t *testing.T) {
			db := state.NewMemoryStateDB()
			sender := types.HexToAddress("0x123456")
			parent := types.HexToAddress("0xc0ffee")
			db.AddBalance(sender, big.NewInt(10000000))
			db.AddBalance(parent, big.NewInt(30))
			db.SetNonce(parent, 1)
			// Existing contracts retain all data under EIP-6780, even in Amsterdam.
			runtime := []byte{0x60, 1, 0x60, 0, 0x55, 0x30, 0xff}
			if revert {
				db.SetCode(parent, lateCreditParent8246(runtime, false, true))
			} else {
				db.SetCode(parent, runtime)
			}
			tx := types.NewTransaction(&types.LegacyTx{To: &parent, Gas: 800000, GasPrice: big.NewInt(1), Value: new(big.Int)})
			tx.SetSender(sender)
			header := newTestHeader()
			receipt, _, err := ApplyTransaction(amsterdam8246(), db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			if revert {
				address := types.Address(gethcrypto.CreateAddress(gethcommon.Address(parent), 1))
				if receipt.Status != types.ReceiptStatusFailed || db.Exist(address) || db.GetNonce(parent) != 1 || db.GetBalance(parent).Int64() != 30 {
					t.Fatalf("nested creation survived REVERT: receipt=%+v child=%t nonce=%d balance=%s", receipt, db.Exist(address), db.GetNonce(parent), db.GetBalance(parent))
				}
			} else if db.GetBalance(parent).Int64() != 30 || db.GetNonce(parent) != 1 || !bytes.Equal(db.GetCode(parent), runtime) || db.GetState(parent, types.Hash{}) != (types.Hash{31: 1}) {
				t.Fatal("existing contract was cleared")
			}
		})
	}
}

func TestEIP8246FinalBAL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    []byte
		destroy bool
		failed  bool
	}{
		{"destroyed_storage", []byte{0x60, 1, 0x60, 0, 0x55, 0x30, 0xff}, true, false},
		{"reverted_storage", []byte{0x60, 1, 0x60, 0, 0x55, 0x60, 0, 0x60, 0, 0xfd}, false, true},
		{"restored_storage", []byte{0x60, 1, 0x60, 0, 0x55, 0x60, 0, 0x60, 0, 0x55, 0}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := state.NewMemoryStateDB()
			sender := types.HexToAddress("0x123456")
			target := types.HexToAddress("0xc0ffee")
			db.AddBalance(sender, big.NewInt(10000000))
			var to *types.Address
			if tc.destroy {
				target = types.Address(gethcrypto.CreateAddress(gethcommon.Address(sender), 0))
			} else {
				to = &target
				db.SetCode(target, tc.code)
			}
			tx := types.NewTransaction(&types.LegacyTx{To: to, Gas: 800000, GasPrice: big.NewInt(1), Value: big.NewInt(7)})
			if tc.destroy {
				tx = types.NewTransaction(&types.LegacyTx{Gas: 800000, GasPrice: big.NewInt(1), Value: big.NewInt(7), Data: tc.code})
			}
			tx.SetSender(sender)
			header := newTestHeader()
			tracker := bal.NewTracker()
			before := new(big.Int).Set(db.GetBalance(sender))
			receipt, _, err := ApplyTransactionWithBAL(amsterdam8246(), db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
			if err != nil || (receipt.Status == types.ReceiptStatusFailed) != tc.failed {
				t.Fatalf("err=%v receipt=%+v", err, receipt)
			}
			preBalances := map[types.Address]*big.Int{sender: before, target: new(big.Int), header.Coinbase: new(big.Int)}
			populateTracker(tracker, db, preBalances, map[types.Address]uint64{sender: 0, target: 0})
			var found bool
			for _, entry := range tracker.Build(1).Entries {
				if entry.Address == sender && (entry.BalanceChange == nil || entry.BalanceChange.OldValue.Cmp(before) != 0 || entry.BalanceChange.NewValue.Cmp(db.GetBalance(sender)) != 0) {
					t.Fatal("upfront fee baseline lost")
				}
				if entry.Address != target {
					continue
				}
				found = true
				if len(entry.StorageChanges) != 0 || len(entry.StorageReads) != 1 || entry.StorageReads[0].Slot != (types.Hash{}) || entry.NonceChange != nil || entry.CodeChange != nil {
					t.Fatalf("temporary or reverted writes leaked into BAL: %+v", entry)
				}
				if tc.failed {
					if entry.BalanceChange != nil {
						t.Fatal("reverted value transfer leaked into BAL")
					}
				} else if entry.BalanceChange == nil || entry.BalanceChange.NewValue.Int64() != 7 {
					t.Fatal("final balance missing from BAL")
				}
			}
			if !found {
				t.Fatal("accessed target missing from BAL")
			}
		})
	}
}

func TestEIP8246CanonicalFrameFinalization(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	cfg.GlamsterdanTime = nil
	parent := types.Address{19: 0x90}
	db.SetNonce(parent, 1)
	db.AddBalance(parent, big.NewInt(30))
	db.SetCode(parent, lateCreditParent8246([]byte{0x60, 1, 0x60, 0, 0x55, 0x30, 0xff}, false, false))
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &parent, Limits: types.FrameGasLimits{Execution: 400000, State: 400000}})
	tracker := bal.NewTracker()
	transaction := signFrameExecution(t, tx, false)
	receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
	if err != nil || receipt.FrameResults8141[1].Status != types.FrameStatusSuccess {
		t.Fatalf("Frame destruction failed: %v receipt=%+v", err, receipt)
	}
	address := types.Address(gethcrypto.CreateAddress(gethcommon.Address(parent), 1))
	assertCleared8246(t, db, address, big.NewInt(10))
	for _, entry := range tracker.Build(1).Entries {
		if entry.Address == address && (len(entry.StorageChanges) != 0 || len(entry.StorageReads) != 1 || entry.NonceChange != nil || entry.CodeChange != nil) {
			t.Fatalf("Frame destruction leaked temporary BAL changes: %+v", entry)
		}
	}
}

func TestEIP8246ActivationBoundary(t *testing.T) {
	for _, timestamp := range []uint64{99, 100} {
		t.Run(fmt.Sprint(timestamp), func(t *testing.T) {
			db := state.NewMemoryStateDB()
			sender := types.HexToAddress("0x123456")
			db.AddBalance(sender, big.NewInt(10000000))
			cfg := amsterdam8246()
			activation := uint64(100)
			cfg.AmsterdamTime = &activation
			tx := types.NewTransaction(&types.LegacyTx{Gas: 300000, GasPrice: big.NewInt(1), Value: big.NewInt(7), Data: []byte{0x30, 0xff}})
			tx.SetSender(sender)
			header := newTestHeader()
			header.Time = timestamp
			receipt, _, err := ApplyTransaction(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil || receipt.Status != types.ReceiptStatusSuccessful {
				t.Fatalf("err=%v receipt=%+v", err, receipt)
			}
			address := types.Address(gethcrypto.CreateAddress(gethcommon.Address(sender), 0))
			want := int64(0)
			if timestamp == 100 {
				want = 7
			}
			assertCleared8246(t, db, address, big.NewInt(want))
			if db.Exist(address) != (timestamp == 100) {
				t.Fatal("incorrect fork-boundary account existence")
			}
		})
	}
}
