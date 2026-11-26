package txpool

import (
	"fmt"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math"
	"math/big"
	"testing"
)

func signKeyedPolicyFrame(t *testing.T, tx *types.FrameTx8250Delta, keyNumber int) {
	t.Helper()
	tx.Signatures = []types.FrameSignature{{Scheme: 1, Signature: make([]byte, 65)}}
	hash, err := tx.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", keyNumber))
	sig, err := gethcrypto.Sign(hash[:], key)
	if err != nil {
		t.Fatal(err)
	}
	tx.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
}
func TestFrameCIsolatedKeyedPolicyIdentityDependenciesAndExhaustion(t *testing.T) {
	for _, legacyRead := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy_read_%t", legacyRead), func(t *testing.T) {
			native, db, base := framePoolFixture(t)
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			frames := append([]types.Frame8141(nil), base.Frames...)
			frames[0].Limits.State = 97920
			frames[0].Limits.Execution = 30000
			if legacyRead {
				db.SetCode(base.Sender, []byte{0x60, 0x0d, 0xb0, 0x50, 0x60, 3, 0x5f, 0x5f, 0xaa})
			}
			tx := &types.FrameTx8250Delta{ChainID: base.ChainID, Sender: base.Sender, NonceKeys: []*big.Int{big.NewInt(1)}, NonceSeq: math.MaxUint64, Fees: base.Fees, Frames: frames}
			signKeyedPolicyFrame(t, tx, 1)
			policy := NewKeyedFramePolicy8250Delta()
			slot := uint64(2)
			head := &types.Header{Number: big.NewInt(1), Time: 100, SlotNumber: &slot, GasLimit: 30000000, BaseFee: big.NewInt(1)}
			if err := policy.Admit(native.frameConfig, db, head, tx, big.NewInt(1)); err != nil {
				t.Fatal(err)
			}
			entry := policy.entries[tx.Sender]
			dep := execution.FrameDependency{Address: tx.Sender, Kind: execution.FrameNonceDependency}
			if _, tracked := entry.validation.Dependencies[dep]; tracked != legacyRead {
				t.Fatalf("legacy nonce tracked=%t want=%t", tracked, legacyRead)
			}
			replacement := *tx
			replacement.NonceKeys = []*big.Int{big.NewInt(2)}
			replacement.Fees = types.FrameFeeBudget{MaxFee: big.NewInt(1100000), MaxPriorityFeePerGas: big.NewInt(3)}
			signKeyedPolicyFrame(t, &replacement, 1)
			if err := policy.Admit(native.frameConfig, db, head, &replacement, big.NewInt(1)); err != ErrReplacementUnderpriced {
				t.Fatalf("disjoint second pending sender: %v", err)
			}
			db.SetState(types.NonceManager8250, types.NonceSlot8250(tx.Sender, big.NewInt(2)), types.Hash{31: 9})
			policy.Revalidate(native.frameConfig, db, head, big.NewInt(1))
			if policy.entries[tx.Sender] != entry || entry.validation == nil {
				t.Fatal("unselected key invalidated transaction")
			}
			before := entry.validation
			db.SetNonce(tx.Sender, 4)
			policy.Revalidate(native.frameConfig, db, head, big.NewInt(1))
			if !policy.Contains(tx.Sender) || (entry.validation != before) != legacyRead {
				t.Fatal("optional legacy dependency revalidation differs")
			}
			// MAX is admissible only for an unused selected key; consumed MAX is exhausted.
			db.SetState(types.NonceManager8250, types.NonceSlot8250(tx.Sender, big.NewInt(1)), types.BytesToHash(new(big.Int).SetUint64(math.MaxUint64).Bytes()))
			policy.Revalidate(native.frameConfig, db, head, big.NewInt(1))
			if policy.Contains(tx.Sender) {
				t.Fatal("consumed single-use key retained")
			}
			raw, _ := tx.Encode()
			public, err := types.DecodeTxRLP(raw)
			if err != nil {
				t.Fatal(err)
			}
			if err := native.AddRemote(public); err == nil {
				t.Fatal("keyed envelope admitted before activation")
			}
		})
	}
}

