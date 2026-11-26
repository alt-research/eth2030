package txpool

import (
	"errors"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
)

// Signer shares are local admission bounds, not consensus constants. PAY
// recoveries count distinct paymasters separately from other-frame recoveries.
func (pool *TxPool) validateFrameSigners8151(frame *types.FrameTx8141, candidate *execution.FramePrefixValidation) error {
	for _, dependency := range candidate.SignerRecoveries {
		limit := pool.config.FrameSignerCanonicalLimit
		if limit <= 0 {
			limit = 64
		}
		otherLimit := pool.config.FrameSignerOtherLimit
		if otherLimit <= 0 {
			otherLimit = 8
		}
		if otherLimit >= limit {
			return errors.New("Frame signer other-digest share must be smaller than canonical share")
		}
		if !dependency.Canonical {
			limit = otherLimit
		}
		count := 0
		paymasters := make(map[types.Address]struct{})
		if dependency.Pay {
			paymasters[dependency.Paymaster] = struct{}{}
		} else {
			count = 1
		}
		for hash, pending := range pool.lookup.all {
			other, ok := pending.Inner().(*types.FrameTx8141)
			if !ok || other.SameReplayIdentity(frame) {
				continue
			}
			validation := pool.lookup.frameValidations[hash]
			if validation == nil {
				continue
			}
			for _, recovery := range validation.SignerRecoveries {
				if recovery.Address != dependency.Address || recovery.Canonical != dependency.Canonical || recovery.Pay != dependency.Pay {
					continue
				}
				if recovery.Pay {
					paymasters[recovery.Paymaster] = struct{}{}
				} else {
					count++
					break
				}
			}
		}
		if count+len(paymasters) > limit {
			return errors.New("Frame recovered signer pending share exhausted")
		}
	}
	return nil
}
