package chain

import (
	"context"
	"errors"
	coreconfig "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	nativecrypto "github.com/eth2030/eth2030/crypto"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

func TestHistoryCacheDetachesCallerCancellationAndOwnsLazyFailures(t *testing.T) {
	db := rawdb.NewMemoryDB()
	memory := state.NewMemoryStateDB()
	address := types.Address{19: 5}
	memory.AddBalance(address, big.NewInt(7))
	root := memory.GetRoot()
	batch := db.NewBatch()
	if err := state.StoreTrieHistory(db, batch, root, memory); err != nil {
		t.Fatal(err)
	}
	if err := batch.Write(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	source, err := state.OpenTrieHistory(ctx, db, root)
	if err != nil {
		t.Fatal(err)
	}
	cache := newStateCache(2)
	hash := types.Hash{31: 1}
	cache.put(hash, 1, source)
	cancel()
	_ = source.GetBalance(address)
	if !errors.Is(state.ReadError(source), context.Canceled) {
		t.Fatal("source did not observe its own cancellation")
	}
	cached, ok := cache.get(hash)
	if !ok || cached.GetBalance(address).Cmp(big.NewInt(7)) != 0 || state.ReadError(cached) != nil {
		t.Fatal("one caller poisoned shared history cache")
	}
	other, ok := cache.get(hash)
	if !ok {
		t.Fatal("snapshot absent")
	}
	other.AddBalance(address, big.NewInt(1))
	again, _ := cache.get(hash)
	if again.GetBalance(address).Cmp(big.NewInt(7)) != 0 {
		t.Fatal("working state leaked into immutable cache")
	}
}

func TestLazyStateFailureIsStorageFailureRatherThanInvalidExecution(t *testing.T) {
	key, err := gethcrypto.HexToECDSA("1111111111111111111111111111111111111111111111111111111111111111")
	if err != nil {
		t.Fatal(err)
	}
	address := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	memory := state.NewMemoryStateDB()
	memory.AddBalance(address, big.NewInt(1000000))
	memory.SetNonce(address, 5)
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
		t.Fatal("missing child proof fixture")
	}
	cfg := &coreconfig.ChainConfig{ChainID: big.NewInt(1), EIP155Block: new(big.Int), HomesteadBlock: new(big.Int), EIP150Block: new(big.Int), EIP158Block: new(big.Int), ByzantiumBlock: new(big.Int)}
	bc := &Blockchain{config: cfg, processor: execution.NewStateProcessor(cfg)}
	to := gethcommon.Address{19: 1}
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.LegacyTx{Nonce: 5, Gas: 21000, GasPrice: big.NewInt(1), Value: new(big.Int), To: &to}), gethtypes.NewEIP155Signer(big.NewInt(1)), key)
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
	block := types.NewBlock(&types.Header{Number: big.NewInt(1), Time: 1, GasLimit: 30000000, Difficulty: new(big.Int)}, &types.Body{Transactions: []*types.Transaction{tx}})
	healthy, err := state.OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := bc.executeStateContext(context.Background(), block, healthy, nil); err != nil {
		t.Fatalf("healthy transition control: %v", err)
	}
	lazy, err := state.OpenTrieHistory(context.Background(), db, root)
	if err != nil {
		t.Fatal(err)
	}
	leaf := nativecrypto.Keccak256Hash(proof[len(proof)-1])
	if err := db.Put(append([]byte("mpt-history-node/"), leaf[:]...), []byte{0x80}); err != nil {
		t.Fatal(err)
	}
	next, result, err := bc.executeStateContext(context.Background(), block, lazy, nil)
	if !errors.Is(err, ErrStorageFailure) || next != nil || result != nil {
		t.Fatalf("unreadable parent state misclassified as consensus invalidity: %v", err)
	}
}
