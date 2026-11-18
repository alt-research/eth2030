package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func frameExecutionFixture(t *testing.T) (*types.FrameTx8141, *state.MemoryStateDB, *config.ChainConfig, *types.Header) {
	t.Helper()
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	db := state.NewMemoryStateDB()
	db.AddBalance(sender, big.NewInt(1_000_000_000))
	tx := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender,
		Fees: types.FrameFeeBudget{MaxFee: big.NewInt(10_000_000), MaxPriorityFeePerGas: big.NewInt(2)},
		Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3,
			Limits: types.FrameGasLimits{Execution: 1000}}},
	}
	zero := uint64(0)
	cfg := *config.TestConfig
	cfg.ChainID, cfg.HogotaTime = big.NewInt(1337), &zero
	header := newTestHeader()
	return tx, db, &cfg, header
}

func signFrameExecution(t *testing.T, tx *types.FrameTx8141, sponsor bool, arbitrary ...[]byte) *types.Transaction {
	t.Helper()
	keys := []string{"0000000000000000000000000000000000000000000000000000000000000001"}
	if sponsor {
		keys = append(keys, "0000000000000000000000000000000000000000000000000000000000000002")
	}
	tx.Signatures = make([]types.FrameSignature, len(keys))
	for i := range keys {
		tx.Signatures[i] = types.FrameSignature{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}
		if i == 1 {
			key, _ := gethcrypto.HexToECDSA(keys[i])
			address := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
			tx.Signatures[i].Signer = &address
		}
	}
	for _, witness := range arbitrary {
		tx.Signatures = append(tx.Signatures, types.FrameSignature{Scheme: types.FrameSignatureArbitrary, Signature: witness})
	}
	hash, err := tx.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	for i, text := range keys {
		key, err := gethcrypto.HexToECDSA(text)
		if err != nil {
			t.Fatal(err)
		}
		sig, err := gethcrypto.Sign(hash[:], key)
		if err != nil {
			t.Fatal(err)
		}
		tx.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
	}
	encoded, err := types.EncodeFrameTx8141(tx)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := types.DecodeTxRLP(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return transaction
}

func TestCanonicalFrame8141SponsorshipAndRollback(t *testing.T) {
	for _, sponsored := range []bool{false, true} {
		t.Run(map[bool]string{false: "self", true: "sponsored"}[sponsored], func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			sponsor := tx.Sender
			if sponsored {
				key, _ := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000002")
				sponsor = types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
				db.AddBalance(sponsor, big.NewInt(1_000_000_000))
				tx.Frames[0].Flags = 2
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1,
					Target: &sponsor, Limits: types.FrameGasLimits{Execution: 3000}})
			}
			target := types.Address{19: 0x90}
			// Write calldata to storage and emit one LOG0.
			db.SetCode(target, []byte{0x60, 0, 0x35, 0x60, 0, 0x55, 0x60, 0, 0x60, 0, 0xa0, 0x00})
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target,
				Data: []byte{1}, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}})
			transaction := signFrameExecution(t, tx, sponsored)
			db.SetTxContext(transaction.Hash(), 0)
			payerBefore := new(big.Int).Set(db.GetBalance(sponsor))
			gp := new(gaspool.GasPool).AddGas(header.GasLimit)
			receipt, used, err := ApplyTransaction(cfg, db, header, transaction, gp)
			if err != nil {
				t.Fatal(err)
			}
			if receipt.Payer != sponsor || db.GetNonce(tx.Sender) != 1 || used == 0 ||
				receipt.FrameResults8141[len(tx.Frames)-1].GasUsed.State != 97920 ||
				len(receipt.Logs) != 1 || db.GetState(target, types.Hash{}).IsZero() {
				t.Fatalf("Frame effects or receipt mismatch: %+v", receipt)
			}
			charge := new(big.Int).Sub(payerBefore, db.GetBalance(sponsor))
			if charge.Cmp(new(big.Int).Mul(new(big.Int).SetUint64(used), big.NewInt(3))) != 0 ||
				db.GetBalance(header.Coinbase).Cmp(new(big.Int).SetUint64(used*2)) != 0 {
				t.Fatalf("payer charge %s, used %d, proposer %s", charge, used, db.GetBalance(header.Coinbase))
			}
			if gp.Gas() != header.GasLimit-receipt.BlockGasUsed || gp.StateGas() != header.GasLimit-97920 {
				t.Fatalf("block gas pool: %d, used %d", gp.Gas(), used)
			}
		})
	}
}

