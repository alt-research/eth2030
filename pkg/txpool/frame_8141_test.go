package txpool

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func framePoolAddress(t *testing.T, key int) types.Address {
	t.Helper()
	private, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", key))
	if err != nil {
		t.Fatal(err)
	}
	return types.Address(gethcrypto.PubkeyToAddress(private.PublicKey))
}

func framePoolFixture(t *testing.T) (*TxPool, *state.MemoryStateDB, *types.FrameTx8141) {
	t.Helper()
	db := state.NewMemoryStateDB()
	for _, key := range []int{1, 2, 3, 4} {
		db.AddBalance(framePoolAddress(t, key), big.NewInt(100_000_000))
	}
	zero := uint64(0)
	cfg := *config.TestConfig
	cfg.ChainID, cfg.HogotaTime = big.NewInt(1337), &zero
	pool := New(DefaultConfig(), db)
	pool.SetForkRules(&cfg, 100)
	pool.SetBaseFee(big.NewInt(1))
	tx := &types.FrameTx8141{ChainID: cfg.ChainID, Sender: framePoolAddress(t, 1),
		Fees:   types.FrameFeeBudget{MaxFee: big.NewInt(1_000_000), MaxPriorityFeePerGas: big.NewInt(2)},
		Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}}
	return pool, db, tx
}

func signPoolFrame(t *testing.T, frame *types.FrameTx8141, keys ...int) *types.Transaction {
	t.Helper()
	frame.Signatures = make([]types.FrameSignature, len(keys))
	for i, key := range keys {
		frame.Signatures[i] = types.FrameSignature{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}
		signer := framePoolAddress(t, key)
		if signer != frame.Sender {
			frame.Signatures[i].Signer = &signer
		}
	}
	hash, err := frame.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range keys {
		private, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", key))
		if err != nil {
			t.Fatal(err)
		}
		sig, err := gethcrypto.Sign(hash[:], private)
		if err != nil {
			t.Fatal(err)
		}
		frame.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
	}
	return types.NewTransaction(frame)
}

func sponsorPoolFrame(t *testing.T, frame *types.FrameTx8141, key int) {
	t.Helper()
	payer := framePoolAddress(t, key)
	frame.Frames[0].Flags = 2
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 4000}})
}

func TestCanonicalFrame8141PoolLastValidNonce(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	frame.Nonce = math.MaxUint64 - 1
	db.SetNonce(frame.Sender, frame.Nonce)
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("last valid scalar nonce was rejected: %v", err)
	}
	if len(pool.PendingFlat()) != 1 || db.GetNonce(frame.Sender) != frame.Nonce {
		t.Fatal("high nonce admission changed live state or lost the pending Frame")
	}
}

func TestCanonicalFrame8141PoolPrefixDoesNotExecuteBody(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	target := types.Address{19: 0x90}
	// INVALID is allowed in the body; admission must stop after payment approval.
	db.SetCode(target, []byte{0xfe})
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 20000}})
	tx := signPoolFrame(t, frame, 1)
	root := db.GetRoot()
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	if len(pool.PendingFlat()) != 1 || db.GetRoot() != root || db.GetNonce(frame.Sender) != 0 {
		t.Fatal("admission executed on live state or lost transaction")
	}
	if pool.lookup.framePayers[tx.Hash()] != frame.Sender {
		t.Fatal("payer exposure missing")
	}
	pool.Remove(tx.Hash())
	if len(pool.lookup.framePayers) != 0 {
		t.Fatal("removed transaction retained exposure")
	}
}

