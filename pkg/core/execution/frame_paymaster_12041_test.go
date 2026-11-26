package execution

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func paymaster12041Key(t *testing.T, n int) *ecdsa.PrivateKey {
	t.Helper()
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", n))
	if err != nil {
		t.Fatal(err)
	}
	return key
}
func paymaster12041Sign(t *testing.T, frame *types.FrameTx8141, sender int, sponsor *ecdsa.PrivateKey, message []byte) *types.Transaction {
	t.Helper()
	keys := []*ecdsa.PrivateKey{paymaster12041Key(t, sender)}
	if sponsor != nil {
		keys = append(keys, sponsor)
	}
	frame.Signatures = make([]types.FrameSignature, len(keys))
	for i, key := range keys {
		address := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
		scheme := types.FrameSignatureSecp256k1
		size := 65
		if key.Curve == elliptic.P256() {
			scheme = types.FrameSignatureP256
			size = 128
		}
		frame.Signatures[i] = types.FrameSignature{Scheme: scheme, Signer: &address, Signature: make([]byte, size)}
		if i == 1 {
			frame.Signatures[i].Message = bytes.Clone(message)
		}
	}
	sign := func(i int, key *ecdsa.PrivateKey, digest []byte) {
		if key.Curve == elliptic.P256() {
			r, s, err := ecdsa.Sign(rand.Reader, key, digest)
			if err != nil {
				t.Fatal(err)
			}
			half := new(big.Int).Rsh(new(big.Int).Set(key.Params().N), 1)
			if s.Cmp(half) > 0 {
				s.Sub(key.Params().N, s)
			}
			sig := frame.Signatures[i].Signature
			r.FillBytes(sig[:32])
			s.FillBytes(sig[32:64])
			key.X.FillBytes(sig[64:96])
			key.Y.FillBytes(sig[96:])
		} else {
			sig, err := gethcrypto.Sign(digest, key)
			if err != nil {
				t.Fatal(err)
			}
			frame.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
		}
	}
	// Explicit-digest witnesses are committed bytes, not elided lengths.
	// Fill them before computing the canonical hash for the remaining entries.
	for i, key := range keys {
		if len(frame.Signatures[i].Message) > 0 {
			sign(i, key, frame.Signatures[i].Message)
		}
	}
	hash, err := frame.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range keys {
		if len(frame.Signatures[i].Message) == 0 {
			sign(i, key, hash[:])
		}
	}
	return types.NewTransaction(frame)
}

