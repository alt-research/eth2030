package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/eth2030/eth2030/ssz"
)

// ConsensusPayload8504 is the pinned Gloas progressive payload with the
// EIP-8504 field appended. It is distinct from the EIP-7807 execution-block
// schema. Source: ethlambda 7aef8bd75316078010193684715359b26b5c2be6,
// crates/common/types/src/beacon/containers/gloas.rs. Nil means the 19-field
// pre-extension schema; non-nil (including empty) means the 20-field schema.
type ConsensusPayload8504 struct {
	Header              *Header
	BlockHash           Hash
	Transactions        [][]byte
	Withdrawals         []*Withdrawal
	BlockAccessList     []byte
	DependencyWitnesses [][]byte
}

var consensusPayloadSizes8504 = []int{32, 20, 32, 32, 256, 32, 8, 8, 8, 8, 0, 32, 32, 0, 0, 8, 8, 0, 8}

func NewConsensusPayload8504(block *Block, encodedBAL []byte) *ConsensusPayload8504 {
	p := &ConsensusPayload8504{Header: block.Header(), BlockHash: block.Hash(), BlockAccessList: bytes.Clone(encodedBAL), DependencyWitnesses: block.DependencyWitnesses8504(), Withdrawals: block.Withdrawals()}
	for _, tx := range block.Transactions() {
		raw, _ := tx.EncodeRLP()
		p.Transactions = append(p.Transactions, raw)
	}
	return p
}

func (p *ConsensusPayload8504) parts() ([][]byte, []int, error) {
	h := p.Header
	if h == nil || h.Number == nil || !h.Number.IsUint64() || h.BaseFee == nil || h.BlobGasUsed == nil || h.ExcessBlobGas == nil || h.SlotNumber == nil || len(h.Extra) > 32 {
		return nil, nil, fmt.Errorf("incomplete Gloas payload")
	}
	if h.DependencyWitnessesHash != nil {
		if p.DependencyWitnesses == nil {
			return nil, nil, fmt.Errorf("active EIP-8504 payload is missing its witness field")
		}
		if DependencyWitnessesHash8504(p.DependencyWitnesses) != *h.DependencyWitnessesHash {
			return nil, nil, fmt.Errorf("Gloas witness field does not match its execution header commitment")
		}
	}
	fee, err := sszUint256(h.BaseFee)
	if err != nil {
		return nil, nil, err
	}
	txs, err := marshalExecutionSSZList(p.Transactions)
	if err != nil {
		return nil, nil, err
	}
	var withdrawals []byte
	for _, w := range p.Withdrawals {
		if w == nil {
			return nil, nil, fmt.Errorf("nil withdrawal")
		}
		withdrawals = append(withdrawals, ssz.MarshalUint64(w.Index)...)
		withdrawals = append(withdrawals, ssz.MarshalUint64(w.ValidatorIndex)...)
		withdrawals = append(withdrawals, w.Address[:]...)
		withdrawals = append(withdrawals, ssz.MarshalUint64(w.Amount)...)
	}
	parts := [][]byte{h.ParentHash[:], h.Coinbase[:], h.Root[:], h.ReceiptHash[:], h.Bloom[:], h.MixDigest[:], ssz.MarshalUint64(h.Number.Uint64()), ssz.MarshalUint64(h.GasLimit), ssz.MarshalUint64(h.GasUsed), ssz.MarshalUint64(h.Time), h.Extra, fee, p.BlockHash[:], txs, withdrawals, ssz.MarshalUint64(*h.BlobGasUsed), ssz.MarshalUint64(*h.ExcessBlobGas), p.BlockAccessList, ssz.MarshalUint64(*h.SlotNumber)}
	sizes := append([]int(nil), consensusPayloadSizes8504...)
	if p.DependencyWitnesses != nil {
		encoded, err := marshalExecutionSSZList(p.DependencyWitnesses)
		if err != nil {
			return nil, nil, err
		}
		parts = append(parts, encoded)
		sizes = append(sizes, 0)
	}
	return parts, sizes, nil
}

func (p *ConsensusPayload8504) MarshalSSZ() ([]byte, error) {
	f, sizes, err := p.parts()
	if err != nil {
		return nil, err
	}
	return marshalExecutionSSZFields(f, sizes)
}

