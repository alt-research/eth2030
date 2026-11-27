package types

import (
	"math/big"
	"testing"
)

func TestDependency8504GloasReconstructsAndChecksWitnessCommitment(t *testing.T) {
	for _, witnesses := range [][][]byte{{}, {{1, 2, 3}}} {
		// CL treats witnesses as opaque; EL performs the length/crypto checks.
		zero := uint64(0)
		hash := DependencyWitnessesHash8504(witnesses)
		h := &Header{Number: big.NewInt(1), BaseFee: big.NewInt(1), BlobGasUsed: &zero, ExcessBlobGas: &zero, SlotNumber: &zero, DependencyWitnessesHash: &hash}
		p := &ConsensusPayload8504{Header: h, DependencyWitnesses: witnesses}
		raw, err := p.MarshalSSZ()
		if err != nil {
			t.Fatal(err)
		}
		var decoded ConsensusPayload8504
		if err := decoded.UnmarshalSSZ(raw); err != nil {
			t.Fatal(err)
		}
		if decoded.Header.DependencyWitnessesHash == nil || *decoded.Header.DependencyWitnessesHash != hash {
			t.Fatal("CL header reconstruction omitted witness commitment")
		}
		p.DependencyWitnesses = nil
		if _, err := p.MarshalSSZ(); err == nil {
			t.Fatal("active field silently downgraded to the pre-extension schema")
		}
		p.DependencyWitnesses = witnesses
		bad := hash
		bad[0] ^= 1
		p.Header.DependencyWitnessesHash = &bad
		if _, err := p.HashTreeRoot(); err == nil {
			t.Fatal("CL payload accepted inconsistent reconstructed witness commitment")
		}
	}
}
