package backend

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type frameSimulationObserver struct {
	frames []rpctypes.FrameSimulationFrame
	tracer *vm.StructLogTracer
	start  int
	bytes  uint64
	err    error
}

func (o *frameSimulationObserver) BeginFrame(index int) {
	if o.tracer != nil {
		o.start = len(o.tracer.Logs)
	}
}
func (o *frameSimulationObserver) EndFrame(index int, output []byte, executionGas, stateGas uint64, err error) {
	if uint64(len(output)) > 1<<20-o.bytes {
		o.err = fmt.Errorf("Frame simulation return data exceeds 1 MiB")
		output = nil
	} else {
		o.bytes += uint64(len(output))
	}
	status := uint64(1)
	reason := ""
	if err != nil {
		status = 0
		reason = err.Error()
	}
	frame := rpctypes.FrameSimulationFrame{Index: hexutil.EncodeUint64(uint64(index)), Status: hexutil.EncodeUint64(status), ExecutionGasUsed: hexutil.EncodeUint64(executionGas), StateGasUsed: hexutil.EncodeUint64(stateGas), ReturnData: hexutil.Encode(output), Logs: []*rpctypes.RPCLog{}, Error: reason}
	if o.tracer != nil {
		logs := make([]map[string]any, 0, len(o.tracer.Logs)-o.start)
		for _, step := range o.tracer.Logs[o.start:] {
			entry := map[string]any{"pc": hexutil.EncodeUint64(step.Pc), "op": step.Op.String(), "gas": hexutil.EncodeUint64(step.Gas), "gasCost": hexutil.EncodeUint64(step.GasCost), "depth": step.Depth}
			if step.Err != nil {
				entry["error"] = step.Err.Error()
			}
			logs = append(logs, entry)
		}
		frame.StructLogs = logs
	}
	o.frames[index] = frame
}

