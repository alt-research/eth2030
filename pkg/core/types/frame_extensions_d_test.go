package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
)

func proofFrameD8504(data []byte) Frame8141 {
	return Frame8141{Mode: ModeProof8504, Data: data, Limits: FrameGasLimits{Execution: uint64(len(data)/96) * 3000}}
}

func TestFrameDIndependentProofFrameWireHashAndGas(t *testing.T) {
	raw, err := os.ReadFile("testdata/frame-d-independent.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Cases []struct {
			Profile, Wire        string
			SigningHash          string `json:"signing_hash"`
			Dependency           string
			Intrinsic, Floor     uint64
			ExecutionReservation uint64 `json:"execution_reservation"`
		}
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors.Cases {
		t.Run(v.Profile, func(t *testing.T) {
			wire, err := hex.DecodeString(strings.TrimPrefix(v.Wire, "0x"))
			if err != nil {
				t.Fatal(err)
			}
			transaction, err := DecodeTxRLP(wire)
			if err != nil {
				t.Fatal(err)
			}
			f := transaction.Inner().(*FrameTx8141)
			if err := f.Validate(); err == nil {
				t.Fatal("proof accepted without explicit extension")
			}
			if err := f.ValidateWithExtensions(FrameExtensions{Proof8504: true}); err != nil {
				t.Fatal(err)
			}
			h, err := f.SigningHash()
			if err != nil || h.Hex() != v.SigningHash {
				t.Fatal("signing hash differs from independent oracle", h, err)
			}
			original := append([]byte(nil), f.Frames[0].Data...)
			deps, err := f.Dependencies8504()
			if err != nil || len(deps) != 1 || "0x"+hex.EncodeToString(deps[0].Bytes()) != v.Dependency {
				t.Fatal("zero replacement or duplicate resolution differs", deps, err)
			}
			if !bytes.Equal(original, f.Frames[0].Data) {
				t.Fatal("deriving dependencies mutated signed frame data")
			}
			budget, err := f.GasBudget()
			if err != nil || budget.Intrinsic != v.Intrinsic || budget.CalldataFloor != v.Floor || budget.ExecutionReservation != v.ExecutionReservation {
				t.Fatalf("gas=%+v err=%v", budget, err)
			}
			encoded, err := EncodeFrameTx8141(f)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatal("proof envelope round trip differs", err)
			}
		})
	}
}

func TestFrameDProofConstraintsAndAssertComposition(t *testing.T) {
	d := Dependency8504{Scheme: 16, VerificationKeyHash: Hash{31: 1}}
	base := &FrameTx8141{ChainID: big.NewInt(1), Frames: []Frame8141{proofFrameD8504(d.Bytes()), {Mode: ModeVerify, Flags: 3, Limits: FrameGasLimits{Execution: 1000}}}, Fees: FrameFeeBudget{MaxFee: big.NewInt(10000000)}}
	ext := FrameExtensions{Proof8504: true, Assert7906: true}
	if err := base.ValidateWithExtensions(ext); err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name  string
		apply func(*FrameTx8141)
	}{
		{"zero_address_target", func(f *FrameTx8141) { a := Address{}; f.Frames[0].Target = &a }},
		{"flags", func(f *FrameTx8141) { f.Frames[0].Flags = 1 }},
		{"state", func(f *FrameTx8141) { f.Frames[0].Limits.State = 1 }},
		{"wrong_execution", func(f *FrameTx8141) { f.Frames[0].Limits.Execution++ }},
		{"padding", func(f *FrameTx8141) { f.Frames[0].Data[0] = 1 }},
		{"reserved_stark", func(f *FrameTx8141) { f.Frames[0].Data[31] = 17 }},
		{"empty", func(f *FrameTx8141) { f.Frames[0].Data = nil }},
		{"truncated", func(f *FrameTx8141) { f.Frames[0].Data = f.Frames[0].Data[:95] }},
		{"too_many", func(f *FrameTx8141) { f.Frames[0] = proofFrameD8504(bytes.Repeat(d.Bytes(), 257)) }},
		{"inside_batch", func(f *FrameTx8141) {
			f.Frames = append([]Frame8141{{Mode: ModeDefault, Flags: FrameAtomicBatch}}, f.Frames...)
		}},
		{"proof_after_assert", func(f *FrameTx8141) { f.Frames = append([]Frame8141{{Mode: ModeAssert7906}}, f.Frames...) }},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			f := base.copy().(*FrameTx8141)
			m.apply(f)
			if err := f.ValidateWithExtensions(ext); err == nil {
				t.Fatal("invalid extension accepted")
			}
		})
	}
	base.Frames = append(base.Frames, Frame8141{Mode: ModeAssert7906}, Frame8141{Mode: ModeAssert7906})
	if err := base.ValidateWithExtensions(ext); err != nil {
		t.Fatal("valid trailing assertions rejected", err)
	}
	base.Frames = append(base.Frames, Frame8141{Mode: ModeDefault})
	if err := base.ValidateWithExtensions(ext); err == nil {
		t.Fatal("non-ASSERT after suffix accepted")
	}
}

