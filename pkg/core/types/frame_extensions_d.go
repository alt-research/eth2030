package types

import (
	"bytes"
	"errors"
	"sort"
)

// FrameExtensions selects explicit fork rules without changing transaction
// bytes or mutating shared transactions during head/candidate revalidation.
type FrameExtensions struct {
	Proof8504  bool
	Assert7906 bool
}

const (
	ModeAssert7906                uint8  = 3
	ModeProof8504                 uint8  = 4
	LeanSPHINCSScheme8504         uint8  = 0x10
	MaxDependenciesPerFrame8504          = 256
	MaxPublicDependencies8504            = 16
	DependencyVerificationGas8504 uint64 = 3000
	DependencyWitnessGas8504      uint64 = 495360
	DependencyWitnessSize8504            = 7740
)

// Dependency8504 contains resolved data, not the encoded zero placeholder.
// The public key is the 32-byte key itself, without an additional hash.
type Dependency8504 struct {
	Scheme              uint8
	DataHash            Hash
	VerificationKeyHash Hash
}

func (d Dependency8504) Bytes() []byte {
	b := make([]byte, 96)
	b[31] = d.Scheme
	copy(b[32:64], d.DataHash[:])
	copy(b[64:], d.VerificationKeyHash[:])
	return b
}

func ParseDependencies8504(data []byte) ([]Dependency8504, error) {
	if len(data) == 0 || len(data)%96 != 0 || len(data)/96 > MaxDependenciesPerFrame8504 {
		return nil, errors.New("EIP-8504 invalid dependency frame length")
	}
	out := make([]Dependency8504, len(data)/96)
	for i := range out {
		t := data[i*96 : (i+1)*96]
		if !bytes.Equal(t[:31], make([]byte, 31)) || t[31] != LeanSPHINCSScheme8504 {
			return nil, errors.New("EIP-8504 unknown scheme or nonzero scheme padding")
		}
		out[i].Scheme = t[31]
		copy(out[i].DataHash[:], t[32:64])
		copy(out[i].VerificationKeyHash[:], t[64:])
	}
	return out, nil
}

func SortDependencies8504(deps []Dependency8504) []Dependency8504 {
	deps = append([]Dependency8504(nil), deps...)
	sort.Slice(deps, func(i, j int) bool {
		a, b := deps[i], deps[j]
		if a.Scheme != b.Scheme {
			return a.Scheme < b.Scheme
		}
		if a.DataHash != b.DataHash {
			return bytes.Compare(a.DataHash[:], b.DataHash[:]) < 0
		}
		return bytes.Compare(a.VerificationKeyHash[:], b.VerificationKeyHash[:]) < 0
	})
	end := 0
	for _, d := range deps {
		if end == 0 || deps[end-1] != d {
			deps[end] = d
			end++
		}
	}
	return deps[:end]
}

func (tx *FrameTx8141) HasProofFrames8504() bool {
	for _, f := range tx.Frames {
		if f.Mode == ModeProof8504 {
			return true
		}
	}
	return false
}

func (tx *FrameTx8141) Dependencies8504() ([]Dependency8504, error) {
	h, err := tx.SigningHash()
	if err != nil {
		return nil, err
	}
	return tx.dependencies8504(h)
}

func (tx *FrameTx8141) dependencies8504(signingHash Hash) ([]Dependency8504, error) {
	var out []Dependency8504
	for _, f := range tx.Frames {
		if f.Mode != ModeProof8504 {
			continue
		}
		deps, err := ParseDependencies8504(f.Data)
		if err != nil {
			return nil, err
		}
		for _, d := range deps {
			if d.DataHash == (Hash{}) {
				d.DataHash = signingHash
			}
			out = append(out, d)
		}
	}
	return SortDependencies8504(out), nil
}

func BlockDependencies8504(transactions []*Transaction) ([]Dependency8504, error) {
	var all []Dependency8504
	for _, transaction := range transactions {
		if f, ok := transaction.Inner().(*FrameTx8141); ok {
			deps, err := f.Dependencies8504()
			if err != nil {
				return nil, err
			}
			all = append(all, deps...)
		}
	}
	return SortDependencies8504(all), nil
}
