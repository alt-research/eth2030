package chain

import (
	"bytes"
	"errors"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fcuPublicationGate struct {
	rawdb.Database
	armed                    bool
	entered, release         chan struct{}
	coldEntered, coldRelease chan struct{}
	coldKey                  []byte
	once, coldOnce           sync.Once
}

func (g *fcuPublicationGate) Get(k []byte) ([]byte, error) {
	if g.armed && bytes.Equal(k, g.coldKey) {
		g.coldOnce.Do(func() { close(g.coldEntered) })
		<-g.coldRelease
		return nil, rawdb.ErrNotFound
	}
	return g.Database.Get(k)
}
func (g *fcuPublicationGate) Error() error { return rawdb.ReadError(g.Database) }
func (g *fcuPublicationGate) NewBatch() rawdb.Batch {
	return &fcuGateBatch{Batch: g.Database.NewBatch(), gate: g}
}

type fcuGateBatch struct {
	rawdb.Batch
	gate *fcuPublicationGate
}

func (b *fcuGateBatch) pause() {
	if b.gate.armed {
		b.gate.once.Do(func() { close(b.gate.entered) })
		<-b.gate.release
	}
}
func (b *fcuGateBatch) Write() error { b.pause(); return b.Batch.Write() }

type fcuGateLocker struct {
	sync.Locker
	batch *fcuGateBatch
}

func (l fcuGateLocker) Lock() { l.batch.pause(); l.Locker.Lock() }
func (b *fcuGateBatch) WriteAndPublish(lock sync.Locker, publish func()) error {
	return rawdb.WriteBatchAndPublish(b.Batch, fcuGateLocker{Locker: lock, batch: b}, publish)
}

func TestFCUDurablePublicationKeepsCommittedReadersResponsive(t *testing.T) {
	full, _, _ := newHistoryChain(t)
	dir := t.TempDir()
	db, err := rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	batch := db.NewBatch()
	it := full.db.NewIterator(nil)
	for it.Next() {
		if err := batch.Put(it.Key(), it.Value()); err != nil {
			t.Fatal(err)
		}
	}
	if err := rawdb.IteratorError(it); err != nil {
		t.Fatal(err)
	}
	it.Release()
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	gate := &fcuPublicationGate{Database: db, entered: make(chan struct{}), release: make(chan struct{}), coldEntered: make(chan struct{}), coldRelease: make(chan struct{})}
	bc, err := NewBlockchain(config.TestConfig, full.genesis, full.execGenesisState.Copy(), gate)
	if err != nil {
		t.Fatal(err)
	}
	old := bc.CurrentBlock()
	child := makeBlockWithState(old, nil, bc.State())
	if err := bc.InsertSyncedBlock(child); err != nil {
		t.Fatal(err)
	}
	missing := types.Hash{0: 0xff, 31: 0x72}
	gate.coldKey = append([]byte("H"), missing[:]...)
	// The real persistent batch has completed its slow portion but has not
	// published its cache/index view; the real writer remains owned. The old
	// chain instead pauses Write while retaining bc.mu, reproducing the capture.
	gate.armed = true
	done := make(chan error, 1)
	go func() { done <- bc.ApplyForkchoice(child.Hash(), old.Hash(), old.Hash()) }()
	var releaseOnce, coldOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(gate.release) }) }
	releaseCold := func() { coldOnce.Do(func() { close(gate.coldRelease) }) }
	defer release()
	defer releaseCold()
	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("durable-publication gate not reached")
	}
	read := make(chan *types.Block, 1)
	go func() { read <- bc.CurrentBlock() }()
	select {
	case got := <-read:
		if got.Hash() != old.Hash() {
			t.Fatal("unpublished head exposed")
		}
	case <-time.After(300 * time.Millisecond):
		release()
		<-done
		t.Fatal("durable FCU I/O retained the chain read lock")
	}
	// CheckStorage must continue seeing the prior committed durable head via
	// the real FileDB cache while that writer is pending.
	storage := make(chan error, 1)
	go func() { storage <- bc.CheckStorage() }()
	select {
	case err := <-storage:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(300 * time.Millisecond):
		release()
		<-done
		t.Fatal("committed head read waits for persistent writer")
	}
	cold := make(chan *types.Block, 1)
	go func() { cold <- bc.GetBlock(missing) }()
	select {
	case <-gate.coldEntered:
	case <-time.After(time.Second):
		t.Fatal("cold lookup did not enter storage")
	}
	release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		releaseCold()
		<-done
		t.Fatal("cold reader retained chain lock and deadlocked publication")
	}
	releaseCold()
	if <-cold != nil {
		t.Fatal("unknown block invented")
	}
	if bc.CurrentBlock().Hash() != child.Hash() || bc.CurrentSafeBlock().Hash() != old.Hash() || bc.CurrentFinalBlock().Hash() != old.Hash() || bc.State().GetRoot() != child.Root() {
		t.Fatal("atomic head/state/checkpoint publication incomplete")
	}
	// Unknown checkpoints are rejected before publication.
	if err := bc.ApplyForkchoice(child.Hash(), old.Hash(), types.Hash{31: 0x33}); err == nil || errors.Is(err, rawdb.ErrNotFound) {
		t.Fatal("unknown checkpoint was not rejected before publication")
	}
	// Force an actual materialization failure of the next head pointer, after
	// execution has persisted successfully. Neither the warm cache nor memory
	// selection may hide the resulting durable-store failure fence.
	grandchild := makeBlockWithState(child, nil, bc.State())
	if err := bc.InsertSyncedBlock(grandchild); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "data", "6862.tmp"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := bc.ApplyForkchoice(grandchild.Hash(), child.Hash(), old.Hash()); err == nil {
		t.Fatal("failed head materialization accepted")
	}
	if bc.CurrentBlock().Hash() != child.Hash() || bc.CurrentSafeBlock().Hash() != old.Hash() || bc.CurrentFinalBlock().Hash() != old.Hash() || bc.State().GetRoot() != child.Root() {
		t.Fatal("failed durable batch published selection")
	}
	if err := bc.CheckStorage(); err == nil {
		t.Fatal("committed cache hid failed materialization")
	}
}

