package main

import (
	"encoding/json"
	"github.com/eth2030/eth2030/zkvm"
	"os"
)

func main() {
	program := []byte{1, 2, 3}
	input := []byte("not an executed guest input")
	accepted := zkvm.VerifyExecution(&zkvm.GuestExecution{Success: true, ProofData: make([]byte, 256)}, program, input)
	json.NewEncoder(os.Stdout).Encode(map[string]any{"source": "b4b88ed0e410e5b735a9c6bf0f033180c89dc8c0", "path": "pkg/zkvm/canonical.go:VerifyExecution", "scope": "standalone research helper; not the qualified Glamsterdam import path", "success_flag": true, "proof_bytes": 256, "proof_all_zero": true, "accepted": accepted})
}
