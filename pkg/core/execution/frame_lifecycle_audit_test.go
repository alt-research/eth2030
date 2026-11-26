package execution

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func frameCreationProgram(init []byte, create2 bool, value byte) []byte {
	code := []byte{0x60, byte(len(init)), 0x60, 0, 0x5f, 0x39}
	if create2 {
		code = append(code, 0x60, 1) // salt
	}
	code = append(code, 0x60, byte(len(init)), 0x5f, 0x60, value)
	if create2 {
		code = append(code, 0xf5)
	} else {
		code = append(code, 0xf0)
	}
	code = append(code, 0x50, 0)
	code[3] = byte(len(code))
	return append(code, init...)
}

// EIP-8037 permits creation at a balance-only account without a new-account
// charge. EIP-7928 compares balances against transaction pre-state, including
// when constructor failure restores the original prefunding.
func TestFrame8141PrefundedCreationBAL(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, fails := range []bool{false, true} {
			for _, value := range []byte{0, 3} {
				t.Run(fmt.Sprintf("create2_%v_revert_%v_value_%d", create2, fails, value), func(t *testing.T) {
					tx, db, cfg, header := frameExecutionFixture(t)
					creator := types.Address{19: 0x90}
					// Return a one-byte STOP runtime, or explicitly revert.
					init := []byte{0x5f, 0x5f, 0x53, 0x60, 1, 0x5f, 0xf3}
					if fails {
						init = []byte{0x5f, 0x5f, 0xfd}
					}
					destination := vm.CreateAddressFromNonce(creator, 0)
					if create2 {
						destination = vm.Create2AddressFromSaltAndCode(creator, big.NewInt(1), init)
					}
					db.AddBalance(destination, big.NewInt(7))
					db.AddBalance(creator, big.NewInt(100))
					db.SetCode(creator, frameCreationProgram(init, create2, value))
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &creator,
						Limits: types.FrameGasLimits{Execution: 200000, State: 1530}})
					tracker := bal.NewTracker()
					receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
					if err != nil {
						t.Fatal(err)
					}
					wantBalance, wantState, wantNonce := int64(7), uint64(0), uint64(0)
					var wantCode []byte
					if !fails {
						wantBalance += int64(value)
						wantState, wantNonce, wantCode = 1530, 1, []byte{0}
					}
					if got := receipt.FrameResults8141[1]; got.Status != 1 || got.GasUsed.State != wantState {
						t.Fatalf("prefunded creation must not charge a new account: %+v", got)
					}
					if db.GetBalance(destination).Cmp(big.NewInt(wantBalance)) != 0 || db.GetNonce(destination) != wantNonce || !bytes.Equal(db.GetCode(destination), wantCode) {
						t.Fatal("constructor state or prefunding was not preserved/restored")
					}
					// Independently check the same CREATE/CREATE2 body in unchanged
					// upstream Amsterdam. Its legacy gas pool pays the state deposit
					// too; remove that known 1530 charge when comparing execution.
					sdk, err := gethstate.New(gethtypes.EmptyRootHash, gethstate.NewDatabaseForTesting())
					if err != nil {
						t.Fatal(err)
					}
					sdk.SetCode(gethcommon.Address(creator), frameCreationProgram(init, create2, value), tracing.CodeChangeUnspecified)
					sdk.SetBalance(gethcommon.Address(creator), uint256.NewInt(100), tracing.BalanceChangeUnspecified)
					sdk.SetBalance(gethcommon.Address(destination), uint256.NewInt(7), tracing.BalanceChangeUnspecified)
					sdkCfg := *params.AllDevChainProtocolChanges
					sdkCfg.AmsterdamTime, sdkCfg.BogotaTime = new(uint64), nil
					_, left, err := runtime.Call(gethcommon.Address(creator), nil, &runtime.Config{
						ChainConfig: &sdkCfg, State: sdk, GasLimit: 197000, Origin: gethcommon.Address(tx.Sender), BlockNumber: big.NewInt(1), Time: 12,
					})
					if err != nil || sdk.GetBalance(gethcommon.Address(destination)).ToBig().Cmp(big.NewInt(wantBalance)) != 0 ||
						sdk.GetNonce(gethcommon.Address(destination)) != wantNonce || !bytes.Equal(sdk.GetCode(gethcommon.Address(destination)), wantCode) ||
						197000-left != receipt.FrameResults8141[1].GasUsed.Execution-3000+wantState {
						t.Fatalf("prefunded creation differs from independent Amsterdam state/gas: left=%d err=%v", left, err)
					}
					seen := false
					for _, entry := range tracker.Build(1).Entries {
						if entry.Address != destination {
							continue
						}
						seen = true
						changed := !fails && value != 0
						if (entry.BalanceChange != nil) != changed {
							t.Fatalf("unchanged prefunding must not be published as a balance change: %+v", entry.BalanceChange)
						}
						if changed && (entry.BalanceChange.OldValue.Cmp(big.NewInt(7)) != 0 || entry.BalanceChange.NewValue.Cmp(big.NewInt(wantBalance)) != 0) {
							t.Fatalf("balance baseline must precede account reconstruction: %+v", entry.BalanceChange)
						}
					}
					if !seen {
						t.Fatal("creation destination is missing from BAL")
					}
				})
			}
		}
	}
}

