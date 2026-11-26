package txpool

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

// EIP-8141 Replacement and Eviction requires invalid entries to leave before
// resource arbitration. An invalid reservation must not evict a valid peer.
func TestFrameCRevalidationDropsInvalidBeforeSharedExposure(t *testing.T) {
	pool, db, base := framePoolFixture(t)
	payer := framePoolAddress(t, 3)
	var txs []*types.Transaction
	for _, key := range []int{1, 2} {
		frame := types.NewTransaction(base).Inner().(*types.FrameTx8141)
		frame.Sender = framePoolAddress(t, key)
		sponsorPoolFrame(t, frame, 3)
		tx := signPoolFrame(t, frame, key, 3)
		if err := pool.AddRemote(tx); err != nil {
			t.Fatal(err)
		}
		txs = append(txs, tx)
	}
	a, b := txs[0].Hash(), txs[1].Hash()
	if bytes.Compare(a[:], b[:]) > 0 {
		txs[0], txs[1] = txs[1], txs[0]
	}
	valid, invalid := txs[0], txs[1] // valid is visited first by the old hash ordering
	db.SetNonce(invalid.FrameSender(), 1)
	db.SubBalance(payer, big.NewInt(99_000_000)) // exactly one signed max_fee
	pool.Reset(db)
	if pool.Count() != 1 || pool.Get(valid.Hash()) == nil || pool.Get(invalid.Hash()) != nil {
		t.Fatal("stale invalid payer exposure evicted the valid transaction")
	}
	if len(pool.lookup.frameValidations) != 1 || len(pool.lookup.framePayers) != 1 {
		t.Fatal("batch revalidation leaked metadata")
	}
}

func TestFrameCSharedPayerRevalidationEvictsByPolicy(t *testing.T) {
	for _, expiryFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "lowest_priority", true: "nearest_expiry"}[expiryFirst], func(t *testing.T) {
			pool, db, base := framePoolFixture(t)
			payer := framePoolAddress(t, 3)
			db.SetCode(types.FrameExpiryVerifier, types.FrameExpiryCode)
			var txs []*types.Transaction
			for i, key := range []int{1, 2} {
				frame := types.NewTransaction(base).Inner().(*types.FrameTx8141)
				frame.Sender = framePoolAddress(t, key)
				frame.Fees.MaxPriorityFeePerGas = big.NewInt(int64(2 + i*10))
				sponsorPoolFrame(t, frame, 3)
				if expiryFirst {
					deadline := make([]byte, 8)
					binary.BigEndian.PutUint64(deadline, uint64(300-i*100)) // higher-tip tx expires first
					frame.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &types.FrameExpiryVerifier, Data: deadline, Limits: types.FrameGasLimits{Execution: 5000}}}, frame.Frames...)
				}
				tx := signPoolFrame(t, frame, key, 3)
				if err := pool.AddRemote(tx); err != nil {
					t.Fatal(err)
				}
				txs = append(txs, tx)
			}
			db.SubBalance(payer, big.NewInt(99_000_000))
			pool.Reset(db)
			survivor := txs[1]
			if expiryFirst {
				survivor = txs[0]
			}
			if pool.Count() != 1 || pool.Get(survivor.Hash()) == nil {
				t.Fatal("shared payer arbitration ignored expiry/priority policy")
			}
		})
	}
}
