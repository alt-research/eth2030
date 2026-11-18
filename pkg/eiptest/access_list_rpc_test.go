package eiptest_test

import (
	"encoding/json"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/eiptest"
	"math/big"
	"strings"
	"testing"
)

func TestRPCActualAccessListDiscovery(t *testing.T) {
	target := types.HexToAddress("0x2222222222222222222222222222222222222222")
	// SLOAD(0), POP, SSTORE(1,7): discover both a read and a write.
	code := []byte{0x60, 0, 0x54, 0x50, 0x60, 7, 0x60, 1, 0x55, 0}
	s := eiptest.New(t, eiptest.FundCode(target, new(big.Int), code, 1))
	rpc := s.RPC()
	raw := rpc.Call(t, "eth_createAccessList", []any{map[string]any{"from": s.Alice().Address.Hex(), "to": target.Hex(), "gas": "0x30d40"}, "latest"})
	var out struct {
		AccessList []struct {
			Address     string   `json:"address"`
			StorageKeys []string `json:"storageKeys"`
		} `json:"accessList"`
		GasUsed string `json:"gasUsed"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.AccessList) != 1 || !strings.EqualFold(out.AccessList[0].Address, target.Hex()) || len(out.AccessList[0].StorageKeys) != 2 {
		t.Fatalf("missing actual storage accesses: %s", raw)
	}
	if out.AccessList[0].StorageKeys[0] != (types.Hash{}).Hex() || out.AccessList[0].StorageKeys[1] != types.HexToHash("0x1").Hex() {
		t.Fatalf("noncanonical keys: %s", raw)
	}
	if out.GasUsed == "0x0" {
		t.Fatal("missing transaction gas accounting")
	}
	stateRaw := rpc.Call(t, "eth_getStorageAt", []any{target.Hex(), "0x1", "latest"})
	var value string
	if err := json.Unmarshal(stateRaw, &value); err != nil {
		t.Fatal(err)
	}
	if value != (types.Hash{}).Hex() {
		t.Fatal("access-list generation changed canonical state")
	}
	again := rpc.Call(t, "eth_createAccessList", []any{map[string]any{"from": s.Alice().Address.Hex(), "to": target.Hex(), "gas": "0x30d40"}, "latest"})
	if string(raw) != string(again) {
		t.Fatal("discovery is not deterministic")
	}
}

func TestRPCAccessListKeepsRevertedChildAccesses(t *testing.T) {
	outer := types.HexToAddress("0x3333333333333333333333333333333333333333")
	child := types.HexToAddress("0x4444444444444444444444444444444444444444")
	call := []byte{0x60, 0, 0x60, 0, 0x60, 0, 0x60, 0, 0x60, 0, 0x73}
	call = append(call, child[:]...)
	call = append(call, 0x61, 0xff, 0xff, 0xf1, 0x50, 0)
	s := eiptest.New(t, eiptest.FundCode(outer, new(big.Int), call, 1), eiptest.FundCode(child, new(big.Int), []byte{0x60, 9, 0x54, 0x50, 0x60, 0, 0x60, 0, 0xfd}, 1))
	raw := s.RPC().Call(t, "eth_createAccessList", []any{map[string]any{"from": s.Alice().Address.Hex(), "to": outer.Hex(), "gas": "0x30d40"}})
	var out struct {
		AccessList []struct {
			Address     string
			StorageKeys []string
		}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.AccessList) != 1 || !strings.EqualFold(out.AccessList[0].Address, child.Hex()) || len(out.AccessList[0].StorageKeys) != 1 || out.AccessList[0].StorageKeys[0] != types.HexToHash("0x9").Hex() {
		t.Fatalf("reverted child access lost: %s", raw)
	}
}
func TestRPCAccessListCreationAndHistoricalState(t *testing.T) {
	target := types.HexToAddress("0x5555555555555555555555555555555555555555")
	code := []byte{0x36, 0x15, 0x60, 0x0b, 0x57, 0x60, 1, 0x60, 0, 0x55, 0, 0x5b, 0x60, 0, 0x54, 0x54, 0x50, 0}
	s := eiptest.New(t, eiptest.FundCode(target, new(big.Int), code, 1))
	s.STF().Tx(s.Alice(), target, []byte{1})
	rpc := s.RPC()
	counts := []int{}
	for _, block := range []string{"0x0", "latest"} {
		raw := rpc.Call(t, "eth_createAccessList", []any{map[string]any{"from": s.Alice().Address.Hex(), "to": target.Hex(), "gas": "0x30d40"}, block})
		var out struct {
			AccessList []struct{ StorageKeys []string }
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		if len(out.AccessList) != 1 {
			t.Fatalf("unexpected list: %s", raw)
		}
		counts = append(counts, len(out.AccessList[0].StorageKeys))
	}
	if counts[0] != 1 || counts[1] != 2 {
		t.Fatalf("historical accesses used the wrong state: %v", counts)
	}
	nonce := s.Nonce(s.Alice().Address)
	created := vm.CreateAddressFromNonce(s.Alice().Address, nonce)
	raw := rpc.Call(t, "eth_createAccessList", []any{map[string]any{"from": s.Alice().Address.Hex(), "input": "0x600160015500", "gas": "0x30d40"}})
	var out struct {
		AccessList []struct {
			Address     string
			StorageKeys []string
		}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.AccessList) != 1 || !strings.EqualFold(out.AccessList[0].Address, created.Hex()) || len(out.AccessList[0].StorageKeys) != 1 {
		t.Fatalf("constructor accesses missing: %s", raw)
	}
	if s.Nonce(s.Alice().Address) != nonce {
		t.Fatal("simulation consumed the sender nonce")
	}
}

func TestRPCAccessListRejectsMalformedArguments(t *testing.T) {
	s := eiptest.New(t)
	rpc := s.RPC()
	for _, args := range []map[string]any{{"gas": "0xno"}, {"from": "0x12"}, {"value": "0xzz"}, {"input": "0x0"}, {"nonce": "0x10000000000000000"}, {"accessList": []any{map[string]any{"address": s.Bob().Address.Hex(), "storageKeys": []string{"0x1"}}}}} {
		if _, err := rpc.CallRaw("eth_createAccessList", []any{args}); err == nil || err.Code != -32602 {
			t.Fatalf("malformed args accepted or wrong error: %v, %v", args, err)
		}
	}
}
