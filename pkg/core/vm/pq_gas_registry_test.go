package vm

import (
	"testing"

	"github.com/eth2030/eth2030/crypto/pqc"
)

// These are local experimental estimates, not EIP-8051 precompile gas.
// Exercise the real VM lookup instead of a test copy of the registry table.
func TestLocalPQGasLookupMatchesRegistry(t *testing.T) {
	registry := pqc.GlobalRegistry()
	if err := registry.ValidateGasCostsMatch(GasPQVerify); err != nil {
		t.Fatal(err)
	}
	mismatch := func(id uint8) uint64 {
		gas := GasPQVerify(id)
		if id == uint8(pqc.MLDSA44) {
			return gas + 1
		}
		return gas
	}
	if err := registry.ValidateGasCostsMatch(mismatch); err == nil {
		t.Fatal("failed to detect a VM gas mismatch")
	}
}
