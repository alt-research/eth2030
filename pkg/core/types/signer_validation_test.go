package types

import (
	"fmt"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"math/big"
	"testing"
)

func TestNativeLegacySignerHashMatchesIndependentUnsignedTransaction(t *testing.T) {
	to := Address{19: 0x42}
	sdkTo := gethcommon.Address(to)
	native := NewTransaction(&LegacyTx{Nonce: 7, GasPrice: big.NewInt(3), Gas: 25000, To: &to, Value: big.NewInt(5), Data: []byte{7}})
	reference := gethtypes.NewTransaction(7, sdkTo, big.NewInt(5), 25000, big.NewInt(3), []byte{7})
	want := Hash(gethtypes.NewEIP155Signer(big.NewInt(1337)).Hash(reference))
	if got := NewEIP155Signer(1337).Hash(native); got != want {
		t.Fatalf("unsigned native EIP-155 hash=%s want independent %s", got, want)
	}
	if got := NewLondonSigner(1337).Hash(native); got != want {
		t.Fatalf("unsigned native London legacy hash=%s want independent %s", got, want)
	}
}
func TestNativeSignerRejectsMalleabilityAndFullWidthAliases(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	to := gethcommon.Address{19: 0x42}
	for _, kind := range []uint8{0, 2} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			var inner gethtypes.TxData = &gethtypes.LegacyTx{Nonce: 0, GasPrice: big.NewInt(3), Gas: 25000, To: &to, Value: new(big.Int)}
			if kind == 2 {
				inner = &gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Gas: 25000, GasFeeCap: big.NewInt(3), GasTipCap: big.NewInt(1), To: &to, Value: new(big.Int)}
			}
			signed, err := gethtypes.SignTx(gethtypes.NewTx(inner), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := signed.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			tx, err := DecodeTxRLP(raw)
			if err != nil {
				t.Fatal(err)
			}
			signer := MakeSigner(1337, kind)
			if _, err := signer.Sender(tx); err != nil {
				t.Fatal(err)
			}
			v, r, s := tx.RawSignatureValues()
			high := new(big.Int).Sub(gethcrypto.S256().Params().N, s)
			switch p := tx.Inner().(type) {
			case *LegacyTx:
				p.S = high
				p.V = new(big.Int).Set(v)
				if v.Bit(0) == 0 {
					p.V.Sub(v, big.NewInt(1))
				} else {
					p.V.Add(v, big.NewInt(1))
				}
			case *DynamicFeeTx:
				p.S = high
				p.V = new(big.Int).Xor(v, big.NewInt(1))
			}
			if _, err := signer.Sender(tx); err == nil {
				t.Fatal("high-S transaction accepted")
			}
			// Generic ecrecover remains permitted to recover a high-S signature.
			parity := byte(0)
			if kind == 0 {
				parity = byte(new(big.Int).Sub(v, big.NewInt(35+2*1337)).Uint64())
			} else {
				parity = byte(v.Uint64())
			}
			if _, err := RecoverPlain(tx.SigningHash(), r, high, parity^1); err != nil {
				t.Fatalf("generic high-S recovery changed: %v", err)
			}
			tx, err = DecodeTxRLP(raw)
			if err != nil {
				t.Fatal(err)
			}
			switch p := tx.Inner().(type) {
			case *LegacyTx:
				p.V = new(big.Int).Add(p.V, new(big.Int).Lsh(big.NewInt(1), 65))
			case *DynamicFeeTx:
				p.ChainID = new(big.Int).Add(p.ChainID, new(big.Int).Lsh(big.NewInt(1), 64))
			}
			if _, err := signer.Sender(tx); err == nil {
				t.Fatal("full-width chain ID alias accepted")
			}
			if kind == 2 {
				tx, err = DecodeTxRLP(raw)
				if err != nil {
					t.Fatal(err)
				}
				p := tx.Inner().(*DynamicFeeTx)
				p.V = new(big.Int).Add(p.V, big.NewInt(256))
				if _, err := signer.Sender(tx); err == nil {
					t.Fatal("wide parity alias accepted")
				}
			}
		})
	}
}
func TestSignedEnvelopeRecipientsAreNotNormalizedFromCreation(t *testing.T) {
	for _, kind := range []byte{BlobTxType, SetCodeTxType} {
		fields := []interface{}{big.NewInt(1), uint64(0), big.NewInt(1), big.NewInt(2), uint64(100000), []byte{}, new(big.Int), []byte{}, []gethtypes.AccessTuple{}}
		if kind == BlobTxType {
			fields = append(fields, big.NewInt(1), []gethcommon.Hash{{0: 1}})
		} else {
			fields = append(fields, []gethtypes.SetCodeAuthorization{{ChainID: *uint256.NewInt(1)}})
		}
		fields = append(fields, uint64(0), big.NewInt(1), big.NewInt(1))
		encoded, err := gethrlp.EncodeToBytes(fields)
		if err != nil {
			t.Fatal(err)
		}
		raw := append([]byte{kind}, encoded...)
		var reference gethtypes.Transaction
		if err := reference.UnmarshalBinary(raw); err == nil {
			t.Fatal("invalid independent recipient fixture")
		}
		if tx, err := DecodeTxRLP(raw); err == nil {
			t.Fatalf("type %d prohibited creation normalized to %s", kind, tx.To())
		}
	}
	legacy := NewTransaction(&LegacyTx{GasPrice: big.NewInt(1), Gas: 21000, Value: new(big.Int)})
	raw, err := legacy.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeTxRLP(append([]byte{0}, raw...)); err == nil {
		t.Fatal("unassigned typed-legacy prefix accepted")
	}
}
