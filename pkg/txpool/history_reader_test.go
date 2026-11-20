package txpool

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	nativecrypto "github.com/eth2030/eth2030/crypto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func TestLazyHistoryReadFailureCannotAdmitZeroFeeTransaction(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	address := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	memory := state.NewMemoryStateDB()
	memory.AddBalance(address, new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil))
	memory.AddBalance(types.Address{19: 8}, big.NewInt(1))
	root := memory.GetRoot()
	db := rawdb.NewMemoryDB()
	batch := db.NewBatch()
	if err := state.StoreTrieHistory(db, batch, root, memory); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := state.ExportStateSnapshot(memory)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := snapshot.AccountProof(nativecrypto.Keccak256Hash(address[:]))
	if err != nil || len(proof) < 2 {
		t.Fatalf("no child-node fixture: %v", err)
	}
	leaf := nativecrypto.Keccak256Hash(proof[len(proof)-1])
	to := gethcommon.Address{19: 1}
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.LegacyTx{Gas: 100000, GasPrice: new(big.Int), Value: new(big.Int), To: &to}), gethtypes.NewEIP155Signer(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := types.DecodeTxRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.MinGasPrice = new(big.Int)
	rules := *config.TestConfig
	rules.ChainID = big.NewInt(1337)
	healthy, err := state.OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	control := New(cfg, healthy)
	control.SetForkRules(&rules, 0)
	if err := control.AddRemote(tx); err != nil {
		t.Fatalf("valid zero-fee control: %v", err)
	}
	damaged, err := state.OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	pool := New(cfg, damaged)
	pool.SetForkRules(&rules, 0)
	if err := db.Put(append([]byte("mpt-history-node/"), leaf[:]...), []byte{0x80}); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddRemote(tx); err == nil || !strings.Contains(err.Error(), "state history") {
		t.Fatalf("bad state admitted or misclassified: %v", err)
	}
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("errored admission published transaction")
	}
}

func TestSparseMetadataSerializesLazyStateCachePopulation(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(strings.Repeat("22", 32))
	if err != nil {
		t.Fatal(err)
	}
	address := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	memory := state.NewMemoryStateDB()
	memory.AddBalance(address, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
	root := memory.GetRoot()
	db := rawdb.NewMemoryDB()
	batch := db.NewBatch()
	if err := state.StoreTrieHistory(db, batch, root, memory); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	lazy, err := state.OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.MinGasPrice = new(big.Int)
	pool := New(cfg, lazy)
	rules := *config.TestConfig
	rules.ChainID = big.NewInt(1337)
	pool.SetForkRules(&rules, 0)
	makeTx := func() *types.Transaction {
		signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), Gas: 100000, GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(100), BlobFeeCap: uint256.NewInt(100), To: gethcommon.Address{19: 1}, Value: new(uint256.Int), BlobHashes: []gethcommon.Hash{{0: 1, 31: 1}}}), gethtypes.NewCancunSigner(big.NewInt(1337)), key)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := signed.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		tx, err := types.DecodeTxRLP(raw)
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	txs := make([]*types.Transaction, 32)
	for i := range txs {
		txs[i] = makeTx()
	}
	var group sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, len(txs))
	for _, tx := range txs {
		group.Add(1)
		go func(tx *types.Transaction) { defer group.Done(); <-start; errs <- pool.ValidateSparseBlobMetadata(tx) }(tx)
	}
	close(start)
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if pool.Get(txs[0].Hash()) != nil {
		t.Fatal("metadata validation admitted an incomplete transaction")
	}
}
