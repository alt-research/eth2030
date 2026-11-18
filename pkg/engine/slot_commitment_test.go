package engine

import (
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"testing"
)

func TestMemoryEngineSlotCommitment(t *testing.T) {
	for _, slot := range []uint64{0, 42, ^uint64(0)} {
		t.Run(fmt.Sprint(slot), func(t *testing.T) {
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
