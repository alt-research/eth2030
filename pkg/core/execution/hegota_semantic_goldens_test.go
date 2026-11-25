package execution

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// These frozen expected effects are derived from the pinned specification, not
// regenerated from native execution. The two extension oracles have a separate
// Python consumer and do not imply activation of a combined transaction format.
func TestHegotaSemanticGoldens(t *testing.T) {
	var corpus struct {
		Vectors []struct {
			ID, Kind string
			EIP      int
			Input    struct {
				Sender         string
				Nonce, Balance uint64
				Codes          map[string]string
				Frames         []struct {
					Target, Data     string
					Mode, Flags      uint8
					Execution, State uint64
				}
			}
			Expected struct {
				Steps []struct {
					Error         string
					Approved      bool
					Payer         *string
					Nonce         uint64
					SenderBalance uint64 `json:"sender_balance"`
					PayerBalance  uint64 `json:"payer_balance"`
					ExecutionGas  uint64 `json:"execution_gas"`
				}
				Escrow        uint64
				Invalid       bool
				Nonce, Logs   uint64
				Payer         string
				RootUnchanged bool   `json:"root_unchanged"`
				GasUsed       uint64 `json:"gas_used"`
				Statuses      []uint64
				StateGas      []uint64 `json:"state_gas"`
				ExecutionGas  []uint64 `json:"execution_gas"`
				Storage       map[string]string
			}
		} `json:"trace_vectors"`
	}
	raw, err := os.ReadFile("../../../specs/hegota/semantic-goldens.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, golden := range corpus.Vectors {
		if golden.EIP != 8141 {
			continue
		}
		count++
		t.Run(golden.ID, func(t *testing.T) {
			decode := func(s string) []byte {
				b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
				if err != nil {
					t.Fatal(err)
				}
				return b
			}
			db := state.NewMemoryStateDB()
			sender := types.HexToAddress(golden.Input.Sender)
			other := types.HexToAddress("0x0000000000000000000000000000000000000092")
			db.AddBalance(sender, new(big.Int).SetUint64(golden.Input.Balance))
			db.AddBalance(other, new(big.Int).SetUint64(golden.Input.Balance))
			db.SetNonce(sender, golden.Input.Nonce)
			for address, code := range golden.Input.Codes {
				db.SetCode(types.HexToAddress(address), decode(code))
			}
			tx := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Nonce: golden.Input.Nonce,
				Fees: types.FrameFeeBudget{MaxFee: big.NewInt(10000000), MaxPriorityFeePerGas: new(big.Int)}}
			for _, frame := range golden.Input.Frames {
				target := types.HexToAddress(frame.Target)
				tx.Frames = append(tx.Frames, types.Frame8141{Target: &target, Data: decode(frame.Data), Mode: frame.Mode,
					Flags: frame.Flags, Limits: types.FrameGasLimits{Execution: frame.Execution, State: frame.State}})
			}
			cfg := *config.TestConfigHogota
			cfg.ChainID = tx.ChainID
			header := &types.Header{Number: big.NewInt(10), Time: 100, GasLimit: 30000000, BaseFee: big.NewInt(1)}
			before := db.GetRoot()
			if golden.Kind == "approval_calls" {
				// Deliberately exercise instruction preconditions, including a
				// wrong-target call that envelope validity would reject earlier.
				payment, err := tx.Payment(header.BaseFee, big.NewInt(1))
				if err != nil {
					t.Fatal(err)
				}
				if golden.Expected.Escrow != 0 && payment.Escrow().Uint64() != golden.Expected.Escrow {
					t.Fatal("fee oracle mismatch")
				}
				evm := vm.NewEVMWithState(vm.BlockContext{BlockNumber: header.Number, Time: header.Time, BaseFee: header.BaseFee}, vm.TxContext{}, vm.Config{}, db)
				if _, err := ConfigureBlockEVM(&cfg, header, evm); err != nil {
					t.Fatal(err)
				}
				ctx := evm.SetFrameTransaction(tx, payment)
				ctx.Results = make([]types.FrameResult8141, len(tx.Frames))
				db.AddAddressToAccessList(sender)
				if len(golden.Expected.Steps) != len(tx.Frames) {
					t.Fatal("missing per-call expectations")
				}
				for i, want := range golden.Expected.Steps {
					left, err := evm.RunFrame8141(i)
					if (want.Error == "reverted" && !errors.Is(err, vm.ErrExecutionReverted)) || (want.Error == "" && err != nil) {
						t.Fatalf("step %d: %v", i, err)
					}
					var payer *string
					if ctx.Payer != nil {
						s := strings.ToLower(ctx.Payer.Hex())
						payer = &s
					}
					if ctx.Approved != want.Approved || !reflect.DeepEqual(payer, want.Payer) || db.GetNonce(sender) != want.Nonce ||
						db.GetBalance(sender).Uint64() != want.SenderBalance || db.GetBalance(other).Uint64() != want.PayerBalance ||
						tx.Frames[i].Limits.Execution-left != want.ExecutionGas {
						t.Fatalf("step %d: approved=%v payer=%v nonce=%d balances=%s/%s execution=%d", i, ctx.Approved, payer,
							db.GetNonce(sender), db.GetBalance(sender), db.GetBalance(other), tx.Frames[i].Limits.Execution-left)
					}
				}
				return
			}
			if golden.Kind != "transaction" {
				t.Fatalf("unknown native oracle kind %q", golden.Kind)
			}
			transaction := types.NewTransaction(tx)
			db.SetTxContext(transaction.Hash(), 0)
			gp := new(gaspool.GasPool).AddGas(header.GasLimit)
			receipt, used, err := ApplyTransaction(&cfg, db, header, transaction, gp)
			if (err != nil) != golden.Expected.Invalid || db.GetNonce(sender) != golden.Expected.Nonce {
				t.Fatalf("validity/nonce: %v, nonce=%d", err, db.GetNonce(sender))
			}
			if golden.Expected.Invalid {
				if !golden.Expected.RootUnchanged || db.GetRoot() != before || used != golden.Expected.GasUsed ||
					gp.Gas() != header.GasLimit || gp.StateGas() != header.GasLimit || len(db.GetLogs(transaction.Hash())) != int(golden.Expected.Logs) {
					t.Fatal("invalid VERIFY retained effects or gas reservation")
				}
				return
			}
			if receipt.Payer != types.HexToAddress(golden.Expected.Payer) || len(receipt.Logs) != int(golden.Expected.Logs) || len(receipt.FrameResults8141) != len(golden.Expected.Statuses) {
				t.Fatal("payer/log/frame count mismatch")
			}
			for i, frame := range receipt.FrameResults8141 {
				if frame.Status != golden.Expected.Statuses[i] || frame.GasUsed.State != golden.Expected.StateGas[i] || frame.GasUsed.Execution != golden.Expected.ExecutionGas[i] || len(frame.Logs) != 0 {
					t.Fatalf("frame %d: %+v", i, frame)
				}
			}
			for address, value := range golden.Expected.Storage {
				if db.GetState(types.HexToAddress(address), types.Hash{}) != types.HexToHash(value) {
					t.Fatalf("storage mismatch: %s", address)
				}
			}
		})
	}
	if count != 7 {
		t.Fatalf("executed %d native semantic goldens, want 7", count)
	}
}
