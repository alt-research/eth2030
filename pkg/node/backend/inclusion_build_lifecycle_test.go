package backend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/chain"
	coreconfig "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/txpool"
	"math/big"
)

func TestGroupEEngineBuildReplacementOwnershipDeadlineAndClose(t *testing.T) {
	for _, mode := range []string{"replacement", "deadline", "close"} {
		t.Run(mode, func(t *testing.T) {
			n := newTestBackendNode(t)
			db := state.NewMemoryStateDB()
			eips.ApplyBuilderRequestGenesis(db)
			genesis := types.NewBlock(&types.Header{Number: new(big.Int), GasLimit: 30000000, Root: db.GetRoot()}, &types.Body{})
			bc, err := chain.NewBlockchain(coreconfig.TestConfig, genesis, db, rawdb.NewMemoryDB())
			if err != nil {
				t.Fatal(err)
			}
			n.blockchain = bc
			n.txPool = txpool.New(txpool.DefaultConfig(), n.blockchain.State())
			b := NewEngineBackend(n)
			t.Cleanup(b.Close)
			b.buildTimeout = time.Second
			if mode == "deadline" {
				b.buildTimeout = 20 * time.Millisecond
			}
			b.buildMu.Lock()
			attrs := &payload.PayloadAttributesV3{PayloadAttributesV2: payload.PayloadAttributesV2{PayloadAttributesV1: payload.PayloadAttributesV1{Timestamp: 12}, Withdrawals: []*payload.Withdrawal{{Index: 1, Amount: 2}}}}
			list := [][]byte{{0xff}}
			slot := uint64(1)
			start := func() *pendingPayload {
				r, err := b.forkchoiceUpdated(payload.ForkchoiceStateV1{HeadBlockHash: n.blockchain.Genesis().Hash()}, attrs, list, &slot, 30000000)
				if err != nil || r.PayloadID == nil {
					b.buildMu.Unlock()
					t.Fatal(r, err)
				}
				b.mu.Lock()
				pp := b.payloads[*r.PayloadID]
				b.mu.Unlock()
				return pp
			}
			first := start()
			// Mutations after FCU must neither alter the registered ID/view nor
			// make a replacement share memory with the original request.
			list[0][0] = 0xfe
			attrs.Timestamp = 24
			attrs.Withdrawals[0].Amount = 3
			slot = 2
			var second *pendingPayload
			if mode == "replacement" {
				second = start()
			}
			closed := make(chan struct{})
			if mode == "close" {
				go func() { b.Close(); close(closed) }()
				<-b.buildContext.Done()
			}
			if mode == "deadline" {
				time.Sleep(35 * time.Millisecond)
			}
			b.buildMu.Unlock()
			select {
			case <-first.done:
			case <-time.After(2 * time.Second):
				t.Fatal("cancelled worker deadlocked")
			}
			want := context.Canceled
			if mode == "deadline" {
				want = context.DeadlineExceeded
			}
			if !errors.Is(first.err, want) || first.block != nil {
				t.Fatal("aborted build published result", first.err)
			}
			if second != nil {
				select {
				case <-second.done:
				case <-time.After(2 * time.Second):
					t.Fatal("replacement deadlocked")
				}
				if second.err != nil || second.block == nil || second.block.Time() != 24 || second.slot != 2 {
					t.Fatal("replacement view lost", second.err)
				}
			}
			if mode == "close" {
				select {
				case <-closed:
				case <-time.After(time.Second):
					t.Fatal("close deadlocked")
				}
				if b.beginWork() == nil {
					t.Fatal("closed backend accepted work")
				}
			}
		})
	}
}
