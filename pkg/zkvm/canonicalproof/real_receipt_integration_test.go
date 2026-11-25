//go:build canonicalproof_integration

package canonicalproof

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRealReceiptIndependentVerificationAndBindings(t *testing.T) {
	prover := Prover{}
	if !prover.Available() {
		t.Fatal("real-proof integration requires the packaged RISC Zero verifier")
	}
	file, err := os.Open("testdata/risc0-echo-v1.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var proof Proof
	if err := json.NewDecoder(reader).Decode(&proof); err != nil {
		t.Fatal(err)
	}
	var program []byte
	for _, word := range []uint32{0x00200893, 0x73, 0x00100893, 0x73, 0x00000513, 0x00000893, 0x73} {
		program = binary.LittleEndian.AppendUint32(program, word)
	}
	cycles := uint64(7)
	expected := Expected{Program: program, Input: []byte{42}, Output: []byte{42}, Cycles: &cycles, Config: proof.Statement.Config, StateHash: proof.Statement.StateHash}
	reference, err := ReferenceExecution(context.Background(), program, []byte{42}, proof.Statement.Config)
	if err != nil || reference != proof.Statement {
		t.Fatal("independent Go machine disagrees with retained real proof", reference, err)
	}
	metadata, err := prover.Metadata(context.Background())
	if err != nil || metadata.ImageID != proof.ImageID {
		t.Fatal("fixture image does not match the pinned verifier", metadata, err)
	}
	if err := prover.Verify(context.Background(), &proof, expected); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Proof){
		"image":   func(p *Proof) { p.ImageID = strings.Repeat("0", 64) },
		"program": func(p *Proof) { p.Statement.ProgramHash = strings.Repeat("0", 64) },
		"input":   func(p *Proof) { p.Statement.InputHash = strings.Repeat("0", 64) },
		"output":  func(p *Proof) { p.Statement.Output = "2b"; p.Statement.OutputHash = hash([]byte{43}) },
		"state":   func(p *Proof) { p.Statement.StateHash = strings.Repeat("0", 64) },
		"cycles":  func(p *Proof) { p.Statement.Cycles++ },
		"bounds":  func(p *Proof) { p.Statement.Config.MaxCycles++ },
		"receipt": func(p *Proof) { p.Receipt = "AA==" },
	} {
		t.Run(name, func(t *testing.T) {
			copy := proof
			change(&copy)
			if err := prover.Verify(context.Background(), &copy, expected); err == nil {
				t.Fatal("accepted changed real execution claim")
			}
		})
	}
	wrong := expected
	wrong.Input = []byte{43}
	if err := prover.Verify(context.Background(), &proof, wrong); err == nil {
		t.Fatal("accepted a different caller input")
	}
}
