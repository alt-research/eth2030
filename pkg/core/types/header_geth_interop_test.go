package types

import (
	"bytes"
	"fmt"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"testing"
)

func TestAmsterdamHeaderGethInterop(t *testing.T) {
	if EIP7706HashFields {
		t.Fatal("canonical header comparison requires standard hash layout")
	}
	for _, slot := range []uint64{0, 42, ^uint64(0)} {
		t.Run(fmt.Sprint(slot), func(t *testing.T) {
			blobUsed, excess := uint64(131072), uint64(17)
			withdrawals, beacon, requests, bal := HexToHash("0x1111"), HexToHash("0x2222"), HexToHash("0x3333"), HexToHash("0x4444")
			native := &Header{ParentHash: HexToHash("0x1234"), UncleHash: EmptyUncleHash, Coinbase: HexToAddress("0x5678"), Root: HexToHash("0x9012"), TxHash: EmptyRootHash, ReceiptHash: EmptyRootHash, Difficulty: new(big.Int), Number: big.NewInt(5), GasLimit: 60000000, GasUsed: 345678, Time: 1000, Extra: []byte("amsterdam"), BaseFee: big.NewInt(7), WithdrawalsHash: &withdrawals, BlobGasUsed: &blobUsed, ExcessBlobGas: &excess, ParentBeaconRoot: &beacon, RequestsHash: &requests, BlockAccessListHash: &bal, SlotNumber: &slot}
			encoded, err := native.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			var independent gethtypes.Header
			if err = gethrlp.DecodeBytes(encoded, &independent); err != nil {
				t.Fatal(err)
			}
			if independent.BlockAccessListHash == nil || Hash(*independent.BlockAccessListHash) != bal || independent.SlotNumber == nil || *independent.SlotNumber != slot || Hash(independent.Hash()) != native.Hash() {
				t.Fatal("independent header lost BAL/slot or computed another block hash")
			}
			reencoded, err := gethrlp.EncodeToBytes(&independent)
			if err != nil || !bytes.Equal(reencoded, encoded) {
				t.Fatalf("independent canonical encoding differs: %v", err)
			}
			decoded, err := DecodeHeaderRLP(reencoded)
			if err != nil || decoded.Hash() != native.Hash() {
				t.Fatalf("independent header could not be read back: %v", err)
			}
			changed := slot ^ 1
			independent.SlotNumber = &changed
			if Hash(independent.Hash()) == native.Hash() {
				t.Fatal("slot tamper did not alter independently computed hash")
			}
			independent.SlotNumber = &slot
			(*independent.BlockAccessListHash)[0] ^= 1
			if Hash(independent.Hash()) == native.Hash() {
				t.Fatal("BAL tamper did not alter independently computed hash")
			}
		})
	}
}
