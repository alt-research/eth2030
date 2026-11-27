package proofs

import (
	"errors"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func TestDependency8504LegacyProofHelpersRejectUnsupportedBlocks(t *testing.T) {
	for _, kind := range []string{"valid", "missing", "tampered", "bare", "active-empty"} {
		block := proof8504.Block()
		h, body := block.Header(), block.Body()
		switch kind {
		case "missing":
			body.DependencyWitnesses = nil
		case "tampered":
			body.DependencyWitnesses[0][0] ^= 1
		case "bare":
			h.DependencyWitnessesHash, body.DependencyWitnesses = nil, nil
		case "active-empty":
			body.Transactions, body.DependencyWitnesses = nil, [][]byte{}
			empty := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
			h.DependencyWitnessesHash = &empty
		}
		block = types.NewBlock(h, body)
		for _, withBAL := range []bool{false, true} {
			var err error
			if withBAL {
				_, err = GenerateExecutionProofWithBAL(block, types.Hash{31: 1}, &bal.BlockAccessList{Entries: []bal.AccessEntry{{Address: types.Address{19: 2}}}})
			} else {
				_, err = GenerateExecutionProof(block, types.Hash{31: 1})
			}
			if !errors.Is(err, ErrExecProofDependency8504) {
				t.Fatal(kind, withBAL, "legacy synthetic proof helper accepted an unsupported dependency block")
			}
		}
	}
}
