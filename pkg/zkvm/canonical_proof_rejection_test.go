package zkvm

import "testing"

func TestCanonicalRejectsStructurallySizedFakeProof(t *testing.T) {
	execution := &GuestExecution{Success: true, ProofData: make([]byte, groth16ProofSize)}
	if VerifyExecution(execution, buildTestSimpleHaltProgram(), nil) {
		t.Fatal("accepted a zero-filled structural placeholder as an execution proof")
	}
}
