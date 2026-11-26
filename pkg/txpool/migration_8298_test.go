package txpool

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestMigration8298PoolEvictsRetiredOriginsAndAllowsFrames(t *testing.T) {
	pool, db, cfg := dependencyPool8504()
	zero := uint64(0)
	cfg.AmsterdamTime, cfg.EIP8298Time = &zero, &zero
	pool.SetForkRules(cfg, 100)
	key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	db.AddBalance(sender, big.NewInt(1_000_000_000))
	signed := func(nonce uint64) *types.Transaction {
		t.Helper()
		to := gethcommon.Address{19: 0x91}
		g := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: nonce, Gas: 50000, GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(2), To: &to, Value: new(big.Int)})
		g, err := gethtypes.SignTx(g, gethtypes.NewLondonSigner(big.NewInt(1337)), key)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := g.MarshalBinary()
		tx, err := types.DecodeTxRLP(raw)
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	for _, nonce := range []uint64{0, 1} {
		if err := pool.AddRemote(signed(nonce)); err != nil {
			t.Fatal(err)
		}
	}
	db.SetCode(sender, proof8504.ApproveCode)
	pool.Reset(db, &types.Header{Number: big.NewInt(1), Time: 101, GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if pool.Count() != 0 || pool.PendingCount() != 0 || pool.QueuedCount() != 0 {
		t.Fatal("retired origin still pending/queued")
	}
	if err := pool.AddRemote(signed(0)); err == nil {
		t.Fatal("retired ECDSA origin admitted")
	}
	f := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1_000_000), MaxPriorityFeePerGas: big.NewInt(2)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 5000}}}}
	if err := pool.AddRemote(types.NewTransaction(f)); err != nil {
		t.Fatal("retirement disabled account Frame path", err)
	}
	pool.Remove(types.NewTransaction(f).Hash())
	db.SetCode(sender, nil)
	pool.Reset(db, &types.Header{Number: big.NewInt(0), Time: 100, GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if err := pool.AddRemote(signed(0)); err != nil {
		t.Fatal("reorg did not restore origin", err)
	}
}

func TestMigration8298ProxyDependencyLimitReplacementEvictionAndReorg(t *testing.T) {
	pool, db, cfg := dependencyPool8504()
	zero := uint64(0)
	cfg.AmsterdamTime, cfg.EIP8298Time = &zero, &zero
	pool.SetForkRules(cfg, 100)
	proxy := types.Address{18: 0xaa, 19: 1}
	db.SetCode(proxy, []byte{byte(vm.STOP), byte(vm.DELEGATECALL)})
	makeTx := func(i int, fee int64) *types.Transaction {
		sender := types.Address{19: byte(i + 0xa0)}
		db.AddBalance(sender, big.NewInt(100_000_000))
		code := append([]byte{byte(vm.PUSH20)}, proxy[:]...)
		code = append(code, byte(vm.EXTCODEHASH), byte(vm.POP), 0x60, 3, 0x5f, 0x5f, byte(vm.APPROVE))
		db.SetCode(sender, code)
		return types.NewTransaction(&types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(fee), MaxPriorityFeePerGas: big.NewInt(2)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 10000}}}})
	}
	for i := 0; i < MaxProxyDependents8298; i++ {
		if err := pool.AddRemote(makeTx(i, 1_000_000)); err != nil {
			t.Fatal(i, err)
		}
	}
	next := makeTx(8, 1_000_000)
	if err := pool.AddRemote(next); err == nil {
		t.Fatal("unbounded proxy dependents")
	}
	replacement := makeTx(0, 2_000_000)
	replacement.Inner().(*types.FrameTx8141).Fees.MaxPriorityFeePerGas = big.NewInt(4)
	if err := pool.AddRemote(replacement); err != nil {
		t.Fatal("replacement leaked reservation", err)
	}
	pool.Remove(replacement.Hash())
	if err := pool.AddRemote(next); err != nil {
		t.Fatal("eviction did not release share", err)
	}
	db.SetCode(proxy, []byte{byte(vm.STOP), byte(vm.SETCODEFROM)})
	pool.Reset(db, &types.Header{Time: 101, Number: big.NewInt(1), GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if pool.Count() != 0 {
		t.Fatal("mutable helper invalidation did not evict dependents")
	}
	// Immediate bytes do not make a helper mutable.
	db.SetCode(proxy, []byte{byte(vm.PUSH1), byte(vm.SETCODEFROM), byte(vm.STOP)})
	if err := pool.AddRemote(next); err != nil {
		t.Fatal("PUSH immediate was treated as an instruction", err)
	}
	pool.Remove(next.Hash())
	db.SetCode(proxy, []byte{byte(vm.STOP), byte(vm.DELEGATECALL)})
	pool.Reset(db, &types.Header{Time: 100, Number: big.NewInt(0), GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if err := pool.AddRemote(next); err != nil {
		t.Fatal("proxy reorg readmission", err)
	}
}

func TestDependency8504ExpiryAndActualWrapperByteBudget(t *testing.T) {
	pool, db, cfg := dependencyPool8504()
	tx := proof8504.Transaction(0)
	f := tx.Inner().(*types.FrameTx8141)
	expiry := make([]byte, 8)
	binary.BigEndian.PutUint64(expiry, 101)
	verify := f.Frames[0]
	proof := f.Frames[1]
	f.Frames = []types.Frame8141{proof, {Mode: types.ModeVerify, Target: &types.FrameExpiryVerifier, Data: expiry, Limits: types.FrameGasLimits{Execution: 5000}}, verify}
	db.SetCode(types.FrameExpiryVerifier, types.FrameExpiryCode)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal("proof before expiry", err)
	}
	if frameExpiry(tx) != 101 || transactionPoolBytes(tx) != uint64(len(pool.DependencyWrapper8504(pool.lookup.dependencyHashes[tx.Hash()]))) {
		t.Fatal("logical expiry or wrapper byte count")
	}
	pool.SetForkRules(cfg, 102)
	pool.Reset(db, &types.Header{Time: 102, Number: big.NewInt(1), GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if pool.Count() != 0 || pool.DependencyBytes8504() != 0 {
		t.Fatal("proof-prefix expiry leaked wrapper")
	}
}

func TestMigration8298AssertionJournalsPreserve8151SignerClass(t *testing.T) {
	pool, _, _ := framePoolFixture(t)
	enableRecovery8151(pool)
	zero := uint64(0)
	pool.frameConfig.AmsterdamTime, pool.frameConfig.EIP8298Time, pool.frameConfig.EIP7906Time = &zero, &zero, &zero
	tx := recoveredSignerFrame8151(t, pool, 1, true, nil)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	signer := framePoolAddress(t, 2)
	validation := pool.lookup.frameValidations[tx.Hash()]
	if validation.Dependencies[execution.FrameDependency{Address: signer, Kind: execution.FrameSignerClassDependency}] != "permitted" {
		t.Fatal("D journals erased paid 8151 signer class")
	}
	if _, found := validation.Dependencies[execution.FrameDependency{Address: signer, Kind: execution.FrameCodeDependency}]; found {
		t.Fatal("D journals replaced signer class with raw code dependency")
	}
}
