package chain

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

func TestCanonicalFrame8141DiskReceiptMetadataAndLogIndices(t *testing.T) {
	path := t.TempDir()
	db, err := rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	header := &types.Header{Number: big.NewInt(1), GasLimit: 1000000, BaseFee: big.NewInt(1), Difficulty: new(big.Int)}
	hash := header.Hash()
	encoded, err := header.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteHeader(db, 1, hash, encoded); err != nil {
		t.Fatal(err)
	}
	var receipts []*types.Receipt
	var txs []*types.Transaction
	for i := uint64(0); i < 2; i++ {
		log := &types.Log{Address: types.Address{19: byte(i + 1)}, Data: []byte{1}}
		txs = append(txs, types.NewTransaction(&types.FrameTx8141{ChainID: big.NewInt(1337), Nonce: i, Sender: types.Address{19: byte(i + 1)},
			Fees:   types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(2)},
			Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 3000, State: 97920}}}}))
		receipts = append(receipts, &types.Receipt{Type: types.FrameTxType, CumulativeGasUsed: (i + 1) * 110495, Payer: types.Address{19: 9},
			FrameResults8141: []types.FrameResult8141{{Status: 1, GasUsed: types.FrameGasLimits{Execution: 100, State: 97920}, Logs: []*types.Log{log}}}})
	}
	(&Blockchain{db: db}).writeReceipts(1, hash, receipts)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restored := (&Blockchain{db: db}).readReceiptsFromDB(1, hash, txs)
	if len(restored) != 2 {
		t.Fatal("persisted Frame receipts missing")
	}
	for i, r := range restored {
		if r.GasUsed != 110495 || r.BlockGasUsed != 12575 || r.EffectiveGasPrice == nil || r.EffectiveGasPrice.Int64() != 3 {
			t.Fatalf("derived fee metadata: %+v", r)
		}
		if len(r.Logs) != 1 || r.Logs[0].Index != uint(i) || r.Logs[0].TxIndex != uint(i) || r.FrameResults8141[0].Logs[0] != r.Logs[0] {
			t.Fatal("Frame RPC log indices changed after disk readback")
		}
	}
}
