package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/frameproof"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func TestDependency8504CanonicalSigningHashWalletBaseAndKeyed(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		tx, expected := proof8504.CanonicalTransaction(keyed)
		if tx.SigningHash() != expected {
			t.Fatal("independent canonical signing hash", keyed, tx.SigningHash(), expected)
		}
		zero := uint64(0)
		cfg := *config.TestConfig
		cfg.ChainID = big.NewInt(1337)
		cfg.HogotaTime, cfg.EIP8504Time = &zero, &zero
		if keyed {
			cfg.EIP8250Time = &zero
		}
		db := state.NewMemoryStateDB()
		db.AddBalance(tx.FrameSender(), new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil))
		db.SetCode(tx.FrameSender(), proof8504.CanonicalWalletCode())
		if keyed {
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			db.SetNonce(types.NonceManager8250, 1)
		}
		if err := frameproof.VerifyTransaction(tx, cfg.FrameExtensions(0), true); err != nil {
			t.Fatal("independent EELS canonical signature", keyed, err)
		}
		h := &types.Header{Number: big.NewInt(1), Time: 0, BaseFee: big.NewInt(1), GasLimit: 30_000_000}
		db.SetTxContext(tx.Hash(), 0)
		r, _, err := ApplyTransaction(&cfg, db, h, tx, new(gaspool.GasPool).AddGas(h.GasLimit))
		if err != nil || r.FrameResults8141[0].Status != 1 || r.FrameResults8141[1].Status != 1 || r.FrameResults8141[1].GasUsed.Execution != 3000 {
			t.Fatal("real canonical PQ wallet did not approve", keyed, r, err)
		}
		if keyed {
			if db.GetState(types.NonceManager8250, types.NonceSlot8250(tx.FrameSender(), big.NewInt(42))) != (types.Hash{31: 1}) {
				t.Fatal("keyed nonce not consumed")
			}
		} else if db.GetNonce(tx.FrameSender()) != 1 {
			t.Fatal("base nonce not consumed")
		}
		// Same key/witness cannot authorize another transaction's nonce.
		other, _ := proof8504.CanonicalTransaction(keyed)
		f := other.Inner().(*types.FrameTx8141)
		if keyed {
			profile := f.NonceProfile8250()
			profile.NonceSeq++
			other = types.NewFrameTransaction8250(profile).WithDependencyWitnesses8504(other.DependencyWitnesses8504())
		} else {
			f.Nonce++
		}
		if err := frameproof.VerifyTransaction(other, cfg.FrameExtensions(0), true); err == nil {
			t.Fatal("canonical witness did not bind replay identity")
		}
	}
}