func TestFrameCIsolatedKeyedSponsorCodeAdoptionCapsPaymaster(t *testing.T) {
	native, db, base := framePoolFixture(t)
	db.SetNonce(types.NonceManager8250, 1)
	db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
	payer := framePoolAddress(t, 3)
	head := &types.Header{Number: big.NewInt(1), Time: 100, GasLimit: 30000000, BaseFee: big.NewInt(1)}
	policy := NewKeyedFramePolicy8250Delta()
	for _, keyNumber := range []int{1, 2} {
		tx := &types.FrameTx8250Delta{ChainID: base.ChainID, Sender: framePoolAddress(t, keyNumber), NonceKeys: []*big.Int{big.NewInt(1)}, Fees: base.Fees, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 2, Limits: types.FrameGasLimits{Execution: 1000}}, {Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 10000, State: 97920}}}}
		signKeyedPolicyFrame(t, tx, keyNumber)
		tx.Signatures = append(tx.Signatures, types.FrameSignature{Scheme: 1, Signer: &payer, Signature: make([]byte, 65)})
		hash, _ := tx.SigningHash()
		for i, k := range []int{keyNumber, 3} {
			key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", k))
			sig, _ := gethcrypto.Sign(hash[:], key)
			tx.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
		}
		if err := policy.Admit(native.frameConfig, db, head, tx, big.NewInt(1)); err != nil {
			t.Fatal(err)
		}
	}
	if len(policy.entries) != 2 {
		t.Fatal("empty-code sponsor incorrectly capped")
	}
	db.SetCode(payer, []byte{0x60, 1, 0x5f, 0x5f, 0xaa})
	policy.Revalidate(native.frameConfig, db, head, big.NewInt(1))
	if len(policy.entries) != 1 {
		t.Fatal("code adoption bypassed noncanonical paymaster cap")
	}
}

// EIP-8250 preserves EIP-8141's independent block reservations. A stable
// validation read set does not make a transaction fit a smaller new block.
func TestFrameCIsolatedKeyedCachedDimensionalLimits(t *testing.T) {
	for _, dimension := range []string{"execution", "state"} {
		t.Run(dimension, func(t *testing.T) {
			native, db, base := framePoolFixture(t)
			db.SetNonce(types.NonceManager8250, 1)
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			frames := append([]types.Frame8141(nil), base.Frames...)
			frames[0].Limits.Execution = 30000
			frames[0].Limits.State = 97920
			tx := &types.FrameTx8250Delta{ChainID: base.ChainID, Sender: base.Sender, NonceKeys: []*big.Int{big.NewInt(1)}, Fees: base.Fees, Frames: frames}
			if dimension == "execution" {
				tx.NonceSeq = 1
				tx.Frames[0].Limits.State = 0
				db.SetState(types.NonceManager8250, types.NonceSlot8250(tx.Sender, tx.NonceKeys[0]), types.Hash{31: 1})
			}
			signKeyedPolicyFrame(t, tx, 1)
			budget, err := tx.GasBudget()
			if err != nil {
				t.Fatal(err)
			}
			limit := budget.ExecutionReservation
			if dimension == "state" {
				limit = budget.StateReservation
				if budget.ExecutionReservation >= limit {
					t.Fatal("state boundary does not isolate the state reservation")
				}
			}
			head := &types.Header{Number: big.NewInt(1), Time: 100, GasLimit: limit, BaseFee: big.NewInt(1)}
			policy := NewKeyedFramePolicy8250Delta()
			if err := policy.Admit(native.frameConfig, db, head, tx, big.NewInt(1)); err != nil {
				t.Fatal("exact boundary admission:", err)
			}
			validation := policy.entries[tx.Sender].validation
			policy.Revalidate(native.frameConfig, db, head, big.NewInt(1))
			if !policy.Contains(tx.Sender) || policy.entries[tx.Sender].validation != validation {
				t.Fatal("exact unchanged boundary did not reuse validation")
			}
			head.GasLimit--
			if _, err := execution.ValidateFrame8250DeltaPrefix(native.frameConfig, db, head, tx, big.NewInt(1)); err == nil {
				t.Fatal("native execution unexpectedly fits below the reservation")
			}
			policy.Revalidate(native.frameConfig, db, head, big.NewInt(1))
			if policy.Contains(tx.Sender) {
				t.Fatal("cached validation retained a transaction exceeding the new dimensional limit")
			}
		})
	}
}
