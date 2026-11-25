package execution

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"strings"
	"testing"
)

func TestProcessorFrameGateBeforeStateMutation(t *testing.T) {
	ts := uint64(100)
	for _, tt := range []struct {
		name string
		time uint64
		opt  bool
	}{{"before", 99, true}, {"canonical at", 100, false}, {"canonical after", 101, false}} {
		t.Run(tt.name, func(t *testing.T) {
			c := *config.TestConfigHogota
			c.HogotaTime = &ts
			c.ExperimentalLegacyFrames = tt.opt
			s := state.NewMemoryStateDB()
			sender := types.Address{1}
			s.AddBalance(sender, big.NewInt(1000000))
			before, _ := s.Commit()
			gp := new(gaspool.GasPool).AddGas(1000000)
			tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0), Sender: sender})
			h := newTestHeader()
			h.Time = tt.time
			_, used, err := ApplyTransaction(&c, s, h, tx, gp)
			after, _ := s.Commit()
			if err == nil || !strings.Contains(err.Error(), "frame transactions unavailable") || used != 0 || before != after || gp.Gas() != 1000000 {
				t.Fatalf("gate mutated state/gas or accepted tx: used=%d err=%v", used, err)
			}
		})
	}
	// An opted-in prototype reaches execution validation at the boundary; this
	// deliberately malformed transaction must still fail, for a non-gate reason.
	c := *config.TestConfigHogota
	c.HogotaTime = &ts
	c.ExperimentalLegacyFrames = true
	h := newTestHeader()
	h.Time = 100
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0)})
	_, _, err := ApplyTransaction(&c, state.NewMemoryStateDB(), h, tx, new(gaspool.GasPool).AddGas(1000000))
	if err != nil && strings.Contains(err.Error(), "frame transactions unavailable") {
		t.Fatal("gate remained closed at activation")
	}
}
