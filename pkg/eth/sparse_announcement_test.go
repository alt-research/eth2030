package eth

import (
	"io"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
	getheth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

type announcementRecorder struct{ messages []p2p.Msg }

func (r *announcementRecorder) ReadMsg() (p2p.Msg, error) { return p2p.Msg{}, io.EOF }
func (r *announcementRecorder) WriteMsg(msg p2p.Msg) error {
	r.messages = append(r.messages, msg)
	return nil
}
func (r *announcementRecorder) Close() error { return nil }

func TestEth72OrdinaryAnnouncementsHaveCanonicalFixedMask(t *testing.T) {
	h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
	tx := makeTestTx(0)
	for _, version := range []uint32{ETH71, ETH72} {
		peer := p2p.NewPeer("ordinary", "127.0.0.1:1", nil)
		peer.SetVersion(version)
		transport := new(announcementRecorder)
		if err := h.sendRelayAnnouncement(NewEthPeer(peer, transport), []*types.Transaction{tx}); err != nil {
			t.Fatal(err)
		}
		if len(transport.messages) != 1 {
			t.Fatalf("sent %d messages", len(transport.messages))
		}
		msg := transport.messages[0]
		if msg.Code != p2p.NewPooledTransactionHashesMsg {
			t.Fatalf("wrong message code %x", msg.Code)
		}
		if version == ETH72 {
			var sdk getheth.NewPooledTransactionHashesPacket72
			if err := gethrlp.DecodeBytes(msg.Payload, &sdk); err != nil {
				t.Fatalf("eth/72 SDK rejected ordinary announcement: %v", err)
			}
			if len(sdk.Hashes) != 1 || types.Hash(sdk.Hashes[0]) != tx.Hash() || sdk.Mask != (types.CellMask{}) {
				t.Fatal("ordinary hash or zero mask changed")
			}
		} else {
			var sdk getheth.NewPooledTransactionHashesPacket71
			if err := gethrlp.DecodeBytes(msg.Payload, &sdk); err != nil {
				t.Fatalf("legacy schema changed: %v", err)
			}
		}
	}
}

func TestEth72OrdinaryAnnouncementsRejectNoncanonicalMasks(t *testing.T) {
	tx := makeTestTx(0)
	for _, width := range []int{0, 1, 15, 16, 17, 32} {
		h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
		h.txPool.(*mockTxPool).txs[tx.Hash()] = tx
		peer := p2p.NewPeer("ordinary", "127.0.0.1:1", nil)
		peer.SetVersion(ETH72)
		mask := make([]byte, width)
		if width > 0 {
			mask[0] = 255
		}
		raw, err := gethrlp.EncodeToBytes([]any{[]byte{types.LegacyTxType}, []uint32{100}, []types.Hash{tx.Hash()}, mask})
		if err != nil {
			t.Fatal(err)
		}
		err = h.handleSparseAnnouncement(NewEthPeer(peer, new(announcementRecorder)), p2p.Msg{Code: p2p.NewPooledTransactionHashesMsg, Payload: raw, Size: uint32(len(raw))})
		if width == 16 && err != nil {
			t.Fatalf("ordinary mask must be ignored: %v", err)
		}
		if width != 16 && err == nil {
			t.Fatalf("accepted %d-byte ordinary mask", width)
		}
		if len(h.relay.records) != 0 || len(h.relay.requests) != 0 {
			t.Fatal("ordinary mask mutated blob sampling state")
		}
	}
}
