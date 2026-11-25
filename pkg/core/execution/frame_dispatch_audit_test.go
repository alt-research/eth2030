package execution

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

func TestFrame8141DirectPrecompileBALGasBoundary(t *testing.T) {
	for _, budget := range []uint64{99, 100, 118} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			target := types.Address{19: 4}
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Data: []byte{1}, Limits: types.FrameGasLimits{Execution: budget}})
			tracker := bal.NewTracker()
			receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
			if err != nil {
				t.Fatal(err)
			}
			seen := false
			for _, entry := range tracker.Build(1).Entries {
				if entry.Address == target {
					seen = true
				}
			}
			if seen != (budget >= 100) {
				t.Fatalf("precompile BAL touch=%v at entry budget %d", seen, budget)
			}
			if (receipt.FrameResults8141[1].Status == 1) != (budget == 118) {
				t.Fatal("identity precompile execution boundary")
			}
		})
	}
}

func TestFrame8141SstoreBALReadSurvivesPostStateFailure(t *testing.T) {
	for _, budget := range []uint64{5305, 5306, 15105} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			target := types.Address{19: 0x90}
			db.SetCode(target, []byte{0x60, 1, 0x5f, 0x55, 0})
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: budget}})
			tracker := bal.NewTracker()
			receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
			if err != nil {
				t.Fatal(err)
			}
			if receipt.FrameResults8141[1].Status != 0 {
				t.Fatal("expected pre/post-state OOG")
			}
			for _, entry := range tracker.Build(1).Entries {
				if entry.Address == target {
					want := 0
					if budget > 5305 {
						want = 1
					}
					if len(entry.StorageReads) != want || len(entry.StorageChanges) != 0 {
						t.Fatalf("budget %d: reads=%d changes=%d", budget, len(entry.StorageReads), len(entry.StorageChanges))
					}
				}
			}
		})
	}
}

func TestFrame8141DelegatedCallGasAndForwarding(t *testing.T) {
	for _, op := range []byte{0xf1, 0xf2, 0xf4, 0xfa} {
		for _, warm := range []bool{false, true} {
			for _, allGas := range []bool{false, true} {
				t.Run(fmt.Sprintf("op_%x_warm_%v_all_%v", op, warm, allGas), func(t *testing.T) {
					tx, db, cfg, header := frameExecutionFixture(t)
					caller, authority, delegate := types.Address{19: 0x90}, types.Address{19: 0x91}, types.Address{19: 0x92}
					db.SetCode(authority, types.AddressToDelegation(delegate))
					// Return child GAS immediately after its two-gas opcode.
					db.SetCode(delegate, []byte{0x5a, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3})
					code := []byte{0x5f, 0x5f, 0x5f, 0x5f}
					valueOps := uint64(0)
					if op == 0xf1 || op == 0xf2 {
						code = append(code, 0x5f)
						valueOps = 2
					}
					code = append(code, 0x73)
					code = append(code, authority[:]...)
					if allGas {
						code = append(code, 0x7f)
						for i := 0; i < 32; i++ {
							code = append(code, 0xff)
						}
					} else {
						code = append(code, 0x61, 3, 0xe8)
					}
					code = append(code, op, 0x50, 0x3d, 0x5f, 0x5f, 0x3e, 0x3d, 0x5f, 0xa0, 0)
					db.SetCode(caller, code)
					if warm {
						tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeDefault, Target: &delegate, Limits: types.FrameGasLimits{Execution: 5000}})
					}
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &caller, Limits: types.FrameGasLimits{Execution: 20000}})
					transaction := signFrameExecution(t, tx, false)
					db.SetTxContext(transaction.Hash(), 0)
					receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
					if err != nil {
						t.Fatal(err)
					}
					cost := uint64(6000)
					if warm {
						cost = 3100
					}
					expected := uint64(3000+14+15+652) + valueOps + cost
					result := receipt.FrameResults8141[len(tx.Frames)-1]
					if result.Status != 1 || result.GasUsed.Execution != expected {
						t.Fatalf("gas=%d status=%d want=%d", result.GasUsed.Execution, result.Status, expected)
					}
					child := uint64(998)
					if allGas {
						available := uint64(20000-3000-14) - valueOps - cost
						child = available - available/64 - 2
					}
					if len(result.Logs) != 1 || len(result.Logs[0].Data) != 32 || new(big.Int).SetBytes(result.Logs[0].Data).Uint64() != child {
						t.Fatalf("wrong pre-forwarding charge: logs=%+v child GAS=%d", result.Logs, child)
					}
				})
			}
		}
	}
}

func TestFrame8141SelfdestructBALPreStateBoundary(t *testing.T) {
	for _, funded := range []bool{false, true} {
		for _, budget := range []uint64{11002, 11003} {
			t.Run(fmt.Sprintf("funded_%v_gas_%d", funded, budget), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				target, beneficiary := types.Address{19: 0x90}, types.Address{19: 0x91}
				code := append([]byte{0x73}, beneficiary[:]...)
				db.SetCode(target, append(code, 0xff))
				if funded {
					db.AddBalance(target, big.NewInt(1))
				}
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: budget}})
				tracker := bal.NewTracker()
				_, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
				if err != nil {
					t.Fatal(err)
				}
				seen := false
				for _, entry := range tracker.Build(1).Entries {
					if entry.Address == beneficiary {
						seen = true
					}
				}
				if seen != (budget >= 11003) {
					t.Fatalf("beneficiary touch=%v at budget %d", seen, budget)
				}
			})
		}
	}
}

func TestFrame8141DelegationBALBeforeFailedChildChecks(t *testing.T) {
	for _, budget := range []uint64{17316, 17317, 20316, 20317} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			caller, authority, delegate := types.Address{19: 0x90}, types.Address{19: 0x91}, types.Address{19: 0x92}
			db.AddBalance(caller, big.NewInt(5))
			db.SetCode(authority, types.AddressToDelegation(delegate))
			db.SetCode(delegate, []byte{0})
			code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x60, 7, 0x73}
			code = append(code, authority[:]...)
			code = append(code, 0x61, 3, 0xe8, 0xf1, 0x50, 0)
			db.SetCode(caller, code)
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &caller, Limits: types.FrameGasLimits{Execution: budget}})
			tracker := bal.NewTracker()
			_, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[types.Address]bool{}
			for _, entry := range tracker.Build(1).Entries {
				seen[entry.Address] = true
			}
			if seen[authority] != (budget >= 17317) || seen[delegate] != (budget >= 20317) {
				t.Fatalf("budget %d authority=%v delegate=%v", budget, seen[authority], seen[delegate])
			}
			if db.GetBalance(caller).Int64() != 5 || db.GetBalance(authority).Sign() != 0 {
				t.Fatal("failed child balance check changed balances")
			}
		})
	}
}
