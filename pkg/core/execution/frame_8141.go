package execution

import (
	"errors"
	"fmt"
	"math/big"

	corconfig "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/frameproof"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// applyFrame8141 is shared by payload construction, import and historical replay.
func applyFrame8141(config *corconfig.ChainConfig, getHash vm.GetHashFunc, db state.StateDB,
	header *types.Header, transaction *types.Transaction, gp *gaspool.GasPool, tracker vm.BALTracker, tracing ...vm.Config,
) (_ *types.Receipt, _ uint64, err error) {
	return executeFrame8141(config, getHash, db, header, transaction, gp, tracker, nil, false, nil, tracing...)
}

// ValidateFrame8141Prefix retains the address-only interface for existing callers.
func ValidateFrame8141Prefix(config *corconfig.ChainConfig, db state.StateDB, header *types.Header, tx *types.Transaction, blobBaseFee *big.Int) (types.Address, error) {
	result, err := ValidateFrame8141Dependencies(config, db, header, tx, blobBaseFee, false)
	if err != nil {
		return types.Address{}, err
	}
	return result.Payer, nil
}

// ValidateFrame8141Dependencies validates on an independent snapshot. The local
// paymaster candidate is opt-in and is never identified as upstream bytecode.
func ValidateFrame8141Dependencies(config *corconfig.ChainConfig, db state.StateDB, header *types.Header, tx *types.Transaction, blobBaseFee *big.Int, experimentalPaymasterV1 bool) (*FramePrefixValidation, error) {
	profile, _ := framepay.SelectProfile(experimentalPaymasterV1, false)
	return ValidateFrame8141DependenciesWithPaymaster(config, db, header, tx, blobBaseFee, profile)
}

// ValidateFrame8141DependenciesWithPaymaster pins an explicitly selected local
// PAY runtime profile. Consensus always executes the actual deployed code.
func ValidateFrame8141DependenciesWithPaymaster(config *corconfig.ChainConfig, db state.StateDB, header *types.Header, tx *types.Transaction, blobBaseFee *big.Int, paymasterProfile framepay.Profile) (*FramePrefixValidation, error) {
	if !paymasterProfile.Valid() {
		return nil, errors.New("invalid Frame paymaster profile")
	}
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok || header == nil {
		return nil, errors.New("not a canonical Frame validation context")
	}
	if err := frameproof.VerifyTransaction(tx, config.FrameExtensions(header.Time), true); err != nil {
		return nil, err
	}
	if _, err := frame.ValidationPrefix8141WithExtensions(header.Time, config.FrameExtensions(header.Time)); err != nil {
		return nil, err
	}
	copied := &frameReadSet{StateDB: db.Dup(), sender: frame.Sender, reads: make(map[FrameDependency]struct{})}
	copied.SetTxContext(tx.Hash(), 0)
	for _, f := range frame.Frames {
		target := frame.Sender
		if f.Target != nil {
			target = *f.Target
		}
		if f.Mode == types.ModeVerify && f.Flags == types.FrameApprovePayment && paymasterProfile.Matches(copied.GetCode(target)) {
			if err := paymasterProfile.ValidatePayUsage(f); err != nil {
				return nil, err
			}
		}
	}
	// Check exact code and complete age/storage predicates even when a malicious
	// runtime would return without executing SLOTNUM or SLOAD.
	for _, f := range frame.Frames {
		if !f.IsRecentRootVerifier() {
			continue
		}
		if header.SlotNumber == nil || string(copied.GetCode(types.RecentRootAddress)) != string(types.RecentRootCode) {
			return nil, errors.New("recent-root verifier requires exact runtime and slot")
		}
		refs, _ := types.DecodeRecentRootReferences(f.Data)
		for _, ref := range refs {
			if !ref.IsRecent(*header.SlotNumber) || copied.GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() {
				return nil, errors.New("recent-root dependency is not valid")
			}
		}
	}
	recoveries := make(map[FrameSignerRecovery8151]struct{})
	canonicalHash := tx.SigningHash()
	tracing := vm.Config{OnFrameRecovery8151: func(address types.Address, digest types.Hash, paymaster types.Address, pay bool) {
		if address == frame.Sender {
			return
		}
		if !pay {
			paymaster = types.Address{}
		}
		recoveries[FrameSignerRecovery8151{Address: address, Canonical: digest == canonicalHash, Pay: pay, Paymaster: paymaster}] = struct{}{}
	}, FramePaymasterProfile: paymasterProfile,
		OnFrameLegacyNonceRead: func(addr types.Address) { copied.record(addr, FrameNonceDependency, types.Hash{}) }}
	receipt, _, err := executeFrame8141(config, nil, copied, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil, nil, true, blobBaseFee, tracing)
	if err != nil {
		return nil, err
	}
	if err := copied.checkPublicBalances(frame.Sender, receipt.Payer); err != nil {
		return nil, err
	}
	if paymasterProfile.Matches(copied.GetCode(receipt.Payer)) {
		// The proposed layout explicitly tracks all four slots, including
		// pending rotation. Values in the result are taken from the pre-state.
		for slot := byte(0); slot < 4; slot++ {
			copied.GetState(receipt.Payer, types.Hash{31: slot})
		}
	}
	result := &FramePrefixValidation{Payer: receipt.Payer, Dependencies: make(map[FrameDependency]string, len(copied.reads))}
	for recovery := range recoveries {
		result.SignerRecoveries = append(result.SignerRecoveries, recovery)
	}
	for dep := range copied.reads {
		result.Dependencies[dep] = dep.Value(db)
	}
	if err := errors.Join(state.ReadError(copied), state.ReadError(db)); err != nil {
		return nil, err
	}
	return result, nil
}

func executeFrame8141(config *corconfig.ChainConfig, getHash vm.GetHashFunc, db state.StateDB,
	header *types.Header, transaction *types.Transaction, gp *gaspool.GasPool, tracker vm.BALTracker,
	profile *types.FrameTx8250Delta, prefixOnly bool, blobBaseFee *big.Int, tracing ...vm.Config,
) (_ *types.Receipt, _ uint64, err error) {
	tx := transaction.Inner().(*types.FrameTx8141)
	if config == nil || !config.IsMerge() || !config.IsHogota(header.Time) {
		return nil, 0, errors.New("EIP-8141 requires Bogotá")
	}
	if config.ChainID == nil || tx.ChainID == nil || tx.ChainID.Cmp(config.ChainID) != 0 {
		return nil, 0, errors.New("EIP-8141 chain ID mismatch")
	}
	// Explicit research callers pass a profile; public callers must match the
	// schema selected at this exact block timestamp. No fork-state installation.
	if profile == nil {
		profile = tx.NonceProfile8250()
		if (profile != nil) != config.IsEIP8250(header.Time) {
			return nil, 0, errors.New("Frame nonce schema does not match EIP-8250 activation")
		}
	}
	var legacyNonce uint64
	var budget types.FrameGasBudget
	if profile == nil {
		if err := tx.ValidateWithExtensions(config.FrameExtensions(header.Time)); err != nil {
			return nil, 0, err
		}
		legacyNonce = db.GetNonce(tx.Sender)
		if legacyNonce != tx.Nonce {
			return nil, 0, errors.New("EIP-8141 nonce mismatch")
		}
		budget, err = tx.GasBudget()
	} else {
		if err := profile.ValidateWithExtensions(config.FrameExtensions(header.Time)); err != nil {
			return nil, 0, err
		}
		if reader, ok := db.(interface{ FrameLegacyNonce8250(types.Address) uint64 }); ok && len(profile.NonceKeys) > 0 {
			legacyNonce = reader.FrameLegacyNonce8250(tx.Sender)
		} else {
			legacyNonce = db.GetNonce(tx.Sender)
		}
		if err := validateNonceSet8250(db, profile); err != nil {
			return nil, 0, err
		}
		budget, err = profile.GasBudget()
	}
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
	if profile != nil {
		payment, err = profile.Payment(header.BaseFee, blobFee)
	}
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
	if profile != nil {
		txHash, err = profile.Hash()
		if err != nil {
			return nil, 0, err
		}
	}
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
	vmConfig := vm.Config{}
	if len(tracing) > 0 {
		vmConfig = tracing[0]
	}
	if config.IsEIP7906(header.Time) {
		journal := newAssertionState7906(db, txHash)
		db, vmConfig.FrameAssertionTrace = journal, journal.index
	}
	evm := vm.NewEVMWithState(blockCtx, vm.TxContext{
		GasPrice: payment.GasPrice(), BlobHashes: tx.BlobVersionedHashes,
	}, vmConfig, db)
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
	ctx.NonceProfile, ctx.LegacyNonce = profile, legacyNonce
	ctx.PublicValidation = prefixOnly
	prefixEnd := -1
	if prefixOnly {
		if profile != nil {
			prefixEnd, err = profile.ValidationPrefix8250WithExtensions(header.Time, config.FrameExtensions(header.Time))
		} else {
			prefixEnd, err = tx.ValidationPrefix8141WithExtensions(header.Time, config.FrameExtensions(header.Time))
		}
		if err != nil {
			return nil, 0, err
		}
	}
	bodySnapshot, bodyStart := -1, 0
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
		if config.IsEIP7906(header.Time) && frame.Mode == types.ModeAssert7906 && frameErr != nil {
			if bodySnapshot < 0 {
				return nil, 0, errors.New("ASSERT failed before payment approval")
			}
			retained := append([]types.FrameResult8141(nil), ctx.Results...)
			evm.StateDB.RevertToSnapshot(bodySnapshot)
			for j := bodyStart; j <= i; j++ {
				ctx.Results[j] = types.FrameResult8141{Status: retained[j].Status, GasUsed: types.FrameGasLimits{Execution: retained[j].GasUsed.Execution}}
			}
			for j := i + 1; j < len(tx.Frames); j++ {
				ctx.Results[j] = types.FrameResult8141{Status: types.FrameStatusSkipped}
			}
			break
		}
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
		if bodySnapshot < 0 && ctx.Payer != nil && config.IsEIP7906(header.Time) {
			bodyStart, bodySnapshot = i+1, evm.StateDB.Snapshot()
		}
		if ctx.Payer == nil {
			bodySnapshot = -1
		}
		if prefixOnly && i == prefixEnd {
			if tx.ValidationDeployIndex8141() >= 0 && db.GetCodeSize(tx.Sender) == 0 {
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
	evm.FinalizeSelfDestructs()
	db.Finalise(config.IsEIP158(header.Number))
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

// ReplayFrame8141 executes authenticated raw Frames using consensus semantics.
// The caller owns the disposable state and any bounded observers/tracers.
func ReplayFrame8141(config *corconfig.ChainConfig, getHash vm.GetHashFunc, db state.StateDB, header *types.Header, tx *types.Transaction, blobBaseFee *big.Int, tracing vm.Config) (*types.Receipt, uint64, error) {
	if _, ok := tx.Inner().(*types.FrameTx8141); !ok {
		return nil, 0, errors.New("canonical Frame required")
	}
	db.SetTxContext(tx.Hash(), 0)
	return executeFrame8141(config, getHash, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil, nil, false, blobBaseFee, tracing)
}
