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
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
)

// APPROVE's memory behavior is specified to match RETURN. Replace only the
// final opcode to compare the same program with unchanged upstream RETURN;
// the unused scope word does not affect RETURN's result or gas.
func checkFrameApproveReturnOracle(t *testing.T, code, output []byte, used uint64) {
	t.Helper()
	sdk, err := gethstate.New(gethtypes.EmptyRootHash, gethstate.NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	returned := bytes.Clone(code)
	returned[len(returned)-1] = 0xf3
	sdk.SetCode(gethcommon.Address{19: 0x90}, returned, 0)
	cfg := *params.AllDevChainProtocolChanges
	cfg.AmsterdamTime, cfg.BogotaTime = new(uint64), nil
	ret, left, err := runtime.Call(gethcommon.Address{19: 0x90}, nil, &runtime.Config{
		ChainConfig: &cfg, State: sdk, GasLimit: 900, BlockNumber: big.NewInt(1), Time: 12,
	})
	if used == 0 {
		if err == nil || left != 0 {
			t.Fatalf("upstream RETURN must reject the same nonempty region: left=%d err=%v", left, err)
		}
		return
	}
	if err != nil || !bytes.Equal(ret, output) || 900-left != used-100 {
		t.Fatalf("APPROVE memory differs from upstream RETURN: output=%x left=%d err=%v", ret, left, err)
	}
}

// EIP-8141 APPROVE follows RETURN's memory rules: an empty region ignores
// its offset, including a full-width stack word, and does not expand memory.
func TestFrame8141ApproveEmptyReturnIgnoresOffset(t *testing.T) {
	for _, offset := range []*big.Int{
		new(big.Int), big.NewInt(32),
		new(big.Int).Lsh(big.NewInt(1), 64),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
	} {
		t.Run(offset.Text(16), func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			// PUSH1 scope; PUSH0 length; PUSH32 offset; APPROVE.
			code := []byte{0x60, 3, 0x5f, 0x7f}
			code = append(code, offset.FillBytes(make([]byte, 32))...)
			code = append(code, 0xaa)
			db.SetCode(tx.Sender, code)
			checkFrameApproveReturnOracle(t, code, nil, 108)
			receipt, _, err := ApplyTransaction(cfg, db, header, types.NewTransaction(tx), new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatalf("empty APPROVE must ignore offset: %v", err)
			}
			if got := receipt.FrameResults8141[0]; got.Status != 1 || got.GasUsed.Execution != 108 || got.GasUsed.State != 0 {
				t.Fatalf("want warm entry 100 + pushes 8, without memory expansion: %+v", got)
			}
			if receipt.Payer != tx.Sender || db.GetNonce(tx.Sender) != 1 {
				t.Fatal("empty return did not commit payment approval")
			}
		})
	}
}

func TestFrame8141ApproveReturnMemoryBoundaries(t *testing.T) {
	large := new(big.Int).Lsh(big.NewInt(1), 64)
	for _, tc := range []struct {
		name           string
		offset, length *big.Int
		output         []byte
		gas            uint64 // zero means the VERIFY frame must invalidate the transaction
	}{
		{"last_byte", big.NewInt(31), big.NewInt(1), []byte{0x42}, 120},
		{"word", new(big.Int), big.NewInt(32), append(make([]byte, 31), 0x42), 120},
		{"expansion", big.NewInt(31), big.NewInt(2), []byte{0x42, 0}, 123},
		{"empty_large_offset", large, new(big.Int), nil, 120},
		{"nonempty_large_offset", large, big.NewInt(1), nil, 0},
		{"large_length", new(big.Int), large, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			// MSTORE(0, 0x42); APPROVE(offset, length, 3).
			code := []byte{0x60, 0x42, 0x5f, 0x52, 0x60, 3, 0x7f}
			code = append(code, tc.length.FillBytes(make([]byte, 32))...)
			code = append(code, 0x7f)
			code = append(code, tc.offset.FillBytes(make([]byte, 32))...)
			code = append(code, 0xaa)
			db.SetCode(tx.Sender, code)
			checkFrameApproveReturnOracle(t, code, tc.output, tc.gas)
			before := db.GetRoot()
			gp := new(gaspool.GasPool).AddGas(header.GasLimit)
			trace := vm.NewCallTracer(false, false)
			receipt, _, err := applyFrame8141(cfg, nil, db, header, types.NewTransaction(tx), gp, nil, vm.Config{Debug: true, Tracer: trace})
			if tc.gas == 0 {
				if err == nil || db.GetRoot() != before || gp.Gas() != header.GasLimit || gp.StateGas() != header.GasLimit {
					t.Fatalf("invalid nonempty region must undo all approval effects and reservations: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := receipt.FrameResults8141[0]; got.Status != 1 || got.GasUsed.Execution != tc.gas {
				t.Fatalf("APPROVE memory gas: %+v, want %d", got, tc.gas)
			}
			if trace.Root == nil || trace.Root.Err != nil || !bytes.Equal(trace.Root.Output, tc.output) {
				t.Fatalf("APPROVE did not return the designated memory region: %+v", trace.Root)
			}
		})
	}
}

// EIP-214 rejects SSTORE before its gas calculation reads storage. The
// exception is catchable by the caller and must not invent EIP-7928 slot reads.
func TestFrame8141StaticSstoreHasNoStorageAccess(t *testing.T) {
	for _, initial := range []types.Hash{{}, {31: 7}} {
		for _, childGas := range []uint64{2300, 50000} {
			t.Run(fmt.Sprintf("initial_%d_gas_%d", initial[31], childGas), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				caller, writer := types.Address{19: 0x90}, types.Address{19: 0x91}
				db.SetCode(writer, []byte{0x60, 1, 0x5f, 0x55, 0})
				db.SetState(writer, types.Hash{}, initial)
				// STATICCALL writer; store its failure result in a log.
				code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
				code = append(code, writer[:]...)
				code = append(code, 0x61, byte(childGas>>8), byte(childGas), 0xfa, 0x5f, 0x52, 0x60, 32, 0x5f, 0xa0, 0)
				db.SetCode(caller, code)
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &caller,
					Limits: types.FrameGasLimits{Execution: 100000, State: 97920}})
				transaction := signFrameExecution(t, tx, false)
				db.SetTxContext(transaction.Hash(), 0)
				tracker := bal.NewTracker()
				receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
				if err != nil {
					t.Fatal(err)
				}
				result := receipt.FrameResults8141[1]
				if result.Status != 1 || result.GasUsed.State != 0 || len(result.Logs) != 1 || new(big.Int).SetBytes(result.Logs[0].Data).Sign() != 0 {
					t.Fatalf("caller must catch static write exception without state gas: %+v", result)
				}
				if db.GetState(writer, types.Hash{}) != initial {
					t.Fatal("static write changed storage")
				}
				seen := false
				for _, entry := range tracker.Build(1).Entries {
					if entry.Address == writer {
						seen = true
						if len(entry.StorageReads) != 0 || len(entry.StorageChanges) != 0 {
							t.Fatalf("static prohibition must precede slot access: %+v", entry)
						}
					}
				}
				if !seen {
					t.Fatal("STATICCALL still accesses the writer account")
				}
			})
		}
	}
}
