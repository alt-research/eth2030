package backend

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func newExecutionTracer(options rpctypes.TraceOptions) (vm.EVMLogger, error) {
	switch options.Tracer {
	case "":
		if options.Limit < 0 {
			return nil, simulationError(-32602, "negative trace step limit")
		}
		logger := vm.NewStructLogTracer()
		logger.DisableStack = options.DisableStack
		logger.DisableStorage = options.DisableStorage
		logger.EnableMemory = options.EnableMemory
		logger.EnableReturnData = options.EnableReturnData
		logger.Limit = options.Limit
		return logger, nil
	case "callTracer":
		return vm.NewCallTracer(options.OnlyTopCall, options.WithLog), nil
	default:
		return nil, simulationError(-32602, "unsupported tracer %q", options.Tracer)
	}
}
func traceContext(parent context.Context, options rpctypes.TraceOptions) (context.Context, context.CancelFunc, error) {
	duration := 5 * time.Second
	if options.Timeout != "" {
		var err error
		duration, err = time.ParseDuration(options.Timeout)
		if err != nil {
			return nil, nil, simulationError(-32602, "invalid trace timeout")
		}
	}
	ctx, cancel := context.WithTimeout(parent, duration)
	return ctx, cancel, nil
}
func traceCompleted(logger vm.EVMLogger) error {
	if failed, ok := logger.(vm.TraceFailureLogger); ok && failed.TraceError() != nil {
		return failed.TraceError()
	}
	if tracer, ok := logger.(*vm.CallTracer); ok && (tracer.Root == nil || tracer.LogCountMismatch) {
		return fmt.Errorf("incomplete execution frame/log trace")
	}
	return nil
}

