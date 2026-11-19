package ethproto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/eth2030/eth2030/core/types"
	"testing"
)

func TestReceipt69And70IndependentGoldenEncoding(t *testing.T) {
	receipt := &types.Receipt{Status: 1, CumulativeGasUsed: 1}
	old, err := (ReceiptsPacketV69{1, [][]*types.Receipt{{receipt}}}).EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("c801c6c5c4800101c0")
	if !bytes.Equal(old, want) {
		t.Fatalf("eth/69 receipt: %x", old)
	}
	modern, err := (ReceiptsPacketV70{1, false, [][]*types.Receipt{{receipt}}}).EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	want, _ = hex.DecodeString("c90180c6c5c4800101c0")
	if !bytes.Equal(modern, want) {
		t.Fatalf("eth/70 receipt: %x", modern)
	}
}

func TestReceiptDecoderUsesRequestedBounds(t *testing.T) {
	receipt := &types.Receipt{Status: 1, CumulativeGasUsed: 21000}
	raw, err := (ReceiptsPacketV70{1, false, [][]*types.Receipt{{receipt, receipt}}}).EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := DecodeReceiptsLimited(raw, 70, [][]uint64{{100000}}); !errors.Is(err, ErrReceiptCountLimit) {
		t.Fatalf("extra transaction decoded: %v", err)
	}
	if _, _, _, err := DecodeReceiptsLimited(raw, 70, [][]uint64{{1, 1}}); !errors.Is(err, ErrReceiptSizeLimit) {
		t.Fatalf("oversized receipt decoded: %v", err)
	}
	if id, flag, pages, err := DecodeReceiptsLimited(raw, 70, [][]uint64{{100000, 100000}}); err != nil || id != 1 || flag || len(pages) != 1 || len(pages[0]) != 2 {
		t.Fatalf("bounded valid decode failed: %v", err)
	}
}
