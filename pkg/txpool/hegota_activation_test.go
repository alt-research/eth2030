package txpool

import (
	config "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestFrameAdmissionTracksCanonicalHeadAndReorg(t *testing.T) {
	s := newMockState()
	sender := types.Address{1}
	s.balances[sender] = new(big.Int).Set(richBalance)
	activation := uint64(100)
	chain := *config.TestConfigHogota
	chain.HogotaTime = &activation
	chain.ExperimentalLegacyFrames = true
	cfg := DefaultConfig()
	pool := New(cfg, s)
	resetFramePoolHead(pool, s, &chain, 99, 0)
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(100), MaxPriorityFeePerGas: big.NewInt(1)})
	if err := pool.AddLocal(tx); err == nil {
		t.Fatal("admitted before fork")
	}
	resetFramePoolHead(pool, s, &chain, 100, 0)
	if err := pool.AddLocal(tx); err != nil {
		t.Fatal(err)
	}
	resetFramePoolHead(pool, s, &chain, 99, 0)
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("prototype survived reorg across fork")
	}
	if err := pool.AddLocal(tx); err == nil {
		t.Fatal("readmitted before fork after reorg")
	}
}

func TestFramePoolGateTracksBlockPrerequisites(t *testing.T) {
	s := newMockState()
	sender := types.Address{1}
	s.balances[sender] = new(big.Int).Set(richBalance)
	chain := *config.TestConfigHogota
	chain.ExperimentalLegacyFrames = true
	chain.LondonBlock = big.NewInt(10)
	cfg := DefaultConfig()
	pool := New(cfg, s)
	resetFramePoolHead(pool, s, &chain, 100, 9)
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(100), MaxPriorityFeePerGas: big.NewInt(1)})
	if err := pool.AddLocal(tx); err == nil {
		t.Fatal("admitted before London")
	}
	resetFramePoolHead(pool, s, &chain, 100, 10)
	if err := pool.AddLocal(tx); err != nil {
		t.Fatal(err)
	}
	resetFramePoolHead(pool, s, &chain, 100, 9)
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("retained transaction after prerequisite deactivation")
	}
}

// The canonical 8141 envelope must bypass the frozen prototype's admission and
// eviction gate even when both transaction types share the same type byte.
func TestCanonicalFramesSurviveLegacyGateReset(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	tx := signPoolFrame(t, frame, 1)
	if err := pool.AddLocal(tx); err != nil {
		t.Fatal(err)
	}
	chain := *pool.frameConfig
	for _, legacy := range []bool{false, true, false} {
		chain.ExperimentalLegacyFrames = legacy
		resetFramePoolHead(pool, db, &chain, 100, 1)
		if pool.Get(tx.Hash()) == nil {
			t.Fatalf("canonical Frame evicted with legacy opt-in=%t", legacy)
		}
	}
}

// Exercise the same public head-update path used by native FCU/reorg callers.
func resetFramePoolHead(pool *TxPool, db StateReader, chain *config.ChainConfig, timestamp, number uint64) {
	pool.SetForkRules(chain, timestamp)
	pool.Reset(db, &types.Header{Number: new(big.Int).SetUint64(number), Time: timestamp, GasLimit: 30_000_000, BaseFee: big.NewInt(1)})
}
