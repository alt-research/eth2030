package types

import (
	"fmt"

	"github.com/eth2030/eth2030/rlp"
)

// BlockAppendSizer retains exact RLP list/header overhead for repeated omitted
// transaction checks. Existing block transactions are encoded once, not once
// for every IL candidate. New gas-used values may change the header's RLP size.
type BlockAppendSizer struct {
	header           *Header
	transactionBytes int
	otherBytes       int
	witnessBytes     int
}

func NewBlockAppendSizer(block *Block) (*BlockAppendSizer, error) {
	if block == nil {
		return nil, fmt.Errorf("nil block")
	}
	total, err := block.RLPEncodedSize()
	if err != nil {
		return nil, err
	}
	payload := -1
	for prefix := 1; prefix <= 9 && prefix <= total; prefix++ {
		if rlp.EstimateListSize(total-prefix) == total {
			payload = total - prefix
			break
		}
	}
	if payload < 0 {
		return nil, fmt.Errorf("invalid RLP size")
	}
	header := block.Header()
	encoded, err := header.EncodeRLP()
	if err != nil {
		return nil, err
	}
	s := &BlockAppendSizer{header: header}
	for _, tx := range block.Transactions() {
		n, err := transactionBlockBytes(tx)
		if err != nil {
			return nil, err
		}
		s.transactionBytes += n
	}
	s.otherBytes = payload - len(encoded) - rlp.EstimateListSize(s.transactionBytes)
	if block.body.DependencyWitnesses != nil {
		w, err := rlp.EncodeToBytes(block.body.DependencyWitnesses)
		if err != nil {
			return nil, err
		}
		s.witnessBytes = len(w)
	}
	return s, nil
}
func transactionBlockBytes(tx *Transaction) (int, error) {
	raw, err := tx.EncodeRLP()
	if err != nil {
		return 0, err
	}
	if tx.Type() == LegacyTxType {
		return len(raw), nil
	}
	return rlp.EstimateStringSize(len(raw)), nil
}
func (s *BlockAppendSizer) Size(tx *Transaction, gasUsed uint64) (int, error) {
	header := s.header.Copy()
	header.GasUsed = gasUsed
	return s.SizeWithHeader(tx, header, nil)
}

// SizeWithHeader permits builders to supply the complete projected header and
// dependency union. A nil witnesses argument retains the original witness field.
func (s *BlockAppendSizer) SizeWithHeader(tx *Transaction, header *Header, witnesses [][]byte) (int, error) {
	n, err := transactionBlockBytes(tx)
	if err != nil {
		return 0, err
	}
	encoded, err := header.EncodeRLP()
	if err != nil {
		return 0, err
	}
	other := s.otherBytes
	if witnesses != nil {
		w, err := rlp.EncodeToBytes(witnesses)
		if err != nil {
			return 0, err
		}
		other += len(w) - s.witnessBytes
	}
	return rlp.EstimateListSize(len(encoded) + rlp.EstimateListSize(s.transactionBytes+n) + other), nil
}
func (s *BlockAppendSizer) CommitAppend(tx *Transaction) error {
	n, err := transactionBlockBytes(tx)
	if err != nil {
		return err
	}
	s.transactionBytes += n
	return nil
}
