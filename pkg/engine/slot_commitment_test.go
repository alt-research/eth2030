package engine

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"testing"
)

func TestMemoryEngineSlotCommitment(t *testing.T) {
	for _, tc := range []struct {
		slot uint64
		wire string
	}{
		{0, "0x0"}, {42, "0x2a"}, {42, "0x2A"}, {^uint64(0), "0xffffffffffffffff"}, {^uint64(0), "0xFfFfFfFfFfFfFfFf"},
	} {
		t.Run(tc.wire, func(t *testing.T) {
			slot := tc.slot
			if got, err := hexutil.DecodeUint64(tc.wire); err != nil || got != slot {
				t.Fatalf("independent QUANTITY decoder: got %d err=%v", got, err)
			}
			cfg := *config.TestConfig
			cfg.GlamsterdanTime = nil
			genesis := makeGenesis()
			db := newBuilderRequestTestState()
			builder := NewEngineBackend(&cfg, db, genesis)
			defer builder.Close()
			receiver := NewEngineBackend(&cfg, db.Dup(), genesis)
			defer receiver.Close()
			attrs := &PayloadAttributesV4{PayloadAttributesV3: PayloadAttributesV3{PayloadAttributesV2: PayloadAttributesV2{PayloadAttributesV1: PayloadAttributesV1{Timestamp: genesis.Time() + 12}, Withdrawals: []*Withdrawal{}}, ParentBeaconBlockRoot: types.Hash{1}}, SlotNumber: slot}
			result, err := builder.ForkchoiceUpdatedV4(ForkchoiceStateV1{HeadBlockHash: genesis.Hash()}, attrs)
			if err != nil || result.PayloadID == nil {
				t.Fatalf("build: %v result=%+v", err, result)
			}
			built, err := builder.GetPayloadV6ByID(*result.PayloadID)
			if err != nil {
				t.Fatal(err)
			}
			pending := builder.payloads[*result.PayloadID].block
			raw, err := pending.Header().EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			var independent gethtypes.Header
			if err = gethrlp.DecodeBytes(raw, &independent); err != nil {
				t.Fatal(err)
			}
			if independent.SlotNumber == nil || *independent.SlotNumber != slot || types.Hash(independent.Hash()) != built.ExecutionPayload.BlockHash {
				t.Fatal("memory builder did not commit the slot it reports")
			}
			for _, malformed := range []string{"0x", "0x00", "0x02a", "0X2A", "2a", "0x10000000000000000", "0x-1"} {
				invalid := *built.ExecutionPayload
				invalid.SlotNumber = malformed
				status, err := receiver.ProcessBlockV5(&invalid, nil, attrs.ParentBeaconBlockRoot, built.ExecutionRequests)
				if err != ErrInvalidParams || status.Status == StatusValid {
					t.Fatalf("malformed slot %q: err=%v status=%+v", malformed, err, status)
				}
			}
			built.ExecutionPayload.SlotNumber = tc.wire
			status, err := receiver.ProcessBlockV5(built.ExecutionPayload, nil, attrs.ParentBeaconBlockRoot, built.ExecutionRequests)
			if err != nil || status.Status != StatusValid {
				t.Fatalf("memory receiver lost slot: err=%v status=%+v", err, status)
			}
			accepted, ok := receiver.getBlock(built.ExecutionPayload.BlockHash)
			if !ok || accepted.Header().SlotNumber == nil || *accepted.Header().SlotNumber != slot {
				t.Fatal("memory receiver did not persist the supplied slot")
			}
		})
	}
}
