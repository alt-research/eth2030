package types

import (
	"bytes"
	"math/big"
	"testing"
)

func TestGroupEExactAppendRLPAllPrefixAndGasBoundaries(t *testing.T) {
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
				h := *header
				h.GasUsed = used
				b := *body
				b.Transactions = []*Transaction{tx}
				actual, e := NewBlock(&h, &b).EncodeRLP()
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
			h := *header
			h.GasUsed = 256
			b := *body
			b.Transactions = []*Transaction{tx, tx}
			actual, e := NewBlock(&h, &b).EncodeRLP()
			if e != nil || size != len(actual) {
				t.Fatal("committed transaction list", size, len(actual), e)
			}
		}
	}
}