func TestCanonicalFrame8141CrossFrameRefillAndAtomicRevert(t *testing.T) {
	for _, revertBatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "refill", true: "refill_reverted"}[revertBatch], func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			target := types.Address{19: 0x90}
			db.SetCode(target, []byte{0x60, 0, 0x35, 0x60, 0, 0x55, 0})
			tx.Frames = append(tx.Frames,
				types.Frame8141{Mode: types.ModeSender, Target: &target, Data: []byte{1}, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}},
				types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 20000}},
			)
			if revertBatch {
				tx.Frames[2].Flags = 4
				bad := types.Address{19: 0x91}
				db.SetCode(bad, []byte{0x60, 0, 0x60, 0, 0xfd})
				tx.Frames = append(tx.Frames,
					types.Frame8141{Mode: types.ModeSender, Target: &bad, Flags: 4, Limits: types.FrameGasLimits{Execution: 5000}},
					types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 5000}})
			}
			receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			wantState := uint64(0)
			if revertBatch {
				wantState = 97920
				if receipt.FrameResults8141[3].Status != 0 || receipt.FrameResults8141[4].Status != 2 {
					t.Fatalf("bad atomic statuses: %+v", receipt.FrameResults8141)
				}
			}
			if receipt.FrameResults8141[1].GasUsed.State != wantState ||
				(db.GetState(target, types.Hash{}).IsZero() == revertBatch) {
				t.Fatalf("cross-frame refill or rollback: %+v, state=%s", receipt.FrameResults8141, db.GetState(target, types.Hash{}))
			}
		})
	}
}

func TestCanonicalFrame8141FailureRestoresApprovalsAndFunds(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	target := types.Address{19: 0x90}
	db.SetCode(target, []byte{0x60, 0, 0x60, 0, 0xfd})
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Target: &target,
		Limits: types.FrameGasLimits{Execution: 5000}})
	transaction := signFrameExecution(t, tx, false)
	initial := db.GetRoot()
	gp := new(gaspool.GasPool).AddGas(header.GasLimit)
	if _, _, err := ApplyTransaction(cfg, db, header, transaction, gp); err == nil {
		t.Fatal("VERIFY failure was not transaction-invalid")
	}
	if db.GetRoot() != initial || db.GetNonce(tx.Sender) != 0 || gp.Gas() != header.GasLimit {
		t.Fatal("invalid Frame changed sender, state or gas pool")
	}
}

