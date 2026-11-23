package beacon

import (
	"github.com/eth2030/eth2030/core/types"
	"testing"
	"time"
)

func TestProgressReadsLastSelectionWhileKnownApplyWaits(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	syncer := NewExecutionSyncer(nil, nil, func(ExecutionTarget) error { close(entered); <-release; return nil })
	old := ExecutionTarget{Head: types.Hash{31: 1}}
	next := ExecutionTarget{Head: types.Hash{31: 2}}
	syncer.Request(old, 10)
	done := make(chan error, 1)
	go func() { done <- syncer.ApplyKnown(next) }()
	<-entered
	result := make(chan [3]uint64, 1)
	go func() {
		active, start, highest := syncer.Progress()
		flag := uint64(0)
		if active {
			flag = 1
		}
		result <- [3]uint64{flag, start, highest}
	}()
	select {
	case got := <-result:
		if got != [3]uint64{1, 10, 0} {
			t.Fatalf("uncommitted progress exposed: %v", got)
		}
	case <-time.After(300 * time.Millisecond):
		close(release)
		<-done
		t.Fatal("progress read waits for known forkchoice durable I/O")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	active, start, highest := syncer.Progress()
	if active || start != 10 || highest != 0 {
		t.Fatal("committed progress not published")
	}
}
