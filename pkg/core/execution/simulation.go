package execution

import (
	"context"
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math/big"
)

// messageOptions is internal to execution. Ordinary consensus entrypoints always
// supply its zero value; only explicit simulation entrypoints select call policy.
type messageOptions struct {
	vmConfig   vm.Config
	simulation bool
	uncapped   bool
	validation bool
	overrides  EVMOverrides
}

// EVMOverrides carries context changes supplied by RPC simulations. The active
// fork's original precompile set still defines transaction warming.
type EVMOverrides struct {
	Context     context.Context
	Precompiles map[types.Address]vm.PrecompiledContract
	BlobBaseFee *big.Int
}

func (o EVMOverrides) apply(evm *vm.EVM) {
	if o.Context != nil {
		evm.Config.Context = o.Context
	}
	if o.Precompiles != nil {
		evm.SetPrecompiles(o.Precompiles)
	}
	if o.BlobBaseFee != nil {
		evm.Context.BlobBaseFee = new(big.Int).Set(o.BlobBaseFee)
	}
}

type SimulationConfig struct {
	TransactionRules bool
	Validation       bool
	VMConfig         vm.Config
	Overrides        EVMOverrides
	Tracker          vm.BALTracker
}

// SimulateMessage applies the same transition on caller-owned isolated state.
// transactionRules selects estimates/access-list semantics; false selects the
// merged Geth 35845 call policy, with uncapped execution and no initial reservoir.
func SimulateMessage(cfg *config.ChainConfig, getHash vm.GetHashFunc, db state.StateDB, header *types.Header, msg *config.Message, gp *gaspool.GasPool, transactionRules bool, vmConfig vm.Config) (*ExecutionResult, error) {
	return SimulateMessageWithConfig(cfg, getHash, db, header, msg, gp, SimulationConfig{TransactionRules: transactionRules, VMConfig: vmConfig})
}

// SimulateMessageWithConfig selects the explicit simulator policy. Validation
// keeps nonce and fee checks while still allowing contract senders.
func SimulateMessageWithConfig(cfg *config.ChainConfig, getHash vm.GetHashFunc, db state.StateDB, header *types.Header, msg *config.Message, gp *gaspool.GasPool, settings SimulationConfig) (*ExecutionResult, error) {
	vmConfig := settings.VMConfig
	if msg.TxType > types.SetCodeTxType {
		return nil, fmt.Errorf("unsupported simulation transaction type %d", msg.TxType)
	}
	if original, ok := db.(interface{ FinalizePreState() }); ok {
		original.FinalizePreState()
	}
	if transaction, ok := db.(interface{ FinalizeTransaction() }); ok {
		defer transaction.FinalizeTransaction()
	}
	db.ClearAccessList()
	db.ClearTransientStorage()
	db.SubRefund(db.GetRefund())
	result, err := applyMessageWithOptions(cfg, getHash, db, header, msg, gp, messageOptions{vmConfig: vmConfig, simulation: true, uncapped: !settings.TransactionRules, validation: settings.Validation, overrides: settings.Overrides}, settings.Tracker)
	if err == nil && vmConfig.Tracer != nil {
		vmConfig.Tracer.CaptureEnd(result.ReturnData, result.UsedGas, result.Err)
	}
	return result, err
}

// ApplyTransactionWithVMConfig preserves production transaction preparation and
// settlement while allowing an observational tracer for historical replay.
func ApplyTransactionWithVMConfig(cfg *config.ChainConfig, getHash vm.GetHashFunc, db state.StateDB, header *types.Header, tx *types.Transaction, gp *gaspool.GasPool, vmConfig vm.Config) (*types.Receipt, uint64, error) {
	return applyTransactionInternal(cfg, getHash, db, header, tx, gp, nil, nil, vmConfig)
}
