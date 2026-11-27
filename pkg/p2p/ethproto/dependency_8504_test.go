package ethproto

import (
	"bytes"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestDependency8504BlockBodiesPreserveActiveEmptyAndBytes(t *testing.T) {
	for _, w := range [][][]byte{nil, {}, {{1, 2, 3}}} {
		packet := BlockBodiesPacket{RequestID: 1, Bodies: []*BlockBody{{Withdrawals: []*types.Withdrawal{}, DependencyWitnesses: w}}}
		raw, err := packet.EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			ID     uint64
			Bodies [][]gethrlp.RawValue
		}
		if err := gethrlp.DecodeBytes(raw, &wire); err != nil {
			t.Fatal(err)
		}
		want := 3
		if w != nil {
			want = 4
		}
		if len(wire.Bodies[0]) != want {
			t.Fatal("body field count", len(wire.Bodies[0]), want)
		}
		var decoded BlockBodiesPacket
		if err := decoded.DecodeRLP(raw); err != nil {
			t.Fatal(err)
		}
		got := decoded.Bodies[0].DependencyWitnesses
		if (got != nil) != (w != nil) || len(got) != len(w) {
			t.Fatal("witness presence lost")
		}
		if len(w) > 0 && !bytes.Equal(w[0], got[0]) {
			t.Fatal("witness bytes changed")
		}
	}
}
