package framepay

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestRuntimeIdentityAndWithdrawalLayout(t *testing.T) {
	code := RuntimeV1()
	if types.Hash(gethcrypto.Keccak256Hash(code)) != types.HexToHash("0xca4aa4d8e001959081cc44528b5aeec6dbda2e5adfc0c5544f70da1e622c3f32") {
		t.Fatal("frozen v1 runtime identity changed")
	}
	if len(code) == 0 || !IsV1(code) {
		t.Fatal("compiled runtime missing")
	}
	code[0] ^= 1
	if IsV1(code) || !IsV1(RuntimeV1()) || IsV1(append(RuntimeV1(), 0)) {
		t.Fatal("runtime matching accepts mutations or exposes its backing memory")
	}
	if _, err := InitV1(types.Address{}); err == nil {
		t.Fatal("zero owner accepted")
	}
	owner := types.Address{19: 1}
	init, err := InitV1(owner)
	if err != nil || !bytes.Equal(init[len(init)-20:], owner[:]) {
		t.Fatal("owner argument encoding")
	}
	db := state.NewMemoryStateDB()
	payer := types.Address{19: 2}
	db.SetState(payer, types.Hash{31: 2}, types.Hash{31: 9})
	if PendingWithdrawal(db, payer).Sign() != 0 {
		t.Fatal("inactive withdrawal reserved funds")
	}
	db.SetState(payer, types.Hash{31: 3}, types.Hash{31: 1})
	if PendingWithdrawal(db, payer).Int64() != 9 {
		t.Fatal("mature but unexecuted withdrawal released funds")
	}
	data, err := RequestWithdrawalData(owner, big.NewInt(9))
	if err != nil || len(data) != 68 || data[35] != 1 || data[67] != 9 {
		t.Fatal("withdrawal ABI encoding")
	}
}