func (b *RPCBackend) SimulateFrame(options rpctypes.FrameSimulationOptions, base *types.Header) (*rpctypes.FrameSimulationResult, error) {
	invalid := func(message string) (*rpctypes.FrameSimulationResult, error) {
		return nil, simulationError(-32602, "%s", message)
	}
	if base == nil {
		return nil, fmt.Errorf("header not found")
	}
	// Bound input before decoding so hex conversion cannot allocate unbounded memory.
	if len(options.RawTransaction) > 2+2*128*1024 {
		return invalid("Frame payload exceeds 128 KiB")
	}
	raw, err := hexutil.Decode(options.RawTransaction)
	if err != nil {
		return invalid("invalid rawTransaction DATA")
	}
	tx, err := types.DecodeTxRLP(raw)
	if err != nil {
		return invalid("invalid canonical Frame envelope: " + err.Error())
	}
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok {
		return invalid("canonical type-6 Frame required")
	}
	plain, err := tx.EncodeRLP()
	if err != nil || !bytes.Equal(raw, plain) {
		return invalid("plain canonical Frame envelope required")
	}
	if err := frame.Validate(); err != nil {
		return invalid("invalid Frame authentication/structure: " + err.Error())
	}
	budget, err := frame.GasBudget()
	if err != nil {
		return invalid(err.Error())
	}
	if budget.ExecutionReservation > 50000000 || budget.StateReservation > 50000000 {
		return invalid("Frame simulation dimensional cap exceeded")
	}
	limit := uint64(1000)
	if options.TraceLimit != nil {
		limit, err = hexutil.DecodeUint64(*options.TraceLimit)
		if err != nil || limit == 0 || limit > 10000 {
			return invalid("traceLimit must be a QUANTITY in 1..10000")
		}
	}
	for _, override := range options.StateOverrides {
		if override.MovePrecompileToAddress != nil {
			return invalid("Frame v1 does not support moving precompiles")
		}
	}
	header := types.NewBlock(base, nil).Header()
	blobFee, err := applyTraceBlockOverrides(header, options.BlockOverrides)
	if err != nil {
		return nil, err
	}
	if options.SlotNumber != nil {
		slot, err := hexutil.DecodeUint64(*options.SlotNumber)
		if err != nil {
			return invalid("invalid slotNumber QUANTITY")
		}
		header.SlotNumber = &slot
	}
	if blobFee == nil {
		blobFee = gas.BlobBaseFeeForHeader(b.node.Blockchain().Config(), header)
	}
	ctx, cancel := context.WithTimeout(b.executionContext(), 5*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, simulationError(-32016, "Frame simulation cancelled")
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
	observer := &frameSimulationObserver{frames: make([]rpctypes.FrameSimulationFrame, len(frame.Frames))}
	for i := range observer.frames {
		observer.frames[i] = rpctypes.FrameSimulationFrame{Index: hexutil.EncodeUint64(uint64(i)), Status: "0x2", ExecutionGasUsed: "0x0", StateGasUsed: "0x0", ReturnData: "0x", Logs: []*rpctypes.RPCLog{}}
	}
	cfg := vm.Config{Context: ctx, FrameObserver: observer}
	if options.Trace {
		observer.tracer = vm.NewStructLogTracer()
		observer.tracer.DisableStack = true
		observer.tracer.DisableStorage = true
		// The general tracer silently truncates at Limit. Retain one bounded
		// sentinel step so this API can reject incomplete traces while accepting
		// a complete trace containing exactly the requested number of steps.
		observer.tracer.Limit = int(limit) + 1
		cfg.Debug = true
		cfg.Tracer = observer.tracer
	}
	receipt, _, executionErr := execution.ReplayFrame8141(b.node.Blockchain().Config(), b.traceHashFn(base), db, header, tx, blobFee, cfg)
	if ctx.Err() != nil {
		return nil, simulationError(-32016, "Frame simulation timeout or cancellation")
	}
	if readErr := state.ReadError(db); readErr != nil {
		return nil, readErr
	}
	if observer.err != nil {
		return nil, observer.err
	}
	if observer.tracer != nil && observer.tracer.TraceError() != nil {
		return nil, observer.tracer.TraceError()
	}
	if observer.tracer != nil && len(observer.tracer.Logs) > int(limit) {
		return nil, vm.ErrTraceCaptureLimit
	}
	hash, _ := frame.SigningHash()
	result := &rpctypes.FrameSimulationResult{Hash: hexutil.Encode(tx.Hash().Bytes()), SigningHash: hexutil.Encode(hash.Bytes()), IntrinsicGas: hexutil.EncodeUint64(budget.Intrinsic), CalldataFloor: hexutil.EncodeUint64(budget.CalldataFloor), ExecutionReservation: hexutil.EncodeUint64(budget.ExecutionReservation), StateReservation: hexutil.EncodeUint64(budget.StateReservation), GasUsed: "0x0", BlobGasUsed: hexutil.EncodeUint64(tx.BlobGas()), BlobCost: hexutil.EncodeBig(new(big.Int).Mul(new(big.Int).SetUint64(tx.BlobGas()), blobFee)), Escrow: "0x0", Refund: "0x0", Tip: "0x0", Frames: observer.frames}
	if executionErr != nil {
		result.Error = executionErr.Error()
		return result, nil
	}
	result.Valid = true
	payer := rpctypes.EncodeAddress(receipt.Payer)
	result.Payer = &payer
	result.GasUsed = hexutil.EncodeUint64(receipt.GasUsed)
	payment, _ := frame.Payment(header.BaseFee, blobFee)
	settlement, _ := payment.Settle(receipt.GasUsed)
	result.Escrow = hexutil.EncodeBig(payment.Escrow())
	result.Refund = hexutil.EncodeBig(settlement.Refund)
	result.Tip = hexutil.EncodeBig(settlement.Tip)
	logIndex := uint(0)
	for i, executed := range receipt.FrameResults8141 {
		result.Frames[i].Status = hexutil.EncodeUint64(executed.Status)
		result.Frames[i].ExecutionGasUsed = hexutil.EncodeUint64(executed.GasUsed.Execution)
		result.Frames[i].StateGasUsed = hexutil.EncodeUint64(executed.GasUsed.State)
		for _, log := range executed.Logs {
			copy := *log
			copy.Index = logIndex
			logIndex++
			result.Frames[i].Logs = append(result.Frames[i].Logs, rpctypes.FormatLog(&copy))
		}
	}
	return result, nil
}
