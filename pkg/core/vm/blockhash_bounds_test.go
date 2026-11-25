package vm

import (
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestBlockhashUint256Bounds(t *testing.T) {
	high := new(big.Int).Lsh(big.NewInt(1), 64)
	cases := []*big.Int{new(big.Int).Set(high), new(big.Int).Add(high, big.NewInt(4)), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), big.NewInt(4)}
	for _, number := range cases {
		t.Run(number.String(), func(t *testing.T) {
			reads := 0
			hash := types.Hash{31: 0x44}
			evm := NewEVM(BlockContext{BlockNumber: big.NewInt(5), GetHash: func(n uint64) types.Hash {
				reads++
				if n != 4 {
					t.Fatalf("unexpected history read %d", n)
				}
				return hash
			}}, TxContext{}, Config{})
			c := NewContract(types.Address{}, types.Address{}, nil, 1000)
			c.Code = append([]byte{byte(PUSH32)}, number.FillBytes(make([]byte, 32))...)
			c.Code = append(c.Code, byte(BLOCKHASH), byte(PUSH1), 0, byte(MSTORE), byte(PUSH1), 32, byte(PUSH1), 0, byte(RETURN))
			result, err := evm.Run(c, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := types.Hash{}
			calls := 0
			if number.Cmp(big.NewInt(4)) == 0 {
				want = hash
				calls = 1
			}
			if len(result) != 32 || types.BytesToHash(result) != want || reads != calls {
				t.Fatalf("hash=%x reads=%d, expected %s reads=%d", result, reads, want, calls)
			}
		})
	}
}
