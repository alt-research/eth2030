package block

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type ePool struct{ txs []*types.Transaction }

func (p *ePool) Pending() []*types.Transaction { return p.txs }
func signedETransaction(t *testing.T, keyIndex int64, nonce uint64, to types.Address, value *big.Int) (*types.Transaction, types.Address, []byte) {
	t.Helper()
	key, e := gethcrypto.ToECDSA(new(big.Int).SetInt64(keyIndex).FillBytes(make([]byte, 32)))
	if e != nil {
		t.Fatal(e)
	}
	dst := gethcommon.Address(to)
	signed, e := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: nonce, Gas: 150000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(1), To: &dst, Value: value}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := signed.MarshalBinary()
	if e != nil {
		t.Fatal(e)
	}
	tx, e := types.DecodeBlockTransaction(raw)
	if e != nil {
		t.Fatal(e)
	}
	return tx, types.Address(gethcrypto.PubkeyToAddress(key.PublicKey)), raw
}
func eBuilder(t *testing.T, db state.StateDB, pool *ePool) (*BlockBuilder, *types.Header, *BuildBlockAttributes, *config.ChainConfig) {
	t.Helper()
	cfg := *config.TestConfigHogota
	cfg.ChainID = big.NewInt(1337)
	builder := NewBlockBuilder(&cfg, nil, pool)
	builder.SetState(db)
	parent := &types.Header{Number: big.NewInt(0), GasLimit: 30000000, BaseFee: big.NewInt(10)}
	slot := uint64(1)
	attrs := &BuildBlockAttributes{Timestamp: 12, GasLimit: 30000000, SlotNumber: &slot}
	return builder, parent, attrs, &cfg
}
func verifyEBuild(t *testing.T, b *types.Block, cfg *config.ChainConfig, before state.StateDB, parent *types.Header) {
	t.Helper()
	getHash := func(n uint64) types.Hash {
		if n == parent.Number.Uint64() {
			return parent.Hash()
		}
		return types.Hash{}
	}
	processor := execution.NewStateProcessor(cfg).WithGetHash(getHash)
	result, e := processor.ProcessWithBAL(b, before)
	if e != nil {
		t.Fatal("independent block reexecution", e)
	}
	if before.GetRoot() != b.Root() || types.BlockGasTotal(result.Receipts, true) != b.GasUsed() {
		t.Fatal("builder post-state/accounting diverged")
	}
}
func TestGroupEBuilderReverseNonceILCascade(t *testing.T) {
	db := newBuilderRequestTestState()
	target := types.Address{19: 42}
	var raw [][]byte
	var sender types.Address
	for i := uint64(0); i < 24; i++ {
		_, from, data := signedETransaction(t, 1, 23-i, target, big.NewInt(1))
		sender = from
		raw = append(raw, data)
	}
	db.AddBalance(sender, big.NewInt(1000000000000))
	before := db.Dup()
	builder, parent, attrs, cfg := eBuilder(t, db, &ePool{})
	attrs.InclusionListTxs = raw
	block, _, e := builder.BuildBlock(parent, attrs)
	if e != nil {
		t.Fatal(e)
	}
	if len(block.Transactions()) != 24 {
		t.Fatal("IL not closed under nonce changes", len(block.Transactions()))
	}
	for i, tx := range block.Transactions() {
		if tx.Nonce() != uint64(i) {
			t.Fatal("nonce order")
		}
	}
	verifyEBuild(t, block, cfg, before, parent)
}
func TestGroupEBuilderPoolFundingReopensILAndDeduplicates(t *testing.T) {
	db := newBuilderRequestTestState()
	target := types.Address{19: 42}
	spend, sender, raw := signedETransaction(t, 1, 0, target, big.NewInt(1))
	fund, funder, _ := signedETransaction(t, 2, 0, sender, big.NewInt(100000000))
	db.AddBalance(funder, big.NewInt(1000000000000))
	db.AddBalance(sender, big.NewInt(1))
	before := db.Dup()
	builder, parent, attrs, cfg := eBuilder(t, db, &ePool{[]*types.Transaction{fund, spend}})
	attrs.InclusionListTxs = [][]byte{raw, {0xff}, raw}
	block, _, e := builder.BuildBlock(parent, attrs)
	if e != nil {
		t.Fatal(e)
	}
	if len(block.Transactions()) != 2 || block.Transactions()[0].Hash() != fund.Hash() || block.Transactions()[1].Hash() != spend.Hash() {
		t.Fatal("funded IL not retried or duplicated", len(block.Transactions()), "balance", db.GetBalance(sender), "nonce", db.GetNonce(sender), "txs", block.Transactions())
	}
	verifyEBuild(t, block, cfg, before, parent)
}

type cancelCodeDB struct {
	state.StateDB
	target types.Address
	cancel context.CancelFunc
	once   sync.Once
}

func (d *cancelCodeDB) GetCode(a types.Address) []byte {
	if a == d.target {
		d.once.Do(d.cancel)
	}
	return d.StateDB.GetCode(a)
}
func TestGroupEBuilderCancellationIsNotAnOmissionExcuse(t *testing.T) {
	db := newBuilderRequestTestState()
	target := types.Address{19: 42}
	tx, sender, _ := signedETransaction(t, 1, 0, target, new(big.Int))
	db.AddBalance(sender, big.NewInt(1000000000000))
	db.SetCode(target, []byte{0x5b, 0x5f, 0x56})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watched := &cancelCodeDB{StateDB: db, target: target, cancel: cancel}
	builder, parent, attrs, _ := eBuilder(t, watched, &ePool{[]*types.Transaction{tx}})
	if block, _, e := builder.BuildBlockWithResultContext(ctx, parent, attrs); !errors.Is(e, context.Canceled) || block != nil {
		t.Fatal("cancelled build published a payload", block, e)
	}
	if db.GetNonce(sender) != 0 {
		t.Fatal("cancelled trial left state changes")
	}
}
