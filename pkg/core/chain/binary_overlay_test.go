package chain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
)

type overlayTestPool []*types.Transaction

func (p overlayTestPool) Pending() []*types.Transaction { return p }

func TestBinaryOverlayBuildImportReorgAndRestart(t *testing.T) {
	activation := uint64(2)
	conversion := uint64(4)
	cfg := *config.TestConfig
	cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime, cfg.AmsterdamTime = nil, nil, nil, nil
	cfg.BinaryOverlayTime, cfg.BinaryConversionTime, cfg.BinaryConversionStride = &activation, &conversion, 1
	mem := state.NewMemoryStateDB()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	contract := types.Address{19: 0xfe}
	slot := types.Hash{}
	mem.AddBalance(sender, big.NewInt(100_000_000))
	// Store BLOCKHASH(2) at slot one and clear slot zero.
	mem.SetCode(contract, []byte{0x60, 2, 0x40, 0x60, 1, 0x55, 0x60, 0, 0x60, 0, 0x55, 0})
	mem.SetState(contract, slot, types.Hash{31: 44})
	genesis := types.NewBlock(&types.Header{
		Number: new(big.Int), Difficulty: new(big.Int), GasLimit: 30_000_000,
		BaseFee: big.NewInt(1), Root: mem.GetRoot(), UncleHash: block.EmptyUncleHash,
	}, &types.Body{})
	path := t.TempDir()
	db, err := rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := NewBlockchain(&cfg, genesis, state.NewTrieStateDBFromMemory(rawdb.NewMemoryDB(), mem.Copy()), db)
	if err != nil {
		t.Fatal(err)
	}
	build := func(parent *types.Block, timestamp uint64, txs ...*types.Transaction) *types.Block {
		t.Helper()
		builder := block.NewBlockBuilder(&cfg, bc, overlayTestPool(txs))
		result, _, err := builder.BuildBlockWithResult(parent.Header(), &block.BuildBlockAttributes{
			Timestamp: timestamp, GasLimit: parent.GasLimit(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Transactions()) != len(txs) {
			t.Fatalf("built %d transactions, want %d", len(result.Transactions()), len(txs))
		}
		return result
	}
	insert := func(b *types.Block) {
		t.Helper()
		if err := bc.InsertBlock(b); err != nil {
			t.Fatalf("insert block %d: %v", b.NumberU64(), err)
		}
	}

	before := build(genesis, 1)
	insert(before)
	if before.Root() != genesis.Root() {
		t.Fatal("pre-fork root changed without writes")
	}
	empty := build(before, activation)
	if empty.Root() != (types.Hash{}) {
		t.Fatal("activation did not start at the empty overlay root")
	}
	insert(empty)

	legacy := &types.LegacyTx{GasPrice: big.NewInt(10), Gas: 100_000, To: &contract, Value: new(big.Int)}
	legacy.V = new(big.Int).Add(new(big.Int).Mul(cfg.ChainID, big.NewInt(2)), big.NewInt(35))
	hash := types.NewTransaction(legacy).SigningHash()
	sig, err := crypto.Sign(hash[:], key)
	if err != nil {
		t.Fatal(err)
	}
	legacy.V.Add(legacy.V, big.NewInt(int64(sig[64])))
	legacy.R, legacy.S = new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:64])
	tx := types.NewTransaction(legacy)
	sideActivation := build(before, activation+1)
	insert(sideActivation)
	parent := build(empty, conversion)
	insert(parent)
	sideParent := build(sideActivation, conversion)
	insert(sideParent)
	active := build(parent, conversion+1, tx)
	if active.Root() == empty.Root() || active.Root() == before.Root() {
		t.Fatal("transaction failed to update the binary state root")
	}
	forgedHeader := active.Header()
	forgedHeader.Root[0] ^= 1
	forged := types.NewBlock(forgedHeader, active.Body())
	if err := bc.InsertBlock(forged); !errors.Is(err, block.ErrInvalidStateRoot) {
		t.Fatalf("forged overlay root: %v", err)
	}
	if err := bc.Reorg(forged); !errors.Is(err, ErrBlockNotFound) || bc.CurrentBlock().Hash() != parent.Hash() {
		t.Fatalf("reorg changed the head for a rejected block: %v", err)
	}
	insert(active)
	checkStorage := func(b *types.Block, want types.Hash) {
		t.Helper()
		at, err := bc.StateAtBlock(b)
		if err != nil {
			t.Fatal(err)
		}
		if at.GetRoot() != b.Root() || at.GetState(contract, slot) != want {
			t.Fatalf("block %d state root %s, slot %s", b.NumberU64(), at.GetRoot(), at.GetState(contract, slot))
		}
	}
	checkBlockHash := func(b *types.Block, want types.Hash) {
		t.Helper()
		at, err := bc.StateAtBlock(b)
		if err != nil {
			t.Fatal(err)
		}
		if got := at.GetState(contract, types.Hash{31: 1}); got != want {
			t.Fatalf("BLOCKHASH(2) at %s: got %s, want %s", b.Hash(), got, want)
		}
	}
	checkStorage(active, types.Hash{})
	checkBlockHash(active, empty.Hash())
	side := build(sideParent, conversion+1, tx)
	insert(side)
	checkBlockHash(side, sideActivation.Hash())
	for _, blk := range []*types.Block{active, side} {
		data, err := db.Get(overlayCheckpointKey(blk.Hash()))
		if err != nil || len(data) != 81 || types.BytesToHash(data[33:65]) != blk.Root() {
			t.Fatalf("block %s checkpoint: %x, %v", blk.Hash(), data, err)
		}
		if account, slot := binary.BigEndian.Uint64(data[65:73]), binary.BigEndian.Uint64(data[73:81]); account+slot == 0 {
			t.Fatal("conversion cursor did not advance")
		}
	}
	sideRecord, err := db.Get(overlayCheckpointKey(side.Hash()))
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Clone(sideRecord)
	corrupt[len(corrupt)-1] ^= 1
	if err := db.Put(overlayCheckpointKey(side.Hash()), corrupt); err != nil {
		t.Fatal(err)
	}
	if err := bc.Reorg(side); !errors.Is(err, state.ErrBinaryConversionCheckpoint) || bc.CurrentBlock().Hash() != active.Hash() {
		t.Fatalf("corrupt side cursor changed the head: %v", err)
	}
	if err := db.Put(overlayCheckpointKey(side.Hash()), sideRecord); err != nil {
		t.Fatal(err)
	}
	nodeKey := append([]byte(overlayNodePrefix+"eth2030:binary-trie:v1:blake3:"), side.Root().Bytes()...)
	node, err := db.Get(nodeKey)
	if err != nil {
		t.Fatal(err)
	}
	broken := bytes.Clone(node)
	broken[len(broken)-1] ^= 1
	if err := db.Put(nodeKey, broken); err != nil {
		t.Fatal(err)
	}
	if err := bc.Reorg(side); err == nil || bc.CurrentBlock().Hash() != active.Hash() {
		t.Fatalf("corrupt binary nodes changed the head: %v", err)
	}
	if err := db.Put(nodeKey, node); err != nil {
		t.Fatal(err)
	}
	if entry, ok := bc.GetTxLookupEntry(tx.Hash()); !ok || entry.BlockHash != active.Hash() {
		t.Fatalf("side-chain import replaced canonical transaction lookup: %+v", entry)
	}
	if pruned, err := bc.PruneHistory(0); err != nil || pruned != 0 {
		t.Fatalf("overlay history must stay replayable: pruned %d, err %v", pruned, err)
	}
	if err := bc.Reorg(sideParent); err != nil {
		t.Fatal(err)
	}
	checkStorage(bc.CurrentBlock(), types.Hash{31: 44})
	if err := bc.Reorg(active); err != nil {
		t.Fatal(err)
	}
	checkStorage(bc.CurrentBlock(), types.Hash{})
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	bc, err = NewBlockchain(&cfg, genesis, state.NewTrieStateDBFromMemory(rawdb.NewMemoryDB(), mem.Copy()), db)
	if err != nil {
		t.Fatal(err)
	}
	headRecord, err := db.Get(overlayCheckpointKey(active.Hash()))
	if err != nil {
		t.Fatal(err)
	}
	corrupt = bytes.Clone(headRecord)
	corrupt[1] ^= 1
	if err := db.Put(overlayCheckpointKey(active.Hash()), corrupt); err != nil {
		t.Fatal(err)
	}
	if _, err := NewBlockchain(&cfg, genesis, mem.Copy(), db); !errors.Is(err, state.ErrBinaryConversionCheckpoint) {
		t.Fatalf("restart accepted a different frozen root: %v", err)
	}
	// A replay-only database from before this format has no checkpoint.
	if err := db.Delete(overlayCheckpointKey(active.Hash())); err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(validatedKey(active.Hash())); err != nil {
		t.Fatal(err)
	}
	bc, err = NewBlockchain(&cfg, genesis, mem.Copy(), db)
	if err != nil {
		t.Fatalf("restart without a legacy checkpoint: %v", err)
	}
	if !bc.HasValidatedBlock(active.Hash()) {
		t.Fatal("replayed legacy head did not acquire a validation marker")
	}
	if bc.CurrentBlock().Hash() != active.Hash() {
		t.Fatalf("reopened head = %s, want %s", bc.CurrentBlock().Hash(), active.Hash())
	}
	checkStorage(bc.CurrentBlock(), types.Hash{})
	checkBlockHash(bc.CurrentBlock(), empty.Hash())
	checkBlockHash(side, sideActivation.Hash())
	if receipts := bc.GetReceipts(side.Hash()); len(receipts) != 1 || receipts[0].TxHash != tx.Hash() {
		t.Fatalf("side-chain receipt was lost on restart: %+v", receipts)
	}
	// Advance several conversion units. The cleared slot stays cleared when
	// the cursor finally reaches the old contract storage.
	parent = bc.CurrentBlock()
	for i := uint64(0); i < 4; i++ {
		next := build(parent, parent.Time()+1)
		insert(next)
		checkStorage(next, types.Hash{})
		parent = next
	}
	if err := bc.SetHead(side.NumberU64() - 1); err != nil {
		t.Fatal(err)
	}
	if bc.CurrentBlock().Hash() != active.ParentHash() || bc.GetBlockByNumber(parent.NumberU64()) != nil {
		t.Fatal("rewind did not restore the pre-transaction canonical head")
	}
	checkStorage(bc.CurrentBlock(), types.Hash{31: 44})

	// Restart must reject a missing body even for an empty canonical block.
	if err := rawdb.DeleteBody(db, empty.NumberU64(), empty.Hash()); err != nil {
		t.Fatal(err)
	}
	if _, err := NewBlockchain(&cfg, genesis, state.NewTrieStateDBFromMemory(rawdb.NewMemoryDB(), mem.Copy()), db); !errors.Is(err, ErrBlockNotFound) {
		t.Fatalf("missing body on restart: %v", err)
	}
}

