package vm

import (
	"fmt"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestEIP7954PublishedCreationBoundaries(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, tc := range []struct {
			name   string
			active bool
			size   int
			want   bool
		}{
			{"old_limit", false, 24576, true}, {"old_limit_plus_one", false, 24577, false},
			{"published_limit", true, 65536, true}, {"published_limit_plus_one", true, 65537, false},
		} {
			t.Run(fmt.Sprintf("%s/create2=%v", tc.name, create2), func(t *testing.T) {
				db := state.NewMemoryStateDB()
				caller := types.HexToAddress("0x1234")
				db.CreateAccount(caller)
				db.AddBalance(caller, big.NewInt(1000000000))
				evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
				evm.SetJumpTable(NewCancunJumpTable())
				evm.SetForkRules(ForkRules{IsEIP7954: tc.active})
				// PUSH3 size; PUSH1 0; RETURN: no large input or unrelated intrinsic check.
				init := []byte{0x62, byte(tc.size >> 16), byte(tc.size >> 8), byte(tc.size), 0x60, 0, 0xf3}
				var addr types.Address
				var err error
				if create2 {
					_, addr, _, err = evm.Create2(caller, init, 30000000, big.NewInt(0), big.NewInt(7))
				} else {
					_, addr, _, err = evm.Create(caller, init, 30000000, big.NewInt(0))
				}
				if tc.want {
					if err != nil {
						t.Fatal(err)
					}
					if got := len(db.GetCode(addr)); got != tc.size {
						t.Fatalf("deployed %d bytes, want %d", got, tc.size)
					}
				} else if err == nil || err.Error() != "max code size exceeded" {
					t.Fatalf("want code-size rejection, got %v", err)
				}
			})
		}
		for _, size := range []int{131072, 131073} {
			t.Run(fmt.Sprintf("init=%d/create2=%v", size, create2), func(t *testing.T) {
				db := state.NewMemoryStateDB()
				caller := types.HexToAddress("0x1234")
				db.CreateAccount(caller)
				evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{}, db)
				evm.SetJumpTable(NewCancunJumpTable())
				evm.SetForkRules(ForkRules{IsEIP7954: true})
				var err error
				if create2 {
					_, _, _, err = evm.Create2(caller, make([]byte, size), 30000000, big.NewInt(0), big.NewInt(7))
				} else {
					_, _, _, err = evm.Create(caller, make([]byte, size), 30000000, big.NewInt(0))
				}
				if size == 131072 && err != nil {
					t.Fatal(err)
				}
				if size == 131073 && err != ErrMaxInitCodeSizeExceeded {
					t.Fatalf("want init-size error, got %v", err)
				}
			})
		}
	}
}
