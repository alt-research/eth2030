package txpool

import (
	config "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

// Canonical and frozen legacy envelopes need independent activation controls.
func TestCanonicalFrameAdmissionTracksForkTimeAndReorg(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	cfg := *config.TestConfig
	activation := uint64(100)
	cfg.HogotaTime = &activation
	pool.SetForkRules(&cfg, 99)
	tx := signPoolFrame(t, frame, 1)
	root := db.GetRoot()
	for _, timestamp := range []uint64{99, 100, 101, 99, 100} {
		// Native head updates set fork rules before resetting state/fees.
		pool.SetForkRules(&cfg, timestamp)
		pool.Reset(db, &types.Header{Number: big.NewInt(10), Time: timestamp, GasLimit: 30000000, BaseFee: big.NewInt(1)})
		if timestamp < activation {
			if pool.Get(tx.Hash()) != nil {
				t.Fatal("reorg retained canonical Frame before activation")
			}
			if err := pool.AddRemote(tx); err == nil {
				t.Fatal("canonical Frame admitted before activation")
			}
		} else if pool.Get(tx.Hash()) == nil {
			if err := pool.AddRemote(tx); err != nil {
				t.Fatalf("canonical Frame rejected at timestamp %d: %v", timestamp, err)
			}
		}
		if db.GetRoot() != root || db.GetNonce(frame.Sender) != 0 {
			t.Fatal("canonical admission/revalidation changed live state")
		}
	}
}

// A timestamp does not activate post-merge VM rules before the London block.
// Public-prefix simulation must use the same head height as block execution.
func TestCanonicalFramePrefixUsesHeadBlockPrerequisites(t *testing.T) {
	pool, db, frame := framePoolFixture(t)
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.HogotaTime, cfg.LondonBlock = &zero, big.NewInt(10)
	pool.SetForkRules(&cfg, 100)
	db.SetCode(frame.Sender, []byte{0x60, 3, 0x5f, 0x5f, 0xaa})
	frame.Frames[0].Limits.Execution = 30000
	tx := signPoolFrame(t, frame, 1)
	root := db.GetRoot()
	for _, head := range []uint64{9, 10, 11, 9, 10} {
		pool.Reset(db, &types.Header{Number: new(big.Int).SetUint64(head), Time: 100, GasLimit: 30000000, BaseFee: big.NewInt(1)})
		if head < 10 {
			if pool.Get(tx.Hash()) != nil {
				t.Fatal("reorg retained a Frame requiring a later VM fork")
			}
			if err := pool.AddRemote(tx); err == nil {
				t.Fatal("pre-London prefix accepted PUSH0")
			}
		} else if pool.Get(tx.Hash()) == nil {
			if err := pool.AddRemote(tx); err != nil {
				t.Fatalf("Frame rejected at active head %d: %v", head, err)
			}
		}
		if db.GetRoot() != root || db.GetNonce(frame.Sender) != 0 {
			t.Fatal("prefix validation mutated live state")
		}
	}
}

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
