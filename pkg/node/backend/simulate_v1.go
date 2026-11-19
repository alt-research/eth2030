package backend

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/eth2030/eth2030/bal"
	coreblock "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpcbackend "github.com/eth2030/eth2030/rpc/backend"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const simulationGasBudget uint64 = 50_000_000
const simulationMaxBlocks = 256
const simulationMaxBlockCalls = 5000
const simulationMaxCalls = 10000

func simulationError(code int, format string, args ...any) error {
	return &rpctypes.RPCError{Code: code, Message: fmt.Sprintf(format, args...)}
}
func simulationValidationError(err error) error {
	switch {
	case errors.Is(err, rpcbackend.ErrInvalidCallArgs):
		return simulationError(-32602, "%v", err)
	case errors.Is(err, execution.ErrNonceTooLow):
		return simulationError(-38010, "%v", err)
	case errors.Is(err, execution.ErrNonceTooHigh):
		return simulationError(-38011, "%v", err)
	case errors.Is(err, execution.ErrInsufficientBalance):
		return simulationError(-38014, "%v", err)
	case errors.Is(err, gaspool.ErrGasPoolExhausted):
		return simulationError(-38015, "%v", err)
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		return simulationError(-38025, "%v", err)
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "base fee"):
		return simulationError(-38012, "%v", err)
	case strings.Contains(text, "intrinsic") || strings.Contains(text, "floor"):
		return simulationError(-38013, "%v", err)
	}
	return simulationError(-32603, "%v", err)
}

// Simulate executes only request-owned state. No pool, forkchoice, raw database,
// canonical block cache or persisted configuration is modified.
func (b *RPCBackend) Simulate(options rpctypes.SimulateOptions, base *types.Header) ([]rpctypes.SimulateBlockResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if len(options.BlockStateCalls) == 0 {
		return nil, simulationError(-32602, "no blocks to simulate")
	}
	if len(options.BlockStateCalls) > simulationMaxBlocks {
		return nil, simulationError(-38026, "too many blocks")
	}
	total := 0
	for _, block := range options.BlockStateCalls {
		if len(block.Calls) > simulationMaxBlockCalls {
			return nil, simulationError(-38026, "too many calls in one block")
		}
		total += len(block.Calls)
		if total > simulationMaxCalls {
			return nil, simulationError(-38026, "too many calls")
		}
	}
	cfg := b.node.Blockchain().Config()
	db, err := b.StateAtHeader(base)
	if err != nil {
		return nil, err
	}
	db = db.Dup()
	parent := types.NewBlock(base, nil).Header()
	headers := map[uint64]*types.Header{base.Number.Uint64(): base}
	getHash := func(number uint64) types.Hash {
		if h := headers[number]; h != nil {
			return h.Hash()
		}
		if number < base.Number.Uint64() {
			if base.Number.Uint64()-number > 256 {
				return types.Hash{}
			}
			cursor := base
			for cursor.Number.Uint64() > number {
				previous := b.node.Blockchain().GetBlock(cursor.ParentHash)
				if previous == nil {
					return types.Hash{}
				}
				cursor = previous.Header()
				headers[cursor.Number.Uint64()] = cursor
			}
			return cursor.Hash()
		}
		return types.Hash{}
	}
	remaining := simulationGasBudget
	var results []rpctypes.SimulateBlockResult
	for _, request := range options.BlockStateCalls {
		number := parent.Number.Uint64() + 1
		if parent.Number.Uint64() == math.MaxUint64 {
			return nil, simulationError(-38020, "block number overflow")
		}
		if request.BlockOverrides != nil && request.BlockOverrides.Number != nil {
			number, err = hexutil.DecodeUint64(*request.BlockOverrides.Number)
			if err != nil {
				return nil, simulationError(-32602, "invalid block number: %v", err)
			}
		}
		if number <= parent.Number.Uint64() {
			return nil, simulationError(-38020, "block number must increase")
		}
		if number-base.Number.Uint64() > simulationMaxBlocks {
			return nil, simulationError(-38026, "too many simulated blocks including gaps")
		}
		for next := parent.Number.Uint64() + 1; next <= number; next++ {
			if err := ctx.Err(); err != nil {
				return nil, simulationError(-32016, "simulation timeout")
			}
			current := rpctypes.SimulateBlock{}
			if next == number {
				current = request
			}
			header, withdrawals, blobPrice, err := makeSimulationHeader(cfg, parent, current.BlockOverrides, options.Validation)
			if err != nil {
				return nil, err
			}
			db, err = execution.PrepareBlockState(cfg, header.Time, db)
			if err != nil {
				return nil, err
			}
			if err := applySimulationOverrides(db, current.StateOverrides); err != nil {
				return nil, simulationValidationError(err)
			}
			overrides := execution.EVMOverrides{BlobBaseFee: blobPrice, Context: ctx}
			overrides.Precompiles, err = simulationPrecompiles(cfg, header, db, current.StateOverrides)
			if err != nil {
				return nil, err
			}
			result, err := b.simulateBlock(cfg, db, header, withdrawals, current.Calls, options, overrides, getHash, &remaining)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
			parent = header
			headers[header.Number.Uint64()] = header
			if next == number {
				break
			}
		}
	}
	return results, nil
}

