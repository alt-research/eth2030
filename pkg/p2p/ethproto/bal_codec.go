package ethproto

import (
	"bytes"
	"fmt"
	"io"

	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

const MaxBlockAccessLists = 64

// EncodeRLP embeds each already-encoded BAL as a list, not a byte string.
func (p BlockAccessListsPacket) EncodeRLP() ([]byte, error) {
	if len(p.AccessLists) > MaxBlockAccessLists {
		return nil, fmt.Errorf("too many BAL entries")
	}
	items := make([]gethrlp.RawValue, len(p.AccessLists))
	for i, encoded := range p.AccessLists {
		_, rest, err := gethrlp.SplitList(encoded)
		if err != nil || len(rest) != 0 {
			return nil, fmt.Errorf("BAL %d is not a complete RLP list", i)
		}
		items[i] = encoded
	}
	return gethrlp.EncodeToBytes(struct {
		ID    uint64
		Lists []gethrlp.RawValue
	}{p.RequestID, items})
}

func (p *BlockAccessListsPacket) DecodeRLP(data []byte) error {
	stream := gethrlp.NewStream(bytes.NewReader(data), uint64(len(data)))
	if _, err := stream.List(); err != nil {
		return err
	}
	id, err := stream.Uint64()
	if err != nil {
		return err
	}
	if _, err := stream.List(); err != nil {
		return err
	}
	items := make([][]byte, 0)
	for {
		encoded, err := stream.Raw()
		if err == gethrlp.EOL {
			break
		}
		if err != nil {
			return err
		}
		if len(items) >= MaxBlockAccessLists {
			return fmt.Errorf("too many BAL entries")
		}
		_, rest, err := gethrlp.SplitList(encoded)
		if err != nil || len(rest) != 0 {
			return fmt.Errorf("BAL is not a complete RLP list")
		}
		items = append(items, encoded)
	}
	if err := stream.ListEnd(); err != nil {
		return err
	}
	if err := stream.ListEnd(); err != nil {
		return err
	}
	if _, _, err := stream.Kind(); err != io.EOF {
		return fmt.Errorf("trailing BAL response data")
	}
	p.RequestID, p.AccessLists = id, items
	return nil
}
