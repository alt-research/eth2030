package txpool

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func scalarLifecycleOrdinary(t *testing.T, nonce uint64) *types.Transaction {
	t.Helper()
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	to := gethcommon.Address{19: 0x90}
	chain := big.NewInt(1337)
	tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{
		ChainID: chain, Nonce: nonce, Gas: 21000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(10), To: &to,
	}), gethtypes.LatestSignerForChainID(chain), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	native, err := types.DecodePooledTransaction(raw)
	if err != nil {
		t.Fatal(err)
	}
	return native
}

func scalarLifecycleFrame(t *testing.T, pool *TxPool, base *types.FrameTx8141, profile string, key int, expiry uint64) *types.Transaction {
	t.Helper()
	frame := *base
	frame.Sender = framePoolAddress(t, key)
	frame.Frames = append([]types.Frame8141(nil), base.Frames...)
	if expiry != 0 {
		deadline := make([]byte, 8)
		binary.BigEndian.PutUint64(deadline, expiry)
		frame.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &types.FrameExpiryVerifier, Data: deadline, Limits: types.FrameGasLimits{Execution: 5000}}}, frame.Frames...)
	}
	if profile == "base" {
		return signPoolFrame(t, &frame, key)
	}
	activation := uint64(0)
	pool.frameConfig.EIP8250Time = &activation
	keyed := &types.FrameTx8250Delta{ChainID: frame.ChainID, Sender: frame.Sender, NonceSeq: frame.Nonce, Frames: frame.Frames, Fees: frame.Fees}
	if profile == "keyed" {
		keyed.NonceKeys = []*big.Int{big.NewInt(42)}
		keyed.Frames[len(keyed.Frames)-1].Limits.State = 97920
		keyed.Frames[len(keyed.Frames)-1].Limits.Execution = 30000
	}
	signKeyedPolicyFrame(t, keyed, key)
	return types.NewFrameTransaction8250(keyed)
}

// EIP-8141 consumes a scalar nonce once at payment approval. EIP-8250
// preserves this dependency only for empty nonce_keys, never for selected keys.
func TestFrameCRemovalReconcilesScalarSuccessors(t *testing.T) {
	for _, profile := range []string{"base", "empty_keys", "keyed"} {
		for _, cause := range []string{"remove", "expiry", "unpaid", "dependency", "included"} {
			t.Run(profile+"/"+cause, func(t *testing.T) {
				pool, db, base := framePoolFixture(t)
				db.SetCode(types.FrameExpiryVerifier, types.FrameExpiryCode)
				db.SetNonce(types.NonceManager8250, 1)
				db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
				frame := scalarLifecycleFrame(t, pool, base, profile, 1, 110)
				if err := pool.AddRemote(frame); err != nil {
					t.Fatal(err)
				}
				nonce := uint64(1)
				if profile == "keyed" {
					nonce = 0
				}
				ordinary := scalarLifecycleOrdinary(t, nonce)
				if err := pool.AddRemote(ordinary); err != nil {
					t.Fatal(err)
				}
				if pool.PendingCount() != 2 || pool.QueuedCount() != 0 {
					t.Fatal("positive control did not admit both processable transactions")
				}
				// A second successor must follow the same fate; a pre-existing gap
				// must not be bridged by removal or subsequent re-admission.
				second := scalarLifecycleOrdinary(t, nonce+1)
				future := scalarLifecycleOrdinary(t, nonce+3)
				for _, tx := range []*types.Transaction{second, future} {
					if err := pool.AddRemote(tx); err != nil {
						t.Fatal(err)
					}
				}
				switch cause {
				case "remove":
					pool.Remove(frame.Hash())
				case "expiry":
					pool.RecordBlock(&types.Header{Number: big.NewInt(1), Time: 111, GasLimit: 30000000, BaseFee: big.NewInt(1)}, nil)
				case "unpaid":
					for range 3 {
						pool.FrameBuildResult(frame.Hash(), false)
					}
				case "dependency":
					db.SetCode(types.FrameExpiryVerifier, []byte{0xfe})
					pool.RecordBlock(&types.Header{Number: big.NewInt(1), Time: 101, GasLimit: 30000000, BaseFee: big.NewInt(1)}, nil)
				case "included":
					if profile != "keyed" {
						db.SetNonce(base.Sender, 1)
					}
					pool.Remove(frame.Hash())
				}
				if pool.Get(frame.Hash()) != nil || pool.Get(ordinary.Hash()) == nil || pool.Get(second.Hash()) == nil || pool.Get(future.Hash()) == nil || len(pool.lookup.frameValidations) != 0 || len(pool.lookup.frameDependencies) != 0 || len(pool.lookup.framePayers) != 0 || len(pool.lookup.frameBuildFailures) != 0 || pool.lookup.frameDeadlines.Len() != 0 || pool.lookup.frameRootExpiries.Len() != 0 {
					t.Fatal("removal lost successor or retained Frame resources")
				}
				wantPending := profile == "keyed" || cause == "included"
				pending, queued := 0, 3
				if wantPending {
					pending, queued = 2, 1
				}
				if pool.PendingCount() != pending || pool.QueuedCount() != queued || len(pool.PendingFlat()) != pending {
					t.Fatalf("scalar gap not reconciled: pending=%d queued=%d wantPending=%t", pool.PendingCount(), pool.QueuedCount(), wantPending)
				}
				if !wantPending {
					// Re-admission, rather than implicit resurrection, fills the gap.
					db.SetCode(types.FrameExpiryVerifier, types.FrameExpiryCode)
					received := scalarLifecycleFrame(t, pool, base, profile, 1, 120)
					if err := pool.AddRemote(received); err != nil {
						t.Fatal("fresh predecessor readmission", err)
					}
					if pool.PendingCount() != 3 || pool.QueuedCount() != 1 {
						t.Fatal("readmitted scalar predecessor did not promote successor")
					}
				}
			})
		}
	}
}

func TestFrameCEvictionIncludesNonTailScalarFrame(t *testing.T) {
	for _, profile := range []string{"base", "empty_keys"} {
		t.Run(profile, func(t *testing.T) {
			pool, db, base := framePoolFixture(t)
			pool.config.MaxSize = 3
			db.SetCode(types.FrameExpiryVerifier, types.FrameExpiryCode)
			head := scalarLifecycleFrame(t, pool, base, profile, 1, 110)
			ordinary := scalarLifecycleOrdinary(t, 1)
			later := scalarLifecycleFrame(t, pool, base, profile, 3, 120)
			for _, tx := range []*types.Transaction{head, ordinary, later} {
				if err := pool.AddRemote(tx); err != nil {
					t.Fatal(err)
				}
			}
			incoming := scalarLifecycleFrame(t, pool, base, profile, 4, 130)
			if err := pool.AddRemote(incoming); err != nil {
				t.Fatal(err)
			}
			if pool.Get(head.Hash()) != nil || pool.Get(later.Hash()) == nil || pool.Get(incoming.Hash()) == nil {
				t.Fatal("capacity eviction skipped earliest expiry at scalar sender head")
			}
			if pool.Get(ordinary.Hash()) == nil || pool.PendingCount() != 2 || pool.QueuedCount() != 1 || len(pool.lookup.frameValidations) != 2 {
				t.Fatal("eviction did not retain and queue orphaned ordinary successor")
			}
		})
	}
}