// Counting the resulting batch catches write amplification even with MemoryDB.
type fcuCountingDB struct {
	rawdb.Database
	puts, deletes int
}
type fcuCountingBatch struct {
	rawdb.Batch
	db *fcuCountingDB
}

func (db *fcuCountingDB) NewBatch() rawdb.Batch   { return &fcuCountingBatch{db.Database.NewBatch(), db} }
func (b *fcuCountingBatch) Put(k, v []byte) error { b.db.puts++; return b.Batch.Put(k, v) }
func (b *fcuCountingBatch) Delete(k []byte) error { b.db.deletes++; return b.Batch.Delete(k) }
func TestFCUUnchangedSelectionDoesNotRewriteDurableKeys(t *testing.T) {
	bc, _, _ := newHistoryChain(t)
	db := &fcuCountingDB{Database: bc.db}
	bc.db = db
	head, genesis := bc.CurrentBlock(), bc.Genesis()
	if err := bc.ApplyForkchoice(head.Hash(), genesis.Hash(), genesis.Hash()); err != nil {
		t.Fatal(err)
	}
	db.puts, db.deletes = 0, 0
	if err := bc.ApplyForkchoice(head.Hash(), head.Hash(), genesis.Hash()); err != nil {
		t.Fatal(err)
	}
	if db.puts != 0 || db.deletes != 0 {
		t.Fatalf("unchanged durable selection rewrote %d puts/%d deletes", db.puts, db.deletes)
	}
	if bc.CurrentSafeBlock().Hash() != head.Hash() {
		t.Fatal("memory-only safe change was skipped")
	}
	if err := bc.ApplyForkchoice(head.Hash(), head.Hash(), head.Hash()); err != nil {
		t.Fatal(err)
	}
	if db.puts != 1 || db.deletes != 0 {
		t.Fatalf("finalized-only change wrote %d puts/%d deletes", db.puts, db.deletes)
	}
}
func TestFCUSharedTransactionIsRestoredAfterSameHeightReorg(t *testing.T) {
	st := newBuilderRequestTestState()
	sender, receiver := types.Address{19: 0xaa}, types.Address{19: 0xab}
	st.AddBalance(sender, big.NewInt(10_000_000))
	genesis := makeGenesis(30_000_000, big.NewInt(1))
	db := rawdb.NewMemoryDB()
	bc, err := NewBlockchain(config.TestConfig, genesis, st, db)
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewTransaction(&types.LegacyTx{GasPrice: big.NewInt(7), Gas: 300000, To: &receiver, Value: big.NewInt(100)})
	tx.SetSender(sender)
	a := makeBlockWithState(genesis, []*types.Transaction{tx}, st.Copy())
	h := a.Header()
	h.Extra = []byte("other valid branch")
	b := types.NewBlock(h, a.Body())
	for _, blk := range []*types.Block{a, b} {
		if err := bc.InsertSyncedBlock(blk); err != nil {
			t.Fatal(err)
		}
	}
	for _, blk := range []*types.Block{a, b} {
		if err := bc.ApplyForkchoice(blk.Hash(), genesis.Hash(), genesis.Hash()); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := rawdb.ReadTxLookup(db, tx.Hash()); err != nil || n != 1 {
		t.Fatalf("shared canonical transaction index deleted: %d %v", n, err)
	}
	// Force the disk fallback, rather than merely checking the publication cache.
	bc.mu.Lock()
	delete(bc.txLookup, tx.Hash())
	bc.mu.Unlock()
	if h, n, _, ok := bc.GetTransactionLookup(tx.Hash()); !ok || h != b.Hash() || n != 1 {
		t.Fatal("canonical transaction fallback lost reorg survivor")
	}
}
