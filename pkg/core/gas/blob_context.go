package gas

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
)

// BlobBaseFeeForHeader is the single current-block price used by execution,
// receipts, EVM contexts, pool head policy and RPC consumers.
func BlobBaseFeeForHeader(cfg *config.ChainConfig, header *types.Header) *big.Int {
	if header == nil || header.ExcessBlobGas == nil {
		return new(big.Int)
	}
	s := GetBlobSchedule(cfg, header.Time)
	return CalcBlobBaseFeeWithSchedule(*header.ExcessBlobGas, BlobScheduleEntry{BaseFeeUpdateFraction: s.UpdateFraction})
}

// CalcExcessBlobGasForHeader uses the child's parameters, including at forks.
// EIP-7918 affects excess only after Osaka; there is no direct price floor.
func CalcExcessBlobGasForHeader(parent *types.Header, cfg *config.ChainConfig, childTime uint64) (uint64, error) {
	if parent == nil {
		return 0, fmt.Errorf("missing parent for blob gas calculation")
	}
	s := GetBlobSchedule(cfg, childTime)
	return CalcExcessBlobGasWithRules(parent, s, cfg != nil && cfg.IsOsaka(childTime))
}

// CalcExcessBlobGasWithRules also serves RPC estimates whose backend supplies
// the active parameters and fork rules separately from the consensus config.
func CalcExcessBlobGasWithRules(parent *types.Header, s BlobSchedule, isOsaka bool) (uint64, error) {
	if parent == nil {
		return 0, fmt.Errorf("missing parent for blob gas calculation")
	}
	if s.Max == 0 || s.Target == 0 || s.Target > s.Max || s.UpdateFraction == 0 || s.Max > math.MaxUint64/GasPerBlob {
		return 0, fmt.Errorf("invalid active blob schedule")
	}
	var excess, used uint64
	if parent.ExcessBlobGas != nil {
		excess = *parent.ExcessBlobGas
	}
	if parent.BlobGasUsed != nil {
		used = *parent.BlobGasUsed
	}
	sum, carry := bits.Add64(excess, used, 0)
	if carry != 0 {
		return 0, fmt.Errorf("excess blob gas overflow")
	}
	if sum < s.Target*GasPerBlob {
		return 0, nil
	}
	if isOsaka && parent.BaseFee != nil {
		price := CalcBlobBaseFeeWithSchedule(excess, BlobScheduleEntry{BaseFeeUpdateFraction: s.UpdateFraction})
		if IsExecutionFeeLed(parent.BaseFee, price) {
			hi, lo := bits.Mul64(used, s.Max-s.Target)
			increase, _ := bits.Div64(hi, lo, s.Max)
			result, carry := bits.Add64(excess, increase, 0)
			if carry != 0 {
				return 0, fmt.Errorf("excess blob gas overflow")
			}
			return result, nil
		}
	}
	return sum - s.Target*GasPerBlob, nil
}

// MaxBlobsPerTransaction distinguishes EIP-7594's Osaka transaction bound
// from the independently configured block capacity.
func MaxBlobsPerTransaction(cfg *config.ChainConfig, time uint64) uint64 {
	if cfg != nil && cfg.IsOsaka(time) {
		return 6
	}
	return MaxBlobsForBlock(cfg, time)
}
