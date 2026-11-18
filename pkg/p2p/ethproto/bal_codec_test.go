package ethproto

import (
	"bytes"
	"encoding/hex"
	"github.com/eth2030/eth2030/p2p/wire"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"testing"
)

func TestBlockAccessListsCanonicalRawItems(t *testing.T) {
	packet := &BlockAccessListsPacket{RequestID: 1, AccessLists: [][]byte{{0xc0}}}
	msg, err := wire.EncodeMessage(BlockAccessListsMsg, packet)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := hex.DecodeString("c301c1c0") // [1,[[]]], not [1,[bytes(c0)]].
	if !bytes.Equal(msg.Payload, expected) {
		t.Fatalf("BALs must be RLP lists, got %x want %x", msg.Payload, expected)
	}
	var decoded BlockAccessListsPacket
	if err := wire.DecodeMessage(wire.Message{Payload: expected}, &decoded); err != nil || len(decoded.AccessLists) != 1 || !bytes.Equal(decoded.AccessLists[0], []byte{0xc0}) {
		t.Fatalf("canonical BAL wire decode: %+v %v", decoded, err)
	}
}

func TestBALCodecRejectsWrappedOversizedAndTrailingEntries(t *testing.T) {
	wrapped, _ := hex.DecodeString("c401c281c0")
	many := make([]gethrlp.RawValue, MaxBlockAccessLists+1)
	for i := range many {
		many[i] = []byte{0xc0}
	}
	oversized, _ := gethrlp.EncodeToBytes(struct {
		ID    uint64
		Lists []gethrlp.RawValue
	}{1, many})
	for _, raw := range [][]byte{wrapped, oversized, {0xc3, 1, 0xc1, 0xc0, 0x80}} {
		var packet BlockAccessListsPacket
		if err := packet.DecodeRLP(raw); err == nil {
			t.Fatalf("invalid BAL packet accepted: %x", raw[:4])
		}
	}
}
