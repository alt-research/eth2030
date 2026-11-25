package types

import (
	"math/big"
	"testing"
)

func TestBlockRLPRejectsTrailingData(t *testing.T) {
	b := NewBlock(&Header{Number: big.NewInt(0), Difficulty: big.NewInt(0)}, &Body{})
	raw, err := b.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = DecodeBlockRLP(raw); err != nil {
		t.Fatal(err)
	}
	for _, tail := range [][]byte{{0}, {0xc0}, {0x80, 0x01}} {
		input := append(append([]byte(nil), raw...), tail...)
		if _, err = DecodeBlockRLP(input); err == nil {
			t.Fatalf("accepted trailing %x", tail)
		}
	}
}
