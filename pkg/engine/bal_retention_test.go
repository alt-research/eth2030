package engine

import (
	"bytes"
	"fmt"
	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestHistoricalBALPublishedRetentionWindow(t *testing.T) {
	const window = uint64(14299 * 32)
	for _, tc := range []struct {
		name                       string
		headNumber, headSlot, slot uint64
		want                       bool
	}{
		{"beyond_old_window", 500000, 500000, 200000, true},
		{"at_minimum_boundary", 500000, 500000, 500000 - window, true},
		{"outside_minimum", 500000, 500000, 500000 - window - 1, false},
		{"missed_slots", 100, 500000, 10, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, raw := makePayloadBodiesAPI(t)
			b := raw.(*EngineBackend)
			t.Cleanup(b.Close)
			list := bal.NewBlockAccessList()
			commit := list.Hash()
			target := types.NewBlock(&types.Header{Number: big.NewInt(1), BlockAccessListHash: &commit, SlotNumber: &tc.slot}, &types.Body{})
			head := types.NewBlock(&types.Header{Number: new(big.Int).SetUint64(tc.headNumber), SlotNumber: &tc.headSlot, ParentHash: target.Hash()}, &types.Body{})
			b.storeBlock(target, list)
			b.storeBlock(head, nil)
			b.setForkchoiceState(head.Hash(), types.Hash{}, types.Hash{})
			got, err := api.GetPayloadBodiesByHashV2([]types.Hash{target.Hash()})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0] == nil {
				t.Fatal("retention must not hide the execution body")
			}
			if tc.want {
				if !bytes.Equal(got[0].BlockAccessList, []byte{0xc0}) {
					t.Fatalf("required BAL unavailable: %x", got[0].BlockAccessList)
				}
			} else if got[0].BlockAccessList != nil {
				t.Fatal("expired BAL unexpectedly served by bounded memory backend")
			}
			page, err := api.GetPayloadBodiesByRangeV2(1, 1)
			if err != nil || len(page) != 1 || page[0] == nil {
				t.Fatalf("range body unavailable: %v", err)
			}
			if !bytes.Equal(page[0].BlockAccessList, got[0].BlockAccessList) {
				t.Fatal("hash/range BAL availability differs")
			}
		})
	}
}

func TestHistoricalEmptyBALCacheMiss(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			api, raw := makePayloadBodiesAPI(t)
			b := raw.(*EngineBackend)
			defer b.Close()
			commit := bal.NewBlockAccessList().Hash()
			if !empty {
				commit[0] ^= 1
			}
			slot := uint64(1)
			target := types.NewBlock(&types.Header{Number: big.NewInt(1), BlockAccessListHash: &commit, SlotNumber: &slot}, &types.Body{})
			b.storeBlock(target, nil)
			b.setForkchoiceState(target.Hash(), types.Hash{}, types.Hash{})
			byHash, err := api.GetPayloadBodiesByHashV2([]types.Hash{target.Hash()})
			if err != nil {
				t.Fatal(err)
			}
			byRange, err := api.GetPayloadBodiesByRangeV2(1, 1)
			if err != nil {
				t.Fatal(err)
			}
			for _, body := range []*ExecutionPayloadBodyV2{byHash[0], byRange[0]} {
				if body == nil {
					t.Fatal("cache miss hid execution body")
				}
				if empty && !bytes.Equal(body.BlockAccessList, []byte{0xc0}) {
					t.Fatal("header-committed empty BAL was unavailable without a cache entry")
				}
				if !empty && body.BlockAccessList != nil {
					t.Fatal("cache miss manufactured an uncommitted empty BAL")
				}
			}
		})
	}
}
