package execution

import (
	"errors"
	"fmt"
	"math/big"

	corconfig "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// applyFrame8141 is shared by payload construction, import and historical replay.
func applyFrame8141(config *corconfig.ChainConfig, getHash vm.GetHashFunc, db state.StateDB,
	header *types.Header, transaction *types.Transaction, gp *gaspool.GasPool, tracker vm.BALTracker,
) (_ *types.Receipt, _ uint64, err error) {
	return executeFrame8141(config, getHash, db, header, transaction, gp, tracker, false, nil)
}

func ValidateFrame8141Prefix(config *corconfig.ChainConfig, db state.StateDB, header *types.Header, tx *types.Transaction, blobBaseFee *big.Int) (types.Address, error) {
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok {
		return types.Address{}, errors.New("not a canonical Frame transaction")
	}
	if _, err := frame.ValidationPrefix8141(header.Time); err != nil {
		return types.Address{}, err
	}
	copied := db.Dup()
	copied.SetTxContext(tx.Hash(), 0)
	receipt, _, err := executeFrame8141(config, nil, copied, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil, true, blobBaseFee)
	if err != nil {
		return types.Address{}, err
	}
	return receipt.Payer, nil
}

func executeFrame8141(config *corconfig.ChainConfig, getHash vm.GetHashFunc, db state.StateDB,
	header *types.Header, transaction *types.Transaction, gp *gaspool.GasPool, tracker vm.BALTracker,
	prefixOnly bool, blobBaseFee *big.Int,
) (_ *types.Receipt, _ uint64, err error) {
	tx := transaction.Inner().(*types.FrameTx8141)
	if config == nil || !config.IsMerge() || !config.IsHogota(header.Time) {
		return nil, 0, errors.New("EIP-8141 requires Bogotá")
	}
	if config.ChainID == nil || tx.ChainID == nil || tx.ChainID.Cmp(config.ChainID) != 0 {
		return nil, 0, errors.New("EIP-8141 chain ID mismatch")
	}
	if err := tx.Validate(); err != nil {
		return nil, 0, err
	}
	if db.GetNonce(tx.Sender) != tx.Nonce {
		return nil, 0, errors.New("EIP-8141 nonce mismatch")
	}
	budget, err := tx.GasBudget()
	if err != nil {
		return nil, 0, err
	}
	blobFee := big.NewInt(1)
	if header.ExcessBlobGas != nil {
		blobFee = gas.BlobBaseFeeForHeader(config, header)
	}
	if blobBaseFee != nil {
		blobFee = new(big.Int).Set(blobBaseFee)
	}
	payment, err := tx.Payment(header.BaseFee, blobFee)
	if err != nil {
		return nil, 0, err
	}
	if err := gp.ReserveFrame(budget.ExecutionReservation, budget.StateReservation); err != nil {
		return nil, 0, err
	}
	txSnapshot := db.Snapshot()
	defer func() {
		if err != nil {
			db.RevertToSnapshot(txSnapshot)
			gp.RefundFrame(budget.ExecutionReservation, budget.StateReservation)
		}
	}()
	if original, ok := db.(interface{ FinalizePreState() }); ok {
		original.FinalizePreState()
	}
	txHash := transaction.Hash()
	transaction.SetSender(tx.Sender)
	db.ClearAccessList()
	db.ClearTransientStorage()
	refundBefore := db.GetRefund()
	db.AddAddressToAccessList(tx.Sender)
	db.AddAddressToAccessList(header.Coinbase)
	blockCtx := vm.BlockContext{
		GetHash: getHash, BlockNumber: header.Number, Time: header.Time,
		Coinbase: header.Coinbase, GasLimit: header.GasLimit, BaseFee: header.BaseFee,
		PrevRandao: header.MixDigest, BlobBaseFee: blobFee,
	}
	if header.SlotNumber != nil {
		blockCtx.SlotNumber = *header.SlotNumber
	}
	evm := vm.NewEVMWithState(blockCtx, vm.TxContext{
		GasPrice: payment.GasPrice(), BlobHashes: tx.BlobVersionedHashes,
	}, vm.Config{}, db)
	precompiles, err := ConfigureBlockEVM(config, header, evm)
	if err != nil {
		return nil, 0, err
	}
	for address := range precompiles {
		db.AddAddressToAccessList(address)
	}
	if tracker != nil {
		evm.SetBALTracker(tracker, 0)
	}
	ctx := evm.SetFrameTransaction(tx, payment)
	ctx.PublicValidation = prefixOnly
	prefixEnd := -1
	if prefixOnly {
		prefixEnd, err = tx.ValidationPrefix8141(header.Time)
		if err != nil {
			return nil, 0, err
		}
	}
	batchSnapshot, batchStart := -1, 0
	var batchExecution []types.FrameGasLimits
	for i := 0; i < len(tx.Frames); i++ {
		frame := tx.Frames[i]
		if frame.Mode == types.ModeSender && !ctx.Approved {
			return nil, 0, errors.New("EIP-8141 sender frame requires execution approval")
		}
		if batchSnapshot < 0 && frame.Flags&types.FrameAtomicBatch != 0 {
			batchStart = i
			batchSnapshot = evm.StateDB.Snapshot()
			batchExecution = nil
		}
		logStart := len(db.GetLogs(txHash))
		db.ClearTransientStorage()
		left, frameErr := evm.RunFrame8141(i)
		result := types.FrameResult8141{Status: types.ReceiptStatusSuccessful,
			GasUsed: types.FrameGasLimits{Execution: frame.Limits.Execution - left,
				State: ctx.Results[i].GasUsed.State},
		}
		if frameErr != nil {
			result.Status = types.ReceiptStatusFailed
			if frame.Mode == types.ModeVerify || prefixOnly {
				return nil, 0, fmt.Errorf("EIP-8141 verify frame %d: %w", i, frameErr)
			}
		} else {
			result.Logs = append([]*types.Log(nil), db.GetLogs(txHash)[logStart:]...)
		}
		if prefixOnly {
			if err := evm.FrameValidationError8141(); err != nil {
				return nil, 0, err
			}
			if frame.Mode == types.ModeVerify && frame.Flags != 0 && ctx.Scopes[i] != frame.Flags {
				return nil, 0, errors.New("Frame prefix did not approve its exact scope")
			}
		}
		ctx.Results[i] = result
		if batchSnapshot >= 0 {
			batchExecution = append(batchExecution, result.GasUsed)
			if frameErr != nil {
				evm.StateDB.RevertToSnapshot(batchSnapshot)
				for j, gas := range batchExecution {
					status := types.ReceiptStatusSuccessful
					if batchStart+j == i {
						status = types.ReceiptStatusFailed
					}
					ctx.Results[batchStart+j] = types.FrameResult8141{Status: status,
						GasUsed: types.FrameGasLimits{Execution: gas.Execution}}
				}
				for i < len(tx.Frames)-1 && tx.Frames[i].Flags&types.FrameAtomicBatch != 0 {
					i++
					ctx.Results[i] = types.FrameResult8141{Status: types.FrameStatusSkipped}
				}
				batchSnapshot = -1
			} else if frame.Flags&types.FrameAtomicBatch == 0 {
				batchSnapshot = -1
			}
		}
		if prefixOnly && i == prefixEnd {
			deployIndex := 0
			if tx.Frames[0].Target != nil && *tx.Frames[0].Target == types.FrameExpiryVerifier {
				deployIndex++
			}
			if deployIndex < len(tx.Frames) && tx.Frames[deployIndex].Mode == types.ModeDefault && db.GetCodeSize(tx.Sender) == 0 {
				return nil, 0, errors.New("Frame deploy did not install sender code")
			}
			payer, err := evm.CheckFrameApproval8141()
			if err != nil {
				return nil, 0, err
			}
			return &types.Receipt{Payer: payer}, 0, nil
		}
	}
	payer, err := evm.CheckFrameApproval8141()
	if err != nil {
		return nil, 0, err
	}
	var executionGas, stateGas uint64
	for _, frame := range ctx.Results {
		executionGas += frame.GasUsed.Execution
		stateGas += frame.GasUsed.State
	}
	beforeRefund := budget.Intrinsic + executionGas + stateGas
	refund := min(db.GetRefund()-refundBefore, beforeRefund/5)
	used := max(beforeRefund-refund, budget.CalldataFloor+stateGas)
	settlement, err := payment.Settle(used)
	if err != nil {
		return nil, 0, err
	}
	evm.StateDB.AddBalance(payer, settlement.Refund)
	evm.StateDB.AddBalance(header.Coinbase, settlement.Tip)
	if finalizer, ok := db.(interface{ FinalizeTransaction() }); ok {
		finalizer.FinalizeTransaction()
	}
	evm.FlushFrameBAL8141()
	blockExecution := max(beforeRefund-stateGas, budget.CalldataFloor)
	gp.RefundFrame(budget.ExecutionReservation-blockExecution, budget.StateReservation-stateGas)
	receipt := types.NewReceipt(types.ReceiptStatusSuccessful, used)
	receipt.Type, receipt.TxHash = types.FrameTxType, txHash
	receipt.GasUsed, receipt.BlockGasUsed = used, blockExecution
	receipt.Payer, receipt.FrameResults8141 = payer, ctx.Results
	receipt.EffectiveGasPrice = payment.GasPrice()
	receipt.BlobGasUsed = uint64(len(tx.BlobVersionedHashes)) * 131072
	if receipt.BlobGasUsed != 0 {
		receipt.BlobGasPrice = blobFee
	}
	receipt.Logs = db.GetLogs(txHash)
	receipt.Bloom = types.LogsBloom(receipt.Logs)
	return receipt, used, nil
}
