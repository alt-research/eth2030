package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
)

func TestOsakaModExpIndependentVectors(t *testing.T) {
	oracle := gethvm.PrecompiledContractsOsaka[gethcommon.Address{19: 5}]
	native := SelectPrecompiles(ForkRules{IsOsaka: true, IsPrague: true})[types.Address{19: 5}]
	for _, lengths := range [][3]uint64{{0, 0, 0}, {1, 1, 1}, {32, 32, 32}, {33, 33, 33}, {64, 64, 64}, {1024, 3, 1024}, {1025, 1, 1}, {0, 1025, 0}, {0, 0, 1025}} {
		t.Run(fmt.Sprint(lengths), func(t *testing.T) {
			input := make([]byte, 96)
			for i, n := range lengths {
				binary.BigEndian.PutUint64(input[i*32+24:], n)
			}
			input = append(input, bytes.Repeat([]byte{0x11}, int(lengths[0]+lengths[1]+lengths[2]))...)
			wantGas := oracle.RequiredGas(input)
			if got := native.RequiredGas(input); got != wantGas {
				t.Fatalf("native gas %d want SDK %d", got, wantGas)
			}
			want, e1 := oracle.Run(input)
			got, e2 := native.Run(input)
			if (e1 == nil) != (e2 == nil) || !bytes.Equal(got, want) {
				t.Fatalf("native %x/%v want SDK %x/%v", got, e2, want, e1)
			}
		})
	}
	for _, input := range [][]byte{nil, {1}, bytes.Repeat([]byte{255}, 96)} {
		if got, want := native.RequiredGas(input), oracle.RequiredGas(input); got != want {
			t.Fatalf("oversized/short input gas %d want %d", got, want)
		}
	}
	// Fork selection must not mutate the shared earlier-fork map.
	if got := SelectPrecompiles(ForkRules{IsPrague: true})[types.Address{19: 5}].RequiredGas(nil); got != 200 {
		t.Fatalf("Prague gas changed: %d", got)
	}
}