func TestCanonicalFrame8141IntrospectionOnExecutedState(t *testing.T) {
	for _, test := range []struct {
		name       string
		expression []byte
		want       func(*types.FrameTx8141) *big.Int
		fail       bool
	}{
		{"transaction type", []byte{0x60, 0, 0xb0}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(6) }, false},
		{"sender", []byte{0x60, 2, 0xb0}, func(tx *types.FrameTx8141) *big.Int { return new(big.Int).SetBytes(tx.Sender[:]) }, false},
		{"max fee", []byte{0x60, 4, 0xb0}, func(tx *types.FrameTx8141) *big.Int { return tx.Fees.MaxFee }, false},
		{"frame count", []byte{0x60, 9, 0xb0}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(2) }, false},
		{"current index", []byte{0x60, 10, 0xb0}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(1) }, false},
		{"signature count", []byte{0x60, 11, 0xb0}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(1) }, false},
		{"state gas left", []byte{0x60, 12, 0xb0}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(200000) }, false},
		{"signing hash", []byte{0x60, 8, 0xb0}, func(tx *types.FrameTx8141) *big.Int {
			hash, _ := tx.SigningHash()
			return new(big.Int).SetBytes(hash[:])
		}, false},
		{"previous status", []byte{0x60, 5, 0x60, 0, 0xb3}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(1) }, false},
		{"previous execution gas", []byte{0x60, 10, 0x60, 0, 0xb3}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(100) }, false},
		{"previous state gas", []byte{0x60, 11, 0x60, 0, 0xb3}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(0) }, false},
		{"allowed approval", []byte{0x60, 6, 0x60, 0, 0xb3}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(3) }, false},
		{"data load", []byte{0x60, 1, 0x60, 0, 0xb1}, func(_ *types.FrameTx8141) *big.Int { return new(big.Int).Lsh(big.NewInt(0x1234), 240) }, false},
		{"data copy", []byte{0x60, 1, 0x60, 32, 0x60, 1, 0x60, 0, 0xb2, 0x60, 0, 0x51}, func(_ *types.FrameTx8141) *big.Int { return new(big.Int).Lsh(big.NewInt(0x34), 248) }, false},
		{"signature signer", []byte{0x60, 0, 0x60, 0, 0xb4}, func(tx *types.FrameTx8141) *big.Int { return new(big.Int).SetBytes(tx.Sender[:]) }, false},
		{"signature scheme", []byte{0x60, 1, 0x60, 0, 0xb4}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(1) }, false},
		{"signature message", []byte{0x60, 2, 0x60, 0, 0xb4}, func(_ *types.FrameTx8141) *big.Int { return big.NewInt(0) }, false},
		{name: "undefined transaction parameter", expression: []byte{0x60, 5, 0xb0}, fail: true},
		{name: "future receipt status", expression: []byte{0x60, 5, 0x60, 1, 0xb3}, fail: true},
		{name: "future receipt gas", expression: []byte{0x60, 11, 0x60, 1, 0xb3}, fail: true},
		{name: "frame out of bounds", expression: []byte{0x60, 2, 0x60, 0, 0xb1}, fail: true},
		{name: "protocol signature length hidden", expression: []byte{0x60, 3, 0x60, 0, 0xb4}, fail: true},
		{name: "protocol signature bytes hidden", expression: []byte{0x60, 0, 0x60, 1, 0x60, 0, 0x60, 0, 0xb5}, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			target := types.Address{19: 0x90}
			code := append([]byte(nil), test.expression...)
			code = append(code, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xa0, 0)
			db.SetCode(target, code)
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Data: []byte{0x12, 0x34}, Limits: types.FrameGasLimits{Execution: 50000, State: 200000}})
			transaction := signFrameExecution(t, tx, false)
			db.SetTxContext(transaction.Hash(), 0)
			receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			if test.fail {
				if receipt.FrameResults8141[1].Status != 0 || receipt.FrameResults8141[1].GasUsed.Execution != 50000 || len(receipt.Logs) != 0 {
					t.Fatalf("introspection exception did not consume frame gas: %+v", receipt)
				}
				return
			}
			if len(receipt.Logs) != 1 || new(big.Int).SetBytes(receipt.Logs[0].Data).Cmp(test.want(tx)) != 0 {
				t.Fatalf("result logs %+v, want %s", receipt.Logs, test.want(tx))
			}
		})
	}
}

func TestCanonicalFrame8141DelegatedApprovalAndNestedRollback(t *testing.T) {
	for _, test := range []struct {
		name   string
		op     byte
		revert bool
		valid  bool
	}{
		{"delegatecall approval", 0xf4, false, true},
		{"call cannot approve for target", 0xf1, false, false},
		{"delegatecall approval rolled back", 0xf4, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			tx.Frames[0].Limits.Execution = 50000
			library := types.Address{19: 0x90}
			db.SetCode(library, []byte{0x60, 3, 0x5f, 0x5f, 0xaa})
			code := []byte{0x5f, 0x5f, 0x5f, 0x5f}
			if test.op == 0xf1 {
				code = append(code, 0x5f)
			}
			code = append(code, 0x73)
			code = append(code, library[:]...)
			code = append(code, 0x61, 0x75, 0x30, test.op, 0x50)
			if test.revert {
				code = append(code, 0x5f, 0x5f, 0xfd)
			} else {
				code = append(code, 0)
			}
			db.SetCode(tx.Sender, code)
			before := db.GetRoot()
			receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
			if (err == nil) != test.valid {
				t.Fatalf("approval validity: %v", err)
			}
			if !test.valid && db.GetRoot() != before {
				t.Fatal("rejected nested approval changed state")
			}
			if test.valid && (receipt.Payer != tx.Sender || db.GetNonce(tx.Sender) != 1) {
				t.Fatal("delegate approval was not committed")
			}
		})
	}
}

