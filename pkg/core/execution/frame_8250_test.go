package execution

import (
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func delta8250(base *types.FrameTx8141, keys ...int64) *types.FrameTx8250Delta {
	tx := &types.FrameTx8250Delta{ChainID: base.ChainID, Sender: base.Sender, Frames: base.Frames, Fees: base.Fees}
	for _, key := range keys {
		tx.NonceKeys = append(tx.NonceKeys, big.NewInt(key))
	}
	return tx
}

func signDelta8250(t *testing.T, tx *types.FrameTx8250Delta) {
	t.Helper()
	key, _ := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	tx.Signatures = []types.FrameSignature{{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}}
	hash, err := tx.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	sig, err := gethcrypto.Sign(hash[:], key)
	if err != nil {
		t.Fatal(err)
	}
	tx.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
}

func TestFrame8250DeltaNativeNonceDomains(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(map[bool]string{false: "sequential", true: "single_use"}[single], func(t *testing.T) {
			base, db, cfg, header := frameExecutionFixture(t)
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			tx := delta8250(base, 0, 7)
			tx.Frames[0].Limits.State = 2 * 97920
			if single {
				tx.NonceSeq = math.MaxUint64
			}
			signDelta8250(t, tx)
			tracker := bal.NewTracker()
			receipt, _, err := ExecuteFrame8250Delta(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
			if err != nil {
				t.Fatal(err)
			}
			want := uint64(1)
			if single {
				want = math.MaxUint64
			}
			if receipt.TxHash == types.NewTransaction(tx.ExecutionFields()).Hash() || db.GetNonce(tx.Sender) != 0 || receipt.FrameResults8141[0].GasUsed.State != 2*97920 {
				t.Fatal("keyed profile used base hash/nonce/state gas")
			}
			for _, key := range tx.NonceKeys {
				slot := types.NonceSlot8250(tx.Sender, key)
				value := db.GetState(types.NonceManager8250, slot)
				if new(big.Int).SetBytes(value[:]).Uint64() != want {
					t.Fatal("wrong consumed nonce")
				}
				if _, warm := db.SlotInAccessList(types.NonceManager8250, slot); warm {
					t.Fatal("protocol nonce bookkeeping warmed slot")
				}
			}
			if db.AddressInAccessList(types.NonceManager8250) {
				t.Fatal("nonce manager warmed")
			}
			found := false
			for _, entry := range tracker.Build(7).Entries {
				if entry.Address == types.NonceManager8250 {
					found = true
					if len(entry.StorageReads) != 0 || len(entry.StorageChanges) != 2 {
						t.Fatalf("nonce BAL: %+v", entry)
					}
					if entry.AccessIndex != 7 {
						t.Fatal("wrong BAL index")
					}
				}
			}
			if !found {
				t.Fatal("missing nonce manager BAL")
			}
			if _, _, err = ExecuteFrame8250Delta(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil); err == nil {
				t.Fatal("replayed consumed nonce")
			}
			if !single {
				tx.NonceSeq = 1
				signDelta8250(t, tx)
				receipt, _, err = ExecuteFrame8250Delta(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
				if err != nil || receipt.FrameResults8141[0].GasUsed.State != 0 {
					t.Fatalf("later use charged creation: %v", err)
				}
			}
		})
	}
}

func TestFrame8250DeltaOverlapAndDisjointSets(t *testing.T) {
	base, db, cfg, header := frameExecutionFixture(t)
	db.SetNonce(types.NonceManager8250, 1)
	db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
	base.Frames[0].Limits.State = 3 * 97920
	for i, keys := range [][]int64{{1, 2}, {3, 4}, {2, 5}, {0}} {
		tx := delta8250(base, keys...)
		signDelta8250(t, tx)
		before := db.GetRoot()
		_, _, err := ExecuteFrame8250Delta(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
		if (err == nil) != (i != 2) {
			t.Fatalf("set %v: %v", keys, err)
		}
		if i == 2 && db.GetRoot() != before {
			t.Fatal("overlap rejection changed state")
		}
	}
}

func TestFrame8250DeltaApprovalAndBodyRollback(t *testing.T) {
	for _, failure := range []string{"state_short", "verify_after_payment", "body_revert", "atomic_revert", "approval_revert"} {
		t.Run(failure, func(t *testing.T) {
			base, db, cfg, header := frameExecutionFixture(t)
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			base.Frames[0].Limits.State = 97920
			target := types.Address{19: 0x90}
			db.SetCode(target, []byte{0x5f, 0x5f, 0xfd})
			switch failure {
			case "state_short":
				base.Frames[0].Limits.State--
			case "verify_after_payment":
				base.Frames = append(base.Frames, types.Frame8141{Mode: types.ModeVerify, Target: &target, Limits: types.FrameGasLimits{Execution: 5000}})
			case "body_revert":
				base.Frames = append(base.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 5000}})
			case "atomic_revert":
				base.Frames = append(base.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Flags: 4, Limits: types.FrameGasLimits{Execution: 5000}}, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 5000}})
			case "approval_revert":
				db.SetCode(target, []byte{0x60, 3, 0x5f, 0x5f, 0xaa})
				code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
				code = append(code, target[:]...)
				code = append(code, 0x61, 0x75, 0x30, 0xf4, 0x50, 0x5f, 0x5f, 0xfd)
				db.SetCode(base.Sender, code)
				base.Frames[0].Limits.Execution = 50000
			}
			tx := delta8250(base, 0)
			signDelta8250(t, tx)
			before := db.GetRoot()
			gp := new(gaspool.GasPool).AddGas(header.GasLimit)
			receipt, _, err := ExecuteFrame8250Delta(cfg, db, header, tx, gp, nil)
			valid := failure == "body_revert" || failure == "atomic_revert"
			if (err == nil) != valid {
				t.Fatal(err)
			}
			value := db.GetState(types.NonceManager8250, types.NonceSlot8250(tx.Sender, big.NewInt(0)))
			if !valid && (db.GetRoot() != before || gp.Gas() != header.GasLimit || gp.StateGas() != header.GasLimit) {
				t.Fatal("invalid transaction retained state or gas reservation")
			}
			if valid && (value != (types.Hash{31: 1}) || receipt.FrameResults8141[0].GasUsed.State != 97920) {
				t.Fatal("later failure reverted approved nonce")
			}
		})
	}
}

