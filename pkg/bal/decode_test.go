package bal

import (
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
)

func TestCanonicalBALDecodeRetainsDeletionsAndUint32Indices(t *testing.T) {
	list := NewBlockAccessList()
	list.AddEntry(AccessEntry{Address: types.Address{19: 1}, AccessIndex: math.MaxUint32, StorageChanges: []StorageChange{{Slot: types.Hash{31: 2}, OldValue: types.Hash{31: 3}, NewValue: types.Hash{}}}, BalanceChange: &BalanceChange{NewValue: big.NewInt(4)}})
	encoded, err := list.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	hash := crypto.Keccak256Hash(encoded)
	decoded, err := DecodeForHeader(encoded, &types.Header{BlockAccessListHash: &hash, GasLimit: 1_000_000})
	if err != nil || len(decoded) != 1 || decoded[0].StorageChanges[0].Changes[0].Value.Sign() != 0 || decoded[0].StorageChanges[0].Changes[0].Index != math.MaxUint32 {
		t.Fatalf("deletion/index lost: %+v %v", decoded, err)
	}
	if _, err := DecodeForHeader(encoded, &types.Header{BlockAccessListHash: &hash, GasLimit: 2000}); err == nil {
		t.Fatal("BAL item budget ignored")
	}
}

func TestCanonicalBALRejectsCommittedMalformedStructure(t *testing.T) {
	for _, accounts := range [][]AccountChanges{
		{{Address: types.Address{19: 2}}, {Address: types.Address{19: 1}}},
		{{StorageChanges: []SlotChanges{{Slot: big.NewInt(1)}}}},
		{{StorageChanges: []SlotChanges{{Slot: big.NewInt(1), Changes: []IndexedChange[*big.Int]{{Index: 1, Value: big.NewInt(1)}}}}, StorageReads: []*big.Int{big.NewInt(1)}}},
		{{BalanceChanges: []IndexedChange[*big.Int]{{Index: math.MaxUint32 + 1, Value: big.NewInt(1)}}}},
		{{BalanceChanges: []IndexedChange[*big.Int]{{Index: 1, Value: new(big.Int).Lsh(big.NewInt(1), 256)}}}},
	} {
		encoded, err := rlp.EncodeToBytes(accounts)
		if err != nil {
			t.Fatal(err)
		}
		hash := crypto.Keccak256Hash(encoded)
		if _, err := DecodeForHeader(encoded, &types.Header{BlockAccessListHash: &hash, GasLimit: 30_000_000}); err == nil {
			t.Fatal("hash-matching malformed BAL accepted")
		}
	}
}
