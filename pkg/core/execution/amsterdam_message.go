package execution

import (
	"errors"
	"math"
	"math/big"

	corconfig "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
)

// applyAmsterdamMessage is the canonical ordinary-transaction transition. The
// caller supplies the shared block/precompile/blob context and BAL reconciliation.
// Older forks and independent experimental transaction envelopes retain their
// existing entrypoints; ordinary Amsterdam transactions always use this path.
func applyAmsterdamMessage(cfg *corconfig.ChainConfig, db state.StateDB, header *types.Header, msg *corconfig.Message, gp *gaspool.GasPool, evm *vm.EVM, precompiles map[types.Address]vm.PrecompiledContract, blobCost, blobMaxCost *big.Int, options messageOptions, trackers []vm.BALTracker) (result *ExecutionResult, err error) {
	value := msg.Value
	if value == nil {
		value = new(big.Int)
	}
	if value.Sign() < 0 || value.BitLen() > 256 {
		return nil, errors.New("invalid transaction value")
	}
	if msg.Nonce == math.MaxUint64 {
		return nil, errors.New("sender nonce overflow")
	}
	nonce := db.GetNonce(msg.From)
	if nonce != msg.Nonce {
		if msg.Nonce < nonce {
			return nil, ErrNonceTooLow
		}
		return nil, ErrNonceTooHigh
	}
	code := db.GetCode(msg.From)
	_, validSenderDelegation := types.ParseDelegation(code)
	if !options.simulation && len(code) > 0 && !validSenderDelegation {
		return nil, errors.New("sender not an EOA")
	}
	if msg.TxType == types.SetCodeTxType && (msg.To == nil || len(msg.AuthList) == 0) {
		return nil, errors.New("set-code transaction requires destination and authorizations")
	}
	for _, auth := range msg.AuthList {
		for _, field := range []*big.Int{auth.ChainID, auth.R, auth.S} {
			if field != nil && (field.Sign() < 0 || field.BitLen() > 256) {
				return nil, errors.New("authorization field exceeds uint256")
			}
		}
		if auth.V != nil && (auth.V.Sign() < 0 || auth.V.BitLen() > 8) {
			return nil, errors.New("authorization parity exceeds uint8")
		}
	}
	feeCap, tipCap := msg.GasFeeCap, msg.GasTipCap
	if feeCap == nil {
		feeCap = msg.GasPrice
	}
	if tipCap == nil {
		tipCap = msg.GasPrice
	}
	if feeCap == nil {
		feeCap = new(big.Int)
	}
	if tipCap == nil {
		tipCap = new(big.Int)
	}
	if feeCap.Sign() < 0 || tipCap.Sign() < 0 || feeCap.BitLen() > 256 || tipCap.BitLen() > 256 || tipCap.Cmp(feeCap) > 0 {
		return nil, errors.New("invalid transaction fee caps")
	}
	if header.BaseFee != nil && feeCap.Cmp(header.BaseFee) < 0 && !(options.simulation && feeCap.Sign() == 0 && tipCap.Sign() == 0) {
		return nil, errors.New("max fee per gas less than block base fee")
	}
	intrinsic, err := gas.CalculateAmsterdamIntrinsic(msg.From, msg.To, value, msg.Data, msg.AccessList, uint64(len(msg.AuthList)))
	if err != nil {
		return nil, err
	}
	executionLeft, reservoir, err := intrinsic.ExecutionBudget(msg.GasLimit)
	if options.uncapped {
		err = nil
		if msg.GasLimit < max(intrinsic.Gas, intrinsic.Floor) {
			return nil, ErrIntrinsicGasTooLow
		}
		executionLeft, reservoir = msg.GasLimit-intrinsic.Gas, 0
	}
	if err != nil {
		return nil, err
	}
	maximum := new(big.Int).Mul(feeCap, new(big.Int).SetUint64(msg.GasLimit))
	maximum.Add(maximum, value)
	maximum.Add(maximum, blobMaxCost)
	calldataCost := new(big.Int)
	if cfg.IsGlamsterdan(header.Time) && header.CalldataExcessGas != nil {
		calldataCost = gas.CalldataGasCost(types.CalldataTokenGas(msg.Data), gas.CalcCalldataBaseFeeFromHeader(header))
		maximum.Add(maximum, calldataCost)
	}
	if db.GetBalance(msg.From).Cmp(maximum) < 0 {
		return nil, ErrInsufficientBalance
	}
	reserveExecution := min(msg.GasLimit, vm.AmsterdamMaxExecutionGas)
	if options.uncapped {
		reserveExecution = msg.GasLimit
	}
	if err := gp.ReserveFrame(reserveExecution, msg.GasLimit); err != nil {
		return nil, err
	}
	included := false
	defer func() {
		if !included {
			gp.RefundFrame(reserveExecution, msg.GasLimit)
		}
	}()
	price := msgEffectiveGasPrice(msg, header.BaseFee)
	upfront := new(big.Int).Mul(price, new(big.Int).SetUint64(msg.GasLimit))
	upfront.Add(upfront, blobCost)
	upfront.Add(upfront, calldataCost)
	db.SubBalance(msg.From, upfront)
	if msg.To != nil {
		db.SetNonce(msg.From, msg.Nonce+1)
	}
	db.ClearTransientStorage()
	if refund := db.GetRefund(); refund != 0 {
		db.SubRefund(refund)
	}
	db.AddAddressToAccessList(msg.From)
	if msg.To != nil {
		db.AddAddressToAccessList(*msg.To)
	}
	db.AddAddressToAccessList(header.Coinbase)
	for address := range precompiles {
		db.AddAddressToAccessList(address)
	}
	for _, entry := range msg.AccessList {
		db.AddAddressToAccessList(entry.Address)
		for _, key := range entry.StorageKeys {
			db.AddSlotToAccessList(entry.Address, key)
		}
	}
	var tracker vm.BALTracker
	if len(trackers) > 0 {
		tracker = trackers[0]
		if tracker != nil {
			evm.SetBALTracker(tracker, 0)
		}
	}
	evm.TxContext.GasPrice = price
	budget := &vm.Contract{Gas: executionLeft, StateGasReservoir: reservoir, StateGasBaseline: reservoir}
	preSnapshot := db.Snapshot()
	preError := processAmsterdamAuthorizations(db, msg, cfg.ChainID, budget, tracker)
	// Only a later frame rollback preserves the authorizations. A pre-execution
	// failure below restores the pre-authorization snapshot and both charge pools.
	if preError == nil {
		budget.CommitStateGas()
	}
	var creationAddress types.Address
	if preError == nil {
		if msg.To != nil {
			if tracker != nil {
				tracker.RecordAddressTouch(*msg.To)
			}
			if value.Sign() > 0 && db.Empty(*msg.To) && !budget.ChargeStateGas(vm.AmsterdamNewAccountStateGas) {
				preError = vm.ErrOutOfGas
			}
			if preError == nil {
				if delegate, ok := types.ParseDelegation(db.GetCode(*msg.To)); ok {
					cost := uint64(100)
					if !db.AddressInAccessList(delegate) {
						cost = vm.AmsterdamColdAccount
						db.AddAddressToAccessList(delegate)
					}
					if !budget.UseGas(cost) {
						preError = vm.ErrOutOfGas
					}
				}
			}
		} else {
			encoded, encodeErr := rlp.EncodeToBytes([]any{msg.From, msg.Nonce})
			if encodeErr != nil {
				return nil, encodeErr
			}
			creationAddress = types.BytesToAddress(crypto.Keccak256(encoded)[12:])
			if tracker != nil {
				tracker.RecordAddressTouch(creationAddress)
			}
			if db.Empty(creationAddress) && !budget.ChargeStateGas(vm.AmsterdamNewAccountStateGas) {
				preError = vm.ErrOutOfGas
			}
		}
	}
	var output []byte
	var created types.Address
	var executionError error
	if preError != nil {
		db.RevertToSnapshot(preSnapshot)
		if msg.To == nil {
			db.SetNonce(msg.From, msg.Nonce+1)
		}
		budget.RestorePreExecutionStateGas(reservoir)
		budget.Gas = 0
		executionError = preError
	} else {
		evm.SetTransactionStateGas(budget)
		if msg.To == nil {
			output, created, _, executionError = evm.Create(msg.From, msg.Data, budget.Gas, value)
		} else {
			output, _, executionError = evm.Call(msg.From, *msg.To, msg.Data, budget.Gas, value)
		}
		if returned := evm.TransactionStateGas(); returned != nil {
			budget = returned
		} else {
			return nil, errors.New("Amsterdam execution omitted gas outcome")
		}
	}
	if budget.StateGasUsed < 0 {
		return nil, errors.New("negative transaction state gas")
	}
	stateUsed := uint64(budget.StateGasUsed)
	if budget.Gas > msg.GasLimit || budget.StateGasReservoir > msg.GasLimit-budget.Gas {
		return nil, errors.New("Amsterdam gas conservation violation")
	}
	gross := msg.GasLimit - budget.Gas - budget.StateGasReservoir
	if stateUsed > gross {
		return nil, errors.New("state gas exceeds transaction consumption")
	}
	refund := min(db.GetRefund(), gross/5)
	paid := max(gross-refund, intrinsic.Floor)
	blockExecution := max(gross-stateUsed, intrinsic.Floor)
	if paid > msg.GasLimit || blockExecution > reserveExecution || stateUsed > msg.GasLimit {
		return nil, errors.New("Amsterdam gas outcome exceeds reservation")
	}
	db.AddBalance(msg.From, new(big.Int).Mul(price, new(big.Int).SetUint64(msg.GasLimit-paid)))
	tip := new(big.Int).Set(price)
	if header.BaseFee != nil {
		tip.Sub(tip, header.BaseFee)
	}
	if tip.Sign() > 0 {
		db.AddBalance(header.Coinbase, new(big.Int).Mul(tip, new(big.Int).SetUint64(paid)))
	}
	gp.RefundFrame(reserveExecution-blockExecution, msg.GasLimit-stateUsed)
	included = true
	evm.FinalizeSelfDestructs()
	return &ExecutionResult{UsedGas: paid, BlockGasUsed: blockExecution, StateGasUsed: stateUsed, Err: executionError, ReturnData: output, ContractAddress: created}, nil
}

