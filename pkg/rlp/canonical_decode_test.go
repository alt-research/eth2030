package rlp

import (
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"testing"
)

func TestCanonicalIntegerDecodingMatchesIndependentRLP(t *testing.T) {
	for _, encoded := range [][]byte{{0x80}, {0}, {1}, {0x81, 0}, {0x82, 0, 1}} {
		var native, reference uint64
		e1, e2 := DecodeBytes(encoded, &native), gethrlp.DecodeBytes(encoded, &reference)
		if (e1 == nil) != (e2 == nil) || e1 == nil && native != reference {
			t.Fatalf("uint %x native=%d/%v reference=%d/%v", encoded, native, e1, reference, e2)
		}
		var a, b big.Int
		e1, e2 = DecodeBytes(encoded, &a), gethrlp.DecodeBytes(encoded, &b)
		if (e1 == nil) != (e2 == nil) || e1 == nil && a.Cmp(&b) != 0 {
			t.Fatalf("bigint %x native=%s/%v reference=%s/%v", encoded, &a, e1, &b, e2)
		}
	}
}
func TestFixedAddressArraysMatchIndependentRLP(t *testing.T) {
	for _, length := range []int{0, 1, 19, 20, 21} {
		encoded, err := gethrlp.EncodeToBytes(make([]byte, length))
		if err != nil {
			t.Fatal(err)
		}
		var a, b [20]byte
		e1, e2 := DecodeBytes(encoded, &a), gethrlp.DecodeBytes(encoded, &b)
		if (e1 == nil) != (e2 == nil) || e1 == nil && a != b {
			t.Fatalf("address length %d native=%v independent=%v", length, e1, e2)
		}
	}
}
