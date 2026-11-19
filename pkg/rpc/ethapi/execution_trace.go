package ethapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpcbackend "github.com/eth2030/eth2030/rpc/backend"
	"github.com/eth2030/eth2030/rpc/debugapi"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func executionTraceOptions(config debugapi.DbgTraceConfig) (rpctypes.TraceOptions, error) {
	options := rpctypes.TraceOptions{Tracer: config.Tracer, Timeout: config.Timeout, DisableStack: config.DisableStack, DisableStorage: config.DisableStorage, EnableReturnData: config.EnableReturnData, EnableMemory: config.EnableMemory && !config.DisableMemory, Limit: config.Limit}
	if len(config.TracerConfig) > 0 {
		var named struct {
			OnlyTopCall bool `json:"onlyTopCall"`
			WithLog     bool `json:"withLog"`
		}
		if err := json.Unmarshal(config.TracerConfig, &named); err != nil {
			return options, err
		}
		options.OnlyTopCall, options.WithLog = named.OnlyTopCall, named.WithLog
	}
	return options, nil
}
func traceResponseError(id json.RawMessage, err error) *Response {
	var detail *rpctypes.RPCError
	if errors.As(err, &detail) {
		return &Response{JSONRPC: "2.0", ID: id, Error: detail}
	}
	return errorResponse(id, ErrCodeExecution, err.Error())
}
func traceFrameResult(frame *vm.CallTraceFrame) *debugapi.DbgCallFrame {
	if frame == nil {
		return nil
	}
	output := &debugapi.DbgCallFrame{Type: frame.Type.String(), From: encodeAddress(frame.From), Gas: encodeUint64(frame.Gas), GasUsed: encodeUint64(frame.GasUsed), Input: encodeBytes(frame.Input)}
	if frame.To != nil {
		output.To = encodeAddress(*frame.To)
	}
	if frame.Value != nil {
		output.Value = encodeBigInt(frame.Value)
	}
	if len(frame.Output) > 0 {
		output.Output = encodeBytes(frame.Output)
	}
	if frame.Err != nil {
		output.Error = frame.Err.Error()
		if errors.Is(frame.Err, vm.ErrExecutionReverted) {
			output.Error = "execution reverted"
			if reason, err := abi.UnpackRevert(frame.Output); err == nil && reason != "" {
				output.RevertReason = reason
			}
		}
	}
	if frame.Reservoir != nil {
		value := encodeUint64(*frame.Reservoir)
		output.StateGasReservoir = &value
	}
	if frame.Usage != nil {
		regular, state, refund := encodeUint64(frame.Usage.RegularGasUsed), encodeUint64(frame.Usage.StateGasUsed), encodeUint64(frame.Usage.GasRefund)
		output.RegularGasUsed = &regular
		output.StateGasUsed = &state
		output.GasRefund = &refund
	}
	for _, child := range frame.Calls {
		output.Calls = append(output.Calls, traceFrameResult(child))
	}
	for _, event := range frame.Logs {
		topics := make([]string, len(event.Log.Topics))
		for i, topic := range event.Log.Topics {
			topics[i] = encodeHash(topic)
		}
		output.Logs = append(output.Logs, &debugapi.DbgCallLog{Address: encodeAddress(event.Log.Address), Topics: topics, Data: encodeBytes(event.Log.Data), Position: encodeUint64(event.Position), Index: encodeUint64(event.Index)})
	}
	return output
}
func opcodeTraceResult(tracer *vm.StructLogTracer) *TraceResult {
	result := &TraceResult{Gas: tracer.GasUsed(), Failed: tracer.Error() != nil, ReturnValue: encodeBytes(tracer.Output()), StructLogs: make([]StructLog, len(tracer.Logs))}
	if tracer.Usage != nil {
		regular, state, refund := tracer.Usage.RegularGasUsed, tracer.Usage.StateGasUsed, tracer.Usage.GasRefund
		result.RegularGasUsed = &regular
		result.StateGasUsed = &state
		result.GasRefund = &refund
	}
	for i, entry := range tracer.Logs {
		stack := make([]string, len(entry.Stack))
		for j, value := range entry.Stack {
			stack[j] = "0x" + value.Text(16)
		}
		step := StructLog{PC: entry.Pc, Op: entry.Op.String(), Gas: entry.Gas, GasCost: entry.GasCost, Depth: entry.Depth, Stack: stack, DisableStack: tracer.DisableStack, EnableMemory: tracer.EnableMemory}
		refund := entry.Refund
		step.Refund = &refund
		if entry.Err != nil {
			step.Error = entry.Err.Error()
		}
		if tracer.EnableReturnData {
			value := encodeBytes(entry.ReturnData)
			step.ReturnData = &value
		}
		if tracer.EnableMemory {
			step.Memory = make([]string, 0, (len(entry.Memory)+31)/32)
			for start := 0; start < len(entry.Memory); start += 32 {
				word := make([]byte, 32)
				copy(word, entry.Memory[start:min(start+32, len(entry.Memory))])
				step.Memory = append(step.Memory, "0x"+hex.EncodeToString(word))
			}
		}
		if entry.Storage != nil {
			step.Storage = make(map[string]string, len(entry.Storage))
			for key, value := range entry.Storage {
				step.Storage[encodeHash(key)] = encodeHash(value)
			}
		}
		if entry.StateGasCost != 0 {
			value := entry.StateGasCost
			step.StateGasCost = &value
		}
		if entry.StateGasReservoir != 0 {
			value := entry.StateGasReservoir
			step.StateGasReservoir = &value
		}
		result.StructLogs[i] = step
	}
	return result
}
func executionTraceResult(logger vm.EVMLogger) (any, error) {
	switch value := logger.(type) {
	case *vm.CallTracer:
		if value.Root == nil {
			return nil, fmt.Errorf("missing call root")
		}
		return traceFrameResult(value.Root), nil
	case *vm.StructLogTracer:
		return opcodeTraceResult(value), nil
	default:
		return nil, fmt.Errorf("unsupported execution tracer")
	}
}