func TestPR12041NativePaymentAuthorization(t *testing.T) {
	for _, kind := range []string{"secp256k1", "p256", "wrong_signer", "explicit_digest", "zero_digest", "arbitrary", "missing", "corrupted", "inert"} {
		t.Run(kind, func(t *testing.T) {
			frame, db, cfg, header := frameExecutionFixture(t)
			payer := types.Address{19: 0x93}
			key := paymaster12041Key(t, 2)
			if kind == "p256" {
				var err error
				key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
			}
			owner := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
			if kind == "inert" {
				owner = types.Address{}
			}
			db.SetCode(payer, framepay.Runtime12041())
			db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
			db.AddBalance(payer, big.NewInt(100000000))
			frame.Frames[0].Flags = 2
			frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 15000}})
			var msg []byte
			if kind == "explicit_digest" {
				msg = bytes.Repeat([]byte{0x42}, 32)
			}
			if kind == "zero_digest" {
				msg = make([]byte, 32)
			}
			if kind == "wrong_signer" {
				key = paymaster12041Key(t, 3)
			}
			if kind == "missing" {
				key = nil
			}
			tx := paymaster12041Sign(t, frame, 1, key, msg)
			if kind == "arbitrary" {
				inner := tx.Inner().(*types.FrameTx8141)
				inner.Signatures[1] = types.FrameSignature{Scheme: types.FrameSignatureArbitrary, Signature: []byte{1}}
				hash, err := inner.SigningHash()
				if err != nil {
					t.Fatal(err)
				}
				sig, err := gethcrypto.Sign(hash[:], paymaster12041Key(t, 1))
				if err != nil {
					t.Fatal(err)
				}
				inner.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
				tx = types.NewTransaction(inner)
			}
			if kind == "corrupted" {
				inner := tx.Inner().(*types.FrameTx8141)
				inner.Signatures[1].Signature[12] ^= 1
				tx = types.NewTransaction(inner)
			}
			if kind != "zero_digest" && kind != "corrupted" {
				if err := tx.Inner().(*types.FrameTx8141).Validate(); err != nil {
					t.Fatal("runtime rejection fixture must have valid protocol signatures", err)
				}
			}
			root := db.GetRoot()
			before := new(big.Int).Set(db.GetBalance(payer))
			validation, err := ValidateFrame8141DependenciesWithPaymaster(cfg, db, header, tx, big.NewInt(1), framepay.PR12041)
			valid := kind == "secp256k1" || kind == "p256"
			if (err == nil) != valid {
				t.Fatalf("authorization %s: %v", kind, err)
			}
			if db.GetRoot() != root {
				t.Fatal("prefix mutated source state")
			}
			if !valid {
				_, _, err = ApplyTransaction(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
				if err == nil || db.GetRoot() != root {
					t.Fatal("invalid approval entered consensus or leaked state")
				}
				return
			}
			if validation.Payer != payer || len(validation.SignerRecoveries) != 0 {
				t.Fatal("SIGPARAM payment invented an ecRecover dependency")
			}
			for slot := byte(0); slot < 4; slot++ {
				if _, ok := validation.Dependencies[FrameDependency{Address: payer, Kind: FrameStorageDependency, Slot: types.Hash{31: slot}}]; !ok {
					t.Fatalf("untracked paymaster slot %d", slot)
				}
			}
			receipt, _, err := ApplyTransaction(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil || receipt.Payer != payer || db.GetNonce(frame.Sender) != 1 || db.GetBalance(payer).Cmp(before) >= 0 {
				t.Fatalf("native settlement: %v %+v", err, receipt)
			}
		})
	}
}

func TestPR12041NativeDeployment(t *testing.T) {
	for _, zero := range []bool{false, true} {
		frame, db, cfg, header := frameExecutionFixture(t)
		factory := types.Address{19: 0x90}
		db.SetCode(factory, []byte{0x36, 0x5f, 0x5f, 0x37, 0x36, 0x5f, 0x5f, 0xf0, 0x50, 0})
		signer := frame.Sender
		if zero {
			signer = types.Address{}
		}
		budget := uint64(183600 + len(framepay.Runtime12041())*1530)
		if !zero {
			budget += 97920
		}
		frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeSender, Target: &factory, Data: framepay.Init12041(signer), Limits: types.FrameGasLimits{Execution: 150000, State: budget}})
		receipt, _, err := ApplyTransaction(cfg, db, header, paymaster12041Sign(t, frame, 1, nil, nil), new(gaspool.GasPool).AddGas(header.GasLimit))
		created := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
		if err != nil || !framepay.Is12041(db.GetCode(created)) || db.GetState(created, types.Hash{}) != types.BytesToHash(signer[:]) || receipt.FrameResults8141[1].GasUsed.State != budget {
			t.Fatalf("deployment zero=%v: %v %+v", zero, err, receipt)
		}
	}
}

