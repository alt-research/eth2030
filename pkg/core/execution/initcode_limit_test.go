package execution

import (
	"errors"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math/big"
	"testing"
)

func TestCreationTransactionInitcodeLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cfg   *config.ChainConfig
		size  int
		valid bool
	}{
		{"pre_limit", config.TestConfig, 49152, true}, {"pre_over", config.TestConfig, 49153, false},
		{"amsterdam_limit", config.TestConfigGlamsterdan, 131072, true}, {"amsterdam_over", config.TestConfigGlamsterdan, 131073, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := state.NewMemoryStateDB()
			from := types.HexToAddress("0x1234")
			balance := big.NewInt(1000000000)
			db.AddBalance(from, balance)
			h := newTestHeader()
			h.GasLimit = 30000000
			gp := new(gaspool.GasPool).AddGas(h.GasLimit)
			msg := &config.Message{From: from, GasLimit: 10000000, GasPrice: big.NewInt(1), Value: new(big.Int), Data: make([]byte, tc.size)}
			result, err := ApplyMessage(tc.cfg, nil, db, h, msg, gp)
			if tc.valid {
				if err != nil {
					t.Fatal(err)
				}
				if result.Err != nil {
					t.Fatal(result.Err)
				}
			} else {
				if !errors.Is(err, vm.ErrMaxInitCodeSizeExceeded) {
					t.Fatalf("want invalid transaction, got result=%+v error=%v", result, err)
				}
				if db.GetNonce(from) != 0 || db.GetBalance(from).Cmp(balance) != 0 || gp.Gas() != h.GasLimit {
					t.Fatal("invalid transaction mutated nonce, balance or gas pool")
				}
			}
		})
	}
}
