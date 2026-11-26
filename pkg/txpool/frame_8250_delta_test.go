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
			if _, err := types.DecodeTxRLP(raw); err == nil {
				t.Fatal("isolated keyed envelope escaped D01 public decoding gate")
			}
		})
	}
}
