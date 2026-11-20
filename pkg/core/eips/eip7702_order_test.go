package eips

import (
	"bytes"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func signedHistoricalAuthorization(t *testing.T, chain, nonce uint64, target types.Address) (types.Authorization, types.Address) {
	t.Helper()
	key, err := gethcrypto.HexToECDSA(strings.Repeat("0", 63) + "2")
	if err != nil {
		t.Fatal(err)
	}
	signed, err := gethtypes.SignSetCode(key, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(chain), Address: gethcommon.Address(target), Nonce: nonce})
	if err != nil {
		t.Fatal(err)
	}
	return types.Authorization{ChainID: signed.ChainID.ToBig(), Address: target, Nonce: nonce, V: big.NewInt(int64(signed.V)), R: signed.R.ToBig(), S: signed.S.ToBig()}, types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
}

func TestHistoricalAuthorizationValidationOrdering(t *testing.T) {
	target := types.HexToAddress("0x123456")
	for _, tc := range []struct {
		name                         string
		chain, authNonce, stateNonce uint64
		code                         []byte
		invalidSig, warm             bool
	}{
		{name: "wrong_nonce_warms", chain: 1337, authNonce: 1, warm: true},
		{name: "contract_code_warms_but_skips", chain: 1337, code: []byte{0x00}, warm: true},
		{name: "max_nonce_skips_before_warming", chain: 1337, authNonce: math.MaxUint64, stateNonce: math.MaxUint64},
		{name: "wrong_chain_skips_before_warming", chain: 2},
		{name: "bad_signature_skips_before_warming", chain: 1337, invalidSig: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			auth, authority := signedHistoricalAuthorization(t, tc.chain, tc.authNonce, target)
			if tc.invalidSig {
				auth.V = big.NewInt(2)
			}
			db := state.NewMemoryStateDB()
			db.AddBalance(authority, big.NewInt(5))
			db.SetNonce(authority, tc.stateNonce)
			db.SetCode(authority, tc.code)
			if err := ProcessAuthorizations(db, []types.Authorization{auth}, big.NewInt(1337)); err != nil {
				t.Fatal(err)
			}
			if db.AddressInAccessList(authority) != tc.warm {
				t.Fatalf("authority warm=%t want %t", db.AddressInAccessList(authority), tc.warm)
			}
			if db.GetNonce(authority) != tc.stateNonce || !bytes.Equal(db.GetCode(authority), tc.code) || db.GetRefund() != 0 {
				t.Fatalf("skipped authorization changed nonce/code/refund: nonce=%d code=%x refund=%d", db.GetNonce(authority), db.GetCode(authority), db.GetRefund())
			}
		})
	}
}

func TestHistoricalAuthorizationClearAndDuplicates(t *testing.T) {
	target := types.HexToAddress("0x123456")
	first, authority := signedHistoricalAuthorization(t, 1337, 0, target)
	repeated, _ := signedHistoricalAuthorization(t, 1337, 0, types.HexToAddress("0x9999"))
	clear, _ := signedHistoricalAuthorization(t, 1337, 1, types.Address{})
	db := state.NewMemoryStateDB()
	if err := ProcessAuthorizations(db, []types.Authorization{first, repeated, clear}, big.NewInt(1337)); err != nil {
		t.Fatal(err)
	}
	if !db.AddressInAccessList(authority) || db.GetNonce(authority) != 2 || len(db.GetCode(authority)) != 0 || db.GetRefund() != 12500 {
		t.Fatalf("duplicate/clear result: warm=%t nonce=%d code=%x refund=%d", db.AddressInAccessList(authority), db.GetNonce(authority), db.GetCode(authority), db.GetRefund())
	}
}
