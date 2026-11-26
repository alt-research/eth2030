package txpool

import (
	"encoding/hex"
	"fmt"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

// A coded sender actually invokes ecRecover. Its arbitrary witness is elided
// by length, permitting an independently signed canonical digest at runtime.
func recoveredSignerFrame8151(t *testing.T, pool *TxPool, senderKey int, canonical bool, paymaster *types.Address) *types.Transaction {
	t.Helper()
	db := pool.state.(interface {
		SetCode(types.Address, []byte)
		AddBalance(types.Address, *big.Int)
	})
	sender := framePoolAddress(t, senderKey)
	code, _ := hex.DecodeString("5f60805f5fb5") // SIGDATACOPY(0, 0, 128, 0)
	if canonical {
		code = append(code, 0x60, 8, 0xb0, 0x5f, 0x52)
	} // TXPARAM(8) -> memory[0]
	call, _ := hex.DecodeString("5f5f60805f600161fffffa50")
	code = append(code, call...)
	flags := byte(3)
	if paymaster != nil {
		flags = 1
	}
	code = append(code, 0x60, flags, 0x5f, 0x5f, 0xaa)
	target := sender
	if paymaster != nil {
		target = *paymaster
	}
	db.SetCode(target, code)
	db.AddBalance(sender, big.NewInt(100000000))
	db.AddBalance(target, big.NewInt(100000000))
	frame := &types.FrameTx8141{ChainID: pool.frameConfig.ChainID, Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(2)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 80000}}}, Signatures: []types.FrameSignature{{Scheme: 0, Signature: make([]byte, 128)}, {Scheme: 1, Signature: make([]byte, 65)}}}
	if paymaster != nil {
		db.SetCode(sender, []byte{0x60, 2, 0x5f, 0x5f, 0xaa})
		frame.Frames = []types.Frame8141{{Mode: types.ModeVerify, Flags: 2, Limits: types.FrameGasLimits{Execution: 1000}}, {Mode: types.ModeVerify, Flags: 1, Target: paymaster, Limits: types.FrameGasLimits{Execution: 80000}}}
	}
	signing, err := frame.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	digest := signing
	if !canonical {
		digest = types.Hash{31: 42}
	}
	recoveryKey, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
	sig, err := gethcrypto.Sign(digest[:], recoveryKey)
	if err != nil {
		t.Fatal(err)
	}
	input := frame.Signatures[0].Signature
	copy(input, digest[:])
	input[63] = 27 + sig[64]
	copy(input[64:], sig[:64])
	key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", senderKey))
	signature, err := gethcrypto.Sign(signing[:], key)
	if err != nil {
		t.Fatal(err)
	}
	frame.Signatures[1].Signature = append([]byte{signature[64]}, signature[:64]...)
	return types.NewTransaction(frame)
}

func enableRecovery8151(pool *TxPool) {
	zero := uint64(0)
	pool.frameConfig.GlamsterdanTime = &zero
	pool.frameConfig.EIP8151Time = &zero
}

func TestFrame8151SignerClassesReorgAndSeparateHelperDependency(t *testing.T) {
	pool, db, _ := framePoolFixture(t)
	enableRecovery8151(pool)
	tx := recoveredSignerFrame8151(t, pool, 1, true, nil)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	signer := framePoolAddress(t, 2)
	dep := execution.FrameDependency{Address: signer, Kind: execution.FrameSignerClassDependency}
	validation := pool.lookup.frameValidations[tx.Hash()]
	if validation.Dependencies[dep] != "permitted" || len(validation.SignerRecoveries) != 1 || !validation.SignerRecoveries[0].Canonical {
		t.Fatalf("missing canonical signer dependency: %+v", validation)
	}
	hashDep := execution.FrameDependency{Address: signer, Kind: execution.FrameCodeDependency}
	if _, ok := validation.Dependencies[hashDep]; ok {
		t.Fatal("recovery added unnecessary code hash dependency")
	}
	db.SetCode(signer, types.AddressToDelegation(types.Address{19: 0x66}))
	pool.Reset(db)
	if pool.lookup.frameValidations[tx.Hash()] != validation {
		t.Fatal("same class forced re-execution")
	}
	db.SetCode(signer, []byte{0})
	pool.Reset(db)
	rejected := pool.lookup.frameValidations[tx.Hash()]
	if rejected == nil || rejected == validation || rejected.Dependencies[dep] != "rejected" {
		t.Fatal("permitted -> rejected was not revalidated")
	}
	db.SetCode(signer, nil)
	pool.Reset(db)
	restored := pool.lookup.frameValidations[tx.Hash()]
	if restored == nil || restored == rejected || restored.Dependencies[dep] != "permitted" {
		t.Fatal("reorg rejected -> permitted was not revalidated")
	}
	// An independent helper hash dependency remains authoritative even when the
	// signer class is stable; the dependency kinds must never overwrite each other.
	restored.Dependencies[hashDep] = hashDep.Value(db)
	db.SetCode(signer, types.AddressToDelegation(types.Address{19: 0x67}))
	if restored.Unchanged(db) {
		t.Fatal("signer-class caching erased helper hash dependency")
	}
	pool.Remove(tx.Hash())
	if len(pool.lookup.frameDependencies) != 0 {
		t.Fatal("removed signer dependency leaked")
	}
}