func TestFrame8250DeltaCreateAndLegacyIntrospection(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "keyed"}[keyed], func(t *testing.T) {
			base, db, cfg, header := frameExecutionFixture(t)
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			db.SetNonce(base.Sender, 9)
			// Before payment: CREATE then APPROVE; afterwards record immutable prestate
			// legacy nonce, selected sequence, count, key-set hash and first key.
			db.SetCode(base.Sender, []byte{0x5f, 0x5f, 0x5f, 0xf0, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa})
			base.Frames[0].Mode = types.ModeDefault
			base.Frames[0].Limits = types.FrameGasLimits{Execution: 50000, State: 281520}
			target := types.Address{19: 0x90}
			var code []byte
			for i, param := range []byte{13, 1, 14, 15, 16} {
				code = append(code, 0x60, param, 0xb0, 0x60, byte(i), 0x55)
			}
			code = append(code, 0)
			db.SetCode(target, code)
			base.Frames = append(base.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 100000, State: 5 * 97920}})
			tx := delta8250(base)
			tx.NonceSeq = 9
			if keyed {
				tx.NonceKeys = []*big.Int{big.NewInt(42)}
				tx.NonceSeq = 0
			}
			signDelta8250(t, tx)
			receipt, _, err := ExecuteFrame8250Delta(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
			if err != nil {
				t.Fatal(err)
			}
			wantNonce := uint64(11)
			if keyed {
				wantNonce = 10
			}
			if db.GetNonce(tx.Sender) != wantNonce || receipt.FrameResults8141[1].Status != 1 {
				t.Fatalf("CREATE/approval: nonce=%d receipt=%+v", db.GetNonce(tx.Sender), receipt.FrameResults8141)
			}
			legacy := db.GetState(target, types.Hash{})
			if legacy != (types.Hash{31: 9}) {
				t.Fatal("legacy introspection observed mutated nonce")
			}
			expected := []types.Hash{types.BytesToHash(new(big.Int).SetUint64(tx.NonceSeq).Bytes()), types.BytesToHash(new(big.Int).SetUint64(uint64(len(tx.NonceKeys))).Bytes()), types.NonceKeysHash8250(tx.NonceKeys), {}}
			if keyed {
				expected[3] = types.Hash{31: 42}
			}
			for i, want := range expected {
				if db.GetState(target, types.Hash{31: byte(i + 1)}) != want {
					t.Fatalf("TXPARAM %d mismatch", i)
				}
			}
		})
	}
}
