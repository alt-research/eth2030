package types

import (
	"bytes"
	"math/big"
	"testing"
)

func TestGroupEExactAppendRLPAllPrefixAndGasBoundaries(t *testing.T) {
	t.Run("blob-and-calldata-header-widths", func(t *testing.T) {
		zero, calldata := uint64(0), uint64(127)
		vec := [3]uint64{127, 0, 127}
		header := &Header{Number: big.NewInt(1), GasLimit: 30000000, GasUsed: 127, BaseFee: big.NewInt(1), BlobGasUsed: &zero, CalldataGasUsed: &calldata, GasUsedVec: &vec}
		tx := NewTransaction(&BlobTx{ChainID: big.NewInt(1), Gas: 100000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(1), BlobFeeCap: big.NewInt(100), Value: new(big.Int), BlobHashes: []Hash{{0: 1}}, Data: []byte{1, 2}})
		sizer, err := NewBlockAppendSizer(NewBlock(header, &Body{}))
		if err != nil {
			t.Fatal(err)
		}
		projected := header.Copy()
		projected.GasUsed = 128
		blobUsed := uint64(131072)
		projected.BlobGasUsed = &blobUsed
		projectedCalldata := calldata + tx.CalldataGas()
		projected.CalldataGasUsed = &projectedCalldata
		projected.GasUsedVec = &[3]uint64{128, 0, projectedCalldata}
		actual, err := NewBlock(projected, &Body{Transactions: []*Transaction{tx}}).EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		size, err := sizer.Size(tx, 128)
		if err != nil || size != len(actual) {
			t.Fatal("all append-sensitive header fields must be projected", size, len(actual), err)
		}
		// A second append must include prior committed blob/calldata usage.
		base := uint64(1<<24) - 2*131072
		header.BlobGasUsed = &base
		sizer, err = NewBlockAppendSizer(NewBlock(header, &Body{}))
		if err != nil {
			t.Fatal(err)
		}
		if err = sizer.CommitAppend(tx); err != nil {
			t.Fatal(err)
		}
		projected = header.Copy()
		projected.GasUsed = 256
		blobUsed = 1 << 24
		projected.BlobGasUsed = &blobUsed
		cd := *header.CalldataGasUsed + 2*tx.CalldataGas()
		projected.CalldataGasUsed = &cd
		projected.GasUsedVec = &[3]uint64{256, 0, 127 + 2*tx.CalldataGas()}
		actual, err = NewBlock(projected, &Body{Transactions: []*Transaction{tx, tx}}).EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		size, err = sizer.Size(tx, 256)
		if err != nil || size != len(actual) {
			t.Fatal("committed gas metadata lost", size, len(actual), err)
		}
	})
	for _, length := range []int{0, 1, 31, 32, 54, 55, 56, 127, 128, 254, 255, 256, 1023, 65535} {
		for _, typed := range []bool{false, true} {
			to := Address{19: 42}
			tx := NewTransaction(&LegacyTx{Gas: 50000, GasPrice: big.NewInt(1), To: &to, Value: new(big.Int), Data: bytes.Repeat([]byte{0xff}, length)})
			if typed {
				tx = NewTransaction(&DynamicFeeTx{ChainID: big.NewInt(1), Gas: 50000, GasFeeCap: big.NewInt(1), GasTipCap: new(big.Int), To: &to, Value: new(big.Int), Data: bytes.Repeat([]byte{0xff}, length)})
			}
			header := &Header{Number: big.NewInt(1), GasLimit: 30000000, BaseFee: big.NewInt(1)}
			body := &Body{Withdrawals: []*Withdrawal{}, DependencyWitnesses: [][]byte{{1, 2, 3}}}
			block := NewBlock(header, body)
			sizer, e := NewBlockAppendSizer(block)
			if e != nil {
				t.Fatal(e)
			}
			for _, used := range []uint64{0, 127, 128, 255, 256, 65535, 65536} {
				size, e := sizer.Size(tx, used)
				if e != nil {
					t.Fatal(e)
				}
				h := header.Copy()
				h.GasUsed = used
				b := *body
				b.Transactions = []*Transaction{tx}
				actual, e := NewBlock(h, &b).EncodeRLP()
				if e != nil || size != len(actual) {
					t.Fatal(length, typed, used, size, len(actual), e)
				}
			}
			if e := sizer.CommitAppend(tx); e != nil {
				t.Fatal(e)
			}
			size, e := sizer.Size(tx, 256)
			if e != nil {
				t.Fatal(e)
			}
			h := header.Copy()
			h.GasUsed = 256
			b := *body
			b.Transactions = []*Transaction{tx, tx}
			actual, e := NewBlock(h, &b).EncodeRLP()
			if e != nil || size != len(actual) {
				t.Fatal("committed transaction list", size, len(actual), e)
			}
		}
	}
}