func (api *EthAPI) detailedTraceCall(req *Request, backend rpcbackend.ExecutionTraceBackend) *Response {
	if len(req.Params) < 1 || len(req.Params) > 3 {
		return errorResponse(req.ID, ErrCodeInvalidParams, "expected call, optional block and trace options")
	}
	var args CallArgs
	if err := json.Unmarshal(req.Params[0], &args); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
	}
	header := api.backend.CurrentHeader()
	if len(req.Params) > 1 {
		var text string
		if err := json.Unmarshal(req.Params[1], &text); err != nil {
			return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
		}
		if len(text) == 66 {
			header = api.backend.HeaderByHash(types.HexToHash(text))
		} else {
			var number BlockNumber
			if err := json.Unmarshal(req.Params[1], &number); err != nil {
				return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
			}
			header = api.backend.HeaderByNumber(number)
		}
	}
	if header == nil {
		return errorResponse(req.ID, ErrCodeInternal, "block not found")
	}
	config := debugapi.DefaultDbgTraceConfig()
	var extra struct {
		StateOverrides StateOverride                    `json:"stateOverrides"`
		BlockOverrides *rpctypes.SimulateBlockOverrides `json:"blockOverrides"`
	}
	if len(req.Params) > 2 {
		if err := json.Unmarshal(req.Params[2], &config); err != nil {
			return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
		}
		if err := json.Unmarshal(req.Params[2], &extra); err != nil {
			return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
		}
	}
	options, err := executionTraceOptions(config)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, err.Error())
	}
	options.StateOverrides, options.BlockOverrides = extra.StateOverrides, extra.BlockOverrides
	logger, err := backend.TraceCallDetailed(args, header, options)
	if err != nil {
		return traceResponseError(req.ID, err)
	}
	result, err := executionTraceResult(logger)
	if err != nil {
		return traceResponseError(req.ID, err)
	}
	return successResponse(req.ID, result)
}
