//go:build integration

package core

import (
	"math/big"
	"testing"

	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

var sstoreContractCode = []byte{0x60, 1, 0x60, 0, 0x55, 0}
var contractAddr = types.HexToAddress("0xca11")

func newGlamHeader() *types.Header {
	return &types.Header{Number: big.NewInt(1), GasLimit: 30_000_000, Time: 1, BaseFee: big.NewInt(1)}
}
func newCallTx(nonce uint64, sender types.Address) *types.Transaction {
	tx := types.NewTransaction(&types.LegacyTx{Nonce: nonce, GasPrice: big.NewInt(10_000_000_000), Gas: 200_000, To: &contractAddr, Value: new(big.Int)})
	tx.SetSender(sender)
	return tx
}
func newSSTOREState() (state.StateDB, types.Address) {
	db := newBuilderRequestTestState()
	sender := types.HexToAddress("0xabcd")
	db.AddBalance(sender, new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18)))
	db.SetCode(contractAddr, sstoreContractCode)
	return db, sender
}

// The old draft DimStorageGas premium is not the canonical EIP-8037 state dimension.
func TestReceipt_StateGas_AmsterdamActive(t *testing.T) {
	db, sender := newSSTOREState()
	header := newGlamHeader()
	receipt, used, err := execution.ApplyTransactionWithBAL(config.TestConfig, db, header, newCallTx(0, sender), new(gaspool.GasPool).AddGas(header.GasLimit), nil)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 || receipt.StateGasUsed != 64*1530 || receipt.BlockGasUsed != 27106 || used != 125026 || receipt.DimStorageGas != 0 {
		t.Fatalf("canonical slot pricing/receipt dimensions: %+v paid=%d", receipt, used)
	}
	if db.GetState(contractAddr, types.Hash{}) != types.HexToHash("0x01") {
		t.Fatal("slot was not actually created")
	}
}

func TestReceipt_StateGas_PreAmsterdam(t *testing.T) {
	db, sender := newSSTOREState()
	header := newGlamHeader()
	cfg := pragueAuthorizationTestConfig()
	receipt, _, err := execution.ApplyTransactionWithBAL(cfg, db, header, newCallTx(0, sender), new(gaspool.GasPool).AddGas(header.GasLimit), nil)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 || receipt.StateGasUsed != 0 || receipt.DimStorageGas != 0 {
		t.Fatalf("Prague acquired Amsterdam state gas: %+v", receipt)
	}
}

