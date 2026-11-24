package zkvm

import (
	"testing"

	"github.com/eth2030/eth2030/zkvm/canonicalproof"
)

// The dedicated real-proof CI job installs the pinned prover and runs every
// one of these positive integration cases. Go-only jobs still test fail-closed
// rejection and execution, without substituting synthetic proofs.
func requireRealCanonicalProof(t *testing.T) {
	t.Helper()
	if !(canonicalproof.Prover{}).Available() {
		t.Skip("real RISC Zero backend absent; positive proving runs in canonical-proof CI")
	}
}