func TestFrame8151ReservedAndOtherDigestShares(t *testing.T) {
	pool, _, _ := framePoolFixture(t)
	enableRecovery8151(pool)
	pool.config.FrameSignerCanonicalLimit = 2
	pool.config.FrameSignerOtherLimit = 1
	// Replaying an unrelated digest exhausts only the smaller share.
	first := recoveredSignerFrame8151(t, pool, 1, false, nil)
	if err := pool.AddRemote(first); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddRemote(recoveredSignerFrame8151(t, pool, 3, false, nil)); err == nil {
		t.Fatal("other digest share exceeded")
	}
	canonical := recoveredSignerFrame8151(t, pool, 3, true, nil)
	if err := pool.AddRemote(canonical); err != nil {
		t.Fatal("unrelated digest consumed reserved share", err)
	}
	if err := pool.AddRemote(recoveredSignerFrame8151(t, pool, 4, true, nil)); err != nil {
		t.Fatal(err)
	}
	if err := pool.AddRemote(recoveredSignerFrame8151(t, pool, 5, true, nil)); err == nil {
		t.Fatal("canonical share exceeded")
	}
	pool.Remove(canonical.Hash())
	if err := pool.AddRemote(recoveredSignerFrame8151(t, pool, 5, true, nil)); err != nil {
		t.Fatal("eviction did not release signer share", err)
	}
}

func TestFrame8151PayRecoveriesCountDistinctPaymasters(t *testing.T) {
	pool, _, _ := framePoolFixture(t)
	enableRecovery8151(pool)
	pool.config.FrameSignerCanonicalLimit = 2
	pool.config.FrameSignerOtherLimit = 1
	payer := types.Address{19: 0xa1}
	tx := recoveredSignerFrame8151(t, pool, 1, false, &payer)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	validation := pool.lookup.frameValidations[tx.Hash()]
	if len(validation.SignerRecoveries) != 1 || !validation.SignerRecoveries[0].Pay || validation.SignerRecoveries[0].Paymaster != payer {
		t.Fatal("PAY recovery misclassified")
	}
	// This checks the signer accounting independently of generic PAY's stricter
	// one-pending transaction rule. Multiple txs of a future pinned paymaster
	// must consume a single signer share, while another paymaster consumes one.
	same := recoveredSignerFrame8151(t, pool, 3, false, &payer)
	frame := same.Inner().(*types.FrameTx8141)
	if err := pool.validateFrameSigners8151(frame, validation); err != nil {
		t.Fatal("same paymaster counted per transaction", err)
	}
	other := *validation
	other.SignerRecoveries = append([]execution.FrameSignerRecovery8151(nil), validation.SignerRecoveries...)
	other.SignerRecoveries[0].Paymaster = types.Address{19: 0xa2}
	if err := pool.validateFrameSigners8151(frame, &other); err == nil {
		t.Fatal("distinct paymasters did not exhaust smaller PAY share")
	}
	if err := pool.AddRemote(recoveredSignerFrame8151(t, pool, 4, false, nil)); err != nil {
		t.Fatal("PAY recovery consumed non-PAY share", err)
	}
}

func TestFrame8151ActivationInvalidatesPrefixCache(t *testing.T) {
	pool, db, _ := framePoolFixture(t)
	enableRecovery8151(pool)
	activation := uint64(100)
	pool.frameConfig.EIP8151Time = &activation
	pool.SetForkRules(pool.frameConfig, 99)
	tx := recoveredSignerFrame8151(t, pool, 1, true, nil)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	before := pool.lookup.frameValidations[tx.Hash()]
	if len(before.SignerRecoveries) != 0 {
		t.Fatal("pre-fork recovery acquired signer dependencies")
	}
	pool.SetForkRules(pool.frameConfig, 100)
	pool.Reset(db)
	after := pool.lookup.frameValidations[tx.Hash()]
	if after == nil || after == before || len(after.SignerRecoveries) != 1 {
		t.Fatal("EIP-8151 activation reused stateless prefix cache")
	}
	pool.SetForkRules(pool.frameConfig, 99)
	pool.Reset(db)
	restored := pool.lookup.frameValidations[tx.Hash()]
	if restored == nil || restored == after || len(restored.SignerRecoveries) != 0 {
		t.Fatal("activation rollback retained stateful prefix cache")
	}
}
