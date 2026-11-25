package ssz

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

type progressiveReferenceProof struct {
	Position int      `json:"position"`
	Gindex   uint64   `json:"gindex"`
	Branch   []string `json:"branch"`
}
type progressiveReferenceContainer struct {
	Name   string                      `json:"name"`
	Active []bool                      `json:"active_fields"`
	Sizes  []int                       `json:"sizes"`
	Fields []string                    `json:"fields"`
	Roots  []string                    `json:"field_roots"`
	SSZ    string                      `json:"ssz"`
	Root   string                      `json:"root"`
	Proofs []progressiveReferenceProof `json:"proofs"`
}

func referenceBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func referenceRoot(t *testing.T, s string) [32]byte {
	t.Helper()
	b := referenceBytes(t, s)
	if len(b) != 32 {
		t.Fatalf("not a root: %s", s)
	}
	var root [32]byte
	copy(root[:], b)
	return root
}
func referenceRoots(t *testing.T, items []string) [][32]byte {
	t.Helper()
	out := make([][32]byte, len(items))
	for i, s := range items {
		out[i] = referenceRoot(t, s)
	}
	return out
}

func TestEIP7495IndependentReferenceCorpus(t *testing.T) {
	file, err := os.Open("testdata/eip7495/vectors.json.gz")
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
	var corpus struct {
		Reference  string                          `json:"reference"`
		Containers []progressiveReferenceContainer `json:"containers"`
		Lists      []struct {
			Count  int                         `json:"count"`
			Roots  []string                    `json:"field_roots"`
			Root   string                      `json:"root"`
			Proofs []progressiveReferenceProof `json:"proofs"`
		} `json:"lists"`
		Invalid []struct {
			Name     string `json:"name"`
			Active   []bool `json:"active_fields"`
			Sizes    []int  `json:"sizes"`
			Rejected bool   `json:"reference_rejected"`
		} `json:"invalid_schemas"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Reference != "ethereum/remerkleable@2f0baeef0082d4278acaef7d822deb7009d7db7e" || len(corpus.Containers) < 20 || len(corpus.Lists) != 10 || len(corpus.Invalid) != 4 {
		t.Fatal("incomplete or unknown reference corpus")
	}
	for _, v := range corpus.Containers {
		t.Run(v.Name, func(t *testing.T) {
			s, err := NewProgressiveContainerSchema(v.Active, v.Sizes)
			if err != nil {
				t.Fatal(err)
			}
			fields := make([][]byte, len(v.Fields))
			for i, f := range v.Fields {
				fields[i] = referenceBytes(t, f)
			}
			wire, err := s.MarshalSSZ(fields)
			if err != nil || !bytes.Equal(wire, referenceBytes(t, v.SSZ)) {
				t.Fatalf("reference wire differs: %v", err)
			}
			decoded, err := s.UnmarshalSSZ(referenceBytes(t, v.SSZ))
			if err != nil {
				t.Fatal(err)
			}
			for i := range fields {
				if !bytes.Equal(decoded[i], fields[i]) {
					t.Fatalf("field %d differs", i)
				}
			}
			roots := referenceRoots(t, v.Roots)
			root, err := s.HashTreeRoot(roots)
			want := referenceRoot(t, v.Root)
			if err != nil || root != want {
				t.Fatalf("reference root differs: %x %v", root, err)
			}
			if len(v.Proofs) != len(roots) {
				t.Fatal("missing reference field proofs")
			}
			for i, expected := range v.Proofs {
				p, g, err := s.GenerateProof(roots, expected.Position)
				if err != nil || g != expected.Gindex {
					t.Fatalf("field %d gindex=%d want %d %v", expected.Position, g, expected.Gindex, err)
				}
				branch := referenceRoots(t, expected.Branch)
				if !equalRoots(p, branch) || !s.VerifyProof(want, roots[i], expected.Position, branch) {
					t.Fatalf("field %d reference proof differs", expected.Position)
				}
				leaf := roots[i]
				leaf[0] ^= 1
				if s.VerifyProof(want, leaf, expected.Position, branch) {
					t.Fatal("changed leaf accepted")
				}
				branch[0][0] ^= 1
				if s.VerifyProof(want, roots[i], expected.Position, branch) {
					t.Fatal("changed reference sibling accepted")
				}
			}
		})
	}
	for _, v := range corpus.Lists {
		t.Run(fmt.Sprintf("List%d", v.Count), func(t *testing.T) {
			roots := referenceRoots(t, v.Roots)
			list := NewProgressiveList(roots)
			want := referenceRoot(t, v.Root)
			if list.HashTreeRoot() != want {
				t.Fatal("list reference root differs")
			}
			for _, e := range v.Proofs {
				p, g, err := list.GenerateProof(e.Position)
				if err != nil || g != e.Gindex || !equalRoots(p, referenceRoots(t, e.Branch)) {
					t.Fatalf("list index %d differs from reference: gindex=%d want %d %v", e.Position, g, e.Gindex, err)
				}
			}
		})
	}
	for _, v := range corpus.Invalid {
		t.Run("Illegal/"+v.Name, func(t *testing.T) {
			if !v.Rejected {
				t.Fatal("missing independent negative")
			}
			if _, err := NewProgressiveContainerSchema(v.Active, v.Sizes); err == nil {
				t.Fatal("accepted reference-rejected type")
			}
		})
	}
}
func equalRoots(a, b [][32]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