func TestPR12041NativeTreasuryAndAdministration(t *testing.T) {
	for _, kind := range []string{"caller_withdraw", "signature_withdraw", "p256_withdraw", "insufficient", "failed_send", "reentrant", "caller_rotation", "signature_rotation", "p256_rotation", "cancel_withdraw", "cancel_rotation", "caller_cancel", "p256_cancel"} {
		t.Run(kind, func(t *testing.T) {
			frame, db, cfg, header := frameExecutionFixture(t)
			payer := types.Address{19: 0x93}
			ownerKey := paymaster12041Key(t, 2)
			if strings.HasPrefix(kind, "p256_") {
				var err error
				ownerKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
			}
			owner := types.Address(gethcrypto.PubkeyToAddress(ownerKey.PublicKey))
			db.SetCode(payer, framepay.Runtime12041())
			db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
			db.AddBalance(payer, big.NewInt(2000000))
			db.AddBalance(owner, big.NewInt(100000000))
			var adminMessage []byte
			apply := func(data []byte, caller int, signature bool, value int64) *types.Receipt {
				t.Helper()
				frame.Sender = types.Address(gethcrypto.PubkeyToAddress(paymaster12041Key(t, caller).PublicKey))
				frame.Nonce = db.GetNonce(frame.Sender)
				frame.Frames = []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}, {Mode: types.ModeSender, Target: &payer, Data: data, Value: big.NewInt(value), Limits: types.FrameGasLimits{Execution: 200000, State: 4 * 97920}}}
				var key *ecdsa.PrivateKey
				if signature {
					key = ownerKey
				}
				tx := paymaster12041Sign(t, frame, caller, key, adminMessage)
				db.SetTxContext(tx.Hash(), 0)
				r, _, err := ApplyTransaction(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
				if err != nil {
					t.Fatal(err)
				}
				return r
			}
			for _, data := range [][]byte{{0xff}, {1}, {2}, {3}, {4}} {
				if r := apply(data, 1, false, 0); r.FrameResults8141[1].Status != 0 {
					t.Fatalf("invalid or unauthorized admin operation %x", data)
				}
			}
			// Value-bearing empty calldata deposits; nonempty admin calldata is non-payable.
			if r := apply(nil, 1, false, 7); r.FrameResults8141[1].Status != 1 || db.GetBalance(payer).Int64() != 2000007 {
				t.Fatal("deposit")
			}
			withdraw, _ := framepay.RequestWithdrawal12041(big.NewInt(400000))
			rotation, _ := framepay.RequestRotation12041(types.Address(gethcrypto.PubkeyToAddress(paymaster12041Key(t, 3).PublicKey)))
			adminMessage = bytes.Repeat([]byte{0x42}, 32)
			for _, data := range [][]byte{withdraw, rotation, framepay.Cancel12041()} {
				if r := apply(data, 1, true, 0); r.FrameResults8141[1].Status != 0 {
					t.Fatal("explicit-digest signature authorized administration")
				}
			}
			adminMessage = nil
			if r := apply(withdraw, 1, false, 0); r.FrameResults8141[1].Status != 0 || !db.GetState(payer, types.Hash{31: 2}).IsZero() {
				t.Fatal("unauthorized administration")
			}
			if r := apply(withdraw, 1, true, 1); r.FrameResults8141[1].Status != 0 || db.GetBalance(payer).Int64() != 2000007 {
				t.Fatal("payable administration")
			}
			rotating := strings.Contains(kind, "rotation")
			data := withdraw
			if rotating {
				data = rotation
			}
			caller, signature := 1, true
			if kind == "caller_withdraw" || kind == "caller_rotation" || kind == "caller_cancel" {
				caller, signature = 2, false
			}
			start := header.Time
			r := apply(data, caller, signature, 0)
			ready := start + framepay.WithdrawalDelay12041
			if r.FrameResults8141[1].Status != 1 || db.GetState(payer, types.Hash{31: 2}) != types.BytesToHash(new(big.Int).SetUint64(ready).Bytes()) {
				t.Fatalf("init: %+v", r.FrameResults8141)
			}
			if r := apply(data, caller, signature, 0); r.FrameResults8141[1].Status != 0 {
				t.Fatal("overwrote pending action")
			}
			if strings.Contains(kind, "cancel") {
				if r := apply(framepay.Cancel12041(), caller, signature, 0); r.FrameResults8141[1].Status != 1 {
					t.Fatal("cancel failed")
				}
				for _, slot := range []byte{1, 2, 3} {
					if !db.GetState(payer, types.Hash{31: slot}).IsZero() {
						t.Fatal("cancel retained pending state")
					}
				}
				return
			}
			header.Time = ready - 1
			if r := apply(framepay.Finalize12041(), 1, false, 0); r.FrameResults8141[1].Status != 0 {
				t.Fatal("early finalize")
			}
			header.Time = ready
			if kind == "insufficient" {
				db.SubBalance(payer, big.NewInt(1900000))
			}
			if kind == "failed_send" {
				db.SetCode(owner, []byte{0x5f, 0x5f, 0xfd})
			}
			if kind == "reentrant" {
				code := []byte{0x60, 4, 0x5f, 0x53, 0x5f, 0x5f, 0x60, 1, 0x5f, 0x5f, 0x73}
				code = append(code, payer[:]...)
				code = append(code, 0x62, 0x01, 0x86, 0xa0, 0xf1, 0x50, 0)
				db.SetCode(owner, code)
			}
			before := new(big.Int).Set(db.GetBalance(owner))
			payerBefore := new(big.Int).Set(db.GetBalance(payer))
			r = apply(framepay.Finalize12041(), 1, false, 0)
			if kind == "insufficient" || kind == "failed_send" {
				if r.FrameResults8141[1].Status != 0 || r.FrameResults8141[1].GasUsed.State != 0 || db.GetBalance(payer).Cmp(payerBefore) != 0 || framepay.PR12041.PendingWithdrawal(db, payer).Int64() != 400000 || db.GetState(payer, types.Hash{31: 2}).IsZero() {
					t.Fatal("failed CALL lost funds/pending state")
				}
				return
			}
			if r.FrameResults8141[1].Status != 1 {
				t.Fatalf("mature finalize: %+v", r.FrameResults8141)
			}
			for _, slot := range []byte{1, 2, 3} {
				if !db.GetState(payer, types.Hash{31: slot}).IsZero() {
					t.Fatal("finalize retained pending state")
				}
			}
			if rotating {
				next := types.Address(gethcrypto.PubkeyToAddress(paymaster12041Key(t, 3).PublicKey))
				if db.GetState(payer, types.Hash{}) != types.BytesToHash(next[:]) || db.GetBalance(payer).Cmp(payerBefore) != 0 {
					t.Fatal("rotation moved funds or lost new authority")
				}
			} else if new(big.Int).Sub(db.GetBalance(owner), before).Int64() != 400000 || new(big.Int).Sub(payerBefore, db.GetBalance(payer)).Int64() != 400000 {
				t.Fatal("withdrawal amount/recipient/reentrancy")
			}
			if r := apply(framepay.Finalize12041(), 1, false, 0); r.FrameResults8141[1].Status != 0 {
				t.Fatal("replayed matured action")
			}
		})
	}
}

