package main

import (
	"encoding/json"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine"
	"os"
)

func main() {
	api := engine.NewEngineAPI(nil)
	var key engine.BLSPubkey
	key[0] = 1
	b, err := api.RegisterBuilderV1(engine.SignedBuilderRegistrationV1{Message: engine.BuilderRegistrationV1{Pubkey: key, GasLimit: 30000000, Timestamp: 1000}})
	if err != nil {
		panic(err)
	}
	signed := engine.SignedExecutionPayloadBid{Message: engine.ExecutionPayloadBid{Slot: 10, BuilderIndex: b.Index, Value: 100, BlockHash: types.HexToHash("0x11"), ParentBlockHash: types.HexToHash("0x22"), GasLimit: 30000000}}
	err = api.SubmitBuilderBidV1(signed)
	if err != nil {
		panic(err)
	}
	best, err := api.GetBestBuilderBidV1(10)
	if err != nil {
		panic(err)
	}
	unsignedAccepted := best.Signature == engine.BLSSignature{}
	signed.Message.Slot = 11
	signed.Signature[0] = 1
	if err := api.SubmitBuilderBidV1(signed); err != nil {
		panic(err)
	}
	header, err := api.GetPayloadHeaderV1(11)
	if err != nil {
		panic(err)
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"observed": true, "scope": "In-process actual exported legacy Engine methods with nil execution backend; no RPC endpoint or live chain touched. This does not establish acceptance by the signed CL consensus path.", "zero_registration_signature_accepted": true, "zero_bid_signature_accepted": unsignedAccepted, "nonzero_invalid_signature_accepted": true, "returned_header_signature_zero": header.Bid.Signature == [96]byte{}, "stored_bid_value": best.Message.Value, "registered_balance": b.Balance.String()})
}
