package chain

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
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
	if bc.CurrentBlock().Hash() != active.Hash() {
		t.Fatalf("reopened head = %s, want %s", bc.CurrentBlock().Hash(), active.Hash())
	}
	checkStorage(bc.CurrentBlock(), types.Hash{})
	checkBlockHash(bc.CurrentBlock(), empty.Hash())
	checkBlockHash(side, sideActivation.Hash())
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

	// Restart must reject a missing ancestor even if the new canonical
	// head contains no transactions.
	if err := rawdb.DeleteHeader(db, empty.NumberU64(), empty.Hash()); err != nil {
		t.Fatal(err)
	}
	if _, err := NewBlockchain(&cfg, genesis, state.NewTrieStateDBFromMemory(rawdb.NewMemoryDB(), mem.Copy()), db); !errors.Is(err, ErrBlockNotFound) {
		t.Fatalf("missing ancestor on restart: %v", err)
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
