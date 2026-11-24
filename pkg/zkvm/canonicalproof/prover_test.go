package canonicalproof

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSyntheticAndMalformedProofsAreRejectedWithoutAProver(t *testing.T) {
	for _, data := range [][]byte{nil, make([]byte, 256), []byte(`{"version":1,"image_id":"00","receipt":"fake","statement":{}}`), []byte(`{} {}`)} {
		if _, err := DecodeProof(data); err == nil {
			t.Fatal("accepted malformed or synthetic proof")
		}
	}
	p := Prover{Binary: "/no/such/canonical-prover"}
	if p.Available() {
		t.Fatal("missing binary advertised available")
	}
	if _, err := p.Prove(context.Background(), []byte{0x73}, nil, DefaultConfig()); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestStatementBindingRejectsClaimsBeforeRunningBackend(t *testing.T) {
	program, input, output := []byte{0x73}, []byte{42}, []byte{42}
	cfg := DefaultConfig()
	proof := Proof{Version: 1, ImageID: strings.Repeat("0", 64), Receipt: "fixture-is-not-a-real-receipt", Statement: Statement{Version: 1, ProgramHash: hash(program), InputHash: hash(input), Output: "2a", OutputHash: hash(output), Cycles: 1, ExitCode: 0, StateHash: strings.Repeat("0", 64), Config: cfg}}
	expected := Expected{Program: program, Input: input, Output: output, Config: cfg}
	for _, change := range []func(*Proof){
		func(p *Proof) { p.Statement.ProgramHash = hash([]byte{1}) },
		func(p *Proof) { p.Statement.InputHash = hash([]byte{1}) },
		func(p *Proof) { p.Statement.Output = "2b" },
		func(p *Proof) { p.Statement.Cycles = cfg.MaxCycles + 1 },
		func(p *Proof) { p.Statement.Config.StackBase++ },
		func(p *Proof) { p.Statement.StateHash = "00" },
		func(p *Proof) { p.Statement.ExitCode = 1 },
	} {
		copy := proof
		change(&copy)
		if err := matchStatement(&copy, expected); err == nil {
			t.Fatal("accepted changed claim")
		}
	}
	// Correct metadata is insufficient: the actual receipt must still verify.
	if err := (Prover{Binary: "/no/such/canonical-prover"}).Verify(context.Background(), &proof, expected); err == nil {
		t.Fatal("accepted metadata without cryptographic verification")
	}
	encoded, _ := json.Marshal(proof)
	encoded = append(encoded, []byte(` {}`)...)
	if _, err := DecodeProof(encoded); err == nil {
		t.Fatal("accepted trailing proof data")
	}
}

func TestResourceBoundsRejectBeforeExecution(t *testing.T) {
	prover := Prover{Binary: "/no/such/canonical-prover"}
	for _, cfg := range []Config{{}, {MaxCycles: MaxCycles + 1, MemoryPages: 1}, {MaxCycles: 1, MemoryPages: 16385}, {MaxCycles: 1, MemoryPages: 1, EntryPC: 1}} {
		if _, err := prover.Prove(context.Background(), []byte{0x73}, nil, cfg); !errors.Is(err, ErrBounds) {
			t.Fatal(err)
		}
	}
}
