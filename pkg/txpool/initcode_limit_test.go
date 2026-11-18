package txpool

import (
	"errors"
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math/big"
	"testing"
)

func TestCreationAdmissionInitcodeBoundaries(t *testing.T) {
	for _, cfg := range []*config.ChainConfig{config.TestConfig, config.TestConfigGlamsterdan} {
		limit := 49152
		if cfg.IsGlamsterdan(100) {
			limit = 131072
		}
		for _, size := range []int{limit, limit + 1} {
			t.Run(fmt.Sprintf("limit=%d/size=%d", limit, size), func(t *testing.T) {
				pool, _ := newRichPool()
				pool.SetForkRules(cfg, 100)
				tx := types.NewTransaction(&types.LegacyTx{GasPrice: big.NewInt(1), Gas: 10000000, Value: new(big.Int), Data: make([]byte, size)})
				tx.SetSender(testSender)
				err := pool.AddLocal(tx)
				if size == limit {
					if err != nil {
						t.Fatal(err)
					}
				} else if !errors.Is(err, vm.ErrMaxInitCodeSizeExceeded) {
					t.Fatalf("want initcode size rejection, got %v", err)
				}
			})
		}
	}
}
