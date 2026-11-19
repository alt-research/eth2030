package vm

import (
	"errors"
	"testing"
)

func TestTraceCaptureCapacityFailsBeforeRetainingAnotherSnapshot(t *testing.T) {
	logger := NewStructLogTracer()
	logger.EnableMemory = true
	memory := NewMemory()
	memory.Resize(MaxMemorySize)
	stack := NewStack()
	logger.CaptureStateV2(0, MSTORE, 100, 3, 0, 0, stack, memory, 1, nil)
	if logger.TraceError() != nil || len(logger.Logs) != 1 {
		t.Fatal("first valid snapshot was rejected")
	}
	logger.CaptureStateV2(1, MSTORE, 97, 3, 0, 0, stack, memory, 1, nil)
	if !errors.Is(logger.TraceError(), ErrTraceCaptureLimit) || len(logger.Logs) != 1 {
		t.Fatal("capture overflow retained another snapshot or did not fail")
	}
}