func makeSimulationHeader(cfg *config.ChainConfig, parent *types.Header, override *rpctypes.SimulateBlockOverrides, validation bool) (*types.Header, []*types.Withdrawal, *big.Int, error) {
	if parent.Time > math.MaxUint64-12 {
		return nil, nil, nil, simulationError(-38021, "timestamp overflow")
	}
	header := &types.Header{ParentHash: parent.Hash(), Number: new(big.Int).Add(parent.Number, big.NewInt(1)), Time: parent.Time + 12, GasLimit: parent.GasLimit, UncleHash: coreblock.EmptyUncleHash, Difficulty: new(big.Int), TxHash: types.EmptyRootHash, ReceiptHash: types.EmptyRootHash}
	if parent.SlotNumber != nil {
		if *parent.SlotNumber == math.MaxUint64 {
			return nil, nil, nil, simulationError(-32602, "slot overflow")
		}
		slot := *parent.SlotNumber + 1
		header.SlotNumber = &slot
	}
	if override != nil {
		for _, item := range []struct {
			raw  *string
			dest *uint64
		}{{override.Time, &header.Time}, {override.GasLimit, &header.GasLimit}} {
			if item.raw != nil {
				value, err := hexutil.DecodeUint64(*item.raw)
				if err != nil {
					return nil, nil, nil, simulationError(-32602, "invalid block quantity: %v", err)
				}
				*item.dest = value
			}
		}
		if override.Number != nil {
			number, err := hexutil.DecodeUint64(*override.Number)
			if err != nil {
				return nil, nil, nil, simulationError(-32602, "invalid block number")
			}
			header.Number.SetUint64(number)
		}
		if override.FeeRecipient != nil {
			raw, err := hexutil.Decode(*override.FeeRecipient)
			if err != nil || len(raw) != 20 {
				return nil, nil, nil, simulationError(-32602, "invalid feeRecipient")
			}
			header.Coinbase = types.BytesToAddress(raw)
		}
		if override.PrevRandao != nil {
			raw, err := hexutil.Decode(*override.PrevRandao)
			if err != nil || len(raw) != 32 {
				return nil, nil, nil, simulationError(-32602, "invalid prevRandao")
			}
			header.MixDigest = types.BytesToHash(raw)
		}
	}
	if header.Time <= parent.Time {
		return nil, nil, nil, simulationError(-38021, "block timestamp must increase")
	}
	if cfg.IsLondon(header.Number) {
		header.BaseFee = new(big.Int)
		if validation {
			header.BaseFee = gas.CalcBaseFee(parent)
		}
	}
	if override != nil && override.BaseFeePerGas != nil {
		value, err := hexutil.DecodeBig(*override.BaseFeePerGas)
		if err != nil || value.BitLen() > 256 {
			return nil, nil, nil, simulationError(-32602, "invalid baseFeePerGas")
		}
		header.BaseFee = value
	}
	blobPrice := new(big.Int)
	if cfg.IsCancun(header.Time) {
		var excess uint64
		if cfg.IsCancun(parent.Time) {
			var err error
			excess, err = gas.CalcExcessBlobGasForHeader(parent, cfg, header.Time)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		zero := uint64(0)
		header.ExcessBlobGas = &excess
		header.BlobGasUsed = &zero
		beacon := types.Hash{}
		header.ParentBeaconRoot = &beacon
		if override != nil && override.BeaconRoot != nil {
			raw, err := hexutil.Decode(*override.BeaconRoot)
			if err != nil || len(raw) != 32 {
				return nil, nil, nil, simulationError(-32602, "invalid beaconRoot")
			}
			beacon = types.BytesToHash(raw)
		}
		if validation {
			blobPrice = gas.BlobBaseFeeForHeader(cfg, header)
		}
	}
	if override != nil && override.BlobBaseFee != nil {
		value, err := hexutil.DecodeUint64(*override.BlobBaseFee)
		if err != nil {
			return nil, nil, nil, simulationError(-32602, "invalid blobBaseFee")
		}
		blobPrice = new(big.Int).SetUint64(value)
	}
	withdrawals := make([]*types.Withdrawal, 0)
	if override != nil && override.Withdrawals != nil {
		for _, wire := range *override.Withdrawals {
			if wire == nil {
				return nil, nil, nil, simulationError(-32602, "null withdrawal")
			}
			index, e1 := hexutil.DecodeUint64(wire.Index)
			validator, e2 := hexutil.DecodeUint64(wire.ValidatorIndex)
			amount, e3 := hexutil.DecodeUint64(wire.Amount)
			address, e4 := hexutil.Decode(wire.Address)
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil || len(address) != 20 {
				return nil, nil, nil, simulationError(-32602, "invalid withdrawal")
			}
			withdrawals = append(withdrawals, &types.Withdrawal{Index: index, ValidatorIndex: validator, Amount: amount, Address: types.BytesToAddress(address)})
		}
	}
	if cfg.IsShanghai(header.Time) {
		root := coreblock.DeriveWithdrawalsRoot(withdrawals)
		header.WithdrawalsHash = &root
	} else if len(withdrawals) > 0 {
		return nil, nil, nil, simulationError(-32602, "withdrawals before Shanghai")
	}
	return header, withdrawals, blobPrice, nil
}

func (b *RPCBackend) simulateBlock(cfg *config.ChainConfig, db state.StateDB, header *types.Header, withdrawals []*types.Withdrawal, calls []rpctypes.CallArgs, options rpctypes.SimulateOptions, overrides execution.EVMOverrides, getHash vm.GetHashFunc, remaining *uint64) (rpctypes.SimulateBlockResult, error) {
	traced := &simulationLogState{StateDB: &simulationBalanceState{StateDB: db}, synthetic: options.TraceTransfers && !cfg.IsAmsterdam(header.Time), snapshots: make(map[int]int)}
	db = traced
	var displayLogs [][]*types.Log
	fail := func(err error) (rpctypes.SimulateBlockResult, error) { return rpctypes.SimulateBlockResult{}, err }
	var blockBAL *bal.BlockAccessList
	var pre *bal.AccessTracker
	if cfg.IsAmsterdam(header.Time) {
		blockBAL = bal.NewBlockAccessList()
		pre = bal.NewTracker()
	}
	if err := execution.ApplyPreExecutionSystems(cfg, getHash, db, header, pre, overrides); err != nil {
		return fail(err)
	}
	if overrides.Context != nil && overrides.Context.Err() != nil {
		return fail(simulationError(-32016, "simulation timeout"))
	}
	merge := func(tracker *bal.AccessTracker, index uint64) {
		if tracker != nil {
			for _, entry := range tracker.Build(index).Entries {
				blockBAL.AddEntry(entry)
			}
		}
	}
	merge(pre, 0)
	pool := new(gaspool.GasPool).AddGas(header.GasLimit)
	var receipts []*types.Receipt
	var transactions []*types.Transaction
	outputs := make([]rpctypes.SimulateCallResult, 0, len(calls))
	var cumulative, blobGasUsed uint64
	for index, input := range calls {
		if overrides.Context != nil && overrides.Context.Err() != nil {
			return fail(simulationError(-32016, "simulation timeout"))
		}
		available := pool.Gas()
		if cfg.IsAmsterdam(header.Time) {
			available = pool.StateGas()
			if !options.Validation || pool.Gas() < vm.AmsterdamMaxExecutionGas {
				available = min(available, pool.Gas())
			}
		}
		requested := available
		if input.Gas != nil {
			var err error
			requested, err = hexutil.DecodeUint64(*input.Gas)
			if err != nil {
				return fail(simulationError(-32602, "invalid gas"))
			}
		}
		if requested > available {
			return fail(simulationError(-38015, "block gas limit reached: remaining %d, requested %d", available, requested))
		}
		requested = min(requested, *remaining)
		encoded := hexutil.EncodeUint64(requested)
		input.Gas = &encoded
		msg, err := parseSimulationMessage(input, header, db, cfg)
		if err != nil {
			return fail(simulationValidationError(err))
		}
		if db.GetNonce(msg.From) == math.MaxUint64 {
			return fail(simulationError(-32603, "sender nonce at maximum"))
		}
		tx := simulationTransaction(cfg, msg, input)
		msg.TxHash = tx.Hash()
		traced.beginCall()
		db.SetTxContext(tx.Hash(), index)
		oldLogs := len(db.GetLogs(tx.Hash()))
		var tracker *bal.AccessTracker
		var balances map[types.Address]*big.Int
		var nonces map[types.Address]uint64
		if blockBAL != nil {
			tracker = bal.NewTracker()
			balances, nonces = execution.CapturePreState(cfg, db, tx, header.Coinbase)
		}
		settings := execution.SimulationConfig{TransactionRules: options.Validation, Validation: options.Validation, Overrides: overrides}
		if tracker != nil {
			settings.Tracker = tracker
		}
		result, err := execution.SimulateMessageWithConfig(cfg, getHash, db, header, msg, pool, settings)
		if err != nil {
			return fail(simulationValidationError(err))
		}
		if overrides.Context != nil && overrides.Context.Err() != nil {
			return fail(simulationError(-32016, "simulation timeout"))
		}
		if result.UsedGas > *remaining {
			return fail(simulationError(-38026, "global simulation gas budget exhausted"))
		}
		*remaining -= result.UsedGas
		status := uint64(1)
		var failure *rpctypes.RPCError
		if result.Err != nil {
			status = 0
			failure = &rpctypes.RPCError{Code: -32015, Message: result.Err.Error()}
			if errors.Is(result.Err, vm.ErrExecutionReverted) {
				failure.Code = 3
				failure.Message = "execution reverted"
				failure.Data = hexutil.Encode(result.ReturnData)
			}
		}
		receipt := types.NewReceipt(status, cumulative+result.UsedGas)
		receipt.Type = tx.Type()
		receipt.TxHash = tx.Hash()
		receipt.GasUsed = result.UsedGas
		receipt.StateGasUsed = result.StateGasUsed
		receipt.BlockGasUsed = result.BlockGasUsed
		receipt.ContractAddress = result.ContractAddress
		receipt.Logs = append([]*types.Log(nil), db.GetLogs(tx.Hash())[oldLogs:]...)
		receipt.Bloom = types.LogsBloom(receipt.Logs)
		cumulative += result.UsedGas
		blobGasUsed += tx.BlobGas()
		outputs = append(outputs, rpctypes.SimulateCallResult{ReturnData: hexutil.Encode(result.Return()), GasUsed: hexutil.EncodeUint64(result.UsedGas), MaxUsedGas: hexutil.EncodeUint64(result.MaxUsedGas), Status: hexutil.EncodeUint64(status), Logs: make([]*rpctypes.RPCLog, 0), Error: failure})
		displayLogs = append(displayLogs, append([]*types.Log(nil), traced.events...))
		receipts = append(receipts, receipt)
		transactions = append(transactions, tx)
		if tracker != nil {
			execution.PopulateTracker(tracker, db, balances, nonces)
			merge(tracker, uint64(index+1))
		}
	}
	requests, err := execution.FinalizeBlockExecution(cfg, getHash, db, header, withdrawals, receipts, blockBAL, overrides)
	if err != nil {
		return fail(err)
	}
	if overrides.Context != nil && overrides.Context.Err() != nil {
		return fail(simulationError(-32016, "simulation timeout"))
	}
	if cfg.IsPrague(header.Time) {
		root := types.ComputeRequestsHash(requests)
		header.RequestsHash = &root
	}
	if blockBAL != nil {
		blockBAL.Sort()
		root := blockBAL.Hash()
		header.BlockAccessListHash = &root
	}
	header.GasUsed = types.BlockGasTotal(receipts, cfg.IsAmsterdam(header.Time) || cfg.IsGlamsterdan(header.Time))
	if header.BlobGasUsed != nil {
		*header.BlobGasUsed = blobGasUsed
	}
	header.Bloom = types.CreateBloom(receipts)
	header.TxHash = coreblock.DeriveTxsRoot(transactions)
	header.ReceiptHash = coreblock.DeriveReceiptsRoot(receipts)
	if err := execution.FinalizeOverlayState(db); err != nil {
		return fail(err)
	}
	header.Root = db.GetRoot()
	block := types.NewBlock(header, &types.Body{Transactions: transactions, Withdrawals: withdrawals})
	types.DeriveReceiptFields(receipts, block.Hash(), header.Number.Uint64(), header.BaseFee, transactions)
	var logIndex uint
	for i, logs := range displayLogs {
		for _, log := range logs {
			log.BlockHash = block.Hash()
			log.BlockNumber = header.Number.Uint64()
			log.BlockTimestamp = header.Time
			log.Index = logIndex
			logIndex++
			outputs[i].Logs = append(outputs[i].Logs, rpctypes.FormatLog(log))
		}
	}
	formatted := rpctypes.FormatBlock(block, options.ReturnFullTransactions, overrides.BlobBaseFee)
	size, err := block.RLPEncodedSize()
	if err != nil {
		return fail(err)
	}
	switch value := formatted.(type) {
	case *rpctypes.RPCBlock:
		value.Size = hexutil.EncodeUint64(uint64(size))
	case *rpctypes.RPCBlockWithTxs:
		value.Size = hexutil.EncodeUint64(uint64(size))
	}
	return rpctypes.SimulateBlockResult{Block: formatted, Calls: outputs}, nil
}

func simulationTransaction(cfg *config.ChainConfig, msg *config.Message, input rpctypes.CallArgs) *types.Transaction {
	var data types.TxData
	cap, tip := msg.GasFeeCap, msg.GasTipCap
	if cap == nil {
		cap = msg.GasPrice
	}
	if tip == nil {
		tip = msg.GasPrice
	}
	switch {
	case msg.TxType == types.SetCodeTxType:
		data = &types.SetCodeTx{ChainID: cfg.ChainID, Nonce: msg.Nonce, GasTipCap: tip, GasFeeCap: cap, Gas: msg.GasLimit, To: *msg.To, Value: msg.Value, Data: msg.Data, AccessList: msg.AccessList, AuthorizationList: msg.AuthList}
	case msg.TxType == types.BlobTxType:
		data = &types.BlobTx{ChainID: cfg.ChainID, Nonce: msg.Nonce, GasTipCap: tip, GasFeeCap: cap, Gas: msg.GasLimit, To: *msg.To, Value: msg.Value, Data: msg.Data, AccessList: msg.AccessList, BlobHashes: msg.BlobHashes, BlobFeeCap: msg.BlobGasFeeCap}
	case input.GasPrice != nil && input.AccessList != nil:
		data = &types.AccessListTx{ChainID: cfg.ChainID, Nonce: msg.Nonce, GasPrice: msg.GasPrice, Gas: msg.GasLimit, To: msg.To, Value: msg.Value, Data: msg.Data, AccessList: msg.AccessList}
	case input.GasPrice != nil:
		data = &types.LegacyTx{Nonce: msg.Nonce, GasPrice: msg.GasPrice, Gas: msg.GasLimit, To: msg.To, Value: msg.Value, Data: msg.Data}
	default:
		data = &types.DynamicFeeTx{ChainID: cfg.ChainID, Nonce: msg.Nonce, GasTipCap: tip, GasFeeCap: cap, Gas: msg.GasLimit, To: msg.To, Value: msg.Value, Data: msg.Data, AccessList: msg.AccessList}
	}
	tx := types.NewTransaction(data)
	tx.SetSender(msg.From)
	return tx
}
