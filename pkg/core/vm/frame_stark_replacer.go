package vm

import (
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/proofs"
)

// BlockValidationFrameProver must prove and verify the actual execution and
// approvals of the VERIFY frames against the supplied block and its trusted
// pre-state. A bare calldata digest is not an execution proof. The provider is
// responsible for binding the proof to all required public inputs and state.
type BlockValidationFrameProver interface {
	proofs.ValidationFrameProver
	ProveBlockValidationFrames(block *types.Block) (*proofs.STARKProofData, error)
	VerifyBlockValidationFrames(block *types.Block, proof *proofs.STARKProofData) bool
}

func inspectVerifyFrames(block *types.Block) (hasVerify, valid bool) {
	if block == nil {
		return false, false
	}
	for _, tx := range block.Transactions() {
		if tx == nil {
			return false, false
		}
		if tx.Type() != types.FrameTxType {
			continue
		}
		for _, frame := range tx.Frames() {
			if frame.Mode == types.ModeVerify {
				hasVerify = true
			}
		}
	}
	return hasVerify, true
}

// ReplaceValidationFrames currently returns the unmodified block. It may
// return a proof only from an explicitly configured block-bound provider; it
// does not strip calldata or make an experimental proof consensus authoritative.
func ReplaceValidationFrames(block *types.Block, prover proofs.ValidationFrameProver) (*types.Block, *proofs.STARKProofData, error) {
	hasVerify, valid := inspectVerifyFrames(block)
	if !valid {
		return block, nil, proofs.ErrFrameProofUnavailable
	}
	if !hasVerify {
		return block, nil, nil
	}
	bound, ok := prover.(BlockValidationFrameProver)
	if !ok {
		return block, nil, proofs.ErrFrameProofUnavailable
	}
	proof, err := bound.ProveBlockValidationFrames(block)
	if err != nil {
		return block, nil, err
	}
	if proof == nil || !bound.VerifyBlockValidationFrames(block, proof) {
		return block, nil, proofs.ErrFrameProofUnavailable
	}
	return block, proof, nil
}

// VerifyBlockFrameProof requires a provider that verifies the block-bound
// execution statement. Legacy proof-only verifiers cannot authenticate it.
func VerifyBlockFrameProof(block *types.Block, proof *proofs.STARKProofData, prover proofs.ValidationFrameProver) bool {
	hasVerify, valid := inspectVerifyFrames(block)
	if !valid {
		return false
	}
	if !hasVerify {
		return proof == nil
	}
	if proof == nil {
		return false
	}
	bound, ok := prover.(BlockValidationFrameProver)
	return ok && bound.VerifyBlockValidationFrames(block, proof)
}
