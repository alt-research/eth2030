package ssz

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
)

func TestEIP7495IllegalSchemas(t *testing.T) {
	cases := []struct {
		name   string
		active []bool
		sizes  []int
	}{
		{"empty", nil, nil}, {"no_fields", []bool{true}, nil}, {"trailing_gap", []bool{true, false}, []int{2}},
		{"count_mismatch", []bool{true, true}, []int{2}}, {"all_inactive", []bool{false}, nil},
		{"over_256", make([]bool, 257), nil}, {"negative_size", []bool{true}, []int{-1}},
		{"fixed_section_overflow", []bool{true, true, true}, []int{int(^uint(0) >> 1), int(^uint(0) >> 1), 4}},
	}
	cases[5].active[256] = true
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewProgressiveContainerSchema(c.active, c.sizes); !errors.Is(err, ErrProgressiveSchema) {
				t.Fatalf("accepted illegal type: %v", err)
			}
		})
	}
}

func TestEIP7495SquareReferenceAndImmutableSchema(t *testing.T) {
	active, sizes := []bool{true, false, true}, []int{2, 1}
	s, err := NewProgressiveContainerSchema(active, sizes)
	if err != nil {
		t.Fatal(err)
	}
	active[0] = false
	sizes[0] = 7 // Caller-owned inputs cannot change the declared type.
	fields := [][]byte{MarshalUint16(42), MarshalUint8(255)}
	wire, err := s.MarshalSSZ(fields)
	if err != nil || !bytes.Equal(wire, []byte{42, 0, 255}) {
		t.Fatalf("wire=%x err=%v", wire, err)
	}
	back, err := s.UnmarshalSSZ(wire)
	if err != nil || !bytes.Equal(back[0], fields[0]) || !bytes.Equal(back[1], fields[1]) {
		t.Fatalf("decoded=%x %v", back, err)
	}
	back[0][0] ^= 1
	if wire[0] != 42 {
		t.Fatal("decoder aliases input")
	}
	root, err := s.HashTreeRoot([][32]byte{HashTreeRootUint16(42), HashTreeRootUint8(255)})
	const want = "550969a02e4f4c4e85e3fcf2c26103d1cd6dba7b6185bbb684a40be4550d37af"
	if err != nil || hex.EncodeToString(root[:]) != want {
		t.Fatalf("root=%x err=%v", root, err)
	}
	mask := s.ActiveFields()
	mask[0] = 0
	if s.ActiveFields()[0] != 5 || s.FieldCount() != 2 {
		t.Fatal("schema exposes mutable metadata")
	}
	p, g, err := s.GenerateProof([][32]byte{HashTreeRootUint16(42), HashTreeRootUint8(255)}, 2)
	if err != nil || g != 41 || !s.VerifyProof(root, HashTreeRootUint8(255), 2, p) {
		t.Fatalf("proof gindex=%d err=%v", g, err)
	}
	p[0][0] ^= 1
	if s.VerifyProof(root, HashTreeRootUint8(255), 2, p) {
		t.Fatal("changed sibling accepted")
	}
}

func TestEIP7495CanonicalContainerOffsets(t *testing.T) {
	s, _ := NewProgressiveContainerSchema([]bool{true, false, true, true}, []int{2, 0, 0})
	fields := [][]byte{{42, 0}, {1, 2}, {}}
	wire, err := s.MarshalSSZ(fields)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire, []byte{42, 0, 10, 0, 0, 0, 12, 0, 0, 0, 1, 2}) {
		t.Fatalf("wire=%x", wire)
	}
	if _, err := s.UnmarshalSSZ(wire); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func([]byte){func(b []byte) { b[2] = 11 }, func(b []byte) { b[2] = 9 }, func(b []byte) { b[6] = 9 }, func(b []byte) { b[6] = 13 }} {
		bad := bytes.Clone(wire)
		change(bad)
		if _, err := s.UnmarshalSSZ(bad); err == nil {
			t.Fatalf("accepted bad offsets: %x", bad)
		}
	}
	if _, err := s.MarshalSSZ(fields[:2]); err == nil {
		t.Fatal("missing field accepted")
	}
	if _, err := s.MarshalSSZ([][]byte{{42}, {}, {}}); err == nil {
		t.Fatal("wrong fixed field size accepted")
	}
	fixed, _ := NewProgressiveContainerSchema([]bool{true}, []int{2})
	if _, err := fixed.UnmarshalSSZ([]byte{42, 0, 1}); err == nil {
		t.Fatal("trailing fixed bytes accepted")
	}
}

func TestEIP7495CheckedSlots(t *testing.T) {
	var mask [32]byte
	mask[0] = 5
	fields := [][32]byte{HashTreeRootUint16(42), {}, HashTreeRootUint8(255)}
	root, err := HashTreeRootProgressiveContainerChecked(fields, mask)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := NewProgressiveContainerSchema([]bool{true, false, true}, []int{2, 1})
	want, _ := s.HashTreeRoot([][32]byte{fields[0], fields[2]})
	if root != want {
		t.Fatal("slot and active-field APIs disagree")
	}
	fields[1][0] = 1
	if _, err := HashTreeRootProgressiveContainerChecked(fields, mask); err == nil {
		t.Fatal("inactive nonzero root accepted")
	}
	fields[1] = [32]byte{}
	mask[31] = 128
	if _, err := HashTreeRootProgressiveContainerChecked(fields, mask); err == nil {
		t.Fatal("undeclared position accepted")
	}
	if _, err := HashTreeRootProgressiveContainerChecked(nil, [32]byte{}); err == nil {
		t.Fatal("empty layout accepted")
	}
}

