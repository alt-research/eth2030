package backend

import (
	"math/big"
	"testing"
	"time"

	blockpkg "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/txpool"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// This is an actually built, imported EIP-7934-sized block, not padding in an
// invalid header. PR #12723's proposed byte-budget omission excuse is unmerged.
func TestGroupELatest7805SizeIsNotAnOmissionExcuse(t *testing.T) {
	const limit = uint64(500000000)
	cfg := *config.TestConfigHogota
	cfg.ChainID = big.NewInt(1337)
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	other, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000002")
	if err != nil {
		t.Fatal(err)
	}
	from := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	to := gethcrypto.PubkeyToAddress(other.PublicKey)
	db := state.NewMemoryStateDB()
	eips.ApplyBuilderRequestGenesis(db)
	db.AddBalance(from, new(big.Int).Lsh(big.NewInt(1), 200))
	db.AddBalance(types.Address(to), new(big.Int).Lsh(big.NewInt(1), 200))
	db.SetNonce(types.Address(to), 1)
	genesis := types.NewBlock(&types.Header{Number: new(big.Int), GasLimit: limit, BaseFee: big.NewInt(1000000000), Root: db.GetRoot()}, &types.Body{})
	bc, err := chain.NewBlockchain(&cfg, genesis, db, rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	n := newTestBackendNode(t)
	n.blockchain = bc
	poolConfig := txpool.DefaultConfig()
	poolConfig.BlockGasLimit, poolConfig.MaxPerSender = limit, 256
	n.txPool = txpool.New(poolConfig, bc.State())
	n.txPool.SetForkRules(&cfg, 12)
	bytes := 0
	for nonce := uint64(0); nonce < 128; nonce++ {
		size := 65536
		if nonce == 127 {
			size = blockpkg.MaxRLPBlockSize - 4096 - bytes - 160
		}
		if size <= 0 || size > 65536 {
			t.Fatal("invalid size fixture", size)
		}
		signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: cfg.ChainID, Nonce: nonce, Gas: 4000000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1), To: &to, Data: make([]byte, size)}), gethtypes.LatestSignerForChainID(cfg.ChainID), key)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := signed.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		bytes += len(raw)
		tx, err := types.DecodeBlockTransaction(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := n.txPool.AddLocal(tx); err != nil {
			t.Fatalf("bulk nonce %d: %v", nonce, err)
		}
	}
	b := NewEngineBackend(n)
	b.buildTimeout = time.Minute
	t.Cleanup(b.Close)
	beacon := types.Hash{31: 1}
	slot := uint64(1)
	attrs := &payload.PayloadAttributesV3{PayloadAttributesV2: payload.PayloadAttributesV2{PayloadAttributesV1: payload.PayloadAttributesV1{Timestamp: 12}, Withdrawals: []*payload.Withdrawal{}}, ParentBeaconBlockRoot: beacon}
	fc, err := b.forkchoiceUpdated(payload.ForkchoiceStateV1{HeadBlockHash: genesis.Hash()}, attrs, nil, &slot, limit)
	if err != nil || fc.PayloadID == nil {
		t.Fatal(fc, err)
	}
	b.mu.Lock()
	pending := b.payloads[*fc.PayloadID]
	b.mu.Unlock()
	select {
	case <-pending.done:
	case <-time.After(75 * time.Second):
		t.Fatal("large native build timed out")
	}
	if pending.err != nil || pending.block == nil {
		t.Fatal(pending.err)
	}
	block := pending.block
	if len(block.Transactions()) != 128 {
		t.Fatal("bulk fixture was not actually included", len(block.Transactions()))
	}
	encoded, err := block.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > blockpkg.MaxRLPBlockSize || blockpkg.MaxRLPBlockSize-len(encoded) > 7000 {
		t.Fatal("fixture did not reach the real RLP boundary", len(encoded))
	}
	if err := bc.InsertBlock(block); err != nil {
		t.Fatal("large fixture did not validate/import", err)
	}
	target := gethcrypto.PubkeyToAddress(key.PublicKey)
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: cfg.ChainID, Nonce: 1, Gas: 500000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1), To: &target, Data: make([]byte, 7800)}), gethtypes.LatestSignerForChainID(cfg.ChainID), other)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 8192 {
		t.Fatal("candidate exceeds the actual IL byte limit")
	}
	tx, err := types.DecodeBlockTransaction(raw)
	if err != nil {
		t.Fatal(err)
	}
	sizer, err := types.NewBlockAppendSizer(block)
	if err != nil {
		t.Fatal(err)
	}
	projected, err := sizer.Size(tx, block.GasUsed())
	if err != nil {
		t.Fatal(err)
	}
	if projected <= blockpkg.MaxRLPBlockSize {
		t.Fatal("counterfactual append must exceed RLP size", projected)
	}
	satisfied, err := b.CheckPayloadInclusionList(block.Hash(), [][]byte{raw})
	if err != nil || satisfied {
		t.Fatal("unmerged byte-size rule excused a pre-execution-valid omission", satisfied, err)
	}
}
