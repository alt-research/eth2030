package block_test

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestCanonicalFrame8141PrefundedCreateBuildImport(t *testing.T) {
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.ChainID, cfg.HogotaTime = big.NewInt(1337), &zero
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	sender, creator := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey)), types.Address{19: 0x90}
	destination := vm.CreateAddressFromNonce(creator, 0)
	alloc := newBuilderRequestTestState()
	alloc.AddBalance(sender, big.NewInt(100000000))
	alloc.AddBalance(destination, big.NewInt(7))
	// Copy and CREATE initcode that returns a one-byte STOP, with no endowment.
	alloc.SetCode(creator, []byte{0x60, 7, 0x60, 14, 0x5f, 0x39, 0x60, 7, 0x5f, 0x5f, 0xf0, 0x50, 0, 0,
		0x5f, 0x5f, 0x53, 0x60, 1, 0x5f, 0xf3})
	h := makeGenesisInteg(30000000, big.NewInt(1)).Header()
	h.Root, h.SlotNumber = alloc.GetRoot(), &zero
	genesis := types.NewBlock(h, &types.Body{Withdrawals: []*types.Withdrawal{}})
	bc, err := chain.NewBlockchain(&cfg, genesis, alloc.Copy(), rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	frame := &types.FrameTx8141{ChainID: cfg.ChainID, Sender: sender,
		Fees: types.FrameFeeBudget{MaxFee: big.NewInt(10000000), MaxPriorityFeePerGas: big.NewInt(2)},
		Frames: []types.Frame8141{
			{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}},
			{Mode: types.ModeSender, Target: &creator, Limits: types.FrameGasLimits{Execution: 200000, State: 1530}},
		}}
	tx := signFrameBlock(t, frame, 1)
	slot := uint64(1)
	built, receipts, err := block.NewBlockBuilder(&cfg, bc, &mockTxPoolInteg{txs: []*types.Transaction{tx}}).BuildBlock(genesis.Header(),
		&block.BuildBlockAttributes{Timestamp: 12, GasLimit: genesis.GasLimit(), SlotNumber: &slot})
	if err != nil || len(receipts) != 1 || len(built.Transactions()) != 1 {
		t.Fatalf("builder omitted prefunded creation: %v", err)
	}
	replay := alloc.Copy()
	result, err := execution.NewStateProcessor(&cfg).ProcessWithBAL(built, replay)
	if err != nil {
		t.Fatal(err)
	}
	if replay.GetRoot() != built.Root() || result.BlockAccessList.Hash() != *built.Header().BlockAccessListHash ||
		receipts[0].FrameResults8141[1].GasUsed.State != 1530 || replay.GetBalance(destination).Cmp(big.NewInt(7)) != 0 ||
		replay.GetNonce(destination) != 1 || len(replay.GetCode(destination)) != 1 {
		t.Fatal("builder/replay state, deposit charge or BAL commitment differs")
	}
	seen := false
	for _, entry := range result.BlockAccessList.Entries {
		if entry.Address == destination {
			seen = true
			if entry.BalanceChange != nil || entry.NonceChange == nil || entry.CodeChange == nil {
				t.Fatalf("published BAL must contain creation fields without unchanged prefunding: %+v", entry)
			}
		}
	}
	if !seen {
		t.Fatal("created account missing from published BAL")
	}
	if err := bc.InsertBlock(built); err != nil {
		t.Fatalf("independent native import rejected creation: %v", err)
	}
	if bc.State().GetBalance(destination).Cmp(big.NewInt(7)) != 0 || bc.State().GetRoot() != built.Root() {
		t.Fatal("import lost prefunding or canonical post-state")
	}
}