func TestFrameDDependenciesSortDeduplicateAndUnion(t *testing.T) {
	a := Dependency8504{Scheme: 16, DataHash: Hash{31: 1}, VerificationKeyHash: Hash{31: 2}}
	b := a
	b.VerificationKeyHash[31] = 1
	c := a
	c.DataHash[31] = 2
	tx := &FrameTx8141{Frames: []Frame8141{proofFrameD8504(append(append(c.Bytes(), a.Bytes()...), b.Bytes()...)), proofFrameD8504(a.Bytes())}}
	deps, err := BlockDependencies8504([]*Transaction{NewTransaction(tx), NewTransaction(tx)})
	if err != nil || len(deps) != 3 || deps[0] != b || deps[1] != a || deps[2] != c {
		t.Fatal("noncanonical dependency union", deps, err)
	}
	// 256 repeated declarations are valid and incur one witness-size charge;
	// verification remains charged for every declared triple.
	tx.Frames = []Frame8141{proofFrameD8504(bytes.Repeat(a.Bytes(), 256))}
	if err := tx.ValidateWithExtensions(FrameExtensions{Proof8504: true}); err != nil {
		t.Fatal(err)
	}
}

func TestFrameDProofPrefixPositionsAndExcludedWork(t *testing.T) {
	d := Dependency8504{Scheme: 16, VerificationKeyHash: Hash{31: 1}}
	proof := proofFrameD8504(bytes.Repeat(d.Bytes(), 256))
	verify := Frame8141{Mode: ModeVerify, Flags: 3, Limits: FrameGasLimits{Execution: 1000}}
	for _, frames := range [][]Frame8141{{proof, verify}, {verify, proof}, {proof, verify, proof}} {
		tx := &FrameTx8141{Frames: frames}
		end, err := tx.ValidationPrefix8141WithExtensions(100, FrameExtensions{Proof8504: true})
		if err != nil || tx.Frames[end].Mode != ModeVerify {
			t.Fatal("proof position or excluded work rejected", end, err)
		}
	}
	tx := &FrameTx8141{Frames: []Frame8141{proof}}
	if _, err := tx.ValidationPrefix8141WithExtensions(100, FrameExtensions{Proof8504: true}); err == nil {
		t.Fatal("proof-only transaction has no approval prefix")
	}
	proof = proofFrameD8504(d.Bytes())
	deadline := make([]byte, 8)
	deadline[7] = 101
	tx.Frames = []Frame8141{proof, {Mode: ModeVerify, Target: &FrameExpiryVerifier, Data: deadline, Limits: FrameGasLimits{Execution: 5000}}, proof, verify}
	if end, err := tx.ValidationPrefix8141WithExtensions(100, FrameExtensions{Proof8504: true}); err != nil || end != 3 {
		t.Fatal("proof frames interfered with first expiry position", end, err)
	}
}