func TestBinaryOverlayFirstChildOfTrieGenesis(t *testing.T) {
	cfg := *config.TestConfig
	cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime, cfg.AmsterdamTime = nil, nil, nil, nil
	activation := uint64(1)
	cfg.BinaryOverlayTime = &activation
	mem := state.NewMemoryStateDB()
	addr := types.Address{19: 0xa1}
	mem.SetCode(addr, bytes.Repeat([]byte{0x5b}, 41))
	trieState := state.NewTrieStateDBFromMemory(rawdb.NewMemoryDB(), mem.Copy())
	genesis := types.NewBlock(&types.Header{
		Number: new(big.Int), Difficulty: new(big.Int), GasLimit: 30_000_000,
		BaseFee: big.NewInt(7), Root: mem.GetRoot(), UncleHash: block.EmptyUncleHash,
	}, &types.Body{})
	bc, err := NewBlockchain(&cfg, genesis, trieState, rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := trieState.Commit(); err != nil {
		t.Fatal(err)
	}
	child, _, err := block.NewBlockBuilder(&cfg, bc, nil).BuildBlock(genesis.Header(), &block.BuildBlockAttributes{Timestamp: activation, GasLimit: genesis.GasLimit()})
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.InsertBlock(child); err != nil {
		t.Fatal(err)
	}
	if child.Root() != (types.Hash{}) || !bytes.Equal(bc.State().GetCode(addr), mem.GetCode(addr)) {
		t.Fatal("empty overlay failed to preserve frozen genesis code")
	}
}

var errOverlayBatchWrite = errors.New("injected overlay batch write failure")

type overlayFailDB struct {
	rawdb.Database
	fail bool
}

func (db *overlayFailDB) NewBatch() rawdb.Batch {
	return overlayFailBatch{Batch: db.Database.NewBatch(), fail: db.fail}
}

type overlayFailBatch struct {
	rawdb.Batch
	fail bool
}

func (b overlayFailBatch) Write() error {
	if b.fail {
		return errOverlayBatchWrite
	}
	return b.Batch.Write()
}

func TestBinaryOverlayFailedPublicationCanRetry(t *testing.T) {
	cfg := *config.TestConfig
	cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime, cfg.AmsterdamTime = nil, nil, nil, nil
	activation := uint64(1)
	cfg.BinaryOverlayTime, cfg.BinaryConversionTime, cfg.BinaryConversionStride = &activation, &activation, 1
	mem := state.NewMemoryStateDB()
	mem.AddBalance(types.Address{19: 9}, big.NewInt(100))
	genesis := types.NewBlock(&types.Header{
		Number: new(big.Int), Difficulty: new(big.Int), GasLimit: 30_000_000,
		BaseFee: big.NewInt(7), Root: mem.GetRoot(), UncleHash: block.EmptyUncleHash,
	}, &types.Body{})
	db := &overlayFailDB{Database: rawdb.NewMemoryDB()}
	bc, err := NewBlockchain(&cfg, genesis, mem, db)
	if err != nil {
		t.Fatal(err)
	}
	build := func(time uint64) *types.Block {
		t.Helper()
		blk, _, err := block.NewBlockBuilder(&cfg, bc, nil).BuildBlock(genesis.Header(), &block.BuildBlockAttributes{Timestamp: time, GasLimit: genesis.GasLimit()})
		if err != nil {
			t.Fatal(err)
		}
		return blk
	}
	child := build(1)
	db.fail = true
	if err := bc.InsertBlock(child); !errors.Is(err, errOverlayBatchWrite) {
		t.Fatalf("failed import: %v", err)
	}
	if bc.GetBlock(child.Hash()) != nil || bc.CurrentBlock().Hash() != genesis.Hash() {
		t.Fatal("failed import published a block or changed the head")
	}
	if _, err := db.Get(overlayCheckpointKey(child.Hash())); !errors.Is(err, rawdb.ErrNotFound) {
		t.Fatalf("failed import published the checkpoint: %v", err)
	}
	nodes := db.NewIterator([]byte(overlayNodePrefix))
	hasNodes := nodes.Next()
	nodes.Release()
	if hasNodes {
		t.Fatal("failed import published binary nodes")
	}
	db.fail = false
	if err := bc.InsertBlock(child); err != nil {
		t.Fatal(err)
	}
	side := build(2)
	if err := bc.InsertBlock(side); err != nil {
		t.Fatal(err)
	}
	db.fail = true
	if err := bc.Reorg(side); !errors.Is(err, errOverlayBatchWrite) {
		t.Fatalf("failed reorg: %v", err)
	}
	head, err := rawdb.ReadHeadBlockHash(db)
	if err != nil || head != child.Hash() || bc.CurrentBlock().Hash() != head || bc.GetBlockByNumber(1).Hash() != head {
		t.Fatalf("failed reorg changed the head: disk %s, memory %s, err %v", head, bc.CurrentBlock().Hash(), err)
	}
	db.fail = false
	if err := bc.Reorg(side); err != nil {
		t.Fatal(err)
	}
	if bc.GetBlockByNumber(1).Hash() != side.Hash() {
		t.Fatal("reorg retry did not publish the side branch")
	}
}

func TestPQDraftRejectsFutureForkCollisionAtStartup(t *testing.T) {
	cfg := *config.TestConfig
	activation, iplus := uint64(2), uint64(20)
	cfg.IPlusTime = &iplus
	cfg.PQDraft = &config.PQDraftConfig{
		Time: &activation, SigRecover: types.Address{18: 2, 19: 0x10},
		MLDSA: types.Address{18: 0x70, 19: 2}, MLDSAETH: types.Address{18: 0x70, 19: 3},
	}
	mem := state.NewMemoryStateDB()
	genesis := types.NewBlock(&types.Header{Number: new(big.Int), Root: mem.GetRoot()}, &types.Body{})
	db := rawdb.NewMemoryDB()
	if _, err := NewBlockchain(&cfg, genesis, mem, db); !errors.Is(err, vm.ErrPQDraftAddress) {
		t.Fatalf("future I+ collision: %v", err)
	}
	if _, err := rawdb.ReadCanonicalHash(db, 0); !errors.Is(err, rawdb.ErrNotFound) {
		t.Fatalf("rejected configuration wrote a genesis record: %v", err)
	}
	cfg.PQDraft.SigRecover = types.Address{18: 0x70, 19: 1}
	if _, err := NewBlockchain(&cfg, genesis, mem, db); err != nil {
		t.Fatalf("valid private profile: %v", err)
	}
}
