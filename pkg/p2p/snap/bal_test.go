package snap

import (
	"bytes"
	"context"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/p2p/wire"
	"github.com/eth2030/eth2030/rlp"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

type snapBALBackend struct {
	StateBackend
	lists map[types.Hash][]byte
}

func (b snapBALBackend) BlockAccessList(hash types.Hash) ([]byte, error) { return b.lists[hash], nil }

func TestSnap2RawBALAndPositionalUnavailability(t *testing.T) {
	packet := BlockAccessListsPacket{ID: 1, Lists: [][]byte{nil, {0xc0}}}
	encoded, err := packet.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, []byte{0xc4, 0x01, 0xc2, 0x80, 0xc0}) {
		t.Fatalf("wrong missing/list wire: %x", encoded)
	}
	var independent struct {
		ID    uint64
		Lists []gethrlp.RawValue
	}
	if err := gethrlp.DecodeBytes(encoded, &independent); err != nil {
		t.Fatal(err)
	}
	var decoded BlockAccessListsPacket
	if err := decoded.DecodeRLP(encoded); err != nil || decoded.Lists[0] != nil || !bytes.Equal(decoded.Lists[1], []byte{0xc0}) {
		t.Fatalf("wire roundtrip: %v", err)
	}
	backend := snapBALBackend{lists: map[types.Hash][]byte{{1}: {0xc0}}}
	h := NewServerHandler(backend)
	served, err := h.HandleGetBlockAccessLists(&GetBlockAccessListsPacket{ID: 2, Hashes: []types.Hash{{9}, {1}, {8}}, Bytes: 2})
	if err != nil || len(served.Lists) != 2 || served.Lists[0] != nil || !bytes.Equal(served.Lists[1], []byte{0xc0}) {
		t.Fatalf("position/budget mismatch: %+v %v", served, err)
	}
}

func TestSnap2ClientValidatesHeaderBeforeReturningBAL(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		left, right := wire.MsgPipe()
		client := &Client{id: "snap2", version: 2, transport: left, pending: make(map[uint64]pendingResponse), done: make(chan struct{})}
		data := []byte{0xc0}
		hash := crypto.Keccak256Hash(data)
		header := &types.Header{BlockAccessListHash: &hash, GasLimit: 1_000_000}
		finished := make(chan error, 1)
		go func() {
			msg, err := right.ReadMsg()
			if err != nil {
				finished <- err
				return
			}
			var req GetBlockAccessListsPacket
			if err := rlp.DecodeBytes(msg.Payload, &req); err != nil {
				finished <- err
				return
			}
			item := data
			if corrupt {
				validOther := bal.NewBlockAccessList()
				validOther.AddEntry(bal.AccessEntry{Address: types.Address{19: 9}})
				item, err = validOther.EncodeRLP()
				if err != nil {
					finished <- err
					return
				}
			}
			finished <- sendSnapMsg(right, BlockAccessListsMsg, &BlockAccessListsPacket{req.ID, [][]byte{item}})
		}()
		go func() {
			msg, err := left.ReadMsg()
			if err == nil {
				_ = client.deliver(msg)
			}
		}()
		result, err := client.RequestBlockAccessLists(context.Background(), []*types.Header{header}, BALSoftResponseLimit)
		if (err != nil) != corrupt || (!corrupt && (len(result) != 1 || !bytes.Equal(result[0], data))) {
			t.Fatalf("commitment validation: %x %v", result, err)
		}
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
		_ = left.Close()
		_ = right.Close()
	}
}

func TestSnap2DoesNotDispatchRemovedTrieRequests(t *testing.T) {
	h := NewServerHandler(snapBALBackend{})
	left, right := wire.MsgPipe()
	defer left.Close()
	defer right.Close()
	if err := h.handleSnapMsgVersion(left, wire.Msg{Code: GetTrieNodesMsg}, 2); err == nil {
		t.Fatal("snap/2 trie healing dispatched")
	}
	protocols := h.Protocols()
	if len(protocols) != 2 || protocols[0].Version != 2 || protocols[0].Length != 10 || protocols[1].Version != 1 {
		t.Fatal("version negotiation descriptors incorrect")
	}
}
