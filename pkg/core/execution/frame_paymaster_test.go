package execution

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFramePaymasterV1ConstructorDeploysExactRuntime(t *testing.T) {
	for _, zeroOwner := range []bool{false, true} {
		tx, db, cfg, header := frameExecutionFixture(t)
		factory := types.Address{19: 0x90}
		db.SetCode(factory, []byte{0x36, 0x5f, 0x5f, 0x37, 0x36, 0x5f, 0x5f, 0xf0, 0x50, 0})
		init, err := framepay.InitV1(tx.Sender)
		if err != nil {
			t.Fatal(err)
		}
		if zeroOwner {
			clear(init[len(init)-32:])
		}
		stateLimit := uint64(183600 + 97920 + len(framepay.RuntimeV1())*1530)
		tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &factory, Data: init, Limits: types.FrameGasLimits{Execution: 100000, State: stateLimit}})
		receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
		if err != nil {
			t.Fatal(err)
		}
		created := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
		if zeroOwner {
			if db.GetCodeSize(created) != 0 || receipt.FrameResults8141[1].GasUsed.State != 0 {
				t.Fatal("invalid constructor retained runtime or state gas")
			}
		} else if !framepay.IsV1(db.GetCode(created)) || db.GetState(created, types.Hash{}) != types.BytesToHash(tx.Sender[:]) || receipt.FrameResults8141[1].GasUsed.State != stateLimit {
			t.Fatalf("constructor/runtime/storage/state charge mismatch: %+v", receipt.FrameResults8141)
		}
	}
}

func TestFramePaymasterV1WithdrawalTimelockAndTransferRollback(t *testing.T) {
	for _, transferFails := range []bool{false, true} {
		tx, db, cfg, header := frameExecutionFixture(t)
		payer, recipient := types.Address{19: 0x93}, types.Address{19: 0x94}
		db.SetCode(payer, framepay.RuntimeV1())
		db.SetState(payer, types.Hash{}, types.BytesToHash(tx.Sender[:]))
		db.AddBalance(payer, big.NewInt(1000000))
		db.AddBalance(recipient, big.NewInt(1))
		apply := func(data []byte, stateGas uint64) *types.Receipt {
			t.Helper()
			tx.Nonce = db.GetNonce(tx.Sender)
			tx.Frames = tx.Frames[:1]
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &payer, Data: data, Limits: types.FrameGasLimits{Execution: 150000, State: stateGas}})
			signed := signFrameExecution(t, tx, false)
			db.SetTxContext(signed.Hash(), 0)
			receipt, _, err := ApplyTransaction(cfg, db, header, signed, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			return receipt
		}
		amount := big.NewInt(400000)
		if transferFails {
			amount = big.NewInt(2000000)
		}
		request, err := framepay.RequestWithdrawalData(recipient, amount)
		if err != nil {
			t.Fatal(err)
		}
		receipt := apply(request, 3*97920)
		ready := header.Time + framepay.WithdrawalDelay
		if receipt.FrameResults8141[1].Status != 1 || receipt.FrameResults8141[1].GasUsed.State != 3*97920 || framepay.PendingWithdrawal(db, payer).Cmp(amount) != 0 || db.GetState(payer, types.Hash{31: 3}) != types.BytesToHash(new(big.Int).SetUint64(ready).Bytes()) {
			t.Fatalf("request did not journal its delayed withdrawal: %+v", receipt.FrameResults8141)
		}
		if len(receipt.Logs) != 1 || len(receipt.Logs[0].Topics) != 2 || !bytes.Equal(receipt.Logs[0].Data[:32], amount.FillBytes(make([]byte, 32))) {
			t.Fatal("withdrawal request event mismatch")
		}
		header.Time = ready - 1
		receipt = apply(framepay.ExecuteWithdrawalData(), 0)
		if receipt.FrameResults8141[1].Status != 0 || framepay.PendingWithdrawal(db, payer).Cmp(amount) != 0 || db.GetBalance(payer).Int64() != 1000000 {
			t.Fatal("withdrawal executed before its exact deadline")
		}
		header.Time = ready
		receipt = apply(framepay.ExecuteWithdrawalData(), 0)
		if transferFails {
			if receipt.FrameResults8141[1].Status != 0 || framepay.PendingWithdrawal(db, payer).Cmp(amount) != 0 || db.GetBalance(payer).Int64() != 1000000 || db.GetBalance(recipient).Int64() != 1 {
				t.Fatal("failed transfer consumed the pending request or funds")
			}
		} else {
			if receipt.FrameResults8141[1].Status != 1 || framepay.PendingWithdrawal(db, payer).Sign() != 0 || db.GetBalance(payer).Int64() != 600000 || db.GetBalance(recipient).Int64() != 400001 {
				t.Fatalf("mature withdrawal value/state mismatch: %+v", receipt.FrameResults8141)
			}
			for _, slot := range []byte{1, 2, 3} {
				if !db.GetState(payer, types.Hash{31: slot}).IsZero() {
					t.Fatal("executed request left pending state")
				}
			}
		}
	}
}

func TestFramePaymasterV1RejectsNonOwnerWithdrawal(t *testing.T) {
	tx, db, cfg, header := frameExecutionFixture(t)
	payer, owner, recipient := types.Address{19: 0x93}, types.Address{19: 0x92}, types.Address{19: 0x94}
	db.SetCode(payer, framepay.RuntimeV1())
	db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
	data, err := framepay.RequestWithdrawalData(recipient, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &payer, Data: data, Limits: types.FrameGasLimits{Execution: 100000, State: 3 * 97920}})
	receipt, _, err := ApplyTransaction(cfg, db, header, signFrameExecution(t, tx, false), new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.FrameResults8141[1].Status != 0 || framepay.PendingWithdrawal(db, payer).Sign() != 0 || !db.GetState(payer, types.Hash{31: 3}).IsZero() {
		t.Fatal("non-owner created a withdrawal request")
	}
}
