package execution

import (
	"github.com/eth2030/eth2030/core/config"
	"math/big"
	"testing"
)

func TestDynamicFeePriceAtZeroBaseFee(t *testing.T) {
	message := &config.Message{GasPrice: big.NewInt(10), GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(3)}
	for _, test := range []struct {
		base *big.Int
		want int64
	}{{nil, 10}, {big.NewInt(0), 3}, {big.NewInt(8), 10}, {big.NewInt(2), 5}} {
		if got := msgEffectiveGasPrice(message, test.base); got.Cmp(big.NewInt(test.want)) != 0 {
			t.Fatalf("base %v effective price %s want %d", test.base, got, test.want)
		}
	}
}