func TestCanonicalFrame8141PoolReplacementChangesPayerAtomically(t *testing.T) {
	pool, _, frame := framePoolFixture(t)
	sponsorPoolFrame(t, frame, 3)
	old := signPoolFrame(t, frame, 1, 3)
	if err := pool.AddLocal(old); err != nil {
		t.Fatal(err)
	}
	for _, fees := range [][2]int64{{1_100_000, 2}, {1_000_000, 3}} {
		replacement := *frame
		replacement.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(fees[0]), MaxPriorityFeePerGas: big.NewInt(fees[1])}
		if err := pool.AddLocal(signPoolFrame(t, &replacement, 1, 3)); err != ErrReplacementUnderpriced {
			t.Fatalf("one-sided bump: %v", err)
		}
		if pool.Get(old.Hash()) == nil || len(pool.lookup.framePayers) != 1 {
			t.Fatal("failed replacement released old payer")
		}
	}
	replacement := *frame
	replacement.Frames = append([]types.Frame8141(nil), frame.Frames...)
	payer := framePoolAddress(t, 4)
	replacement.Frames[1].Target = &payer
	replacement.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(1_100_000), MaxPriorityFeePerGas: big.NewInt(3)}
	next := signPoolFrame(t, &replacement, 1, 4)
	if err := pool.AddLocal(next); err != nil {
		t.Fatal(err)
	}
	if pool.Get(old.Hash()) != nil || pool.Get(next.Hash()) == nil || len(pool.lookup.framePayers) != 1 || pool.lookup.framePayers[next.Hash()] != payer {
		t.Fatal("payer reservation was not transferred")
	}
}

func TestCanonicalFrame8141PoolAggregateExposureAndRevalidation(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	sponsorPoolFrame(t, frame, 3)
	payer := framePoolAddress(t, 3)
	db.SubBalance(payer, big.NewInt(98_500_000)) // enough escrow, but only one max_fee reservation
	first := signPoolFrame(t, frame, 1, 3)
	if err := pool.AddRemote(first); err != nil {
		t.Fatal(err)
	}
	other := *frame
	other.Sender = framePoolAddress(t, 2)
	second := signPoolFrame(t, &other, 2, 3)
	if err := pool.AddRemote(second); err != ErrInsufficientFunds {
		t.Fatalf("oversubscribed payer: %v", err)
	}
	pool.Remove(first.Hash())
	if err := pool.AddRemote(second); err != nil {
		t.Fatalf("exposure was not released: %v", err)
	}
	db.SetNonce(other.Sender, 1)
	pool.Reset(db)
	if pool.Get(second.Hash()) != nil || len(pool.lookup.framePayers) != 0 {
		t.Fatal("head nonce change retained invalid transaction")
	}
}

func TestCanonicalFrame8141PoolTraceRestrictions(t *testing.T) {
	for _, test := range []struct {
		name    string
		prefix  []byte
		allowed bool
	}{
		{"sender storage", []byte{0x5f, 0x54, 0x50}, true},
		{"timestamp", []byte{0x42, 0x50}, false},
		{"balance", []byte{0x47, 0x50}, false},
		{"block number", []byte{0x43, 0x50}, false},
		{"gas observation", []byte{0x5a, 0x50}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool, db, frame := framePoolFixture(t)
			frame.Frames[0].Limits.Execution = 30000
			db.SetCode(frame.Sender, append(test.prefix, 0x60, 3, 0x5f, 0x5f, 0xaa))
			err := pool.AddRemote(signPoolFrame(t, frame, 1))
			if (err == nil) != test.allowed {
				t.Fatalf("admission error %v, allowed=%v", err, test.allowed)
			}
		})
	}
	t.Run("caught third party storage", func(t *testing.T) {
		pool, db, frame := framePoolFixture(t)
		frame.Frames[0].Limits.Execution = 30000
		helper := types.Address{19: 0x90}
		db.SetCode(helper, []byte{0x5f, 0x54, 0x50, 0x00})
		// STATICCALL helper, discard failure and then APPROVE. The trace must still be rejected.
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
		code = append(code, helper[:]...)
		code = append(code, 0x61, 0x27, 0x10, 0xfa, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa)
		db.SetCode(frame.Sender, code)
		if err := pool.AddRemote(signPoolFrame(t, frame, 1)); err == nil {
			t.Fatal("caught forbidden child access passed admission")
		}
	})
}

func TestCanonicalFrame8141PoolExpiryAndFeeChanges(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	expiry := types.FrameExpiryVerifier
	deadline := make([]byte, 8)
	binary.BigEndian.PutUint64(deadline, 110)
	frame.Frames = append([]types.Frame8141{{Mode: types.ModeVerify, Target: &expiry, Data: deadline, Limits: types.FrameGasLimits{Execution: 5000}}}, frame.Frames...)
	db.SetCode(expiry, types.FrameExpiryCode)
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	pool.SetForkRules(pool.frameConfig, 111)
	pool.Reset(db)
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("expired Frame remained pending")
	}
	pool, _, frame = framePoolFixture(t)
	tx = signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	pool.SetBaseFee(big.NewInt(1000))
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("unfunded fee budget remained pending")
	}
}

