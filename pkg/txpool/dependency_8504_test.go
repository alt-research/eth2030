package txpool

import (
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func dependencyPool8504() (*TxPool, *state.MemoryStateDB, *config.ChainConfig) {
	db := state.NewMemoryStateDB()
	tx := proof8504.Transaction(0)
	f := tx.Inner().(*types.FrameTx8141)
	db.AddBalance(f.Sender, big.NewInt(1_000_000_000))
	db.SetCode(f.Sender, proof8504.ApproveCode)
	zero := uint64(0)
	cfg := *config.TestConfig
	cfg.ChainID = big.NewInt(1337)
	cfg.HogotaTime = &zero
	cfg.EIP8504Time = &zero
	pool := New(DefaultConfig(), db)
	pool.SetForkRules(&cfg, 100)
	pool.SetBaseFee(big.NewInt(1))
	return pool, db, &cfg
}

func TestDependency8504PoolAdmissionLifecycleAndBytes(t *testing.T) {
	pool, db, cfg := dependencyPool8504()
	tx := proof8504.Transaction(0)
	hash, _ := types.DependencyWrapperHash8504(tx)
	raw, _ := types.EncodeDependencyWrapper8504(tx)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	if pool.DependencyBytes8504() != uint64(len(raw)) || len(pool.DependencyWrapper8504(hash)) != len(raw) {
		t.Fatal("wrapper not accounted or served")
	}
	metadata := pool.lookup.frameValidations[tx.Hash()]
	pool.Reset(db, &types.Header{Number: big.NewInt(1), Time: 101, GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if pool.Get(tx.Hash()) == nil || pool.lookup.frameValidations[tx.Hash()] != metadata {
		t.Fatal("unchanged head lost stateless/prefix cache")
	}
	bad := tx.DependencyWitnesses8504()
	bad[0][0] ^= 1
	if err := pool.AddRemote(tx.WithDependencyWitnesses8504(bad)); !errors.Is(err, ErrAlreadyKnown) {
		t.Fatal("second wrapper revalidated", err)
	}
	if pool.DependencyBytes8504() != uint64(len(raw)) {
		t.Fatal("duplicate changed byte count")
	}
	pool.Remove(tx.Hash())
	if pool.DependencyBytes8504() != 0 || pool.DependencyWrapper8504(hash) != nil {
		t.Fatal("eviction leaked wrapper")
	}
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	db.SetNonce(tx.FrameSender(), 1)
	pool.Reset(db, &types.Header{Number: big.NewInt(2), Time: 102, GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
	if pool.Count() != 0 || pool.DependencyBytes8504() != 0 {
		t.Fatal("mined/reorg invalidation leaked wrapper")
	}
	db.SetNonce(tx.FrameSender(), 0)
	pool.SetForkRules(cfg, 102)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal("reorg readmission", err)
	}
	pool.Remove(tx.Hash())
	pool.config.MaxDependencyBytes = uint64(len(raw) - 1)
	if err := pool.AddRemote(tx); err == nil || pool.Count() != 0 || pool.DependencyBytes8504() != 0 {
		t.Fatal("byte limit not atomic")
	}
}

func TestDependency8504PoolRejectsBodyWitnessBeforeSimulation(t *testing.T) {
	for _, kind := range []string{"missing", "bad", "extra", "inactive", "overcap"} {
		pool, db, cfg := dependencyPool8504()
		tx := proof8504.Transaction(0)
		switch kind {
		case "missing":
			tx = tx.WithDependencyWitnesses8504(nil)
		case "bad":
			w := tx.DependencyWitnesses8504()
			w[0][0] ^= 1
			tx = tx.WithDependencyWitnesses8504(w)
		case "extra":
			w := tx.DependencyWitnesses8504()
			tx = tx.WithDependencyWitnesses8504(append(w, w[0]))
		case "inactive":
			cfg.EIP8504Time = nil
			pool.SetForkRules(cfg, 100)
		case "overcap":
			f := tx.Inner().(*types.FrameTx8141)
			d, w := proof8504.Dependency()
			f.Frames[1].Data = nil
			witnesses := make([][]byte, 17)
			for i := range witnesses {
				d.DataHash[0] = byte(i + 1)
				f.Frames[1].Data = append(f.Frames[1].Data, d.Bytes()...)
				witnesses[i] = w
			}
			f.Frames[1].Limits.Execution = 17 * 3000
			tx = tx.WithDependencyWitnesses8504(witnesses)
		}
		root := db.GetRoot()
		if err := pool.AddRemote(tx); err == nil || pool.Count() != 0 || pool.DependencyBytes8504() != 0 || db.GetRoot() != root {
			t.Fatal(kind, "admitted or contaminated state", err)
		}
	}
}
