package eth

import (
	"math/big"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/txpool"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func dependencyHandler8504(t *testing.T) (*Handler, *txpool.TxPool) {
	t.Helper()
	db := state.NewMemoryStateDB()
	tx := proof8504.Transaction(0)
	db.AddBalance(tx.FrameSender(), big.NewInt(1_000_000_000))
	db.SetCode(tx.FrameSender(), proof8504.ApproveCode)
	zero := uint64(0)
	cfg := *config.TestConfig
	cfg.ChainID = big.NewInt(1337)
	cfg.HogotaTime = &zero
	cfg.EIP8504Time = &zero
	pool := txpool.New(txpool.DefaultConfig(), db)
	pool.SetForkRules(&cfg, 100)
	pool.SetBaseFee(big.NewInt(1))
	h := NewHandler(newMockBlockchain(), pool, 1337)
	h.EnableDependencyProtocol8504(true)
	return h, pool
}

func TestDependency8504NativePeerRequestAdmissionAndIsolation(t *testing.T) {
	sender, source := dependencyHandler8504(t)
	receiver, destination := dependencyHandler8504(t)
	tx := proof8504.Transaction(0)
	if err := source.AddRemote(tx); err != nil {
		t.Fatal(err)
	}
	left, right := p2p.MsgPipe()
	defer left.Close()
	defer right.Close()
	done := make(chan error, 2)
	go func() { done <- sender.runDependencyPeer8504("receiver", left) }()
	go func() { done <- receiver.runDependencyPeer8504("sender", right) }()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
wait:
	for {
		select {
		case err := <-done:
			t.Fatal("wrapper exchange", err)
		case <-deadline.C:
			t.Fatal("wrapper never admitted")
		case <-ticker.C:
			if destination.Get(tx.Hash()) != nil {
				break wait
			}
		}
	}
	hash, _ := types.DependencyWrapperHash8504(tx)
	if len(destination.DependencyWrapper8504(hash)) == 0 {
		t.Fatal("received wrapper not available for retrieval")
	}
	// The negotiated namespace is separate from every official eth capability.
	protocols := sender.Protocols()
	if protocols[len(protocols)-1].Name != "hgtp" || protocols[len(protocols)-1].Length != 4 {
		t.Fatal("capability collision")
	}
	disabled := NewHandler(newMockBlockchain(), source, 1337)
	for _, p := range disabled.Protocols() {
		if p.Name == "hgtp" {
			t.Fatal("unconfigured node advertises experiment")
		}
	}
	if _, err := encodeTransactions([]*types.Transaction{tx}); err == nil {
		t.Fatal("dependency leaked through Transactions")
	}
	if _, err := (p2p.PooledTransactionsPacket{Transactions: []*types.Transaction{tx}}).EncodeRLP(); err == nil {
		t.Fatal("dependency leaked through PooledTransactions")
	}
	_ = left.Close()
	_ = right.Close()
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("peer did not stop")
		}
	}
}

func TestDependency8504PeerRejectsWrongProfileAndUnsolicitedResponse(t *testing.T) {
	for _, wrongProfile := range []bool{false, true} {
		h, _ := dependencyHandler8504(t)
		left, right := p2p.MsgPipe()
		done := make(chan error, 1)
		go func() { done <- h.runDependencyPeer8504("remote", left) }()
		status, err := right.ReadMsg()
		if err != nil {
			t.Fatal(err)
		}
		if wrongProfile {
			var p dependencyStatusPacket8504
			_ = gethrlp.DecodeBytes(status.Payload, &p)
			p.Profile[0] ^= 1
			status.Payload, _ = gethrlp.EncodeToBytes(p)
			status.Size = uint32(len(status.Payload))
		}
		if err := right.WriteMsg(status); err != nil {
			t.Fatal(err)
		}
		if !wrongProfile {
			raw, _ := gethrlp.EncodeToBytes(dependencyResponsePacket8504{ID: 99})
			if err := right.WriteMsg(p2p.Msg{Code: dependencyResponse8504, Size: uint32(len(raw)), Payload: raw}); err != nil {
				t.Fatal(err)
			}
		}
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("invalid peer accepted")
			}
		case <-time.After(time.Second):
			t.Fatal("invalid peer not rejected")
		}
		_ = left.Close()
		_ = right.Close()
	}
}
