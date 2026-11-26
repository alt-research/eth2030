package txpool

import (
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	"math"
	"math/big"
	"testing"
)

func TestFrame8250PublicIdentityDependenciesAndForkEviction(t *testing.T) {
	for _, legacyRead := range []bool{false, true} {
		t.Run(map[bool]string{false: "keys_only", true: "legacy_read"}[legacyRead], func(t *testing.T) {
			pool, db, base := framePoolFixture(t)
			activation := uint64(100)
			pool.frameConfig.EIP8250Time = &activation
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			db.SetNonce(base.Sender, 55)
			if legacyRead {
				db.SetCode(base.Sender, []byte{0x60, 0x0d, 0xb0, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa})
			}
			profile := &types.FrameTx8250Delta{ChainID: base.ChainID, Sender: base.Sender, NonceKeys: []*big.Int{big.NewInt(42)}, NonceSeq: math.MaxUint64, Frames: base.Frames, Fees: base.Fees}
			profile.Frames[0].Limits.Execution = 30000
			profile.Frames[0].Limits.State = 97920
			signKeyedPolicyFrame(t, profile, 1)
			tx := types.NewFrameTransaction8250(profile)
			pool.SetForkRules(pool.frameConfig, 99)
			if err := pool.AddRemote(tx); err == nil {
				t.Fatal("pre-fork keyed admission")
			}
			pool.SetForkRules(pool.frameConfig, 100)
			base.Nonce = db.GetNonce(base.Sender)
			old := signPoolFrame(t, base, 1)
			if !legacyRead {
				pool.SetForkRules(pool.frameConfig, 99)
				if err := pool.AddRemote(old); err != nil {
					t.Fatal("valid pre-fork base admission", err)
				}
				pool.SetForkRules(pool.frameConfig, 100)
				pool.Reset(db)
				if pool.Count() != 0 {
					t.Fatal("activation retained pre-fork signatures/envelope")
				}
			}
			if err := pool.AddRemote(old); err == nil {
				t.Fatal("post-fork seven-field admission")
			}
			if err := pool.AddRemote(tx); err != nil {
				t.Fatal(err)
			}
			if pool.PendingCount() != 1 || pool.QueuedCount() != 0 {
				t.Fatal("keyed sequence treated as future scalar nonce")
			}
			validation := pool.lookup.frameValidations[tx.Hash()]
			legacyDep := execution.FrameDependency{Address: base.Sender, Kind: execution.FrameNonceDependency}
			if _, ok := validation.Dependencies[legacyDep]; ok != legacyRead {
				t.Fatalf("legacy dependency=%t want %t", ok, legacyRead)
			}
			// A scalar collision must coexist without replacement/removal aliasing.
			ordinary := types.NewTransaction(&types.LegacyTx{Nonce: math.MaxUint64, Gas: 21000, GasPrice: big.NewInt(5)})
			list := &txSortedList{}
			list.Add(ordinary)
			list.Add(tx)
			if list.Len() != 2 || list.Get(math.MaxUint64) != ordinary || !list.RemoveTx(tx) || list.Get(math.MaxUint64) != ordinary {
				t.Fatal("keyed identity aliases ordinary nonce")
			}
			// Disjoint sets are independent in consensus, but not replacement identities.
			replacement := *profile
			replacement.NonceKeys = []*big.Int{big.NewInt(43)}
			replacement.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(2000000), MaxPriorityFeePerGas: big.NewInt(4)}
			signKeyedPolicyFrame(t, &replacement, 1)
			if err := pool.AddRemote(types.NewFrameTransaction8250(&replacement)); err == nil {
				t.Fatal("disjoint key set replaced pending sender")
			}
			if pool.Get(tx.Hash()) == nil {
				t.Fatal("rejected candidate evicted original")
			}
			replacement.NonceKeys = profile.NonceKeys
			signKeyedPolicyFrame(t, &replacement, 1)
			next := types.NewFrameTransaction8250(&replacement)
			if err := pool.AddRemote(next); err != nil {
				t.Fatal("same identity replacement", err)
			}
			if pool.Get(tx.Hash()) != nil || pool.PendingCount() != 1 {
				t.Fatal("replacement leaked old identity")
			}
			before := pool.lookup.frameValidations[next.Hash()]
			db.SetNonce(base.Sender, 56)
			pool.Reset(db)
			if pool.Get(next.Hash()) == nil || (pool.lookup.frameValidations[next.Hash()] != before) != legacyRead {
				t.Fatal("legacy dependency cache invalidation differs")
			}
			db.SetState(types.NonceManager8250, types.NonceSlot8250(base.Sender, big.NewInt(42)), types.Hash{31: 1})
			pool.Reset(db)
			if pool.Count() != 0 || pool.PendingCount() != 0 || len(pool.lookup.frameDependencies) != 0 {
				t.Fatal("consumed selected key retained metadata")
			}
			db.SetState(types.NonceManager8250, types.NonceSlot8250(base.Sender, big.NewInt(42)), types.Hash{})
			if err := pool.AddRemote(next); err != nil {
				t.Fatal("reorg state readmission", err)
			}
			pool.SetForkRules(pool.frameConfig, 99)
			pool.Reset(db)
			if pool.Count() != 0 {
				t.Fatal("fork rollback retained post-fork envelope")
			}
		})
	}
}
