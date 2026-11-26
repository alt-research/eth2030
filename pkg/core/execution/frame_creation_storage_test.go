package execution

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// EIP-8037 (merged #12624) explicitly composes EIP-7610: arbitrary
// pre-existing storage collides, with the account charge determined by
// existence and restored by the failed child's rollback. The parent can
// spend that restored state budget after catching CREATE's zero result.
func TestFrameCCreationStorageCollisionAndRollback(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, test := range []struct {
			name       string
			storage    bool
			prefunded  bool
			budget     uint64
			wantStatus uint64
			wantState  uint64
		}{
			{"storage_only", true, false, 183600, 1, 97920},
			{"prefunded_storage", true, true, 97920, 1, 97920},
			{"account_charge_exhausted", true, false, 183599, 0, 0},
			{"empty_target", false, false, 281520, 1, 281520},
			{"prefunded_empty", false, true, 97920, 1, 97920},
		} {
			t.Run(fmt.Sprintf("create2_%t/%s", create2, test.name), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				factory := types.Address{19: 0x90}
				code := []byte{0x5f, 0x5f, 0x5f, 0xf0}
				target := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
				if create2 {
					code = []byte{0x5f, 0x5f, 0x5f, 0x5f, 0xf5}
					target = types.Address(gethcrypto.CreateAddress2([20]byte(factory), [32]byte{}, gethcrypto.Keccak256(nil)))
				}
				// POP the CREATE result, then spend state gas on a separate slot.
				db.SetCode(factory, append(code, 0x50, 0x60, 1, 0x60, 0xfe, 0x55, 0))
				key := types.Hash(gethcrypto.Keccak256Hash([]byte("arbitrary collision slot")))
				value := types.Hash{31: 0x42}
				if test.storage {
					db.SetState(target, key, value)
				}
				if test.prefunded {
					db.AddBalance(target, big.NewInt(1))
				}
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &factory,
					Limits: types.FrameGasLimits{Execution: 2_000_000, State: test.budget}})
				tracker := bal.NewTracker()
				receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
				if err != nil {
					t.Fatal(err)
				}
				result := receipt.FrameResults8141[1]
				if uint64(result.Status) != test.wantStatus || result.GasUsed.State != test.wantState {
					t.Fatalf("collision/rollback status=%d state=%d, want %d/%d", result.Status, result.GasUsed.State, test.wantStatus, test.wantState)
				}
				if test.storage && (db.GetState(target, key) != value || db.GetNonce(target) != 0 || len(db.GetCode(target)) != 0) {
					t.Fatal("creation destroyed or replaced pre-existing storage")
				}
				if !test.storage && db.GetNonce(target) != 1 {
					t.Fatal("empty or balance-only positive control failed to deploy")
				}
				if db.GetNonce(factory) != test.wantStatus || db.GetState(factory, types.Hash{31: 0xfe}).IsZero() != (test.wantStatus == 0) {
					t.Fatal("failed child lost parent progress, or exhausted parent kept writes/nonce")
				}
				seen := false
				for _, entry := range tracker.Build(1).Entries {
					if entry.Address == target {
						seen = true
						if len(entry.StorageReads) != 0 || len(entry.StorageChanges) != 0 {
							t.Fatal("storage-root collision probe fabricated individual BAL slot reads/writes")
						}
					}
				}
				if !seen {
					t.Fatal("collision destination account access missing from BAL")
				}
			})
		}
	}
}

// The latest dependency composition is selected by the existing native
// Hegota timestamp. Historical pinned forks remain replayable. Both the
// Amsterdam reservoir path and the scalar VM path consume the same rule.
func TestFrameCCreationCollisionForkBoundary(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		for _, create2 := range []bool{false, true} {
			for _, timestamp := range []uint64{99, 100} {
				t.Run(fmt.Sprintf("amsterdam_%t/create2_%t/time_%d", amsterdam, create2, timestamp), func(t *testing.T) {
					_, db, cfg, header := frameExecutionFixture(t)
					activation := uint64(100)
					cfg.HogotaTime = &activation
					if !amsterdam {
						cfg.AmsterdamTime = nil
					}
					header.Time = timestamp
					caller := types.Address{19: 0x90}
					target := types.Address(gethcrypto.CreateAddress([20]byte(caller), 0))
					if create2 {
						target = types.Address(gethcrypto.CreateAddress2([20]byte(caller), [32]byte{}, gethcrypto.Keccak256(nil)))
					}
					key, value := types.Hash{0: 0x80, 31: 0xff}, types.Hash{31: 1}
					db.SetState(target, key, value)
					evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, db)
					if _, err := ConfigureBlockEVM(cfg, header, evm); err != nil {
						t.Fatal(err)
					}
					var err error
					var left uint64
					if create2 {
						_, _, left, err = evm.Create2(caller, nil, 300000, new(big.Int), new(big.Int))
					} else {
						_, _, left, err = evm.Create(caller, nil, 300000, new(big.Int))
					}
					if timestamp == 100 {
						if err == nil || left != 0 || db.GetNonce(target) != 0 || db.GetState(target, key) != value {
							t.Fatal("selected native fork did not reject storage-only creation", err, left)
						}
						if result := evm.TransactionStateGas(); amsterdam && (result == nil || result.StateGasUsed != 0) {
							t.Fatal("top-level collision retained a rolled-back account charge")
						}
					} else if err != nil || db.GetNonce(target) != 1 || !db.GetState(target, key).IsZero() {
						t.Fatal("historical pinned fork behavior changed", err)
					}
				})
			}
		}
	}
}
