package eiptest_test

import (
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/eiptest"
	"math/big"
	"testing"
)

func TestEIP8024_DUPNSWAPNExchange(t *testing.T) {
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkAmsterdam, 0))
	for _, tc := range []struct {
		name          string
		depth         int
		op, immediate byte
		want          uint64
	}{
		{"dupn", 17, 0xe6, 0x80, 1},
		{"swapn", 18, 0xe7, 0x80, 1},
		{"exchange", 3, 0xe8, 0x8e, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var code []byte
			for i := 1; i <= tc.depth; i++ {
				code = append(code, 0x60, byte(i))
			}
			code = append(code, tc.op, tc.immediate)
			if tc.op == 0xe8 {
				code = append(code, 0x50)
			} // expose the exchanged second item.
			code = append(code, 0x60, 0, 0x55, 0)
			address := s.STF().Deploy(s.Alice(), code)
			s.STF().Tx(s.Alice(), address, nil)
			if got := s.StorageAt(address, [32]byte{}); got != eiptest.Uint64ToHash(tc.want) {
				t.Fatalf("stored=%s want=%d", got, tc.want)
			}
		})
	}
}

func TestEIP8024NativeAmsterdamBoundary(t *testing.T) {
	target := types.HexToAddress("0xc0ffee")
	var runtime []byte
	for i := 1; i <= 17; i++ {
		runtime = append(runtime, 0x60, byte(i))
	}
	runtime = append(runtime, 0xe6, 0x80, 0x60, 0, 0x55, 0)
	s := eiptest.New(t, eiptest.WithGenesisTime(100), eiptest.ForkAt(eiptest.ForkAmsterdam, 24), eiptest.FundCode(target, new(big.Int), runtime, 1))
	for i, want := range []uint64{0, 1} {
		tx := eiptest.DynamicFee(s.Config().ChainID, s.Alice(), s.Nonce(s.Alice().Address), &target, new(big.Int), nil, 200000, bigFeeCap, bigTipCap)
		receipt := s.Commit(tx).Tx(0).Receipt
		if receipt.Status != want {
			t.Fatalf("block %d status=%d want=%d", i+1, receipt.Status, want)
		}
	}
	if got := s.StorageAt(target, [32]byte{}); got != eiptest.Uint64ToHash(1) {
		t.Fatalf("Amsterdam DUPN stored=%s", got)
	}
}
