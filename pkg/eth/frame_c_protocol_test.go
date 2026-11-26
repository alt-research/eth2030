package eth

import (
	"fmt"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
	"github.com/eth2030/eth2030/rlp"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"testing"
)

func frameCWireTx() *types.Transaction {
	return types.NewTransaction(&types.FrameTx8141{ChainID: big.NewInt(1337), Sender: types.Address{19: 9}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(2)}})
}
func frameCAnnouncement(t *testing.T, version uint32, kinds []byte, sizes []uint32, hashes []types.Hash) p2p.Msg {
	t.Helper()
	var value any = &NewPooledTxHashesMsg68{Types: kinds, Sizes: sizes, Hashes: hashes}
	if version >= ETH72 {
		value = &NewPooledTxHashesMsg72{Types: kinds, Sizes: sizes, Hashes: hashes}
	}
	raw, err := gethrlp.EncodeToBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	return p2p.Msg{Code: p2p.NewPooledTransactionHashesMsg, Size: uint32(len(raw)), Payload: raw}
}
func TestFrameCAnnouncementFetchMetadataAndBounds(t *testing.T) {
	for _, version := range []uint32{ETH68, ETH69, ETH70, ETH71, ETH72} {
		for _, fault := range []string{"none", "type", "size"} {
			t.Run(fmt.Sprintf("%d_%s", version, fault), func(t *testing.T) {
				h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
				peer := p2p.NewPeer("frame-peer", "127.0.0.1:1", nil)
				peer.SetVersion(version)
				recorder := new(announcementRecorder)
				ep := NewEthPeer(peer, recorder)
				tx := frameCWireTx()
				raw, _ := tx.EncodeRLP()
				kind, size := byte(types.FrameTxType), uint32(len(raw))
				if fault == "type" {
					kind = types.LegacyTxType
				}
				if fault == "size" {
					size++
				}
				ann := frameCAnnouncement(t, version, []byte{kind, kind}, []uint32{size, size}, []types.Hash{tx.Hash(), tx.Hash()})
				if err := h.handleNewPooledTransactionHashes(ep, ann); err != nil {
					t.Fatal(err)
				}
				if len(recorder.messages) != 1 {
					t.Fatalf("duplicate announcement emitted %d requests", len(recorder.messages))
				}
				var req p2p.GetPooledTransactionsPacket
				if err := gethrlp.DecodeBytes(recorder.messages[0].Payload, &req); err != nil {
					t.Fatal(err)
				}
				if len(req.Hashes) != 1 {
					t.Fatal("duplicate hash was not deduplicated")
				}
				var packet any = &p2p.PooledTransactionsPacket{RequestID: req.RequestID, Transactions: []*types.Transaction{tx}}
				if version >= ETH72 {
					packet = &pooledTransactions72{RequestID: req.RequestID, Transactions: []*types.Transaction{tx}}
				}
				encoded, err := packet.(interface{ EncodeRLP() ([]byte, error) }).EncodeRLP()
				if err != nil {
					t.Fatal(err)
				}
				err = h.handlePooledTransactions(ep, p2p.Msg{Code: p2p.PooledTransactionsMsg, Size: uint32(len(encoded)), Payload: encoded})
				if (err == nil) != (fault == "none") {
					t.Fatalf("fault=%s response=%v", fault, err)
				}
				if (h.txPool.Get(tx.Hash()) != nil) != (fault == "none") {
					t.Fatal("metadata mismatch reached pool")
				}
			})
		}
	}
	for _, version := range []uint32{ETH68, ETH72} {
		t.Run(fmt.Sprintf("limits_%d", version), func(t *testing.T) {
			h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
			peer := p2p.NewPeer("bounds", "127.0.0.1:1", nil)
			peer.SetVersion(version)
			recorder := new(announcementRecorder)
			ep := NewEthPeer(peer, recorder)
			hash := types.Hash{31: 1}
			if err := h.handleNewPooledTransactionHashes(ep, frameCAnnouncement(t, version, []byte{255}, []uint32{100}, []types.Hash{hash})); err != nil {
				t.Fatal(err)
			}
			if len(recorder.messages) != 0 {
				t.Fatal("unknown transaction type requested")
			}
			for _, sizes := range [][]uint32{nil, {0}, {MaxMessageSize + 1}} {
				if err := h.handleNewPooledTransactionHashes(ep, frameCAnnouncement(t, version, []byte{types.FrameTxType}, sizes, []types.Hash{hash})); err == nil {
					t.Fatal("invalid size/length accepted")
				}
			}
			kinds := make([]byte, 257)
			sizes := make([]uint32, 257)
			hashes := make([]types.Hash, 257)
			for i := range kinds {
				kinds[i] = types.FrameTxType
				sizes[i] = 100
				hashes[i] = types.BytesToHash(new(big.Int).SetInt64(int64(i + 1)).Bytes())
			}
			if err := h.handleNewPooledTransactionHashes(ep, frameCAnnouncement(t, version, kinds, sizes, hashes)); err != nil {
				t.Fatal(err)
			}
			if len(recorder.messages) != 2 {
				t.Fatal("large announcement not split into bounded requests")
			}
			for _, msg := range recorder.messages {
				var req p2p.GetPooledTransactionsPacket
				if err := gethrlp.DecodeBytes(msg.Payload, &req); err != nil || len(req.Hashes) > 256 {
					t.Fatal("request limit violated")
				}
			}
		})
	}
}
func TestFrameCEvictedAnnouncementSuppression(t *testing.T) {
	h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
	peer := p2p.NewPeer("removed", "127.0.0.1:1", nil)
	peer.SetVersion(ETH72)
	recorder := new(announcementRecorder)
	ep := NewEthPeer(peer, recorder)
	tx := frameCWireTx()
	if err := h.sendRelayAnnouncement(ep, []*types.Transaction{tx}); err != nil {
		t.Fatal(err)
	}
	if len(recorder.messages) != 0 {
		t.Fatal("evicted Frame repropagated from stale caller snapshot")
	}
}

