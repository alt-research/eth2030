package vm

import (
	"bytes"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// EVMLogger captures EVM execution traces step by step.
type EVMLogger interface {
	// CaptureStart is called at the beginning of a top-level call.
	CaptureStart(from, to types.Address, create bool, input []byte, gas uint64, value *big.Int)
	// CaptureState is called before each opcode execution.
	CaptureState(pc uint64, op OpCode, gas, cost uint64, stack *Stack, memory *Memory, depth int, err error)
	// CaptureEnd is called at the end of a top-level call.
	CaptureEnd(output []byte, gasUsed uint64, err error)
}

// StructLogEntry is a single step recorded by StructLogTracer.
type StructLogEntry struct {
	Pc                uint64
	Op                OpCode
	Gas               uint64
	GasCost           uint64
	Depth             int
	Stack             []*big.Int
	Err               error
	StateGasCost      int64
	StateGasReservoir uint64
	Memory            []byte
	ReturnData        []byte
	Refund            uint64
	Storage           map[types.Hash]types.Hash
}

// StructLogTracer collects step-by-step EVM execution logs.
type StructLogTracer struct {
	captureBytes     uint64
	captureError     error
	DisableStack     bool
	DisableStorage   bool
	EnableMemory     bool
	EnableReturnData bool
	Limit            int
	stepAddress      types.Address
	stepKey          *types.Hash
	stepReturnData   []byte
	stepRefund       uint64
	lastStep         int
	storage          map[types.Address]map[types.Hash]types.Hash

	Usage   *TraceGasUsage
	Logs    []StructLogEntry
	output  []byte
	err     error
	gasUsed uint64
}

// NewStructLogTracer returns a new StructLogTracer.
func NewStructLogTracer() *StructLogTracer {
	return &StructLogTracer{}
}

// CaptureStart records the start of a top-level call.
func (t *StructLogTracer) CaptureStart(from, to types.Address, create bool, input []byte, gas uint64, value *big.Int) {
	// Nothing to record at start; metadata is implicit in the trace result.
}

// CaptureState records one opcode step.
func (t *StructLogTracer) CaptureState(pc uint64, op OpCode, gas, cost uint64, stack *Stack, memory *Memory, depth int, err error) {
	// Copy the stack to avoid aliasing with mutable big.Ints.
	if t.captureError != nil {
		t.lastStep = -1
		return
	}
	if t.Limit > 0 && len(t.Logs) >= t.Limit {
		t.lastStep = -1
		return
	}
	size := uint64(192)
	if !t.DisableStack {
		size += uint64(stack.Len()) * 80
	}
	if t.EnableMemory {
		size += uint64(memory.Len())
	}
	if t.EnableReturnData {
		size += uint64(len(t.stepReturnData))
	}
	if !reserveTraceBytes(&t.captureBytes, size) {
		t.captureError = ErrTraceCaptureLimit
		t.lastStep = -1
		return
	}
	data := stack.Data()
	if t.DisableStack {
		data = nil
	}
	stackCopy := make([]*big.Int, len(data))
	for i, v := range data {
		stackCopy[i] = new(big.Int).Set(v)
	}

	t.lastStep = len(t.Logs)
	t.Logs = append(t.Logs, StructLogEntry{
		Pc:      pc,
		Op:      op,
		Gas:     gas,
		GasCost: cost,
		Depth:   depth,
		Stack:   stackCopy,
		Err:     err,
	})
}

// CaptureEnd records the end of a top-level call.
func (t *StructLogTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.output = output
	t.gasUsed = gasUsed
	t.err = err
}

// Output returns the return data from the traced execution.
func (t *StructLogTracer) Output() []byte { return t.output }

// GasUsed returns the total gas consumed by the traced execution.
func (t *StructLogTracer) GasUsed() uint64 { return t.gasUsed }

// Error returns the error from the traced execution, if any.
func (t *StructLogTracer) Error() error { return t.err }

func (t *StructLogTracer) CaptureTransactionStart(TraceTransaction) { t.Usage = nil; t.Logs = nil }
func (t *StructLogTracer) CaptureTransactionEnd(usage TraceGasUsage, output []byte, err error, _ int) {
	if usage.Amsterdam {
		copy := usage
		t.Usage = &copy
	}
	t.CaptureEnd(output, usage.UsedGas, err)
}
func (t *StructLogTracer) CaptureStateV2(pc uint64, op OpCode, gas, cost uint64, stateCost int64, reservoir uint64, stack *Stack, memory *Memory, depth int, err error) {
	t.CaptureState(pc, op, gas, cost, stack, memory, depth, err)
	if t.lastStep < 0 {
		return
	}
	entry := &t.Logs[t.lastStep]
	entry.Refund = t.stepRefund
	if t.EnableMemory {
		entry.Memory = bytes.Clone(memory.Data())
	}
	if t.EnableReturnData {
		entry.ReturnData = bytes.Clone(t.stepReturnData)
	}
	entry.StateGasCost = stateCost
	entry.StateGasReservoir = reservoir
}

func (t *StructLogTracer) CaptureOpcodeContext(evm *EVM, contract *Contract, op OpCode, stack *Stack) {
	t.stepAddress = contract.Address
	t.stepKey = nil
	t.stepReturnData = evm.returnData
	t.stepRefund = 0
	if evm.StateDB != nil {
		t.stepRefund = evm.StateDB.GetRefund()
	}
	if !t.DisableStorage && (op == SLOAD || op == SSTORE) && stack.Len() > 0 {
		key := bigToHash(stack.Back(0))
		t.stepKey = &key
	}
}
func (t *StructLogTracer) CaptureOpcodeResult(evm *EVM, _ *Contract, op OpCode, err error) {
	if t.lastStep < 0 || t.lastStep >= len(t.Logs) {
		return
	}
	entry := &t.Logs[t.lastStep]
	if err != nil {
		entry.Err = err
	}
	if t.DisableStorage || t.stepKey == nil || evm.StateDB == nil || (op != SLOAD && op != SSTORE) {
		return
	}
	if t.storage == nil {
		t.storage = make(map[types.Address]map[types.Hash]types.Hash)
	}
	values := t.storage[t.stepAddress]
	if values == nil {
		values = make(map[types.Hash]types.Hash)
		t.storage[t.stepAddress] = values
	}
	values[*t.stepKey] = evm.StateDB.GetState(t.stepAddress, *t.stepKey)
	if !reserveTraceBytes(&t.captureBytes, uint64(len(values))*128) {
		t.captureError = ErrTraceCaptureLimit
		return
	}
	entry.Storage = make(map[types.Hash]types.Hash, len(values))
	for key, value := range values {
		entry.Storage[key] = value
	}
}

func (t *StructLogTracer) TraceError() error { return t.captureError }