func (p *ConsensusPayload8504) UnmarshalSSZ(data []byte) error {
	sizes := append([]int(nil), consensusPayloadSizes8504...)
	fixed := 0
	for _, size := range sizes {
		if size == 0 {
			fixed += 4
		} else {
			fixed += size
		}
	}
	// The first variable field (extra_data) starts at fixed-part byte 436.
	if len(data) < 440 {
		return ssz.ErrBufferTooSmall
	}
	first := int(binary.LittleEndian.Uint32(data[436:440]))
	if first == fixed+4 {
		sizes = append(sizes, 0)
	} else if first != fixed {
		return ssz.ErrOffset
	}
	f, err := ssz.UnmarshalVariableContainer(data, len(sizes), sizes)
	if err != nil {
		return err
	}
	if len(f[10]) > 32 || len(f[14])%44 != 0 {
		return ssz.ErrSize
	}
	txs, err := unmarshalExecutionSSZList(f[13])
	if err != nil {
		return err
	}
	h := &Header{ParentHash: BytesToHash(f[0]), Coinbase: BytesToAddress(f[1]), Root: BytesToHash(f[2]), ReceiptHash: BytesToHash(f[3]), MixDigest: BytesToHash(f[5]), Number: new(big.Int).SetUint64(binary.LittleEndian.Uint64(f[6])), GasLimit: binary.LittleEndian.Uint64(f[7]), GasUsed: binary.LittleEndian.Uint64(f[8]), Time: binary.LittleEndian.Uint64(f[9]), Extra: bytes.Clone(f[10]), BaseFee: bigIntFrom32LE(f[11])}
	copy(h.Bloom[:], f[4])
	blob, excess, slot := binary.LittleEndian.Uint64(f[15]), binary.LittleEndian.Uint64(f[16]), binary.LittleEndian.Uint64(f[18])
	h.BlobGasUsed, h.ExcessBlobGas, h.SlotNumber = &blob, &excess, &slot
	var withdrawals []*Withdrawal
	for at := 0; at < len(f[14]); at += 44 {
		b := f[14][at : at+44]
		withdrawals = append(withdrawals, &Withdrawal{Index: binary.LittleEndian.Uint64(b), ValidatorIndex: binary.LittleEndian.Uint64(b[8:]), Address: BytesToAddress(b[16:36]), Amount: binary.LittleEndian.Uint64(b[36:])})
	}
	var witnesses [][]byte
	if len(f) == 20 {
		witnesses, err = unmarshalExecutionSSZList(f[19])
		if err != nil {
			return err
		}
		if witnesses == nil {
			witnesses = make([][]byte, 0)
		}
		hash := DependencyWitnessesHash8504(witnesses)
		h.DependencyWitnessesHash = &hash
	}
	*p = ConsensusPayload8504{Header: h, BlockHash: BytesToHash(f[12]), Transactions: cloneWitnesses8504(txs), Withdrawals: withdrawals, BlockAccessList: bytes.Clone(f[17]), DependencyWitnesses: cloneWitnesses8504(witnesses)}
	return nil
}

func (p *ConsensusPayload8504) HashTreeRoot() (Hash, error) {
	f, _, err := p.parts()
	if err != nil {
		return Hash{}, err
	}
	roots := make([][32]byte, len(f))
	for i, part := range f {
		switch i {
		case 4:
			roots[i] = ssz.Merkleize(ssz.Pack(part), 8)
		case 10:
			roots[i] = ssz.HashTreeRootByteList(part, 32)
		case 13:
			roots[i] = [32]byte(executionEnvelopeRoot(p.Transactions))
		case 14:
			wr := make([][32]byte, len(p.Withdrawals))
			for j, w := range p.Withdrawals {
				var address [32]byte
				copy(address[:], w.Address[:])
				wr[j] = ssz.HashTreeRootContainer([][32]byte{ssz.HashTreeRootUint64(w.Index), ssz.HashTreeRootUint64(w.ValidatorIndex), address, ssz.HashTreeRootUint64(w.Amount)})
			}
			roots[i] = ssz.HashTreeRootProgressiveList(wr)
		case 17:
			roots[i] = progressiveBytesRoot(part)
		case 19:
			roots[i] = [32]byte(executionEnvelopeRoot(p.DependencyWitnesses))
		default:
			copy(roots[i][:], part)
		}
	}
	root, err := executionProgressiveRoot(roots, uint32(1<<len(roots)-1))
	return Hash(root), err
}
