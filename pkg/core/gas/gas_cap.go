package gas

import (
	"errors"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// EIP-7825: Transaction Gas Limit Cap
//
// Introduces a protocol-level cap on the maximum gas used by a single
// transaction to 16,777,216 (2^24). Transactions exceeding this limit
// are rejected at both the txpool and block validation level.

// MaxTransactionGas is the protocol-level cap on individual transaction
// gas limits, per EIP-7825.
const MaxTransactionGas uint64 = 1 << 24 // 16,777,216

// ErrTxGasLimitExceeded is returned when a transaction's gas limit exceeds
// the EIP-7825 cap.
var ErrTxGasLimitExceeded = errors.New("transaction gas limit exceeds maximum (EIP-7825)")

// ValidateTransactionGasLimit checks whether a transaction's gas limit
// is within the EIP-7825 cap. Returns ErrTxGasLimitExceeded if gasLimit
// exceeds MaxTransactionGas.
func ValidateTransactionGasLimit(gasLimit uint64) error {
	if gasLimit > MaxTransactionGas {
		return ErrTxGasLimitExceeded
	}
	return nil
}

// IsGasLimitCapped returns true if the EIP-7825 transaction gas cap is
// active for the given chain config and block timestamp. The gas cap
// activates with the Osaka fork.
func IsGasLimitCapped(config *config.ChainConfig, time uint64) bool {
	return config != nil && config.IsOsaka(time)
}

// ValidateTransactionGasForFork preserves Osaka's declared-gas cap until
// Amsterdam, when ordinary envelopes retain only the fixed total-gas cap.
// Their execution cap is enforced on intrinsic gas and the running budget.
func ValidateTransactionGasForFork(cfg *config.ChainConfig, time uint64, tx *types.Transaction) error {
	if cfg != nil && cfg.IsMerge() && cfg.IsAmsterdam(time) && tx.Type() <= types.SetCodeTxType {
		if tx.Gas() > vm.AmsterdamMaxTotalGas {
			return ErrTxGasLimitExceeded
		}
		return nil
	}
	if !IsGasLimitCapped(cfg, time) {
		return nil
	}
	limit := tx.Gas()
	if frame, ok := tx.Inner().(*types.FrameTx8141); ok {
		budget, err := frame.GasBudget()
		if err != nil {
			return err
		}
		limit = budget.ExecutionReservation
	}
	return ValidateTransactionGasLimit(limit)
}
