package zkvm

import (
	"errors"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func TestDependency8504LegacyGuestRejectsUnsupportedBlockBeforeExecution(t *testing.T) {
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
		raw, err := types.NewBlock(h, body).EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := types.DecodeBlockRLP(raw); err != nil {
			t.Fatal("fixture not decodable", err)
		}
		ctx := NewGuestContext(types.Hash{31: 1}, []byte("state witness"))
		if _, err := ExecuteBlock(ctx, raw); !errors.Is(err, ErrGuestDependency8504) || ctx.IsExecuted() {
			t.Fatal(kind, "legacy hash-only guest accepted dependency block or consumed context", err)
		}
		fullContext := NewGuestContext(types.Hash{31: 1}, []byte("state witness"))
		result, err := ExecuteBlockFull(fullContext, raw)
		if !errors.Is(err, ErrGuestDependency8504) || result == nil || result.Success || fullContext.IsExecuted() {
			t.Fatal(kind, "full legacy wrapper bypassed dependency rejection", result, err)
		}
	}
}

func TestDependency8504STFBoundariesRejectUnsupportedInputs(t *testing.T) {
	legacy := NewSTFExecutor(DefaultSTFConfig())
	real, err := NewRealSTFExecutor(DefaultRealSTFConfig(), NewGuestRegistry())
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"valid", "bare", "active-empty"} {
		block := proof8504.Block()
		h, body := block.Header(), block.Body()
		if kind == "bare" {
			h.DependencyWitnessesHash, body.DependencyWitnesses = nil, nil
		}
		if kind == "active-empty" {
			body.Transactions, body.DependencyWitnesses = nil, [][]byte{}
			empty := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
			h.DependencyWitnessesHash = &empty
		}
		block = types.NewBlock(h, body)
		pre := types.Hash{31: 1}
		if _, err := legacy.GenerateWitness(pre, block); !errors.Is(err, ErrSTFDependency8504) {
			t.Error(kind, "legacy witness generator dropped dependency fields", err)
		}
		if _, err := real.GenerateSTFWitness(pre, block); !errors.Is(err, ErrSTFDependency8504) {
			t.Error(kind, "real witness schema dropped dependency fields", err)
		}
		input := STFInput{PreStateRoot: pre, BlockHeader: h, Transactions: body.Transactions, Witnesses: [][]byte{[]byte("state witness")}}
		input.PostStateRoot = computePostStateRoot(pre, input.Transactions, input.Witnesses)
		if _, err := legacy.ValidateTransition(input); !errors.Is(err, ErrSTFDependency8504) {
			t.Error(kind, "legacy transition accepted or misclassified dependency input", err)
		}
		// Refuse the unsupported schema before program/backend/prover work.
		if _, err := real.ExecuteSTF(&input); !errors.Is(err, ErrSTFDependency8504) {
			t.Error(kind, "real STF did not reject before backend setup", err)
		}
	}
}