func TestFrameCStructuredReceiptThroughHandler(t *testing.T) {
	h := NewHandler(newMockBlockchain(), newMockTxPool(), 1)
	hash := h.chain.Genesis().Hash()
	receipt := &types.Receipt{Type: types.FrameTxType, Payer: types.Address{19: 7}, CumulativeGasUsed: 12345, FrameResults8141: []types.FrameResult8141{{Status: 1, GasUsed: types.FrameGasLimits{Execution: 100, State: 200}, Logs: []*types.Log{{Address: types.Address{19: 8}, Data: []byte{9}}}}, {Status: types.FrameStatusSkipped}}}
	h.SetReceiptProvider(fixedReceiptProvider{hash, []*types.Receipt{receipt}})
	recorder := new(announcementRecorder)
	peer := p2p.NewPeer("receipt-frame", "127.0.0.1:1", nil)
	peer.SetVersion(ETH70)
	ep := NewEthPeer(peer, recorder)
	req := &p2p.GetReceiptsPacketV70{RequestID: 7, Hashes: []types.Hash{hash}}
	raw, err := rlp.EncodeToBytes(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.handleGetReceiptsV70(ep, p2p.Msg{Code: p2p.GetReceiptsMsg, Payload: raw, Size: uint32(len(raw))}); err != nil {
		t.Fatal(err)
	}
	if len(recorder.messages) != 1 {
		t.Fatal("receipt handler did not answer")
	}
	var packet struct {
		RequestID  uint64
		Incomplete bool
		Pages      [][]gethrlp.RawValue
	}
	if err := gethrlp.DecodeBytes(recorder.messages[0].Payload, &packet); err != nil {
		t.Fatal(err)
	}
	var fields []gethrlp.RawValue
	if err := gethrlp.DecodeBytes(packet.Pages[0][0], &fields); err != nil || len(fields) != 4 {
		t.Fatalf("receipt not a structured four-field tuple: %v", err)
	}
	var kind uint64
	if err := gethrlp.DecodeBytes(fields[0], &kind); err != nil || kind != 6 {
		t.Fatal("structured type lost")
	}
	var payer types.Address
	if err := gethrlp.DecodeBytes(fields[2], &payer); err != nil || payer != receipt.Payer {
		t.Fatal("structured payer lost")
	}
	var decoded p2p.ReceiptsPacketV70
	if err := decoded.DecodeRLP(recorder.messages[0].Payload); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Receipts[0][0].FrameResults8141) != 2 || decoded.Receipts[0][0].FrameResults8141[0].GasUsed.State != 200 || decoded.Receipts[0][0].FrameResults8141[1].Status != 2 {
		t.Fatal("handler receipt lost Frame dimensions/status")
	}
}