func TestFrame8141PrefundedCreationRollbackBAL(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, batch := range []bool{false, true} {
			t.Run(fmt.Sprintf("create2_%v_batch_%v", create2, batch), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				creator, failure := types.Address{19: 0x90}, types.Address{19: 0x91}
				init := []byte{0x5f, 0x5f, 0x53, 0x60, 1, 0x5f, 0xf3}
				code := frameCreationProgram(init, create2, 3)
				if !batch {
					code[int(code[3])-1] = 0xfe // exception after successful creation
				}
				destination := vm.CreateAddressFromNonce(creator, 0)
				if create2 {
					destination = vm.Create2AddressFromSaltAndCode(creator, big.NewInt(1), init)
				}
				db.AddBalance(destination, big.NewInt(7))
				db.AddBalance(creator, big.NewInt(100))
				db.SetCode(creator, code)
				db.SetCode(failure, []byte{0x5f, 0x5f, 0xfd})
				frame := types.Frame8141{Mode: types.ModeSender, Target: &creator, Limits: types.FrameGasLimits{Execution: 200000, State: 1530}}
				if batch {
					frame.Flags = types.FrameAtomicBatch
				}
				tx.Frames = append(tx.Frames, frame)
				if batch {
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &failure, Limits: types.FrameGasLimits{Execution: 5000}})
				}
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeDefault, Target: &destination, Limits: types.FrameGasLimits{Execution: 5000}})
				tracker := bal.NewTracker()
				receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
				if err != nil {
					t.Fatal(err)
				}
				wantStatus := uint64(0)
				if batch {
					wantStatus = 1
				}
				if got := receipt.FrameResults8141[1]; got.Status != wantStatus || got.GasUsed.State != 0 {
					t.Fatalf("frame/batch rollback must remove deposit state gas: %+v", got)
				}
				if db.GetNonce(creator) != 0 || db.GetNonce(destination) != 0 || len(db.GetCode(destination)) != 0 ||
					db.GetBalance(creator).Cmp(big.NewInt(100)) != 0 || db.GetBalance(destination).Cmp(big.NewInt(7)) != 0 || db.GetNonce(tx.Sender) != 1 {
					t.Fatal("rollback did not restore creator/destination while retaining prior payment approval")
				}
				seen := false
				for _, entry := range tracker.Build(1).Entries {
					if entry.Address == destination {
						seen = true
						if entry.BalanceChange != nil || entry.NonceChange != nil || entry.CodeChange != nil {
							t.Fatalf("rolled-back reconstruction must retain only the account access: %+v", entry)
						}
					}
				}
				if !seen || receipt.FrameResults8141[len(tx.Frames)-1].Status != 1 {
					t.Fatal("subsequent frame must still access the restored balance-only destination")
				}
			})
		}
	}
}
