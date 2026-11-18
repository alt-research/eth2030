package snap

import (
	"bytes"
	"fmt"
	"io"

	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

// Missing entries are RLP empty strings; an available empty BAL is a raw c0 list.
func (p BlockAccessListsPacket) EncodeRLP() ([]byte, error) {
	if len(p.Lists) > MaxBlockAccessLists {
		return nil, ErrRequestTooLarge
	}
	items := make([]gethrlp.RawValue, len(p.Lists))
	for i, item := range p.Lists {
		if len(item) == 0 {
			items[i] = gethrlp.RawValue{0x80}
			continue
		}
		if _, rest, err := gethrlp.SplitList(item); err != nil || len(rest) != 0 {
			return nil, fmt.Errorf("snap/2 malformed BAL list")
		}
		items[i] = item
	}
	encoded, err := gethrlp.EncodeToBytes(struct {
		ID    uint64
		Lists []gethrlp.RawValue
	}{p.ID, items})
	if err == nil && len(encoded) > BALHardResponseLimit {
		return nil, ErrRequestTooLarge
	}
	return encoded, err
}

func (p *BlockAccessListsPacket) DecodeRLP(data []byte) error {
	if len(data) > BALHardResponseLimit {
		return ErrRequestTooLarge
	}
	s := gethrlp.NewStream(bytes.NewReader(data), uint64(len(data)))
	if _, err := s.List(); err != nil {
		return err
	}
	id, err := s.Uint64()
	if err != nil {
		return err
	}
	if _, err := s.List(); err != nil {
		return err
	}
	var lists [][]byte
	for {
		item, err := s.Raw()
		if err == gethrlp.EOL {
			break
		}
		if err != nil {
			return err
		}
		if len(lists) >= MaxBlockAccessLists {
			return ErrRequestTooLarge
		}
		if bytes.Equal(item, []byte{0x80}) {
			lists = append(lists, nil)
			continue
		}
		if _, rest, err := gethrlp.SplitList(item); err != nil || len(rest) != 0 {
			return fmt.Errorf("snap/2 malformed BAL entry")
		}
		lists = append(lists, item)
	}
	if err := s.ListEnd(); err != nil {
		return err
	}
	if err := s.ListEnd(); err != nil {
		return err
	}
	if _, _, err := s.Kind(); err != io.EOF {
		return fmt.Errorf("snap/2 trailing response data")
	}
	p.ID, p.Lists = id, lists
	return nil
}
