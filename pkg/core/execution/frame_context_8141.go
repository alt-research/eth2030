package execution

import (
	"errors"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
)

// ValidateFrameContext8141 checks only immutable fork/chain/nonce-schema rules.
// It neither establishes full validity nor requires state, dependency witnesses
// or EVM execution. IL consumers can ignore these definitely invalid envelopes
// before encountering an unspecified omitted-witness composition.
func ValidateFrameContext8141(cfg *config.ChainConfig, header *types.Header, tx *types.FrameTx8141) error {
	return validateFrameContext8141(cfg, header, tx, false)
}

func validateFrameContext8141(cfg *config.ChainConfig, header *types.Header, tx *types.FrameTx8141, explicitNonceProfile bool) error {
	if cfg == nil || header == nil || tx == nil || !cfg.IsMerge() || !cfg.IsHogota(header.Time) {
		return errors.New("EIP-8141 requires Bogotá")
	}
	if cfg.ChainID == nil || tx.ChainID == nil || tx.ChainID.Cmp(cfg.ChainID) != 0 {
		return errors.New("EIP-8141 chain ID mismatch")
	}
	// Independent research profile callers retain their explicit context. Public
	// consensus/admission/IL consumers always enforce the selected live schema.
	if !explicitNonceProfile && (tx.NonceProfile8250() != nil) != cfg.IsEIP8250(header.Time) {
		return errors.New("Frame nonce schema does not match EIP-8250 activation")
	}
	return nil
}
