package focil

import (
	"fmt"
	"math"
	"math/big"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

type Profile1Kind uint8

const (
	Profile1Outside Profile1Kind = iota
	Profile1Exempt
	Profile1Enforceable
)

type Profile1Result struct {
	Kind                 Profile1Kind
	Transaction          *types.Transaction
	Sender               types.Address
	MinimumGas           uint64
	Reason               string
	ExecutionReservation uint64
	StateReservation     uint64
}

// EvaluateProfile1 implements the determinate EIP-8369 Profile 1 predicate at
// the payload's transaction-end state before withdrawals/post-execution systems. It authenticates the actual envelope (never a cached
// Sender), supports uint256 chain IDs/balances, and executes no EVM code.
// It does not select or activate the informational Profile 2 extension.
func EvaluateProfile1(raw []byte, cfg *config.ChainConfig, header *types.Header, db state.StateDB) (result Profile1Result, err error) {
	return evaluateProfile1(raw, cfg, header, db, nil)
}

// EvaluateProfile1WithCapacity uses the actual independent EIP-8037 pools.
// The header's maximum consumed dimension cannot reconstruct either pool.
func EvaluateProfile1WithCapacity(raw []byte, cfg *config.ChainConfig, header *types.Header, db state.StateDB, capacity *gaspool.GasPool) (Profile1Result, error) {
	if capacity == nil {
		return Profile1Result{}, fmt.Errorf("Profile 1 requires independent gas capacity")
	}
	return evaluateProfile1(raw, cfg, header, db, capacity)
}

func evaluateProfile1(raw []byte, cfg *config.ChainConfig, header *types.Header, db state.StateDB, capacity *gaspool.GasPool) (result Profile1Result, err error) {
	result.Kind = Profile1Exempt
	tx, decodeErr := types.DecodeBlockTransaction(raw)
	if decodeErr != nil {
		result.Reason = "malformed envelope"
		return result, nil
	}
	result.Transaction = tx
	switch tx.Type() {
	case types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType, types.SetCodeTxType:
	default:
		result.Kind = Profile1Outside
		return result, nil
	}
	if cfg == nil || cfg.ChainID == nil || header == nil || header.Number == nil || db == nil || !cfg.IsMerge() || !cfg.IsAmsterdam(header.Time) {
		return result, fmt.Errorf("Profile 1 requires selected Amsterdam state/context")
	}
	defer func() {
		if readErr := state.ReadError(db); readErr != nil {
			err = readErr
		}
	}()
	if header.GasUsed > header.GasLimit {
		return result, fmt.Errorf("invalid payload gas accounting")
	}
	result.ExecutionReservation = min(tx.Gas(), vm.AmsterdamMaxExecutionGas)
	result.StateReservation = tx.Gas()
	if capacity == nil {
		capacity = new(gaspool.GasPool).AddGas(header.GasLimit - header.GasUsed)
	}
	if capacity.Gas() > header.GasLimit || capacity.StateGas() > header.GasLimit {
		return result, fmt.Errorf("invalid independent payload gas capacity")
	}
	if !capacity.FitsFrame(result.ExecutionReservation, result.StateReservation) {
		result.Reason = "declared gas does not fit"
		return result, nil
	}
	if types.ValidateOrdinaryEnvelope(tx) != nil {
		result.Reason = "invalid envelope fields"
		return result, nil
	}
	if tx.Type() == types.SetCodeTxType && (!cfg.IsPrague(header.Time) || tx.To() == nil) {
		result.Reason = "invalid set-code envelope"
		return result, nil
	}
	rules := cfg.Rules(header.Number, cfg.IsMerge(), header.Time)
	if tx.Type() == types.AccessListTxType && !rules.IsBerlin || tx.Type() == types.DynamicFeeTxType && !rules.IsLondon {
		result.Reason = "transaction type is not active"
		return result, nil
	}
	// Geth is used only for independent envelope/signature recovery, not execution.
	var wire gethtypes.Transaction
	if e := wire.UnmarshalBinary(raw); e != nil {
		result.Reason = "invalid signed envelope"
		return result, nil
	}
	sender, e := gethtypes.Sender(gethtypes.LatestSignerForChainID(cfg.ChainID), &wire)
	if e != nil {
		result.Reason = "signature or chain ID"
		return result, nil
	}
	result.Sender = types.Address(sender)
	value, fee, tip := tx.Value(), tx.GasFeeCap(), tx.GasTipCap()
	if value == nil {
		value = new(big.Int)
	}
	for _, v := range []*big.Int{value, fee, tip} {
		if v == nil || v.Sign() < 0 || v.BitLen() > 256 {
			result.Reason = "invalid value/fee width"
			return result, nil
		}
	}
	if tip.Cmp(fee) > 0 || header.BaseFee != nil && fee.Cmp(header.BaseFee) < 0 {
		result.Reason = "invalid fee"
		return result, nil
	}
	if tx.Nonce() == math.MaxUint64 || db.GetNonce(result.Sender) != tx.Nonce() {
		result.Reason = "nonce"
		return result, nil
	}
	code := db.GetCode(result.Sender)
	_, delegated := types.ParseDelegation(code)
	if len(code) > 0 && !delegated {
		result.Reason = "sender code"
		return result, nil
	}
	if tx.To() == nil && cfg.IsShanghai(header.Time) {
		limit := vm.MaxInitCodeSize
		if cfg.Rules(header.Number, cfg.IsMerge(), header.Time).IsEIP7954 {
			limit = vm.MaxInitCodeSizeGlamsterdam
		}
		if len(tx.Data()) > limit {
			result.Reason = "initcode limit"
			return result, nil
		}
	}
	intrinsic, e := gas.CalculateAmsterdamIntrinsic(result.Sender, tx.To(), value, tx.Data(), tx.AccessList(), uint64(len(tx.AuthorizationList())))
	if e != nil || intrinsic.Validate(tx.Gas()) != nil {
		result.Reason = "intrinsic/floor gas"
		return result, nil
	}
	result.MinimumGas = max(intrinsic.Gas, intrinsic.Floor)
	maximum := new(big.Int).Mul(fee, new(big.Int).SetUint64(tx.Gas()))
	maximum.Add(maximum, value)
	if cfg.IsGlamsterdan(header.Time) && header.CalldataExcessGas != nil {
		maximum.Add(maximum, gas.CalldataGasCost(types.CalldataTokenGas(tx.Data()), gas.CalcCalldataBaseFeeFromHeader(header)))
	}
	if db.GetBalance(result.Sender).Cmp(maximum) < 0 {
		result.Reason = "balance"
		return result, nil
	}
	result.Kind = Profile1Enforceable
	return result, nil
}
