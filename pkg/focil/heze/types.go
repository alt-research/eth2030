// Package heze implements the pinned consensus-specs Heze FOCIL companion.
// It is distinct from the older committee-root/Keccak/RLP research helpers in
// focil. No beacon committee or fork schedule is inferred by an execution node.
package heze

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/consensus"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/ssz"
)

const (
	CommitteeSize       = 16
	MaxTransactionBytes = 8192
	MaxSignedSize       = 41112
	MinRequestSlots     = 1
	DueBasisPoints      = 7500
)

var DomainType = [4]byte{0x10, 0, 0, 0}
var ErrEncoding = errors.New("heze: invalid inclusion list SSZ")
var ErrResourceUnavailable = errors.New("heze: inclusion list outside request window")

type InclusionList struct {
	Slot           uint64
	ValidatorIndex uint64
	DependentRoot  types.Hash
	Transactions   [][]byte
}

type SignedInclusionList struct {
	Message   InclusionList
	Signature [96]byte
}

type Committee [CommitteeSize]uint64
type Bits [CommitteeSize / 8]byte

func (b Bits) Has(i int) bool { return i >= 0 && i < CommitteeSize && b[i/8]&(1<<uint(i%8)) != 0 }
func (b *Bits) Set(i int) {
	if i >= 0 && i < CommitteeSize {
		b[i/8] |= 1 << uint(i%8)
	}
}
func (b Bits) Includes(local Bits) bool {
	return b[0]&local[0] == local[0] && b[1]&local[1] == local[1]
}

// SelectCommittee consumes the ordered beacon committees for the requested
// slot. Shuffling and dependent-state selection remain consensus responsibilities.
func SelectCommittee(committees [][]uint64) (Committee, error) {
	var indices []uint64
	for _, c := range committees {
		indices = append(indices, c...)
	}
	var out Committee
	if len(indices) == 0 {
		return out, errors.New("heze: empty beacon committees")
	}
	for i := range out {
		out[i] = indices[i%len(indices)]
	}
	return out, nil
}

func CloneTransactions(txs [][]byte) [][]byte {
	if txs == nil {
		return nil
	}
	out := make([][]byte, len(txs))
	for i := range txs {
		out[i] = bytes.Clone(txs[i])
	}
	return out
}
func (s SignedInclusionList) Clone() SignedInclusionList {
	s.Message.Transactions = CloneTransactions(s.Message.Transactions)
	return s
}

// CheckTransactions deliberately does not decode or execute IL transactions:
// malformed, invalid-signature and duplicate envelopes are permitted by gossip.
func CheckTransactions(txs [][]byte) error {
	n := 0
	for _, tx := range txs {
		if len(tx) == 0 {
			return errors.New("heze: empty IL transaction")
		}
		if len(tx) > MaxTransactionBytes-n {
			return errors.New("heze: IL transaction bytes exceed 8192")
		}
		n += len(tx)
	}
	if n == 0 {
		return errors.New("heze: no IL transactions")
	}
	return nil
}

// EncodeTransactions uses the SSZ variable-list offset layout, also used by
// ProgressiveList[ProgressiveList[Byte]]. Empty lists are valid SSZ values.
func EncodeTransactions(txs [][]byte) ([]byte, error) {
	if len(txs) > MaxTransactionBytes {
		return nil, ErrEncoding
	}
	n := 4 * len(txs)
	for _, tx := range txs {
		if len(tx) > MaxSignedSize-n {
			return nil, ErrEncoding
		}
		n += len(tx)
	}
	if n > MaxSignedSize {
		return nil, ErrEncoding
	}
	out := make([]byte, 4*len(txs), n)
	offset := len(out)
	for i, tx := range txs {
		binary.LittleEndian.PutUint32(out[4*i:], uint32(offset))
		out = append(out, tx...)
		offset += len(tx)
	}
	return out, nil
}