func TestPR12041AuthorizationBindsEntireTransaction(t *testing.T) {
	for _, kind := range []string{"chain", "nonce", "sender", "fees", "pay_target", "body"} {
		t.Run(kind, func(t *testing.T) {
			frame, _, _, _ := frameExecutionFixture(t)
			payer := types.Address{19: 0x93}
			frame.Frames[0].Flags = 2
			frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 15000}})
			tx := paymaster12041Sign(t, frame, 1, paymaster12041Key(t, 2), nil)
			inner := tx.Inner().(*types.FrameTx8141)
			before := tx.SigningHash()
			switch kind {
			case "chain":
				inner.ChainID = new(big.Int).Add(inner.ChainID, big.NewInt(1))
			case "nonce":
				inner.Nonce++
			case "sender":
				inner.Sender = types.Address{19: 7}
			case "fees":
				inner.Fees.MaxFee = new(big.Int).Add(inner.Fees.MaxFee, big.NewInt(1))
			case "pay_target":
				next := types.Address{19: 0x94}
				inner.Frames[1].Target = &next
			case "body":
				inner.Frames = append(inner.Frames, types.Frame8141{Mode: types.ModeSender, Limits: types.FrameGasLimits{Execution: 1000}})
			}
			after, err := inner.SigningHash()
			if err != nil {
				t.Fatal(err)
			}
			if after == before || inner.Validate() == nil {
				t.Fatal("old sponsor authorization survived a transaction change")
			}
		})
	}
}

