package vm

import (
	"bytes"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
)

type CallTraceLog struct {
	Log      *types.Log
	Position uint64
	Index    uint64
}
type CallTraceFrame struct {
	Type      OpCode
	From      types.Address
	To        *types.Address
	Value     *big.Int
	Input     []byte
	Output    []byte
	Gas       uint64
	GasUsed   uint64
	Err       error
	Calls     []*CallTraceFrame
	Logs      []*CallTraceLog
	Reservoir *uint64
	Usage     *TraceGasUsage
}
type callTraceScope struct {
	frame      *CallTraceFrame
	eventStart int
}
type CallTracer struct {
	captureBytes     uint64
	captureError     error
	Root             *CallTraceFrame
	OnlyTopCall      bool
	WithLog          bool
	metadata         TraceTransaction
	stack            []callTraceScope
	events           []*CallTraceLog
	LogOffset        uint64
	LogCountMismatch bool
}

func NewCallTracer(onlyTopCall, withLog bool) *CallTracer {
	return &CallTracer{OnlyTopCall: onlyTopCall, WithLog: withLog}
}
func (t *CallTracer) CaptureStart(types.Address, types.Address, bool, []byte, uint64, *big.Int) {}
func (t *CallTracer) CaptureState(uint64, OpCode, uint64, uint64, *Stack, *Memory, int, error)  {}
func (t *CallTracer) CaptureEnd([]byte, uint64, error)                                          {}
func (t *CallTracer) CaptureTransactionStart(info TraceTransaction) {
	t.metadata = info
	t.stack = nil
	t.events = nil
	t.Root = nil
	t.LogCountMismatch = false
	t.captureBytes = 0
	t.captureError = nil
}
func (t *CallTracer) CaptureEnter(entry TraceFrameEntry) {
	if t.captureError != nil {
		return
	}
	size := uint64(64)
	if !t.OnlyTopCall || entry.Depth == 0 {
		size = 512 + uint64(len(entry.Input))
	}
	if !reserveTraceBytes(&t.captureBytes, size) {
		t.captureError = ErrTraceCaptureLimit
		return
	}
	var frame *CallTraceFrame
	if !t.OnlyTopCall || entry.Depth == 0 {
		to := entry.To
		frame = &CallTraceFrame{Type: entry.Type, From: entry.From, To: &to, Input: bytes.Clone(entry.Input), Gas: entry.Gas}
		if entry.Type != STATICCALL {
			frame.Value = new(big.Int)
			if entry.Value != nil {
				frame.Value.Set(entry.Value)
			}
		}
		if entry.Amsterdam {
			reservoir := entry.Reservoir
			frame.Reservoir = &reservoir
		}
		if entry.Depth == 0 {
			frame.Gas = t.metadata.Gas
			t.Root = frame
		} else if len(t.stack) > 0 && t.stack[len(t.stack)-1].frame != nil {
			parent := t.stack[len(t.stack)-1].frame
			parent.Calls = append(parent.Calls, frame)
		}
	}
	t.stack = append(t.stack, callTraceScope{frame: frame, eventStart: len(t.events)})
}
func clearCallLogs(frame *CallTraceFrame) {
	if frame == nil {
		return
	}
	frame.Logs = nil
	for _, child := range frame.Calls {
		clearCallLogs(child)
	}
}
func (t *CallTracer) CaptureExit(depth int, output []byte, left uint64, err error) {
	if t.captureError != nil {
		return
	}
	if len(t.stack) == 0 {
		return
	}
	scope := t.stack[len(t.stack)-1]
	t.stack = t.stack[:len(t.stack)-1]
	if err != nil {
		t.events = t.events[:scope.eventStart]
		clearCallLogs(scope.frame)
	}
	if scope.frame == nil {
		return
	}
	frame := scope.frame
	if !reserveTraceBytes(&t.captureBytes, uint64(len(output))) {
		t.captureError = ErrTraceCaptureLimit
		return
	}
	frame.Err = err
	frame.Output = bytes.Clone(output)
	if frame.Gas >= left {
		frame.GasUsed = frame.Gas - left
	}
	if err != nil && (frame.Type == CREATE || frame.Type == CREATE2) {
		frame.To = nil
	}
}
func (t *CallTracer) CaptureLog(log *types.Log) {
	if t.captureError != nil {
		return
	}
	if len(t.stack) == 0 {
		return
	}
	scope := t.stack[len(t.stack)-1]
	if !reserveTraceBytes(&t.captureBytes, uint64(128+len(log.Data)+len(log.Topics)*32)) {
		t.captureError = ErrTraceCaptureLimit
		return
	}
	var entry *CallTraceLog
	if t.WithLog && scope.frame != nil {
		entry = &CallTraceLog{Log: log, Position: uint64(len(scope.frame.Calls))}
		if t.OnlyTopCall {
			entry.Position = 0
		}
		scope.frame.Logs = append(scope.frame.Logs, entry)
	}
	t.events = append(t.events, entry)
}
func (t *CallTracer) CaptureTransactionEnd(usage TraceGasUsage, output []byte, err error, logCount int) {
	if t.captureError != nil {
		return
	}
	if t.Root == nil {
		typ := CALL
		if t.metadata.To == nil {
			typ = CREATE
		}
		t.Root = &CallTraceFrame{Type: typ, From: t.metadata.From, To: t.metadata.To, Value: new(big.Int), Input: bytes.Clone(t.metadata.Input), Gas: t.metadata.Gas, Err: err, Output: bytes.Clone(output)}
		if t.metadata.Value != nil {
			t.Root.Value.Set(t.metadata.Value)
		}
		if usage.Amsterdam {
			reservoir := t.metadata.Reservoir
			t.Root.Reservoir = &reservoir
		}
	}
	t.Root.Gas = t.metadata.Gas
	t.Root.GasUsed = usage.UsedGas
	t.Root.Err = err
	t.Root.Output = bytes.Clone(output)
	if usage.Amsterdam {
		copy := usage
		t.Root.Usage = &copy
	}
	if err != nil {
		clearCallLogs(t.Root)
	}
	for index, event := range t.events {
		if event != nil {
			event.Index = t.LogOffset + uint64(index)
		}
	}
	t.LogCountMismatch = len(t.events) != logCount
}

func (t *CallTracer) TraceError() error { return t.captureError }