func TestCanonicalFrame8141PoolRejectsInvalidPrefix(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*types.FrameTx8141)
	}{
		{"future nonce", func(tx *types.FrameTx8141) { tx.Nonce = 1 }},
		{"unbounded verification", func(tx *types.FrameTx8141) { tx.Frames[0].Limits.Execution = 100000 }},
		{"unbounded state", func(tx *types.FrameTx8141) { tx.Frames[0].Limits.State = 500001 }},
		{"verify after payment", func(tx *types.FrameTx8141) { tx.Frames = append(tx.Frames, tx.Frames[0]) }},
		{"wrong chain", func(tx *types.FrameTx8141) { tx.ChainID = big.NewInt(1) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool, _, frame := framePoolFixture(t)
			test.change(frame)
			if err := pool.AddRemote(signPoolFrame(t, frame, 1)); err == nil {
				t.Fatal("invalid prefix accepted")
			}
		})
	}
}

func TestCanonicalFrame8141CodedPaymasterExposureLimit(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	sponsorPoolFrame(t, frame, 3)
	payer := framePoolAddress(t, 3)
	db.SetCode(payer, []byte{0x60, 1, 0x5f, 0x5f, 0xaa})
	frame.Frames[1].Limits.Execution = 30000
	first := signPoolFrame(t, frame, 1, 3)
	if err := pool.AddRemote(first); err != nil {
		t.Fatal(err)
	}
	secondFrame := *frame
	secondFrame.Sender = framePoolAddress(t, 2)
	second := signPoolFrame(t, &secondFrame, 2, 3)
	if err := pool.AddRemote(second); err == nil {
		t.Fatal("unrecognized coded sponsor was granted unlimited pending exposure")
	}
	pool.Remove(first.Hash())
	if err := pool.AddRemote(second); err != nil {
		t.Fatalf("coded sponsor reservation was not released: %v", err)
	}
}

func TestCanonicalFrame8141PoolUsesCanonicalHeadFees(t *testing.T) {
	pool, _, frame := framePoolFixture(t)
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	// RecordBlock is the production FCU hook, not a test-only fee setter.
	pool.RecordBlock(&types.Header{Number: big.NewInt(2), Time: 101, GasLimit: 30000000, BaseFee: big.NewInt(1000)}, nil)
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("canonical head fee change retained an unfunded Frame")
	}
	if err := pool.AddRemote(tx); err == nil {
		t.Fatal("admission ignored the production head fee context")
	}
}

func TestCanonicalFrame8141PoolRequiresOneCombinedApproval(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	frame.Frames[0].Limits.Execution = 40000
	library := types.Address{19: 0x90}
	db.SetCode(library, []byte{0x60, 2, 0x5f, 0x5f, 0xaa})
	// DELEGATECALL approves execution, then the root approves payment. Both
	// effects are valid at consensus, but public self_verify requires scope 3.
	code := []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x73}
	code = append(code, library[:]...)
	code = append(code, 0x61, 0x75, 0x30, 0xf4, 0x50, 0x60, 1, 0x5f, 0x5f, 0xaa)
	db.SetCode(frame.Sender, code)
	root := db.GetRoot()
	if err := pool.AddRemote(signPoolFrame(t, frame, 1)); err == nil {
		t.Fatal("separate scopes were mistaken for the required combined approval")
	}
	if db.GetRoot() != root {
		t.Fatal("rejected validation modified live state")
	}
}

func TestCanonicalFrame8141ResetRefreshesHeadFeesWithoutFCU(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	// P2P/native-consensus/asynchronous-sync head changes do not call RecordBlock.
	pool.Reset(db, &types.Header{Number: big.NewInt(2), Time: 101, GasLimit: 30000000, BaseFee: big.NewInt(1000)})
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("non-FCU head reset kept stale fee context")
	}
	if err := pool.AddRemote(tx); err == nil {
		t.Fatal("admission used fees from before the state reset")
	}
}
