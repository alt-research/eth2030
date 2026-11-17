package rpctypes

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	coretypes "github.com/eth2030/eth2030/core/types"
)

func TestFrame8141ReceiptReadBack(t *testing.T) {
	path := t.TempDir()
	open := func() *rawdb.FileDB {
		t.Helper()
		db, err := rawdb.NewFileDB(path)
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	db := open()
	payer := coretypes.Address{19: 9}
	log := &coretypes.Log{Address: coretypes.Address{19: 8}, Data: []byte{6}}
	receipt := &coretypes.Receipt{
		Type: coretypes.FrameTxType, Payer: payer, CumulativeGasUsed: 123456,
		FrameResults8141: []coretypes.FrameResult8141{
			{Status: 1, GasUsed: coretypes.FrameGasLimits{Execution: 300, State: 400}, Logs: []*coretypes.Log{log}},
			{Status: coretypes.FrameStatusSkipped},
		},
	}
	codec := new(coretypes.ReceiptCodec)
	encoded, err := codec.EncodeReceipts([]*coretypes.Receipt{receipt})
	if err != nil {
		t.Fatal(err)
	}
	hash := coretypes.Hash{31: 1}
	if err := rawdb.WriteReceipts(db, 42, hash, encoded); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db = open()
	defer db.Close()
	persisted, err := rawdb.ReadReceipts(db, 42, hash)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := codec.DecodeReceipts(persisted)
	if err != nil || len(actual) != 1 {
		t.Fatalf("decode persisted receipts: %v", err)
	}
	rpc := FormatReceipt(actual[0], nil, 95)
	if rpc.Payer == nil || *rpc.Payer != EncodeAddress(payer) || rpc.FrameResults == nil {
		t.Fatalf("missing payer or frame results: %+v", rpc)
	}
	frames := *rpc.FrameResults
	if len(frames) != 2 || frames[0].GasUsed != "0x12c" ||
		frames[0].StateGasUsed == nil || *frames[0].StateGasUsed != "0x190" ||
		frames[1].Status != "0x2" || len(frames[0].Logs) != 1 ||
		frames[0].Logs[0] != rpc.Logs[0] || frames[0].Logs[0].Data != "0x06" {
		t.Fatalf("wrong persisted frame results: %+v", frames)
	}
}

func TestFrame8141TransactionJSONPreservesEnvelope(t *testing.T) {
	frame := &coretypes.FrameTx8141{ChainID: big.NewInt(1337), Sender: coretypes.Address{19: 1},
		Fees:       coretypes.FrameFeeBudget{MaxFee: big.NewInt(10_000_000), MaxPriorityFeePerGas: big.NewInt(2)},
		Frames:     []coretypes.Frame8141{{Mode: 1, Flags: 3, Limits: coretypes.FrameGasLimits{Execution: 1000, State: 183600}}},
		Signatures: []coretypes.FrameSignature{{Scheme: 0, Signature: []byte{0x12, 0x34}}}}
	rpc := FormatTransaction(coretypes.NewTransaction(frame), nil, nil, nil, 0, big.NewInt(1))
	data, err := json.Marshal(rpc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["sender"] != EncodeAddress(frame.Sender) || decoded["from"] != EncodeAddress(frame.Sender) || decoded["maxFeePerGas"] != nil {
		t.Fatalf("Frame payer budget was misrepresented: %s", data)
	}
	frames := decoded["frames"].([]any)
	first := frames[0].(map[string]any)
	if first["target"] != nil || first["flags"] != "0x3" || first["limits"].(map[string]any)["state"] != "0x2cd30" {
		t.Fatalf("Frame fields lost: %s", data)
	}
	signatures := decoded["signatures"].([]any)
	if signatures[0].(map[string]any)["signature"] != "0x1234" || decoded["fees"].(map[string]any)["maxFee"] != "0x989680" {
		t.Fatalf("signature or fee fields lost: %s", data)
	}
}
