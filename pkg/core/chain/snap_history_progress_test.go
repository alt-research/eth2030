package chain

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

// Pause the saved-prefix validator after a real database read, outside the
// underlying database lock. The imported tip needs neither this old mapping
// nor permission from a cancelled worker to execute.
type prefixReadBarrier struct {
	rawdb.Database
	parent           types.Hash
	entered, release chan struct{}
	used             atomic.Bool
}

func (db *prefixReadBarrier) Get(key []byte) ([]byte, error) {
	value, err := db.Database.Get(key)
	if err == nil && len(key) == 9 && key[0] == 'c' && bytes.Equal(value, db.parent[:]) && db.used.CompareAndSwap(false, true) {
		close(db.entered)
		<-db.release
	}
	return value, err
}

func TestSnapHistoryPrefixReadDoesNotBlockTipImportOrForkchoice(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "authenticated", true: "cancelled"}[cancel], func(t *testing.T) {
			target, full, _, blocks := snapHistoryFixture(t)
			pivot, parent, child := blocks[4], blocks[3], blocks[5]
			if err := target.CompleteSnapHistoryBlock(pivot.Hash()); err != nil {
				t.Fatal(err)
			}
			if err := target.StageSnapHistoryParent(pivot.Hash(), parent); err != nil {
				t.Fatal(err)
			}
			bal, err := full.BlockAccessList(parent.Hash())
			if err != nil {
				t.Fatal(err)
			}
			if err := target.StoreBlockAccessListRLP(parent.Hash(), bal); err != nil {
				t.Fatal(err)
			}
			if err := target.CompleteSnapHistoryBlock(parent.Hash()); err != nil {
				t.Fatal(err)
			}
			gate := &prefixReadBarrier{Database: target.db, parent: parent.Hash(), entered: make(chan struct{}), release: make(chan struct{})}
			target.db = gate
			var once sync.Once
			release := func() { once.Do(func() { close(gate.release) }) }
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			historical := make(chan error, 1)
			go func() {
				cursor, err := target.SnapHistoryCursor(ctx)
				if err == nil && (cursor == nil || cursor.Hash() != parent.Hash()) {
					err = errors.New("wrong authenticated cursor")
				}
				historical <- err
			}()
			defer func() {
				release()
				err := <-historical
				if cancel {
					if !errors.Is(err, context.Canceled) {
						t.Errorf("cancelled prefix: %v", err)
					}
				} else if err != nil {
					t.Error(err)
				}
			}()
			select {
			case <-gate.entered:
			case <-time.After(time.Second):
				t.Fatal("saved-prefix barrier was not reached")
			}
			if cancel {
				stop()
			}
			imported := make(chan error, 1)
			go func() {
				result, err := target.InsertSyncedBlockWithResult(child)
				if err == nil && result == nil {
					err = errors.New("native execution was skipped")
				}
				if err == nil && target.CurrentBlock().Hash() != pivot.Hash() {
					err = errors.New("import optimistically selected a head")
				}
				if err == nil {
					err = target.ApplyForkchoice(child.Hash(), pivot.Hash(), pivot.Hash())
				}
				imported <- err
			}()
			select {
			case err := <-imported:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				release()
				<-imported
				t.Fatal("saved-prefix read held the execution/FCU mutation lock")
			}
			if target.CurrentBlock().Hash() != child.Hash() || target.State().GetRoot() != child.Root() || target.CurrentFinalBlock().Hash() != pivot.Hash() || target.HasValidatedBlock(parent.Hash()) {
				t.Fatal("progress changed head/state/finality or granted historical execution authority")
			}
		})
	}
}
