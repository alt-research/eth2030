package focil

import (
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// InclusionListUnsatisfied is the status returned by engine_newPayload when a
// valid IL transaction is absent from the block (EIP-7805 §engine-api).
const InclusionListUnsatisfied = "INCLUSION_LIST_UNSATISFIED"

// ILSatisfactionResult indicates whether the inclusion list was satisfied.
type ILSatisfactionResult int

const (
	// ILSatisfied means all IL txs are either in the block or exempt.
	ILSatisfied ILSatisfactionResult = iota
	// ILUnsatisfied means a valid IL tx was absent and gas was available.
	ILUnsatisfied
	// ILValidationUnavailable is neither execution invalidity nor proven satisfaction.
	ILValidationUnavailable
)

// PostStateReader provides account state needed for IL satisfaction checks.
type PostStateReader interface {
	GetNonce(addr [20]byte) uint64
	GetBalance(addr [20]byte) uint64
}

// FrameVerifier is the legacy code-size interface, retained for source compatibility.
// Code size alone cannot establish validation-prefix eligibility.
type FrameVerifier interface {
	// GetCodeSize returns the byte length of contract code at addr.
	// Returns 0 for EOAs and non-existent accounts.
	GetCodeSize(addr types.Address) int
}

// CheckILSatisfaction handles presence/gas exemptions and known post-state checks.
// Missing validation context is ILValidationUnavailable, never a proven verdict.
// This helper is not the complete selected Bogotá/AA-VOPS execution algorithm.
func CheckILSatisfaction(block *types.Block, ils []*InclusionList, postState PostStateReader, gasRemaining uint64) ILSatisfactionResult {
	return CheckILSatisfactionWithVerifier(block, ils, postState, nil, gasRemaining)
}

// CheckILSatisfactionWithVerifier retains the legacy signature. The code-size
// verifier cannot authenticate Frames; absent Frames return unavailable.
func CheckILSatisfactionWithVerifier(block *types.Block, ils []*InclusionList, postState PostStateReader, verifier FrameVerifier, gasRemaining uint64) ILSatisfactionResult {
	if block == nil {
		return ILValidationUnavailable
	}
	// Build set of tx hashes in the block.
	blockTxHashes := make(map[types.Hash]bool, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		blockTxHashes[tx.Hash()] = true
	}

	for _, il := range ils {
		if il == nil {
			return ILValidationUnavailable
		}
		for _, entry := range il.Entries {
			tx, err := types.DecodeTxRLP(entry.Transaction)
			if err != nil {
				// Invalid tx — skip (per spec).
				continue
			}
			// Rule 1: tx in block → satisfied.
			if blockTxHashes[tx.Hash()] {
				continue
			}
			// Rule 2: insufficient gas remaining → skip (gas exemption).
			if gasRemaining < tx.Gas() {
				continue
			}
			// Code existence cannot establish Frame validation-prefix eligibility.
			// Until HGT-046 supplies the selected execution model, do not guess.
			if postState == nil || tx.Type() == types.FrameTxType {
				return ILValidationUnavailable
			}
			// Rule 3: validate state for includability.
			if postState != nil {
				// Resolve sender: frame txs embed the sender directly,
				// while EOA txs recover it from the ECDSA signature.
				var from types.Address
				if senderPtr := tx.Sender(); senderPtr != nil {
					from = *senderPtr
				} else {
					// Can't determine sender → can't prove tx is invalid.
					// Per EIP-7805: ILs can contain any tx, and we can only
					// exempt txs we can prove are state-invalid. Without a
					// sender, return an unavailable result rather than a consensus verdict.
					return ILValidationUnavailable
				}
				fromKey := [20]byte(from)

				// Nonce check: applies to all tx types including frame txs.
				if postState.GetNonce(fromKey) != tx.Nonce() {
					continue // invalid nonce → exempt
				}

				// EOA tx: standard balance check.
				gasPrice := tx.GasPrice()
				if gasPrice == nil {
					gasPrice = new(big.Int)
				}
				cost := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), gasPrice)
				if v := tx.Value(); v != nil {
					cost.Add(cost, v)
				}
				if !cost.IsUint64() {
					return ILValidationUnavailable
				}
				if postState.GetBalance(fromKey) < cost.Uint64() {
					continue // insufficient balance → exempt
				}
			}
			// Valid tx absent from block → unsatisfied.
			return ILUnsatisfied
		}
	}
	return ILSatisfied
}
