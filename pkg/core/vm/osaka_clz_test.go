package vm

import (
	"encoding/binary"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"testing"
)

func TestCanonicalOsakaAndAmsterdamCLZIndependentExecution(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		rules := ForkRules{IsOsaka: true, IsPrague: true, IsAmsterdam: amsterdam}
		table := SelectJumpTable(rules)
		if table[CLZ] == nil {
			t.Fatal("CLZ not activated")
		}
		for _, bit := range []int{-1, 0, 63, 64, 127, 128, 254, 255} {
			word := new(big.Int)
			if bit >= 0 {
				word.SetBit(word, bit, 1)
			}
			code := append([]byte{byte(PUSH32)}, word.FillBytes(make([]byte, 32))...)
			code = append(code, byte(CLZ), byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN))
			db := state.NewMemoryStateDB()
			address := types.Address{19: 0x42}
			db.SetCode(address, code)
			db.SetNonce(address, 1)
			evm := NewEVMWithState(BlockContext{BlockNumber: big.NewInt(1), GasLimit: 1000000}, TxContext{}, Config{}, db)
			evm.SetForkRules(rules)
			evm.SetJumpTable(table)
			got, left, err := evm.Call(types.Address{}, address, nil, 100000, new(big.Int))
			if err != nil {
				t.Fatal(err)
			}
			cfg := *params.AllDevChainProtocolChanges
			cfg.AmsterdamTime = nil
 cfg.BogotaTime = nil
			zero := uint64(0)
			cfg.OsakaTime = &zero
			want, _, err := runtime.Execute(code, nil, &runtime.Config{ChainConfig: &cfg, GasLimit: 100000, Origin: gethcommon.Address{}})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 32 || len(want) != 32 || binary.BigEndian.Uint64(got[24:]) != uint64(255-bit) || string(got) != string(want) {
				t.Fatalf("bit %d native=%x reference=%x", bit, got, want)
			}
			if left != 99979 {
				t.Fatalf("CLZ execution gas %d want 21", 100000-left)
			}
		}
	}
	if SelectJumpTable(ForkRules{IsPrague: true})[CLZ] != nil {
		t.Fatal("CLZ activated before Osaka")
	}
}
