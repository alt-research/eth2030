package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// Persistent backends must implement the same EIP-8141 rollback boundaries
// as memory state, including cold accesses after failure and transaction-start
// storage values after committing and reopening the first transaction.
func TestCanonicalFrame8141PersistentBackendJournals(t *testing.T) {
	type outcome struct {
		root         types.Hash
		receipt, bal []byte
	}
	var reference []outcome
	for _, kind := range []string{"memory", "trie", "snap"} {
		t.Run(kind, func(t *testing.T) {
			tx, initial, cfg, header := frameExecutionFixture(t)
			reverted, reader, skipped := types.Address{19: 0x90}, types.Address{19: 0x91}, types.Address{19: 0x92}
			caller, helper, stored := types.Address{19: 0x93}, types.Address{19: 0x94}, types.Address{19: 0x95}
			initial.SetCode(reverted, []byte{0x5f, 0x5f, 0xfd})
			initial.SetCode(reader, []byte{0x5f, 0x54, 0x50, 0})
			initial.SetState(reader, types.Hash{}, types.Hash{31: 7})
			initial.SetCode(skipped, []byte{0})
			initial.SetCode(stored, []byte{0x5f, 0x35, 0x5f, 0x55, 0})
			initial.SetState(stored, types.Hash{}, types.Hash{31: 5})
			// A delegated child overwrites the parent's transient value then
			// reverts. The parent must retain 42; a later Frame must read zero.
			initial.SetCode(helper, []byte{0x60, 99, 0x5f, 0x5d, 0x5f, 0x5f, 0xfd})
			code := []byte{0x36, 0x60, 0, 0x57, 0x60, 42, 0x5f, 0x5d, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
			code = append(code, helper[:]...)
			code = append(code, 0x61, 0x27, 0x10, 0xf4, 0x50)
			code[2] = byte(len(code))
			code = append(code, 0x5b, 0x5f, 0x5c, 0x5f, 0x52, 0x60, 32, 0x5f, 0xa0, 0)
			initial.SetCode(caller, code)
			initialRoot, err := initial.Commit()
			if err != nil {
				t.Fatal(err)
			}
			store := rawdb.NewMemoryDB()
			var db state.StateDB = initial.Copy()
			switch kind {
			case "trie":
				persisted := state.NewTrieStateDBFromMemory(store, initial.Copy())
				if root, err := persisted.Commit(); err != nil || root != initialRoot {
					t.Fatalf("initial trie persistence: root=%s err=%v", root, err)
				}
				db = state.NewTrieStateDB(store)
			case "snap":
				snapshot, err := state.ExportStateSnapshot(initial)
				if err != nil {
					t.Fatal(err)
				}
				if err := state.StoreStateSnapshot(store, initialRoot, snapshot); err != nil {
					t.Fatal(err)
				}
				db, err = state.OpenSnapState(store, initialRoot)
				if err != nil {
					t.Fatal(err)
				}
			}
			frame := func(target *types.Address, flags uint8) types.Frame8141 {
				return types.Frame8141{Mode: types.ModeSender, Target: target, Flags: flags, Limits: types.FrameGasLimits{Execution: 50000}}
			}
			tx.Frames = append(tx.Frames, frame(&reverted, 0), frame(&reverted, 0), frame(&reader, 4), frame(&reverted, 4), frame(&skipped, 0), frame(&reader, 0), frame(&reader, 0), frame(&caller, 0), frame(&caller, 0))
			tx.Frames[9].Data = []byte{1}
			first := signFrameExecution(t, tx, false)
			tx.Nonce, tx.Frames = 1, tx.Frames[:1]
			tx.Frames = append(tx.Frames, frame(&stored, 0), frame(&stored, 0))
			tx.Frames[2].Data = make([]byte, 32)
			tx.Frames[2].Data[31] = 5
			second := signFrameExecution(t, tx, false)
			pool := new(gaspool.GasPool).AddGas(header.GasLimit)
			var got []outcome
			for phase, transaction := range []*types.Transaction{first, second} {
				db.SetTxContext(transaction.Hash(), phase)
				tracker := bal.NewTracker()
				receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, transaction, pool, tracker)
				if err != nil {
					t.Fatal(err)
				}
				wantGas := []uint64{100, 3004, 3004, 5104, 3004, 0, 5104, 204}
				if phase == 1 {
					wantGas = []uint64{100, 15107, 207}
				}
				for i, want := range wantGas {
					if result := receipt.FrameResults8141[i]; result.GasUsed.Execution != want || result.GasUsed.State != 0 {
						t.Fatalf("phase %d Frame %d: gas=%+v want execution=%d state=0", phase, i, result.GasUsed, want)
					}
				}
				if phase == 0 {
					for i, want := range []int64{42, 0} {
						logs := receipt.FrameResults8141[8+i].Logs
						if len(logs) != 1 || len(logs[0].Data) != 32 || new(big.Int).SetBytes(logs[0].Data).Cmp(big.NewInt(want)) != 0 {
							t.Fatalf("nested/trans-Frame transient journal: Frame %d logs=%+v", 8+i, logs)
						}
					}
				} else if db.GetState(stored, types.Hash{}) != (types.Hash{31: 5}) {
					t.Fatal("original slot did not survive commit/reopen and clear/restore")
				}
				if err := state.ReadError(db); err != nil {
					t.Fatal(err)
				}
				rawReceipt, err := receipt.EncodeRLP()
				if err != nil {
					t.Fatal(err)
				}
				rawBAL, err := tracker.Build(uint64(phase + 1)).EncodeRLP()
				if err != nil {
					t.Fatal(err)
				}
				root, err := db.Commit()
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, outcome{root, rawReceipt, rawBAL})
				if kind == "trie" {
					db = state.NewTrieStateDB(store)
				} else if kind == "snap" {
					db, err = state.OpenSnapState(store, root)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if db.GetNonce(tx.Sender) != 2 {
				t.Fatal("committed sender nonce did not survive reopening")
			}
			if reference == nil {
				reference = got
			}
			for i := range got {
				if got[i].root != reference[i].root || !bytes.Equal(got[i].receipt, reference[i].receipt) || !bytes.Equal(got[i].bal, reference[i].bal) {
					t.Fatalf("phase %d changed state/receipt/BAL across backends", i)
				}
			}
		})
	}
}
