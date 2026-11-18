package txpool

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func paymasterPoolFixture(t *testing.T) (*TxPool, *state.MemoryStateDB, *types.FrameTx8141, types.Address) {
	t.Helper()
	pool, db, frame := framePoolFixture(t)
	payer := types.Address{19: 0x93}
	owner := framePoolAddress(t, 2)
	db.SetCode(payer, framepay.RuntimeV1())
	db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
	db.AddBalance(payer, big.NewInt(100000000))
	frame.Frames[0].Flags = 2
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Data: framepay.ValidationData(), Limits: types.FrameGasLimits{Execution: 10000}})
	return pool, db, frame, payer
}

func TestFramePaymasterV1AuthenticatesCanonicalSignature(t *testing.T) {
	pool, db, frame, payer := paymasterPoolFixture(t)
	root := db.GetRoot()
	tx := signPoolFrame(t, frame, 1, 2)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("valid paymaster prefix: %v", err)
	}
	if db.GetRoot() != root || pool.lookup.framePayers[tx.Hash()] != payer {
		t.Fatal("simulation changed live state or lost payer")
	}
	pool.Remove(tx.Hash())
	if err := pool.AddRemote(signPoolFrame(t, frame, 1, 3)); err == nil {
		t.Fatal("different valid secp signer authorized sponsorship")
	}
	// A valid owner signature over an unrelated digest must not authorize the tx.
	candidate := signPoolFrame(t, frame, 1, 2)
	actual := candidate.Inner().(*types.FrameTx8141)
	actual.Signatures[1].Message = bytes.Repeat([]byte{0x42}, 32)
	key2, _ := gethcrypto.HexToECDSA(strings.Repeat("0", 63) + "2")
	sig, err := gethcrypto.Sign(actual.Signatures[1].Message, key2)
	if err != nil {
		t.Fatal(err)
	}
	actual.Signatures[1].Signature = append([]byte{sig[64]}, sig[:64]...)
	hash, err := actual.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	key1, _ := gethcrypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	sig, err = gethcrypto.Sign(hash[:], key1)
	if err != nil {
		t.Fatal(err)
	}
	actual.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
	if err := actual.Validate(); err != nil {
		t.Fatalf("unrelated digest fixture has invalid protocol signature: %v", err)
	}
	if err := pool.AddRemote(types.NewTransaction(actual)); err == nil {
		t.Fatal("explicit digest bypassed canonical tx authorization")
	}
	db.SetCode(payer, append(framepay.RuntimeV1(), 0))
	err = pool.AddRemote(signPoolFrame(t, frame, 1, 2))
	if err == nil || !strings.Contains(err.Error(), "third-party state") {
		t.Fatalf("nonmatching code received canonical trace exemption: %v", err)
	}
}

func TestFramePaymasterV1ReservationsIncludeDelayedWithdrawal(t *testing.T) {
	pool, db, frame, payer := paymasterPoolFixture(t)
	first := signPoolFrame(t, frame, 1, 2)
	if err := pool.AddRemote(first); err != nil {
		t.Fatal(err)
	}
	other := *frame
	other.Sender = framePoolAddress(t, 3)
	second := signPoolFrame(t, &other, 3, 2)
	if err := pool.AddRemote(second); err != nil {
		t.Fatalf("qualified runtime was limited to one sender: %v", err)
	}
	// The same availability bound applies before and after maturity.
	for _, ready := range []types.Hash{{31: 255}, {31: 1}} {
		db.SetState(payer, types.Hash{31: 2}, types.BytesToHash(big.NewInt(99000000).Bytes()))
		db.SetState(payer, types.Hash{31: 3}, ready)
		pool.Reset(db)
		if len(pool.PendingFlat()) != 1 {
			t.Fatal("aggregate exposure exceeded balance minus pending withdrawal")
		}
	}
	for _, pending := range pool.PendingFlat() {
		pool.Remove(pending.Hash())
	}
	if len(pool.lookup.framePayers) != 0 {
		t.Fatal("eviction retained reservation")
	}
	db.SetState(payer, types.Hash{31: 3}, types.Hash{})
	db.SetState(payer, types.Hash{31: 2}, types.Hash{})
	if err := pool.AddRemote(first); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddRemote(second); err != nil {
		t.Fatal(err)
	}
	// A canonical code change invalidates the exception and both cached prefixes.
	db.SetCode(payer, append(framepay.RuntimeV1(), 0))
	pool.Reset(db)
	if len(pool.PendingFlat()) != 0 || len(pool.lookup.framePayers) != 0 {
		t.Fatal("code change retained canonical paymaster privileges")
	}
}