func TestCanonicalFrame8141StateExhaustionAndTransientIsolation(t *testing.T) {
	t.Run("state exhaustion", func(t *testing.T) {
		tx, db, cfg, header := frameExecutionFixture(t)
		target := types.Address{19: 0x90}
		db.SetCode(target, []byte{0x60, 1, 0x5f, 0x55, 0})
		tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 30000, State: 97919}})
		receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
		if err != nil {
			t.Fatal(err)
		}
		result := receipt.FrameResults8141[1]
		if result.Status != 0 || result.GasUsed.Execution != 30000 || result.GasUsed.State != 0 || !db.GetState(target, types.Hash{}).IsZero() || db.GetNonce(tx.Sender) != 1 {
			t.Fatalf("state OOG effects: %+v", result)
		}
	})
	t.Run("transient state cleared between frames", func(t *testing.T) {
		tx, db, cfg, header := frameExecutionFixture(t)
		target := types.Address{19: 0x90}
		// TSTORE only if calldata is present; always log TLOAD(0).
		db.SetCode(target, []byte{0x36, 0x15, 0x60, 10, 0x57, 0x60, 1, 0x5f, 0x5d, 0x00, 0x5b, 0x5f, 0x5c, 0x5f, 0x52, 0x60, 32, 0x5f, 0xa0, 0})
		tx.Frames = append(tx.Frames,
			types.Frame8141{Mode: types.ModeSender, Target: &target, Data: []byte{1}, Limits: types.FrameGasLimits{Execution: 5000}},
			types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 5000}})
		transaction := signFrameExecution(t, tx, false)
		db.SetTxContext(transaction.Hash(), 0)
		receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
		if err != nil {
			t.Fatal(err)
		}
		if len(receipt.Logs) != 1 || new(big.Int).SetBytes(receipt.Logs[0].Data).Sign() != 0 {
			t.Fatal("transient storage crossed the frame boundary")
		}
	})
}

func TestCanonicalFrame8141ArbitrarySignatureIntrospection(t *testing.T) {
	for _, expression := range [][]byte{
		{0x60, 3, 0x60, 1, 0xb4},                                // arbitrary witness length
		{0x60, 1, 0x60, 32, 0x60, 0, 0x60, 0, 0xb5, 0x5f, 0x51}, // copy and load
	} {
		tx, db, cfg, header := frameExecutionFixture(t)
		target := types.Address{19: 0x90}
		db.SetCode(target, append(append([]byte(nil), expression...), 0x5f, 0x52, 0x60, 32, 0x5f, 0xa0, 0))
		tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 10000}})
		transaction := signFrameExecution(t, tx, false, []byte{0x33, 0x44})
		db.SetTxContext(transaction.Hash(), 0)
		receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
		if err != nil {
			t.Fatal(err)
		}
		want := big.NewInt(2)
		if len(expression) > 5 {
			want = new(big.Int).Lsh(big.NewInt(0x3344), 240)
		}
		if len(receipt.Logs) != 1 || new(big.Int).SetBytes(receipt.Logs[0].Data).Cmp(want) != 0 {
			t.Fatalf("arbitrary witness result: %+v", receipt.Logs)
		}
	}
}

