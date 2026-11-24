//go:build canonicalproof_integration

package zkvm

import (
	"encoding/hex"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/zkvm/canonicalproof"
)

func TestRealExplicitBackendAndProductionVerifier(t *testing.T) {
	if !(canonicalproof.Prover{}).Available() {
		t.Fatal("mandatory real-proof backend unavailable")
	}
	program, input := buildTestSimpleHaltProgram(), []byte("production verifier")
	result, err := ProveExecution(&ProofRequest{Program: program, Input: input, ProgramHash: HashProgram(program)})
	if err != nil {
		t.Fatal(err)
	}
	if valid, err := VerifyExecProof(result, HashProgram(program)); err != nil || !valid {
		t.Fatal(valid, err)
	}
	proof, err := canonicalproof.DecodeProof(result.ProofBytes)
	if err != nil {
		t.Fatal(err)
	}
	key, err := hex.DecodeString(proof.ImageID)
	if err != nil {
		t.Fatal(err)
	}
	vk := &VerificationKey{Data: key, ProgramHash: types.Hash(HashProgram(program))}
	claim := &Proof{Data: result.ProofBytes, PublicInputs: input}
	if valid, err := VerifyProof(vk, claim); err != nil || !valid {
		t.Fatal("production verifier did not accept actual proof", valid, err)
	}
	changed := *claim
	changed.PublicInputs = []byte("another input")
	if valid, _ := VerifyProof(vk, &changed); valid {
		t.Fatal("production verifier ignored expected input")
	}
	wrongKey := *vk
	wrongKey.ProgramHash[0] ^= 1
	if valid, _ := VerifyProof(&wrongKey, claim); valid {
		t.Fatal("production verifier ignored program identity")
	}
	for name, change := range map[string]func(*ProofResult){
		"state":         func(p *ProofResult) { p.TraceCommitment[0] ^= 1 },
		"public inputs": func(p *ProofResult) { p.PublicInputsHash[0] ^= 1 },
		"key": func(p *ProofResult) {
			p.VerificationKey = append([]byte(nil), p.VerificationKey...)
			p.VerificationKey[0] ^= 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := *result
			change(&copy)
			if valid, _ := VerifyExecProof(&copy, HashProgram(program)); valid {
				t.Fatal("accepted tampered backend metadata")
			}
		})
	}
}
