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

func TestCanonicalFrameProcessorForkTimeBeforeMutation(t *testing.T) {
	for _, timestamp := range []uint64{99, 100, 101} {
		frame, db, cfg, header := frameExecutionFixture(t)
		activation := uint64(100)
		cfg.HogotaTime, header.Time = &activation, timestamp
		tx := signFrameExecution(t, frame, false)
		before := db.GetRoot()
		gp := new(gaspool.GasPool).AddGas(header.GasLimit)
		receipt, used, err := ApplyTransaction(cfg, db, header, tx, gp)
		if timestamp < activation {
			if err == nil || receipt != nil || used != 0 || db.GetRoot() != before ||
				gp.Gas() != header.GasLimit || gp.StateGas() != header.GasLimit {
				t.Fatal("pre-fork canonical Frame changed state, receipt or gas")
			}
		} else if err != nil || receipt == nil || used == 0 || db.GetNonce(frame.Sender) != 1 {
			t.Fatalf("canonical execution at %d: receipt=%v gas=%d err=%v", timestamp, receipt, used, err)
		}
	}
}

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

// This vector deliberately exercises the frozen legacy prototype, whose scope
// numbering differs from current EIP-8141. It is not a current-spec oracle.
func TestProcessorOptedInLegacyFrameExecutesAtFork(t *testing.T) {
	c := *config.TestConfigHogota
	c.ExperimentalLegacyFrames = true
	activation := uint64(100)
	c.HogotaTime = &activation
	sender := types.EntryPointAddress
	for _, timestamp := range []uint64{100, 101} {
		s := state.NewMemoryStateDB()
		s.AddBalance(sender, big.NewInt(1000000000))
		// Legacy combined scope=2; push scope, length, offset, then APPROVE.
		s.SetCode(sender, []byte{0x60, 0x02, 0x60, 0x00, 0x60, 0x00, 0xaa})
		tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(100), MaxPriorityFeePerGas: big.NewInt(1)})
		tx.SetSender(sender)
		h := newTestHeader()
		h.Time = timestamp
		gp := new(gaspool.GasPool).AddGas(h.GasLimit)
		receipt, used, err := ApplyTransaction(&c, s, h, tx, gp)
		if err != nil || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful || used == 0 || s.GetNonce(sender) != 1 {
			t.Fatalf("selected prototype did not execute at %d: receipt=%+v used=%d nonce=%d err=%v", timestamp, receipt, used, s.GetNonce(sender), err)
		}
	}
}
