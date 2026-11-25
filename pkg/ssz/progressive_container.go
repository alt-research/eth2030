package ssz

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

var ErrProgressiveSchema = errors.New("ssz: invalid ProgressiveContainer schema")

// ProgressiveContainerSchema declares an immutable EIP-7495 layout. Field sizes
// describe active fields in declaration order: positive sizes are fixed, zero
// denotes a variable-size field. Field-specific codecs and JSON mapping remain
// those of the corresponding ordinary Container. Inactive positions have no
// value or wire bytes; active_fields is type metadata, never message data.
type ProgressiveContainerSchema struct {
	positions []int
	sizes     []int
	width     int
	active    [32]byte
}

func NewProgressiveContainerSchema(activeFields []bool, fieldSizes []int) (*ProgressiveContainerSchema, error) {
	if len(activeFields) == 0 || len(activeFields) > 256 || !activeFields[len(activeFields)-1] {
		return nil, ErrProgressiveSchema
	}
	s := &ProgressiveContainerSchema{width: len(activeFields), sizes: append([]int(nil), fieldSizes...)}
	for i, active := range activeFields {
		if active {
			s.positions = append(s.positions, i)
			s.active[i/8] |= 1 << uint(i%8)
		}
	}
	if len(s.positions) != len(fieldSizes) {
		return nil, fmt.Errorf("%w: active-field count differs from declared fields", ErrProgressiveSchema)
	}
	var fixed uint64
	for _, size := range fieldSizes {
		if size < 0 || uint64(size) > math.MaxUint32 {
			return nil, ErrProgressiveSchema
		}
		if size == 0 {
			fixed += 4
		} else {
			fixed += uint64(size)
		}
		if fixed > math.MaxUint32 {
			return nil, ErrProgressiveSchema
		}
	}
	return s, nil
}

func (s *ProgressiveContainerSchema) valid() bool {
	return s != nil && s.width > 0 && len(s.positions) > 0 && len(s.positions) == len(s.sizes)
}
func (s *ProgressiveContainerSchema) FieldCount() int {
	if s == nil {
		return 0
	}
	return len(s.positions)
}
func (s *ProgressiveContainerSchema) ActiveFields() [32]byte {
	if s == nil {
		return [32]byte{}
	}
	return s.active
}

func (s *ProgressiveContainerSchema) slots(fieldRoots [][32]byte) ([][32]byte, error) {
	if !s.valid() {
		return nil, ErrProgressiveSchema
	}
	if len(fieldRoots) != len(s.positions) {
		return nil, ErrSize
	}
	slots := make([][32]byte, s.width)
	for i, pos := range s.positions {
		slots[pos] = fieldRoots[i]
	}
	return slots, nil
}

func (s *ProgressiveContainerSchema) HashTreeRoot(fieldRoots [][32]byte) ([32]byte, error) {
	slots, err := s.slots(fieldRoots)
	if err != nil {
		return [32]byte{}, err
	}
	return HashTreeRootProgressiveContainer(slots, s.active), nil
}

// MarshalSSZ is ordinary Container serialization of only the active fields.
func (s *ProgressiveContainerSchema) MarshalSSZ(fields [][]byte) ([]byte, error) {
	if !s.valid() {
		return nil, ErrProgressiveSchema
	}
	if len(fields) != len(s.sizes) {
		return nil, ErrSize
	}
	fixed := make([][]byte, len(fields))
	var variable [][]byte
	var indices []int
	var total uint64
	for i, size := range s.sizes {
		total += uint64(len(fields[i]))
		if size == 0 {
			total += 4
			variable = append(variable, fields[i])
			indices = append(indices, i)
		} else if len(fields[i]) != size {
			return nil, ErrSize
		} else {
			fixed[i] = fields[i]
		}
		if total > math.MaxUint32 {
			return nil, ErrSize
		}
	}
	return MarshalVariableContainer(fixed, variable, indices), nil
}

func (s *ProgressiveContainerSchema) UnmarshalSSZ(data []byte) ([][]byte, error) {
	if !s.valid() {
		return nil, ErrProgressiveSchema
	}
	if uint64(len(data)) > math.MaxUint32 {
		return nil, ErrSize
	}
	return UnmarshalVariableContainer(data, len(s.sizes), s.sizes)
}

// FieldGindex addresses a declared field by its stable layout position, not by
// its ordinal among active fields. Inactive and out-of-range positions fail.
func (s *ProgressiveContainerSchema) FieldGindex(position int) (uint64, error) {
	if !s.valid() {
		return 0, ErrProgressiveSchema
	}
	if position < 0 || position >= s.width || s.active[position/8]&(1<<uint(position%8)) == 0 {
		return 0, ErrFieldIndexOOB
	}
	remaining, leaves, base := position, 1, uint64(1)
	for remaining >= leaves {
		remaining -= leaves
		leaves *= 4
		base = base*2 + 1
	}
	depth := bits.Len(uint(leaves)) - 1
	return progressiveMixinGindex((base*2)<<uint(depth) | uint64(remaining))
}

func (s *ProgressiveContainerSchema) GenerateProof(fieldRoots [][32]byte, position int) ([][32]byte, uint64, error) {
	if _, err := s.FieldGindex(position); err != nil {
		return nil, 0, err
	}
	slots, err := s.slots(fieldRoots)
	if err != nil {
		return nil, 0, err
	}
	proof, index := progressiveProof(slots, 1, position, 1)
	index, err = progressiveMixinGindex(index)
	if err != nil {
		return nil, 0, err
	}
	return append(proof, s.active), index, nil
}

func (s *ProgressiveContainerSchema) VerifyProof(root, leaf [32]byte, position int, proof [][32]byte) bool {
	index, err := s.FieldGindex(position)
	if err != nil || len(proof) != bits.Len64(index)-1 || len(proof) == 0 || proof[len(proof)-1] != s.active {
		return false
	}
	for _, sibling := range proof {
		if index&1 == 0 {
			leaf = hash(leaf, sibling)
		} else {
			leaf = hash(sibling, leaf)
		}
		index >>= 1
	}
	return index == 1 && leaf == root
}

// HashTreeRootProgressiveContainerChecked is the checked field-slot form for
// concrete SSZ implementations. Slots include zero roots at inactive positions.
// The highest active bit must be the final slot; bits beyond it are forbidden.
func HashTreeRootProgressiveContainerChecked(fieldRoots [][32]byte, activeFields [32]byte) ([32]byte, error) {
	width := len(fieldRoots)
	if width == 0 || width > 256 || activeFields[(width-1)/8]&(1<<uint((width-1)%8)) == 0 {
		return [32]byte{}, ErrProgressiveSchema
	}
	for i := 0; i < 256; i++ {
		active := activeFields[i/8]&(1<<uint(i%8)) != 0
		if i >= width && active || i < width && !active && fieldRoots[i] != ([32]byte{}) {
			return [32]byte{}, ErrProgressiveSchema
		}
	}
	return HashTreeRootProgressiveContainer(fieldRoots, activeFields), nil
}
