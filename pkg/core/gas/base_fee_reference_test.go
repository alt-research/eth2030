package gas

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethfee "github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestSmallBaseFeesMatchIndependentEIP1559Reference(t *testing.T) {
	config := &params.ChainConfig{LondonBlock: big.NewInt(0)}
	for fee := int64(0); fee <= 12; fee++ {
		for _, used := range []uint64{0, 15000000, 30000000} {
			header := &types.Header{Number: big.NewInt(2), GasLimit: 30000000, GasUsed: used, BaseFee: big.NewInt(fee)}
			expected := gethfee.CalcBaseFee(config, &gethtypes.Header{Number: big.NewInt(2), GasLimit: header.GasLimit, GasUsed: used, BaseFee: big.NewInt(fee)})
			if actual := CalcBaseFee(header); actual.Cmp(expected) != 0 {
				t.Fatalf("parent fee=%d used=%d native=%s independent=%s", fee, used, actual, expected)
			}
		}
	}
}
