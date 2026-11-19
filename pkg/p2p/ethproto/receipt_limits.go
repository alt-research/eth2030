package ethproto

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/eth2030/eth2030/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

var ErrReceiptSizeLimit = errors.New("receipt exceeds requested transaction gas allowance")
var ErrReceiptCountLimit = errors.New("receipt response exceeds requested pages or transactions")

// DecodeReceiptsLimited bounds allocation using the known requested blocks,
// before constructing receipt/log objects. Limits are per remaining transaction.
func DecodeReceiptsLimited(data []byte, version uint32, limits [][]uint64) (uint64, bool, [][]*types.Receipt, error) {
	s := gethrlp.NewStream(bytes.NewReader(data), uint64(len(data)))
	if _, err := s.List(); err != nil {
		return 0, false, nil, err
	}
	id, err := s.Uint64()
	if err != nil {
		return 0, false, nil, err
	}
	incomplete := false
	if version >= 70 {
		incomplete, err = s.Bool()
		if err != nil {
			return 0, false, nil, err
		}
	}
	if _, err := s.List(); err != nil {
		return 0, false, nil, err
	}
	pages := make([][]gethrlp.RawValue, 0, len(limits))
	for {
		if _, _, err := s.Kind(); err == gethrlp.EOL {
			break
		} else if err != nil {
			return 0, false, nil, err
		}
		if len(pages) >= len(limits) {
			return 0, false, nil, ErrReceiptCountLimit
		}
		if _, err := s.List(); err != nil {
			return 0, false, nil, err
		}
		page := make([]gethrlp.RawValue, 0)
		for {
			if _, _, err := s.Kind(); err == gethrlp.EOL {
				break
			} else if err != nil {
				return 0, false, nil, err
			}
			if len(page) >= len(limits[len(pages)]) {
				return 0, false, nil, ErrReceiptCountLimit
			}
			raw, err := s.Raw()
			if err != nil {
				return 0, false, nil, err
			}
			if uint64(len(raw)) > limits[len(pages)][len(page)] {
				return 0, false, nil, ErrReceiptSizeLimit
			}
			page = append(page, raw)
		}
		if err := s.ListEnd(); err != nil {
			return 0, false, nil, err
		}
		pages = append(pages, page)
	}
	if err := s.ListEnd(); err != nil {
		return 0, false, nil, err
	}
	if err := s.ListEnd(); err != nil {
		return 0, false, nil, err
	}
	if _, _, err := s.Kind(); err != io.EOF {
		return 0, false, nil, fmt.Errorf("trailing receipt data")
	}
	var receipts [][]*types.Receipt
	if version >= 69 {
		receipts, err = decodeModernReceiptPages(pages)
	} else {
		receipts, err = decodeReceiptPages(pages)
	}
	return id, incomplete, receipts, err
}
