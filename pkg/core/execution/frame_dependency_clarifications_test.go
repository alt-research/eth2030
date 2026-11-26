package execution

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

// EIP-8038 #12626 clarifies that SELFDESTRUCT's ACCOUNT_WRITE applies
// only to a positive transfer creating a dead beneficiary, even when warm.
func TestFrameCLatestSelfdestructAccountWrite(t *testing.T) {
	for _, funded := range []bool{false, true} {
		for _, existing := range []bool{false, true} {
			for _, warm := range []bool{false, true} {
				t.Run(fmt.Sprintf("funded_%t/existing_%t/warm_%t", funded, existing, warm), func(t *testing.T) {
					tx, db, cfg, header := frameExecutionFixture(t)
					target, beneficiary := types.Address{19: 0x90}, types.Address{19: 0x91}
					code := append(append([]byte{0x73}, beneficiary[:]...), 0xff)
					db.SetCode(target, code)
					if funded {
						db.AddBalance(target, big.NewInt(1))
					}
					if existing {
						db.AddBalance(beneficiary, big.NewInt(1))
					}
					if warm {
						tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeDefault, Target: &beneficiary, Limits: types.FrameGasLimits{Execution: 5000}})
					}
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 30000, State: 183600}})
					receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
					if err != nil {
						t.Fatal(err)
					}
					wantExecution, wantState := uint64(11003), uint64(0) // target 3000, PUSH20 3, base 5000, beneficiary 3000
					if warm {
						wantExecution -= 3000
					}
					if funded && !existing {
						wantExecution += 9000
						wantState = 183600
					}
					result := receipt.FrameResults8141[len(tx.Frames)-1]
					if result.Status != 1 || result.GasUsed.Execution != wantExecution || result.GasUsed.State != wantState {
						t.Fatalf("SELFDESTRUCT result=%+v, want execution/state=%d/%d", result, wantExecution, wantState)
					}
					if string(db.GetCode(target)) != string(code) {
						t.Fatal("pre-existing contract deleted by SELFDESTRUCT")
					}
				})
			}
		}
	}
}

// EIP-2780 #12622 explicitly removes EIP-7702's transaction-start
// delegation-target warming. Explicit access-list warming remains valid.
func TestFrameCLatestDelegationRecipientWarming(t *testing.T) {
	for _, listed := range []bool{false, true} {
		t.Run(fmt.Sprintf("access_list_%t", listed), func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			authority, delegate := types.Address{19: 0x90}, types.Address{19: 0x91}
			db.SetCode(authority, types.AddressToDelegation(delegate))
			db.SetCode(delegate, []byte{0})
			msg := &config.Message{From: tx.Sender, To: &authority, GasLimit: 100000, Value: new(big.Int),
				GasPrice: big.NewInt(3), GasFeeCap: big.NewInt(3), GasTipCap: big.NewInt(2), TxType: types.DynamicFeeTxType}
			if listed {
				msg.AccessList = types.AccessList{{Address: delegate}}
			}
			result, err := ApplyMessage(cfg, nil, db, header, msg, new(gaspool.GasPool).AddGas(header.GasLimit))
			// Unlisted: 15000 intrinsic + 3000 cold resolution. Listed:
			// 15000 + 2900 + 1280 (EIP-7981: 20 bytes * 64) intrinsic,
			// plus 100 warm resolution.
			want := uint64(18000)
			if listed {
				want += 1280
			}
			if err != nil || result.Err != nil || result.UsedGas != want || result.StateGasUsed != 0 {
				t.Fatalf("delegated recipient result=%+v err=%v", result, err)
			}
		})
	}
}