func processAmsterdamAuthorizations(db state.StateDB, msg *corconfig.Message, chainID *big.Int, budget *vm.Contract, tracker vm.BALTracker) error {
	written := map[types.Address]bool{msg.From: true}
	if msg.To != nil && msg.Value != nil && msg.Value.Sign() > 0 {
		written[*msg.To] = true
	}
	initialIndicator := make(map[types.Address]bool)
	observed := make(map[types.Address]bool)
	indicatorCharged := make(map[types.Address]bool)
	for i := range msg.AuthList {
		auth := &msg.AuthList[i]
		if auth.ChainID != nil && auth.ChainID.Sign() != 0 && (chainID == nil || auth.ChainID.Cmp(chainID) != 0) {
			continue
		}
		if auth.Nonce == math.MaxUint64 {
			continue
		}
		authority, err := eips.RecoverAuthority(auth)
		if err != nil {
			continue
		}
		db.AddAddressToAccessList(authority)
		if tracker != nil {
			tracker.RecordAddressTouch(authority)
		}
		code := db.GetCode(authority)
		if !observed[authority] {
			_, initialIndicator[authority] = types.ParseDelegation(code)
			observed[authority] = true
		}
		_, validDelegation := types.ParseDelegation(code)
		if len(code) != 0 && !validDelegation {
			continue
		}
		nonce := db.GetNonce(authority)
		if nonce != auth.Nonce {
			continue
		}
		if db.Empty(authority) && !budget.ChargeStateGas(vm.AmsterdamNewAccountStateGas) {
			return vm.ErrOutOfGas
		}
		if !written[authority] {
			if !budget.UseGas(vm.AmsterdamAccountWrite) {
				return vm.ErrOutOfGas
			}
			written[authority] = true
		}
		if auth.Address != (types.Address{}) && !initialIndicator[authority] && !indicatorCharged[authority] {
			if !budget.ChargeStateGas(vm.AmsterdamDelegationStateGas) {
				return vm.ErrOutOfGas
			}
			indicatorCharged[authority] = true
		}
		nextCode := []byte(nil)
		if auth.Address != (types.Address{}) {
			nextCode = types.AddressToDelegation(auth.Address)
		}
		db.SetCode(authority, nextCode)
		db.SetNonce(authority, nonce+1)
		if t, ok := tracker.(interface {
			RecordCodeChange(types.Address, []byte, []byte)
		}); ok {
			t.RecordCodeChange(authority, code, nextCode)
		}
		if t, ok := tracker.(interface {
			RecordNonceChange(types.Address, uint64, uint64)
		}); ok {
			t.RecordNonceChange(authority, nonce, nonce+1)
		}
	}
	return nil
}
