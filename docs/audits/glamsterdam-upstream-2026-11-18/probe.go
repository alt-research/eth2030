// Run from pkg: go run ../docs/audits/glamsterdam-upstream-2026-11-18/probe.go
// This read-only probe reports current behavior; it is not a conformance test.
package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
)

func main() {
	first, second := types.HexToHash("0x01"), types.HexToHash("0x02")
	makeHeader := func(commitment *types.Hash) *types.Header {
		return &types.Header{Number: big.NewInt(100), Time: 100, GasLimit: 60_000_000, BaseFee: big.NewInt(1), BlockAccessListHash: commitment}
	}
	a, b := makeHeader(&first), makeHeader(&second)
	fmt.Printf("different_BAL_commitments_same_header_hash=%v\n", a.Hash() == b.Hash())
	aRLP, _ := a.EncodeRLP()
	bRLP, _ := b.EncodeRLP()
	fmt.Printf("different_BAL_commitments_same_header_RLP=%v\n", string(aRLP) == string(bRLP))
	_, hasSlot := reflect.TypeOf(types.Header{}).FieldByName("SlotNumber")
	fmt.Printf("native_header_has_SlotNumber=%v\n", hasSlot)
	fmt.Printf("BAL_retention_epochs=%d\n", rawdb.BALRetentionEpochs)
	body := payload.BlockToPayloadBodyV2(types.NewBlock(a, &types.Body{}))
	wire, _ := json.Marshal(body)
	fmt.Printf("body_V2_for_header_with_BAL_hash=%s\n", wire)

	s := state.NewMemoryStateDB()
	sender := types.HexToAddress("0x1111111111111111111111111111111111111111")
	to := types.HexToAddress("0x2222222222222222222222222222222222222222")
	s.CreateAccount(sender)
	s.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil))
	s.CreateAccount(to)
	s.SetCode(to, []byte{0x4b, 0x60, 0, 0x52, 0x60, 0x20, 0x60, 0, 0xf3})
	msg := &config.Message{From: sender, To: &to, GasLimit: 100_000, GasPrice: big.NewInt(1), Value: big.NewInt(0)}
	result, err := execution.ApplyMessage(config.TestConfigGlamsterdan, nil, s, a, msg, new(gaspool.GasPool).AddGas(a.GasLimit))
	if err != nil {
		panic(err)
	}
	fmt.Printf("native_ApplyMessage_SLOTNUM=%s error=%v\n", new(big.Int).SetBytes(result.ReturnData), result.Err)
}