func TestEIP7495AllPositionsAndProofRejections(t *testing.T) {
	active := make([]bool, 256)
	sizes := make([]int, 256)
	roots := make([][32]byte, 256)
	for i := range active {
		active[i] = true
		sizes[i] = 2
		roots[i] = HashTreeRootUint16(uint16(i + 1))
	}
	s, err := NewProgressiveContainerSchema(active, sizes)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := s.HashTreeRoot(roots)
	for i := range roots {
		p, g, err := s.GenerateProof(roots, i)
		if err != nil || !s.VerifyProof(root, roots[i], i, p) {
			t.Fatalf("position=%d gindex=%d %v", i, g, err)
		}
		wrong := roots[i]
		wrong[0] ^= 1
		if s.VerifyProof(root, wrong, i, p) {
			t.Fatalf("changed leaf %d accepted", i)
		}
		if s.VerifyProof(root, roots[i], i, p[:len(p)-1]) {
			t.Fatal("short proof accepted")
		}
		p[len(p)-1][0] ^= 1
		if s.VerifyProof(root, roots[i], i, p) {
			t.Fatal("changed type mix-in accepted")
		}
	}
	sparse, _ := NewProgressiveContainerSchema([]bool{true, false, true}, []int{2, 2})
	if _, err := sparse.FieldGindex(1); err == nil {
		t.Fatal("inactive position accepted")
	}
	if _, _, err := sparse.GenerateProof(nil, 0); err == nil {
		t.Fatal("missing field roots accepted")
	}
	var zero ProgressiveContainerSchema
	if _, err := zero.HashTreeRoot(nil); err == nil {
		t.Fatal("zero schema accepted")
	}
	if _, err := zero.UnmarshalSSZ(nil); err == nil {
		t.Fatal("zero decoder accepted")
	}
	if _, err := zero.MarshalSSZ(nil); err == nil {
		t.Fatal("zero encoder accepted")
	}
}

func TestProgressiveProofsAllIndicesAndGrowth(t *testing.T) {
	previous := map[int]uint64{}
	for _, n := range []int{1, 2, 3, 5, 6, 21, 22, 85, 86, 256} {
		leaves := make([][32]byte, n)
		for i := range leaves {
			leaves[i] = HashTreeRootUint64(uint64(i + 1))
		}
		list := NewProgressiveList(leaves)
		for i := range leaves {
			p, g, err := list.GenerateProof(i)
			if err != nil {
				t.Fatal(err)
			}
			if old, ok := previous[i]; ok && old != g {
				t.Fatalf("index %d moved from %d to %d", i, old, g)
			}
			previous[i] = g
			index, leaf := g, leaves[i]
			for _, sibling := range p {
				if index&1 == 0 {
					leaf = hash(leaf, sibling)
				} else {
					leaf = hash(sibling, leaf)
				}
				index >>= 1
			}
			if index != 1 || leaf != list.HashTreeRoot() {
				t.Fatalf("count=%d index=%d returned invalid gindex=%d", n, i, g)
			}
		}
	}
	if previous[1] != 40 {
		t.Fatalf("second element gindex=%d", previous[1])
	}
	for _, index := range []uint64{0, 1 << 63, ^uint64(0)} {
		if _, err := progressiveMixinGindex(index); err == nil {
			t.Fatal("unrepresentable gindex accepted")
		}
	}
}

func TestEIP7495OrdinaryContainerJSONAndDefaults(t *testing.T) {
	type Square struct {
		Side  uint16 `json:"side"`
		Color uint8  `json:"color"`
	}
	value := Square{42, 255}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) != `{"side":42,"color":255}` {
		t.Fatalf("container JSON=%s %v", encoded, err)
	}
	schema, _ := NewProgressiveContainerSchema([]bool{true, false, true}, []int{2, 1})
	var zero Square
	wire, err := schema.MarshalSSZ([][]byte{MarshalUint16(zero.Side), MarshalUint8(zero.Color)})
	if err != nil || !bytes.Equal(wire, []byte{0, 0, 0}) {
		t.Fatal("ordinary Container default encoding differs")
	}
	squareRoot, _ := schema.HashTreeRoot([][32]byte{HashTreeRootUint16(value.Side), HashTreeRootUint8(value.Color)})
	circle, _ := NewProgressiveContainerSchema([]bool{false, true, true}, []int{2, 1})
	proof, g, err := schema.GenerateProof([][32]byte{HashTreeRootUint16(value.Side), HashTreeRootUint8(value.Color)}, 2)
	otherIndex, _ := circle.FieldGindex(2)
	if err != nil || g != otherIndex {
		t.Fatal("shared field gindex moved")
	}
	if circle.VerifyProof(squareRoot, HashTreeRootUint8(value.Color), 2, proof) {
		t.Fatal("wrong container layout accepted")
	}
}