func TestReceipt_StateGas_NewAccountTransfer(t *testing.T) {
	db, sender := newSSTOREState()
	receiver := types.HexToAddress("0xccdd")
	tx := types.NewTransaction(&types.LegacyTx{Gas: 300_000, GasPrice: big.NewInt(10_000_000_000), To: &receiver, Value: big.NewInt(1e9)})
	tx.SetSender(sender)
	header := newGlamHeader()
	receipt, used, err := execution.ApplyTransactionWithBAL(config.TestConfig, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != 1 || receipt.StateGasUsed != 120*1530 || used != 204600 || db.GetBalance(receiver).Cmp(big.NewInt(1e9)) != 0 {
		t.Fatalf("fresh account state gas/transfer: %+v paid=%d", receipt, used)
	}
}

func TestBlockBuilder_StateGas_Accumulates(t *testing.T) {
	db, sender := newSSTOREState()
	txs := []*types.Transaction{newCallTx(0, sender), newCallTx(1, sender), newCallTx(2, sender)}
	builder := coreblock.NewBlockBuilder(config.TestConfig, nil, nil)
	builder.SetState(db)
	block, receipts, err := builder.BuildBlockLegacy(&types.Header{Number: new(big.Int), GasLimit: 30_000_000, BaseFee: big.NewInt(1)}, txs, 1, types.HexToAddress("0xff"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 3 {
		t.Fatalf("included %d receipts", len(receipts))
	}
	for i, r := range receipts {
		want := uint64(0)
		if i == 0 {
			want = 97920
		}
		if r.Status != 1 || r.StateGasUsed != want {
			t.Fatalf("slot reuse %d: %+v", i, r)
		}
	}
	if block.GasUsed() != 97920 {
		t.Fatalf("header gas=%d want state bottleneck 97920", block.GasUsed())
	}
}

func TestBlockBuilder_StateGas_RejectExcess(t *testing.T) {
	db := newBuilderRequestTestState()
	sender := types.HexToAddress("0xdead")
	db.AddBalance(sender, new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)))
	var txs []*types.Transaction
	for i := 0; i < 20; i++ {
		address := types.Address{18: 0xc0, 19: byte(i)}
		db.SetCode(address, sstoreContractCode)
		tx := types.NewTransaction(&types.LegacyTx{Nonce: uint64(i), GasPrice: big.NewInt(10_000_000_000), Gas: 150000, To: &address, Value: new(big.Int)})
		tx.SetSender(sender)
		txs = append(txs, tx)
	}
	historical := db.Dup()
	historicalBuilder := coreblock.NewBlockBuilder(pragueAuthorizationTestConfig(), nil, nil)
	historicalBuilder.SetState(historical)
	historicalBlock, historicalReceipts, historicalErr := historicalBuilder.BuildBlockLegacy(&types.Header{Number: new(big.Int), GasLimit: 1_000_000, BaseFee: big.NewInt(1)}, txs, 1, types.HexToAddress("0xff"), nil)
	if historicalErr != nil || len(historicalReceipts) != 20 || historicalBlock.GasUsed() >= 1_000_000 {
		t.Fatalf("Prague control should fit all twenty slots: receipts=%d error=%v", len(historicalReceipts), historicalErr)
	}
	for _, r := range historicalReceipts {
		if r.Status != 1 || r.StateGasUsed != 0 {
			t.Fatalf("historical receipt=%+v", r)
		}
	}
	builder := coreblock.NewBlockBuilder(config.TestConfig, nil, nil)
	builder.SetState(db)
	block, receipts, err := builder.BuildBlockLegacy(&types.Header{Number: new(big.Int), GasLimit: 1_000_000, BaseFee: big.NewInt(1)}, txs, 1, types.HexToAddress("0xff"), nil)
	if err != nil {
		t.Fatal(err)
	}
	// After nine slots the remaining state budget is 118720, below the next
	// transaction's declared 150000 reservation. Execution still has ample room.
	if len(receipts) != 9 || len(block.Transactions()) != 9 || block.GasUsed() != 9*97920 || db.GetNonce(sender) != 9 {
		t.Fatalf("state capacity/adoption: txs=%d receipts=%d header=%d nonce=%d", len(block.Transactions()), len(receipts), block.GasUsed(), db.GetNonce(sender))
	}
	for i, r := range receipts {
		if r.Status != 1 || r.StateGasUsed != 97920 || db.GetState(*txs[i].To(), types.Hash{}) != types.HexToHash("0x01") {
			t.Fatalf("included slot %d: %+v", i, r)
		}
	}
	if db.GetState(*txs[9].To(), types.Hash{}) != (types.Hash{}) {
		t.Fatal("excluded slot changed state")
	}
}

func TestProcessorWiresReservoir(t *testing.T) {
	// Funding a reservoir changes where charges are drawn, never the state price,
	// paid gas, resulting storage, or execution/state block dimensions.
	for _, limit := range []uint64{200000, 20_000_000} {
		db, sender := newSSTOREState()
		header := newGlamHeader()
		tx := types.NewTransaction(&types.LegacyTx{Gas: limit, GasPrice: big.NewInt(10_000_000_000), To: &contractAddr, Value: new(big.Int)})
		tx.SetSender(sender)
		receipt, used, err := execution.ApplyTransactionWithBAL(config.TestConfig, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Status != 1 || receipt.StateGasUsed != vm.AmsterdamNewSlotStateGas || receipt.BlockGasUsed != 27106 || used != 125026 {
			t.Fatalf("reservoir limit=%d result=%+v paid=%d", limit, receipt, used)
		}
	}
}
