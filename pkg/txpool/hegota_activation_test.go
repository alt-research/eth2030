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
	cfg.ChainConfig = &chain
	cfg.HeadTime = 99
	pool := New(cfg, s)
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(100), MaxPriorityFeePerGas: big.NewInt(1)})
	if err := pool.AddLocal(tx); err == nil {
		t.Fatal("admitted before fork")
	}
	pool.ResetAtHead(s, &chain, 100, 0)
	if err := pool.AddLocal(tx); err != nil {
		t.Fatal(err)
	}
	pool.ResetAtHead(s, &chain, 99, 0)
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
	cfg.ChainConfig = &chain
	cfg.HeadTime = 100
	cfg.HeadNumber = 9
	pool := New(cfg, s)
	tx := types.NewTransaction(&types.FrameTx{ChainID: big.NewInt(1), Nonce: big.NewInt(0), Sender: sender, Frames: []types.Frame{{Mode: types.ModeVerify, GasLimit: 50000}}, MaxFeePerGas: big.NewInt(100), MaxPriorityFeePerGas: big.NewInt(1)})
	if err := pool.AddLocal(tx); err == nil {
		t.Fatal("admitted before London")
	}
	pool.ResetAtHead(s, &chain, 100, 10)
	if err := pool.AddLocal(tx); err != nil {
		t.Fatal(err)
	}
	pool.ResetAtHead(s, &chain, 100, 9)
	if pool.Get(tx.Hash()) != nil {
		t.Fatal("retained transaction after prerequisite deactivation")
	}
}
