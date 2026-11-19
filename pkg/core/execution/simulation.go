package execution

import (
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// messageOptions is internal to execution. Ordinary consensus entrypoints always
// supply its zero value; only explicit simulation entrypoints select call policy.
type messageOptions struct {
	vmConfig   vm.Config
	simulation bool
	uncapped   bool
}

// SimulateMessage applies the same transition on caller-owned isolated state.
// transactionRules selects estimates/access-list semantics; false selects the
// merged Geth 35845 call policy, with uncapped execution and no initial reservoir.
func SimulateMessage(cfg *config.ChainConfig, getHash vm.GetHashFunc, db state.StateDB, header *types.Header, msg *config.Message, gp *gaspool.GasPool, transactionRules bool, vmConfig vm.Config) (*ExecutionResult, error) {
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
	result, err := applyMessageWithOptions(cfg, getHash, db, header, msg, gp, messageOptions{vmConfig: vmConfig, simulation: true, uncapped: !transactionRules})
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
