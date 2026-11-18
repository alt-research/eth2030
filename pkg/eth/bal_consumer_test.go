package eth

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/p2p"
)

type restoringChain struct {
	*mockBlockchain
	restored map[types.Hash][]byte
}

func (c *restoringChain) StoreBlockAccessListRLP(hash types.Hash, data []byte) error {
	c.restored[hash] = append([]byte(nil), data...)
	return nil
}

func TestCorrelatedBALConsumerRejectsTamperAndReplay(t *testing.T) {
	for _, tc := range []string{"valid", "tampered", "reordered", "unavailable", "unsolicited"} {
		t.Run(tc, func(t *testing.T) {
			chain := &restoringChain{newMockBlockchain(), make(map[types.Hash][]byte)}
			a, b := []byte{0xc1, 1}, []byte{0xc1, 2}
			ha, hb := crypto.Keccak256Hash(a), crypto.Keccak256Hash(b)
			first := types.NewBlock(&types.Header{Number: big.NewInt(1), BlockAccessListHash: &ha}, &types.Body{})
			second := types.NewBlock(&types.Header{Number: big.NewInt(2), BlockAccessListHash: &hb}, &types.Body{})
			chain.addBlock(first)
			chain.addBlock(second)
			h := NewHandler(chain, newMockTxPool(), 1)
			left, right := p2p.MsgPipe()
			defer left.Close()
			defer right.Close()
			ep := NewEthPeer(p2p.NewPeer("bal-client", "127.0.0.1:1", nil), left)
			ep.Peer().SetVersion(ETH71)
			id, err := ep.RequestBlockAccessLists([]types.Hash{first.Hash(), second.Hash()})
			if err != nil {
				t.Fatal(err)
			}
			data := [][]byte{a, b}
			switch tc {
			case "tampered":
				data = [][]byte{a, {0xc1, 3}}
			case "reordered":
				data = [][]byte{b, a}
			case "unavailable":
				data = [][]byte{{0xc0}, {0xc0}}
			case "unsolicited":
				id++
			}
			msg, err := p2p.EncodeMessage(p2p.BlockAccessListsMsg, &p2p.BlockAccessListsPacket{RequestID: id, AccessLists: data})
			if err != nil {
				t.Fatal(err)
			}
			wire := p2p.Msg{Code: msg.Code, Size: msg.Size, Payload: msg.Payload}
			err = h.HandleMsg(ep, wire)
			switch tc {
			case "valid":
				if err != nil || len(chain.restored) != 2 || !bytes.Equal(chain.restored[first.Hash()], a) {
					t.Fatalf("verified restore failed: %v", err)
				}
				if err := h.HandleMsg(ep, wire); err == nil {
					t.Fatal("replayed response accepted")
				}
			case "unavailable":
				if err != nil || len(chain.restored) != 0 {
					t.Fatalf("unavailable data treated as verified: %v", err)
				}
			case "unsolicited":
				if err == nil || len(chain.restored) != 0 {
					t.Fatal("unsolicited response accepted")
				}
			default:
				if !errors.Is(err, ErrBALHashMismatch) || len(chain.restored) != 0 {
					t.Fatalf("tampered/reordered response affected storage: %v", err)
				}
			}
		})
	}
}

func TestEmptyBALCommitmentAndRangeValidation(t *testing.T) {
	commit := crypto.Keccak256Hash([]byte{0xc0})
	if err := ValidateBlockAccessListForHeader([]byte{0xc0}, &types.Header{BlockAccessListHash: &commit}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
	a, b := p2p.MsgPipe()
	defer a.Close()
	defer b.Close()
	ep := NewEthPeer(p2p.NewPeer("range", "127.0.0.1:1", nil), a)
	ep.Peer().SetVersion(ETH69)
	msg, _ := p2p.EncodeMessage(blockRangeUpdateMsg, blockRange{10, 9, types.Hash{1}})
	if err := h.HandleMsg(ep, p2p.Msg{Code: msg.Code, Size: msg.Size, Payload: msg.Payload}); err == nil {
		t.Fatal("invalid range accepted")
	}
	ep.queueRange(blockRange{0, 32, types.Hash{1}})
	select {
	case got := <-ep.rangeUpdates:
		if got.Latest != 32 {
			t.Fatal(got)
		}
	default:
		t.Fatal("canonical range notification was not queued")
	}
}
