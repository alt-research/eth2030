package ethproto

import (
	"bytes"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

func TestCanonicalFrame8141ReceiptPages(t *testing.T) {
	canonical := &types.Receipt{Type: types.FrameTxType, CumulativeGasUsed: 123456, Payer: types.Address{19: 7},
		FrameResults8141: []types.FrameResult8141{
			{Status: 1, GasUsed: types.FrameGasLimits{Execution: 100, State: 183600}, Logs: []*types.Log{{Address: types.Address{19: 1}, Data: []byte{8}}}},
			{Status: types.FrameStatusSkipped},
		}}
	legacy := &types.Receipt{Status: 1, CumulativeGasUsed: 21000}
	packet := ReceiptsPacketV70{RequestID: 42, LastBlockIncomplete: true, Receipts: [][]*types.Receipt{{legacy, canonical}}}
	raw, err := packet.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	var decoded ReceiptsPacketV70
	if err := decoded.DecodeRLP(raw); err != nil {
		t.Fatal(err)
	}
	if decoded.RequestID != 42 || !decoded.LastBlockIncomplete || len(decoded.Receipts) != 1 || len(decoded.Receipts[0]) != 2 {
		t.Fatal("pagination context was lost")
	}
	actual := decoded.Receipts[0][1]
	wantBytes, err := canonical.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	gotBytes, err := actual.EncodeRLP()
	if err != nil || !bytes.Equal(wantBytes, gotBytes) {
		t.Fatalf("Frame consensus receipt changed on the wire: %v", err)
	}
	if actual.Payer != canonical.Payer || len(actual.Logs) != 1 || actual.FrameResults8141[0].GasUsed.State != 183600 || actual.FrameResults8141[1].Status != 2 {
		t.Fatal("Frame fields lost in paged receipts")
	}
}