func TestCanonicalFrame8141SponsoredSenderCreation(t *testing.T) {
	for _, budget := range []uint64{183599, 183600} {
		tx, db, cfg, header := frameExecutionFixture(t)
		db.SubBalance(tx.Sender, db.GetBalance(tx.Sender))
		key, _ := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000002")
		payer := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
		db.AddBalance(payer, big.NewInt(1_000_000_000))
		tx.Frames[0].Flags = 2
		tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 4000, State: budget}})
		root := db.GetRoot()
		receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, true), new(gaspool.GasPool).AddGas(header.GasLimit))
		if budget < 183600 {
			if err == nil || db.GetRoot() != root {
				t.Fatal("approval OOG created the sender or charged the payer")
			}
		} else if err != nil || db.GetNonce(tx.Sender) != 1 || receipt.FrameResults8141[1].GasUsed.State != 183600 {
			t.Fatalf("sponsored nonce creation charge: %+v, %v", receipt, err)
		}
	}
}

func TestCanonicalFrame8141CreationChargesAndCodeDeposit(t *testing.T) {
	for _, budget := range []uint64{186659, 186660} {
		tx, db, cfg, header := frameExecutionFixture(t)
		factory := types.Address{19: 0x90}
		// Deploy two bytes of STOP code from initcode stored in memory.
		// Initcode: PUSH1 2 PUSH0 RETURN. Returned memory is zero-filled.
		db.SetCode(factory, []byte{0x63, 0x60, 2, 0x5f, 0xf3, 0x5f, 0x52, 0x60, 4, 0x60, 28, 0x5f, 0xf0, 0x50, 0})
		tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &factory, Limits: types.FrameGasLimits{Execution: 50000, State: budget}})
		receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
		if err != nil {
			t.Fatal(err)
		}
		created := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
		result := receipt.FrameResults8141[1]
		if budget < 186660 {
			if db.GetCodeSize(created) != 0 || result.GasUsed.State != 0 {
				t.Fatal("failed code deposit retained state or gas charge")
			}
		} else if db.GetCodeSize(created) != 2 || result.GasUsed.State != 186660 || db.GetNonce(factory) != 1 {
			t.Fatalf("conditional account plus code-deposit charge: %+v, code=%x", result, db.GetCode(created))
		}
	}
}

func TestCanonicalFrame8141AtomicBALRetainsReadsAndNetChanges(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	writer, bad, skipped := types.Address{19: 0x90}, types.Address{19: 0x91}, types.Address{19: 0x92}
	db.SetCode(writer, []byte{0x60, 1, 0x5f, 0x55, 0})
	db.SetCode(bad, []byte{0x5f, 0x5f, 0xfd})
	db.SetCode(skipped, []byte{0})
	tx.Frames = append(tx.Frames,
		types.Frame8141{Mode: types.ModeSender, Flags: 4, Target: &writer, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}},
		types.Frame8141{Mode: types.ModeSender, Flags: 4, Target: &bad, Limits: types.FrameGasLimits{Execution: 10000}},
		types.Frame8141{Mode: types.ModeSender, Target: &skipped, Limits: types.FrameGasLimits{Execution: 10000}})
	tracker := bal.NewTracker()
	receipt, _, err := ApplyTransactionWithBAL(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.FrameResults8141[3].Status != types.FrameStatusSkipped {
		t.Fatal("fixture did not skip the final target")
	}
	var sawWriter, sawSender bool
	for _, entry := range tracker.Build(1).Entries {
		if entry.Address == skipped {
			t.Fatal("skipped Frame target was spuriously committed to the BAL")
		}
		if entry.Address == writer {
			sawWriter = true
			if len(entry.StorageChanges) != 0 || len(entry.StorageReads) != 1 || !entry.StorageReads[0].Value.IsZero() {
				t.Fatalf("reverted write leaked into BAL changes: %+v", entry)
			}
		}
		if entry.Address == tx.Sender {
			sawSender = true
			if entry.NonceChange == nil || entry.NonceChange.OldValue != 0 || entry.NonceChange.NewValue != 1 || entry.BalanceChange == nil || entry.BalanceChange.OldValue.Int64() != 1000000000 {
				t.Fatalf("fee approval BAL did not use transaction-start values: %+v", entry)
			}
		}
	}
	if !sawWriter || !sawSender {
		t.Fatal("BAL lost the executed access or sender approval")
	}
}

