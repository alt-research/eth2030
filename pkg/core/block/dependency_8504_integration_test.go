package block_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func TestDependency8504NativeBuildImportInvalidControlsAndRestart(t *testing.T) {
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.ChainID = big.NewInt(1337)
	cfg.HogotaTime = &zero
	cfg.GlamsterdanTime = &zero
	cfg.AmsterdamTime = &zero
	cfg.EIP8504Time = &zero
	tx := proof8504.Transaction(0)
	alloc := newBuilderRequestTestState()
	alloc.AddBalance(tx.FrameSender(), big.NewInt(1_000_000_000))
	alloc.SetCode(tx.FrameSender(), proof8504.ApproveCode)
	h := makeGenesisInteg(30_000_000, big.NewInt(1)).Header()
	h.Root = alloc.GetRoot()
	h.SlotNumber = &zero
	empty := types.DependencyWitnessesHash8504([][]byte{})
	h.DependencyWitnessesHash = &empty
	genesis := types.NewBlock(h, &types.Body{Withdrawals: []*types.Withdrawal{}, DependencyWitnesses: [][]byte{}})
	path := t.TempDir()
	db, err := rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := chain.NewBlockchain(&cfg, genesis, alloc.Copy(), db)
	if err != nil {
		t.Fatal(err)
	}
	slot := uint64(1)
	built, receipts, err := block.NewBlockBuilder(&cfg, bc, &mockTxPoolInteg{txs: []*types.Transaction{tx}}).BuildBlock(genesis.Header(), &block.BuildBlockAttributes{Timestamp: 12, GasLimit: 30_000_000, SlotNumber: &slot})
	if err != nil || len(built.Transactions()) != 1 || len(receipts) != 1 || len(built.DependencyWitnesses8504()) != 1 {
		t.Fatal("native proof build", err)
	}
	replay := alloc.Copy()
	result, err := execution.NewStateProcessor(&cfg).ProcessWithBAL(built, replay)
	if err != nil || replay.GetRoot() != built.Root() || result.BlockAccessList.Hash() != *built.Header().BlockAccessListHash {
		t.Fatal("proof replay differs", err)
	}
	for _, kind := range []string{"missing", "empty", "bad", "extra", "hash", "wrong_transaction"} {
		header, body := built.Header(), built.Body()
		switch kind {
		case "missing":
			body.DependencyWitnesses = nil
		case "empty":
			body.DependencyWitnesses = [][]byte{}
		case "bad":
			body.DependencyWitnesses[0][0] ^= 1
			hash := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
			header.DependencyWitnessesHash = &hash
		case "extra":
			body.DependencyWitnesses = append(body.DependencyWitnesses, body.DependencyWitnesses[0])
			hash := types.DependencyWitnessesHash8504(body.DependencyWitnesses)
			header.DependencyWitnessesHash = &hash
		case "hash":
			header.DependencyWitnessesHash = &empty
		case "wrong_transaction":
			other := proof8504.Transaction(0)
			other.Inner().(*types.FrameTx8141).Frames[1].Data[32] ^= 1
			body.Transactions = []*types.Transaction{other}
			header.TxHash = block.DeriveTxsRoot(body.Transactions)
		}
		root := bc.State().GetRoot()
		if err := bc.InsertBlock(types.NewBlock(header, body)); err == nil || bc.State().GetRoot() != root {
			t.Fatal("invalid proof block accepted or polluted state", kind, err)
		}
	}
	if err := bc.InsertBlock(built); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = rawdb.NewFileDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reopened, err := chain.NewBlockchain(&cfg, genesis, alloc.Copy(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	loaded := reopened.GetBlock(built.Hash())
	if loaded == nil || len(loaded.DependencyWitnesses8504()) != 1 || !bytes.Equal(loaded.DependencyWitnesses8504()[0], built.DependencyWitnesses8504()[0]) {
		t.Fatal("disk restart lost witness bytes")
	}
	if reopened.CurrentBlock().Hash() != built.Hash() || reopened.State().GetRoot() != built.Root() {
		t.Fatalf("restart head %s / %s root %s / %s", reopened.CurrentBlock().Hash(), built.Hash(), reopened.State().GetRoot(), built.Root())
	}
}
