package bal

import (
	"bytes"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
)

func TestCodeAdoption8298CanonicalWireAndAvailability(t *testing.T) {
	code := []byte{0x60, 1}
	hash := crypto.Keccak256Hash(code)
	list := &BlockAccessList{Entries: []AccessEntry{{Address: types.Address{19: 1}, AccessIndex: 2, CodeChange: &CodeChange{NewCodeHash: &hash}}}}
	encoded, err := list.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := rlp.EncodeToBytes([]any{[]any{types.Address{19: 1}, []any{}, []any{}, []any{}, []any{}, []any{[]any{uint64(2), []byte{}, hash}}}})
	if !bytes.Equal(encoded, want) {
		t.Fatal("adoption encoded bytecode or wrong tuple")
	}
	accounts, err := DecodeRLP(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if round, err := EncodeAccountChanges(accounts); err != nil || !bytes.Equal(encoded, round) {
		t.Fatal("adoption lost on roundtrip", err)
	}
	absent := func(types.Hash) ([]byte, bool) { return nil, false }
	if _, err := ResolveCodeChanges(accounts, false, absent); err == nil {
		t.Fatal("accepted inactive adoption")
	}
	if _, err := ResolveCodeChanges(accounts, true, absent); err == nil {
		t.Fatal("accepted unavailable code")
	}
	for _, index := range []uint64{1, 2, 3} {
		combined := append(accounts, AccountChanges{Address: types.Address{19: 2}, CodeChanges: []IndexedChange[[]byte]{{index, code}}})
		_, err := ResolveCodeChanges(combined, true, absent)
		if (err == nil) != (index == 1) {
			t.Fatal("same/later index resolved adoption", index, err)
		}
	}
	if _, err := ResolveCodeChanges(accounts, true, func(h types.Hash) ([]byte, bool) { return code, h == hash }); err != nil {
		t.Fatal(err)
	}
	for _, tuple := range [][]any{{2, code, hash}, {2, []byte{}, []byte{1}}, {2, []byte{}, types.EmptyCodeHash}, {2, []byte{}, hash, 1}} {
		bad, _ := rlp.EncodeToBytes([]any{[]any{types.Address{}, []any{}, []any{}, []any{}, []any{}, []any{tuple}}})
		if _, err := DecodeRLP(bad); err == nil {
			t.Fatal("accepted malformed adoption")
		}
	}
}