func DecodeTransactions(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return [][]byte{}, nil
	}
	if len(data) < 4 || len(data) > MaxSignedSize {
		return nil, ErrEncoding
	}
	first := int(binary.LittleEndian.Uint32(data))
	if first < 4 || first%4 != 0 || first > len(data) {
		return nil, ErrEncoding
	}
	n := first / 4
	if n > MaxTransactionBytes {
		return nil, ErrEncoding
	}
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		start := int(binary.LittleEndian.Uint32(data[i*4:]))
		end := len(data)
		if i+1 < n {
			end = int(binary.LittleEndian.Uint32(data[(i+1)*4:]))
		}
		if start < first || start > end || end > len(data) {
			return nil, ErrEncoding
		}
		out[i] = bytes.Clone(data[start:end])
	}
	return out, nil
}

func (m InclusionList) MarshalSSZ() ([]byte, error) {
	txs, err := EncodeTransactions(m.Transactions)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 52, 52+len(txs))
	binary.LittleEndian.PutUint64(out, m.Slot)
	binary.LittleEndian.PutUint64(out[8:], m.ValidatorIndex)
	copy(out[16:], m.DependentRoot[:])
	binary.LittleEndian.PutUint32(out[48:], 52)
	return append(out, txs...), nil
}
func (m *InclusionList) UnmarshalSSZ(data []byte) error {
	if len(data) < 52 || len(data) > MaxSignedSize-100 || binary.LittleEndian.Uint32(data[48:]) != 52 {
		return ErrEncoding
	}
	txs, err := DecodeTransactions(data[52:])
	if err != nil {
		return err
	}
	var out InclusionList
	out.Slot = binary.LittleEndian.Uint64(data)
	out.ValidatorIndex = binary.LittleEndian.Uint64(data[8:])
	copy(out.DependentRoot[:], data[16:48])
	out.Transactions = txs
	*m = out
	return nil
}
func (s SignedInclusionList) MarshalSSZ() ([]byte, error) {
	m, err := s.Message.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	if 100+len(m) > MaxSignedSize {
		return nil, ErrEncoding
	}
	out := make([]byte, 100, 100+len(m))
	binary.LittleEndian.PutUint32(out, 100)
	copy(out[4:], s.Signature[:])
	return append(out, m...), nil
}
func (s *SignedInclusionList) UnmarshalSSZ(data []byte) error {
	if len(data) < 152 || len(data) > MaxSignedSize || binary.LittleEndian.Uint32(data) != 100 {
		return ErrEncoding
	}
	var out SignedInclusionList
	if err := out.Message.UnmarshalSSZ(data[100:]); err != nil {
		return err
	}
	copy(out.Signature[:], data[4:100])
	*s = out
	return nil
}
func (m InclusionList) HashTreeRoot() types.Hash {
	roots := make([][32]byte, len(m.Transactions))
	for i, tx := range m.Transactions {
		roots[i] = ssz.HashTreeRootProgressiveBasicList(tx, len(tx))
	}
	return types.Hash(ssz.HashTreeRootContainer([][32]byte{ssz.HashTreeRootUint64(m.Slot), ssz.HashTreeRootUint64(m.ValidatorIndex), m.DependentRoot, ssz.HashTreeRootProgressiveList(roots)}))
}
func (m InclusionList) SigningRoot(fork [4]byte, genesis [32]byte) types.Hash {
	domain := consensus.DomainSeparation(DomainType, fork, genesis)
	return consensus.ComputeSigningRoot(m.HashTreeRoot(), domain)
}

type Request struct {
	Slot          uint64
	DependentRoot types.Hash
	Indices       Bits
}

func (r Request) MarshalSSZ() []byte {
	out := make([]byte, 42)
	binary.LittleEndian.PutUint64(out, r.Slot)
	copy(out[8:], r.DependentRoot[:])
	copy(out[40:], r.Indices[:])
	return out
}
func (r *Request) UnmarshalSSZ(data []byte) error {
	if len(data) != 42 {
		return fmt.Errorf("%w: request length", ErrEncoding)
	}
	r.Slot = binary.LittleEndian.Uint64(data)
	copy(r.DependentRoot[:], data[8:40])
	copy(r.Indices[:], data[40:])
	return nil
}
