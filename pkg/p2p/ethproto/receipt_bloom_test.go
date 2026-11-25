package ethproto

import (
	"bytes"
	"github.com/eth2030/eth2030/core/types"
	"testing"
)

func TestModernReceiptRebuildsBloomFromLogs(t *testing.T) {
	original := &types.Receipt{Type: types.DynamicFeeTxType, Status: 1, CumulativeGasUsed: 21000, Logs: []*types.Log{{Address: types.Address{19: 42}, Topics: []types.Hash{{31: 7}}, Data: []byte{1, 2, 3}}}}
	original.Bloom = types.LogsBloom(original.Logs)
	packet := ReceiptsPacketV70{RequestID: 1, Receipts: [][]*types.Receipt{{original}}}
	raw, err := packet.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	var decoded ReceiptsPacketV70
	if err := decoded.DecodeRLP(raw); err != nil {
		t.Fatal(err)
	}
	got := decoded.Receipts[0][0]
	if got.Bloom == (types.Bloom{}) || got.Bloom != original.Bloom {
		t.Fatal("modern wire receipt lost its log bloom")
	}
	wantBytes, _ := original.EncodeRLP()
	gotBytes, _ := got.EncodeRLP()
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Fatal("consensus receipt commitment changed after wire transfer")
	}
}
