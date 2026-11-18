package chain

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/crypto"
)

func TestValidatedBALStorageDetectsCorruptionAndPreservesPrunedBody(t *testing.T) {
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.AmsterdamTime, cfg.TerminalTotalDifficulty = &zero, new(big.Int)
	db := rawdb.NewMemoryDB()
	genesis := makeGenesis(30_000_000, big.NewInt(1))
	bc, err := NewBlockchain(&cfg, genesis, state.NewMemoryStateDB(), db)
	if err != nil {
		t.Fatal(err)
	}
	builder := block.NewBlockBuilder(&cfg, bc, overlayTestPool(nil))
	child, _, err := builder.BuildBlockWithResult(genesis.Header(), &block.BuildBlockAttributes{Timestamp: 12, GasLimit: genesis.GasLimit()})
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.InsertBlock(child); err != nil {
		t.Fatal(err)
	}
	encoded, err := bc.BlockAccessList(child.Hash())
	if err != nil || len(encoded) == 0 || crypto.Keccak256Hash(encoded) != *child.Header().BlockAccessListHash {
		t.Fatalf("validated BAL not retained: %x %v", encoded, err)
	}
	corrupt := append([]byte(nil), encoded...)
	corrupt[0] ^= 1
	if err := db.Put(blockAccessListKey(child.Hash()), corrupt); err != nil {
		t.Fatal(err)
	}
	if _, err := bc.BlockAccessList(child.Hash()); err == nil {
		t.Fatal("served a corrupted historical BAL")
	}
	if err := db.Delete(blockAccessListKey(child.Hash())); err != nil {
		t.Fatal(err)
	}
	if encoded, err := bc.BlockAccessList(child.Hash()); err != nil || encoded != nil || bc.GetBlock(child.Hash()) == nil {
		t.Fatal("a pruned BAL must leave the execution body available")
	}
}
