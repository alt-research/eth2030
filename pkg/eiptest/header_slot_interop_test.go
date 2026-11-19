package eiptest_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/eiptest"
	"github.com/eth2030/eth2030/engine"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestEngineSlotAndBALIndependentReadback(t *testing.T) {
	target := types.HexToAddress("0xc0ffee")
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkAmsterdam, 0), eiptest.FundCode(target, new(big.Int), []byte{0x4b, 0x60, 0, 0x55, 0}, 1))
	probe := s.Engine()
	rpc := s.RPC()
	for nonce, slot := range []uint64{0, 42, ^uint64(0)} {
		tx := eiptest.DynamicFee(s.Config().ChainID, s.Alice(), uint64(nonce), &target, new(big.Int), nil, 200000, bigFeeCap, bigTipCap)
		probe.Submit(tx)
		attrs := probe.Attributes()
		attrs.SlotNumber = slot
		id, err := probe.FCUWith(attrs)
		if err != nil {
			t.Fatal(err)
		}
		built, err := probe.GetPayload(id)
		if err != nil {
			t.Fatal(err)
		}
		status, err := probe.Process(built.ExecutionPayload)
		if err != nil {
			t.Fatal(err)
		}
		probe.ExpectStatusValid(t, status)
		if built.ExecutionPayload.SlotNumber != fmt.Sprintf("0x%x", slot) || s.StorageAt(target, types.Hash{}) != eiptest.Uint64ToHash(slot) {
			t.Fatalf("CL slot %d did not reach payload and SLOTNUM execution", slot)
		}
		header := probe.HeadBlock().Header()
		encoded, err := header.EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		var independent gethtypes.Header
		if err = gethrlp.DecodeBytes(encoded, &independent); err != nil {
			t.Fatal(err)
		}
		if independent.SlotNumber == nil || *independent.SlotNumber != slot || types.Hash(independent.Hash()) != built.ExecutionPayload.BlockHash {
			t.Fatalf("payload/header mismatch: slot wanted=%d native=%v independent=%v native hash=%s independent hash=%s payload hash=%s", slot, header.SlotNumber, independent.SlotNumber, header.Hash(), independent.Hash(), built.ExecutionPayload.BlockHash)
		}
		raw := rpc.Call(t, "eth_getBlockByHash", []any{header.Hash().Hex(), false})
		var block struct {
			Hash string `json:"hash"`
			Slot string `json:"slotNumber"`
		}
		if err = json.Unmarshal(raw, &block); err != nil || block.Hash != header.Hash().Hex() || block.Slot != fmt.Sprintf("0x%x", slot) {
			t.Fatalf("block RPC lost slot/hash: %s error=%v", raw, err)
		}
		for _, byHash := range []bool{false, true} {
			var bodies []*engine.ExecutionPayloadBodyV2
			method := "engine_getPayloadBodiesByRangeV2"
			params := []any{fmt.Sprintf("0x%x", header.Number.Uint64()), "0x1"}
			if byHash {
				method = "engine_getPayloadBodiesByHashV2"
				params = []any{[]types.Hash{header.Hash()}}
			}
			if err = probe.Call(method, &bodies, params...); err != nil {
				t.Fatal(err)
			}
			if len(bodies) != 1 || bodies[0] == nil || independent.BlockAccessListHash == nil || crypto.Keccak256Hash(bodies[0].BlockAccessList) != types.Hash(*independent.BlockAccessListHash) {
				t.Fatalf("%s does not serve the BAL committed by the independent header", method)
			}
		}
		// The current payload BAL DATA must commit to the same independently read field.
		var data string
		if err = json.Unmarshal(built.ExecutionPayload.BlockAccessList, &data); err != nil {
			t.Fatal(err)
		}
		bal, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
		if err != nil || crypto.Keccak256Hash(bal) != types.Hash(*independent.BlockAccessListHash) {
			t.Fatalf("built BAL differs from committed historical body: %v", err)
		}
	}
}