func TestCanonicalFrame8141ZeroValuePrecompileDoesNotCreateState(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	identity := types.Address{19: 4}
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &identity, Data: []byte{1}, Limits: types.FrameGasLimits{Execution: 1000}})
	receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.FrameResults8141[1].Status != 1 || db.Exist(identity) || receipt.FrameResults8141[1].GasUsed.State != 0 {
		t.Fatal("zero-value precompile call created unmetered state")
	}
}

func TestCanonicalFrame8141ZeroFeeMetadata(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	header.BaseFee = new(big.Int)
	tx.Fees = types.FrameFeeBudget{}
	target := types.Address{19: 0x90}
	db.SetCode(target, []byte{0x60, 3, 0xb0, 0x60, 4, 0xb0, 0x01, 0x5f, 0x52, 0x60, 32, 0x5f, 0xa0, 0})
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 10000}})
	signFrameExecution(t, tx, false)
	// Exercise the Go API's nil-as-zero fee convention before RLP normalization.
	transaction := types.NewTransaction(tx)
	db.SetTxContext(transaction.Hash(), 0)
	receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Logs) != 1 || new(big.Int).SetBytes(receipt.Logs[0].Data).Sign() != 0 || db.Exist(header.Coinbase) {
		t.Fatal("zero fee metadata panicked or created an empty beneficiary")
	}
}

func TestCanonicalFrame8141BinaryOverlayAndTransactionOriginalSlots(t *testing.T) {
	tx, initial, cfg, header := frameExecutionFixture(t)
	target := types.Address{19: 0x90}
	initial.SetCode(target, []byte{0x5f, 0x35, 0x5f, 0x55, 0})
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Data: []byte{1}, Limits: types.FrameGasLimits{Execution: 30000, State: 97920}})
	first := signFrameExecution(t, tx, false)
	tx.Nonce = 1
	tx.Frames[1].Limits.State = 0
	tx.Frames[1].Data = nil
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Data: []byte{1}, Limits: types.FrameGasLimits{Execution: 30000}})
	second := signFrameExecution(t, tx, false)
	run := func(db state.StateDB) (types.Hash, [][]byte) {
		t.Helper()
		var encoded [][]byte
		for i, transaction := range []*types.Transaction{first, second} {
			db.SetTxContext(transaction.Hash(), i)
			receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 && receipt.FrameResults8141[1].GasUsed.State != 97920 {
				t.Fatal("initial allocation was not charged")
			}
			if i == 1 && (receipt.FrameResults8141[1].GasUsed.State != 0 || receipt.FrameResults8141[2].GasUsed.State != 0) {
				t.Fatal("a later transaction lost its nonzero original slot")
			}
			raw, err := receipt.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			encoded = append(encoded, raw)
		}
		if db.GetNonce(tx.Sender) != 2 || db.GetState(target, types.Hash{}).IsZero() {
			t.Fatal("Frame effects missing from state backend")
		}
		root, err := db.Commit()
		if err != nil {
			t.Fatal(err)
		}
		return root, encoded
	}
	_, memoryReceipts := run(initial.Copy())
	binary, err := state.NewBinaryOverlayState(initial.Copy())
	if err != nil {
		t.Fatal(err)
	}
	binaryRoot, binaryReceipts := run(binary)
	replay, err := state.NewBinaryOverlayState(initial.Copy())
	if err != nil {
		t.Fatal(err)
	}
	replayRoot, _ := run(replay)
	if binaryRoot != replayRoot || binaryRoot == (types.Hash{}) {
		t.Fatal("binary Frame replay produced a different overlay root")
	}
	for i := range memoryReceipts {
		if string(memoryReceipts[i]) != string(binaryReceipts[i]) {
			t.Fatal("state backend changed canonical Frame gas or receipt")
		}
	}
}