func TestPR12041PublicPolicyDoesNotChangeConsensus(t *testing.T) {
	for _, profile := range []framepay.Profile{framepay.NoPaymaster, framepay.PR12041} {
		frame, db, cfg, header := frameExecutionFixture(t)
		payer := types.Address{19: 0x93}
		owner := types.Address(gethcrypto.PubkeyToAddress(paymaster12041Key(t, 2).PublicKey))
		db.SetCode(payer, framepay.Runtime12041())
		db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
		db.AddBalance(payer, big.NewInt(100000000))
		frame.Frames[0].Flags = 2
		frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 15001}})
		tx := paymaster12041Sign(t, frame, 1, paymaster12041Key(t, 2), nil)
		if _, err := ValidateFrame8141DependenciesWithPaymaster(cfg, db, header, tx, big.NewInt(1), profile); err == nil {
			t.Fatal("public profile policy bypassed")
		}
		if _, _, err := ApplyTransaction(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit)); err != nil {
			t.Fatal("local recognition/gas policy changed consensus", err)
		}
	}
}

func TestPR12041OrdinaryCallerAdministration(t *testing.T) {
	_, db, cfg, header := frameExecutionFixture(t)
	payer := types.Address{19: 0x93}
	ownerKey := paymaster12041Key(t, 2)
	owner := types.Address(gethcrypto.PubkeyToAddress(ownerKey.PublicKey))
	db.SetCode(payer, framepay.Runtime12041())
	db.SetState(payer, types.Hash{}, types.BytesToHash(owner[:]))
	db.AddBalance(payer, big.NewInt(2000000))
	db.AddBalance(owner, big.NewInt(1000000000))
	apply := func(caller int, data []byte, value int64) *types.Receipt {
		t.Helper()
		key := paymaster12041Key(t, caller)
		from := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
		to := gethcommon.Address(payer)
		unsigned := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: cfg.ChainID, Nonce: db.GetNonce(from), To: &to, Gas: 1000000, GasFeeCap: big.NewInt(3), GasTipCap: big.NewInt(2), Data: data, Value: big.NewInt(value)})
		signed, err := gethtypes.SignTx(unsigned, gethtypes.LatestSignerForChainID(cfg.ChainID), key)
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
		db.SetTxContext(tx.Hash(), 0)
		receipt, _, err := ApplyTransaction(cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
		if err != nil {
			t.Fatal(err)
		}
		return receipt
	}
	data, _ := framepay.RequestWithdrawal12041(big.NewInt(400000))
	if r := apply(1, data, 0); r.Status != 0 || !db.GetState(payer, types.Hash{31: 2}).IsZero() {
		t.Fatal("ordinary non-owner authorized without a Frame signature context")
	}
	if r := apply(2, data, 0); r.Status != 1 || framepay.PR12041.PendingWithdrawal(db, payer).Int64() != 400000 {
		t.Fatal("ordinary owner route failed")
	}
	if r := apply(1, nil, 7); r.Status != 1 || db.GetBalance(payer).Int64() != 2000007 {
		t.Fatal("ordinary deposit failed")
	}
	header.Time += framepay.WithdrawalDelay12041
	before := new(big.Int).Set(db.GetBalance(owner))
	if r := apply(1, framepay.Finalize12041(), 0); r.Status != 1 || new(big.Int).Sub(db.GetBalance(owner), before).Int64() != 400000 || framepay.PR12041.PendingWithdrawal(db, payer).Sign() != 0 {
		t.Fatal("permissionless ordinary finalization failed")
	}
}
