package ssz

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/bits"
	"os"
	"testing"
)

func TestEIP7495CapturedGloasStateAndProofs(t *testing.T) {
	file, err := os.Open("testdata/eip7495/captured.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	zip, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer zip.Close()
	data, err := io.ReadAll(zip)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Reference string                        `json:"reference"`
		SSZHash   string                        `json:"source_fixture_sha256"`
		Capture   progressiveReferenceContainer `json:"capture"`
		Nested    map[string]struct {
			Gindex uint64   `json:"gindex"`
			Leaf   string   `json:"leaf"`
			Branch []string `json:"branch"`
		} `json:"original_nested_proofs"`
		Boundary struct {
			Root string `json:"root"`
			SSZ  string `json:"ssz"`
		} `json:"secondary_256_without_more"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	if v.Reference != "ethereum/consensus-specs@d2021d2719235f0df4726a1d330502cb15aa8ea3" || len(v.Capture.Roots) != 46 || len(v.Nested) != 3 {
		t.Fatal("missing captured evidence")
	}
	c := v.Capture
	s, err := NewProgressiveContainerSchema(c.Active, c.Sizes)
	if err != nil {
		t.Fatal(err)
	}
	fields := make([][]byte, len(c.Fields))
	for i, f := range c.Fields {
		fields[i] = referenceBytes(t, f)
	}
	wire, err := s.MarshalSSZ(fields)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wire)
	if hex.EncodeToString(sum[:]) != v.SSZHash {
		t.Fatal("captured CL wire bytes differ")
	}
	back, err := s.UnmarshalSSZ(wire)
	if err != nil {
		t.Fatal(err)
	}
	for i := range fields {
		if !bytes.Equal(fields[i], back[i]) {
			t.Fatalf("captured field %d differs", i)
		}
	}
	roots := referenceRoots(t, c.Roots)
	root, err := s.HashTreeRoot(roots)
	if err != nil || root != referenceRoot(t, c.Root) {
		t.Fatal("captured CL state root differs")
	}
	for i, e := range c.Proofs {
		branch, g, err := s.GenerateProof(roots, e.Position)
		if err != nil || g != e.Gindex || !equalRoots(branch, referenceRoots(t, e.Branch)) || !s.VerifyProof(root, roots[i], e.Position, branch) {
			t.Fatalf("captured field %d proof differs", i)
		}
	}
	// Recompose the original CL current/next/finalized branches using the new
	// Go outer-field proof, preserving each reference inner subtree branch.
	for name, expected := range v.Nested {
		t.Run(name, func(t *testing.T) {
			original := referenceRoots(t, expected.Branch)
			found := false
			for _, field := range c.Proofs {
				difference := bits.Len64(expected.Gindex) - bits.Len64(field.Gindex)
				if difference < 0 || expected.Gindex>>uint(difference) != field.Gindex {
					continue
				}
				outer, _, err := s.GenerateProof(roots, field.Position)
				if err != nil {
					t.Fatal(err)
				}
				combined := append(append([][32]byte(nil), original[:difference]...), outer...)
				if !equalRoots(original, combined) {
					t.Fatal("CL complete branch differs")
				}
				index, leaf := expected.Gindex, referenceRoot(t, expected.Leaf)
				for _, sibling := range combined {
					if index&1 == 0 {
						leaf = hash(leaf, sibling)
					} else {
						leaf = hash(sibling, leaf)
					}
					index >>= 1
				}
				if index != 1 || leaf != root {
					t.Fatal("actual CL nested proof does not verify")
				}
				found = true
				break
			}
			if !found {
				t.Fatal("missing containing field")
			}
		})
	}
	active, sizes, leaves, values := make([]bool, 256), make([]int, 256), make([][32]byte, 256), make([][]byte, 256)
	for i := range active {
		active[i] = true
		sizes[i] = 2
		leaves[i] = HashTreeRootUint16(uint16(i + 1))
		values[i] = MarshalUint16(uint16(i + 1))
	}
	boundary, err := NewProgressiveContainerSchema(active, sizes)
	if err != nil {
		t.Fatal(err)
	}
	boundaryRoot, _ := boundary.HashTreeRoot(leaves)
	boundaryWire, _ := boundary.MarshalSSZ(values)
	if boundaryRoot != referenceRoot(t, v.Boundary.Root) || !bytes.Equal(boundaryWire, referenceBytes(t, v.Boundary.SSZ)) {
		t.Fatal("normative 256-field case differs from independent second reference")
	}
}