func (b *RPCBackend) TraceCallDetailed(args rpctypes.CallArgs, base *types.Header, options rpctypes.TraceOptions) (vm.EVMLogger, error) {
	logger, err := newExecutionTracer(options)
	if err != nil {
		return nil, err
	}
	ctx, cancel, err := traceContext(b.executionContext(), options)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if ctx.Err() != nil {
		return nil, simulationError(-32016, "trace timeout")
	}
	db, err := b.StateAtHeader(base)
	if err != nil {
		return nil, err
	}
	db = db.Dup()
	state.SetReadContext(db, ctx)
	if err := applySimulationOverrides(db, options.StateOverrides); err != nil {
		return nil, err
	}
	header := types.NewBlock(base, nil).Header()
	blobPrice, err := applyTraceBlockOverrides(header, options.BlockOverrides)
	if err != nil {
		return nil, err
	}
	msg, err := parseSimulationMessage(args, header, db, b.node.Blockchain().Config())
	if err != nil {
		return nil, err
	}
	msg.GasLimit = min(msg.GasLimit, max(base.GasLimit, uint64(50_000_000)))
	db.SetTxContext(msg.TxHash, 0)
	runtime, err := simulationPrecompiles(b.node.Blockchain().Config(), header, db, options.StateOverrides)
	if err != nil {
		return nil, err
	}
	getHash := b.traceHashFn(base)
	result, err := execution.SimulateMessageWithConfig(b.node.Blockchain().Config(), getHash, &simulationBalanceState{StateDB: db}, header, msg, new(gaspool.GasPool).AddGas(msg.GasLimit), execution.SimulationConfig{VMConfig: vm.Config{Debug: true, Tracer: logger}, Overrides: execution.EVMOverrides{Context: ctx, Precompiles: runtime, BlobBaseFee: blobPrice}})
	if ctx.Err() != nil {
		return nil, simulationError(-32016, "trace timeout")
	}
	if err != nil {
		return nil, err
	}
	logger.CaptureEnd(result.ReturnData, result.UsedGas, result.Err)
	if err := traceCompleted(logger); err != nil {
		return nil, err
	}
	return logger, nil
}
func applyTraceBlockOverrides(header *types.Header, override *rpctypes.SimulateBlockOverrides) (*big.Int, error) {
	if override == nil {
		return nil, nil
	}
	if override.Withdrawals != nil || override.BeaconRoot != nil {
		return nil, simulationError(-32602, "withdrawals and beaconRoot do not apply to a single traced call")
	}
	for _, item := range []struct {
		raw  *string
		dest *uint64
	}{{override.Time, &header.Time}, {override.GasLimit, &header.GasLimit}} {
		if item.raw != nil {
			value, err := hexutil.DecodeUint64(*item.raw)
			if err != nil {
				return nil, simulationError(-32602, "invalid block override")
			}
			*item.dest = value
		}
	}
	if override.Number != nil {
		number, err := hexutil.DecodeUint64(*override.Number)
		if err != nil {
			return nil, simulationError(-32602, "invalid block number")
		}
		header.Number = new(big.Int).SetUint64(number)
	}
	if override.FeeRecipient != nil {
		raw, err := hexutil.Decode(*override.FeeRecipient)
		if err != nil || len(raw) != 20 {
			return nil, simulationError(-32602, "invalid feeRecipient")
		}
		header.Coinbase = types.BytesToAddress(raw)
	}
	if override.PrevRandao != nil {
		raw, err := hexutil.Decode(*override.PrevRandao)
		if err != nil || len(raw) != 32 {
			return nil, simulationError(-32602, "invalid prevRandao")
		}
		header.MixDigest = types.BytesToHash(raw)
	}
	if override.BaseFeePerGas != nil {
		value, err := hexutil.DecodeBig(*override.BaseFeePerGas)
		if err != nil || value.BitLen() > 256 {
			return nil, simulationError(-32602, "invalid baseFeePerGas")
		}
		header.BaseFee = value
	}
	if override.BlobBaseFee != nil {
		value, err := hexutil.DecodeUint64(*override.BlobBaseFee)
		if err != nil {
			return nil, simulationError(-32602, "invalid blobBaseFee")
		}
		return new(big.Int).SetUint64(value), nil
	}
	return nil, nil
}
func (b *RPCBackend) TraceTransactionDetailed(hash types.Hash, options rpctypes.TraceOptions) (vm.EVMLogger, error) {
	entry, ok := b.node.Blockchain().GetTxLookupEntry(hash)
	if !ok {
		return nil, fmt.Errorf("transaction not found")
	}
	block := b.node.Blockchain().GetBlock(entry.BlockHash)
	if block == nil {
		return nil, fmt.Errorf("block not found")
	}
	results, err := b.replayExecutionTraces(block, &entry.TxIndex, options)
	if err != nil {
		return nil, err
	}
	return results[0], nil
}
func (b *RPCBackend) TraceBlockDetailed(hash types.Hash, options rpctypes.TraceOptions) ([]vm.EVMLogger, error) {
	block := b.node.Blockchain().GetBlock(hash)
	if block == nil {
		return nil, fmt.Errorf("block not found")
	}
	return b.replayExecutionTraces(block, nil, options)
}
func (b *RPCBackend) replayExecutionTraces(block *types.Block, target *uint64, options rpctypes.TraceOptions) ([]vm.EVMLogger, error) {
	if block.NumberU64() == 0 {
		return nil, fmt.Errorf("genesis block cannot be traced")
	}
	if _, err := newExecutionTracer(options); err != nil {
		return nil, err
	}
	ctx, cancel, err := traceContext(b.executionContext(), options)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if ctx.Err() != nil {
		return nil, simulationError(-32016, "trace timeout")
	}
	bc := b.node.Blockchain()
	txs := block.Transactions()
	if len(txs) == 0 {
		return []vm.EVMLogger{}, nil
	}
	last := uint64(len(txs) - 1)
	if target != nil {
		last = *target
	}
	if last >= uint64(len(txs)) {
		return nil, fmt.Errorf("transaction index out of range")
	}
	header := block.Header()
	parent := bc.GetBlock(header.ParentHash)
	if parent == nil {
		return nil, fmt.Errorf("parent block not found")
	}
	db, err := b.StateAtHeader(parent.Header())
	if err != nil {
		return nil, err
	}
	db = db.Dup()
	state.SetReadContext(db, ctx)
	if err := execution.ApplyPreExecutionSystems(bc.Config(), b.traceHashFn(header), db, header, nil, execution.EVMOverrides{Context: ctx}); err != nil {
		return nil, err
	}
	pool := new(gaspool.GasPool).AddGas(header.GasLimit)
	var outputs []vm.EVMLogger
	var logOffset uint64
	for i := uint64(0); i <= last; i++ {
		if ctx.Err() != nil {
			return nil, simulationError(-32016, "trace timeout")
		}
		tx := txs[i]
		db.SetTxContext(tx.Hash(), int(i))
		db.ClearTransientStorage()
		db.SubRefund(db.GetRefund())
		config := vm.Config{Context: ctx}
		var logger vm.EVMLogger
		if target == nil || i == last {
			if tx.Type() > types.SetCodeTxType {
				return nil, fmt.Errorf("transaction envelope has no qualified tracer")
			}
			logger, err = newExecutionTracer(options)
			if err != nil {
				return nil, err
			}
			if calls, ok := logger.(*vm.CallTracer); ok {
				calls.LogOffset = logOffset
			}
			config.Debug = true
			config.Tracer = logger
		}
		receipt, used, err := execution.ApplyTransactionWithVMConfig(bc.Config(), b.traceHashFn(header), db, header, tx, pool, config)
		if ctx.Err() != nil {
			return nil, simulationError(-32016, "trace timeout")
		}
		if err != nil {
			return nil, fmt.Errorf("replay transaction %d: %w", i, err)
		}
		execution.ApplyAAPostExecution(db, tx, receipt, used, nil)
		logOffset += uint64(len(receipt.Logs))
		if logger != nil {
			if err := traceCompleted(logger); err != nil {
				return nil, err
			}
			outputs = append(outputs, logger)
		}
	}
	return outputs, nil
}

func (b *RPCBackend) traceHashFn(base *types.Header) vm.GetHashFunc {
	cache := make(map[uint64]types.Hash)
	return func(number uint64) types.Hash {
		if number >= base.Number.Uint64() || base.Number.Uint64()-number > 256 {
			return types.Hash{}
		}
		if hash, ok := cache[number]; ok {
			return hash
		}
		cursor := base
		for cursor.Number.Uint64() > number {
			parent := b.node.Blockchain().GetBlock(cursor.ParentHash)
			if parent == nil {
				return types.Hash{}
			}
			cursor = parent.Header()
			cache[cursor.Number.Uint64()] = cursor.Hash()
		}
		return cursor.Hash()
	}
}