func TestCanonicalFrame8141DeployPrefixConsumesCurrentSenderNonce(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	factory := types.Address{19: 0x90}
	sender := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
	db.AddBalance(sender, big.NewInt(1000000000))
	db.SetCode(factory, []byte{0x36, 0x5f, 0x5f, 0x37, 0x36, 0x5f, 0x5f, 0xf0, 0x50, 0})
	// A deliberately permissionless fixture verifier: PUSH1 3 PUSH0 PUSH0 APPROVE.
	init := []byte{0x64, 0x60, 3, 0x5f, 0x5f, 0xaa, 0x5f, 0x52, 0x60, 5, 0x60, 27, 0xf3}
	tx.Sender = sender
	tx.Frames = []types.Frame8141{
		{Mode: types.ModeDefault, Target: &factory, Data: init, Limits: types.FrameGasLimits{Execution: 30000, State: 7650}},
		{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}},
	}
	transaction := types.NewTransaction(tx)
	if _, err := ValidateFrame8141Prefix(cfg, db, header, transaction, big.NewInt(1)); err != nil {
		t.Fatalf("valid deploy prefix: %v", err)
	}
	if db.GetNonce(sender) != 0 || db.GetCodeSize(sender) != 0 {
		t.Fatal("prefix simulation modified live deployment state")
	}
	receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	// CREATE initializes the contract nonce to 1; APPROVE increments that value.
	if db.GetNonce(sender) != 2 || receipt.FrameResults8141[0].GasUsed.State != 7650 || db.GetCodeSize(sender) != 5 {
		t.Fatalf("deployment/approval nonce or state charge: nonce=%d, receipt=%+v", db.GetNonce(sender), receipt)
	}
	// Independent StateAccount RLP/StackTrie oracle generated with go-ethereum.
	if root := db.GetRoot(); root != types.HexToHash("0xf51b379d6cdf16258d77fea404eb7f548028fffb2b52eb61d24df45c136a4583") {
		t.Fatalf("deployed Frame post-state differs from Geth trie oracle: %s", root)
	}
}

func TestCanonicalFrame8141ApprovalCannotWrapNonceAfterDefaultCalls(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	tx.Nonce = ^uint64(0) - 2
	db.SetNonce(tx.Sender, tx.Nonce)
	// Two ordinary creations consume the last two available creator nonces.
	code := []byte{0x5f, 0x5f, 0x5f, 0xf0, 0x50, 0x5f, 0x5f, 0x5f, 0xf0, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa}
	db.SetCode(tx.Sender, code)
	tx.Frames = []types.Frame8141{{Mode: types.ModeDefault, Flags: 3, Limits: types.FrameGasLimits{Execution: 50000, State: 367200}}}
	root := db.GetRoot()
	gp := new(gaspool.GasPool).AddGas(header.GasLimit)
	if _, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), gp); err == nil {
		t.Fatal("approval lowered or wrapped the current sender nonce")
	}
	if db.GetRoot() != root || gp.Gas() != header.GasLimit || gp.StateGas() != header.GasLimit {
		t.Fatal("failed payment approval retained effects")
	}
}

func TestCanonicalFrame8141ClippedTipUsesActualEscrowSettlement(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	tx.Fees.MaxPriorityFeePerGas = big.NewInt(10)
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Limits: types.FrameGasLimits{Execution: 100000}})
	signFrameExecution(t, tx, false)
	budget, err := tx.GasBudget()
	if err != nil {
		t.Fatal(err)
	}
	tx.Fees.MaxFee = new(big.Int).Add(new(big.Int).SetUint64(budget.MaxGas), big.NewInt(1))
	before := new(big.Int).Set(db.GetBalance(tx.Sender))
	receipt, used, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	paid := new(big.Int).Sub(before, db.GetBalance(tx.Sender))
	// The declared execution budget is mostly unused. The clipped escrow is
	// entirely spent, with actual tip = max_fee - used_gas * base_fee.
	wantTip := new(big.Int).Sub(tx.Fees.MaxFee, new(big.Int).SetUint64(used))
	if paid.Cmp(tx.Fees.MaxFee) != 0 || db.GetBalance(header.Coinbase).Cmp(wantTip) != 0 {
		t.Fatalf("clipped settlement: paid=%s tip=%s", paid, db.GetBalance(header.Coinbase))
	}
	rounded := new(big.Int).Mul(new(big.Int).SetUint64(used), receipt.EffectiveGasPrice)
	if rounded.Cmp(paid) == 0 {
		t.Fatal("fixture did not distinguish GASPRICE from actual settlement")
	}
}
