package node

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	bridge "github.com/eth2030/eth2030/geth"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/holiman/uint256"
)

// The pinned SDK lacks the four proposed dimensional JSON fields. Compare every
// classic field exactly, and test those explicit extensions separately against
// the selected equations and real receipt/block consumers.
func removeTraceDimensions(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	if message, ok := object["error"].(string); ok && message != "execution reverted" {
		switch {
		case strings.Contains(message, "balance"):
			object["error"] = "balance failure"
		case strings.Contains(message, "write protection"):
			object["error"] = "write protection"
		case strings.Contains(message, "out of gas"):
			object["error"] = "out of gas"
		}
	}
	for _, key := range []string{"regularGasUsed", "stateGasUsed", "gasRefund", "stateGasReservoir"} {
		delete(object, key)
	}
	if children, ok := object["calls"].([]any); ok {
		for _, child := range children {
			removeTraceDimensions(child)
		}
	}
}
func verifyNestedTraceSDK(t *testing.T, rpc track4RPC, raw []byte, sender, parent, child types.Address, parentCode []byte) {
	t.Helper()
	childLog := []byte{0x5f, 0x5f, 0xa0}
	parentLog := append(append([]byte(nil), parentCode[:len(parentCode)-1]...), 0x5f, 0x5f, 0xa0, 0)
	cases := []struct {
		name                        string
		parent, child               []byte
		withLog, onlyTop            bool
		parentBalance, childBalance int64
	}{
		{"state-spill", parentCode, []byte{0x60, 1, 0x60, 0, 0x55, 0}, false, false, 0, 0},
		{"logs", parentLog, append(append([]byte(nil), childLog...), 0), true, false, 0, 0},
		{"reverted-child-log", parentLog, append(append([]byte(nil), childLog...), 0x5f, 0x5f, 0xfd), true, false, 0, 0},
		{"only-top-log-index", parentLog, append(append([]byte(nil), childLog...), 0), true, true, 0, 0},
	}

	makeCall := func(kind byte, value byte) []byte {
		code := []byte{0x5f, 0x5f, 0x5f, 0x5f}
		if kind == 0xf2 {
			code = append(code, 0x60, value)
		}
		code = append(code, 0x73)
		code = append(code, child[:]...)
		return append(code, 0x5a, kind, 0x50, 0)
	}
	destruction := append([]byte{0x73}, sender[:]...)
	destruction = append(destruction, 0xff)
	cases = append(cases,
		struct {
			name                        string
			parent, child               []byte
			withLog, onlyTop            bool
			parentBalance, childBalance int64
		}{"delegate-context", makeCall(0xf4, 0), []byte{0x5f, 0x5f, 0xa0, 0}, true, false, 0, 0},
		struct {
			name                        string
			parent, child               []byte
			withLog, onlyTop            bool
			parentBalance, childBalance int64
		}{"callcode-context", makeCall(0xf2, 1), []byte{0x5f, 0x5f, 0xa0, 0}, true, false, 10, 0},
		struct {
			name                        string
			parent, child               []byte
			withLog, onlyTop            bool
			parentBalance, childBalance int64
		}{"static-write-failure", makeCall(0xfa, 0), []byte{0x60, 1, 0x5f, 0x55, 0}, false, false, 0, 0},
		struct {
			name                        string
			parent, child               []byte
			withLog, onlyTop            bool
			parentBalance, childBalance int64
		}{"create2-memory-padding", []byte{0x5f, 0x60, 1, 0x5f, 0x5f, 0xf5, 0x50, 0}, nil, false, false, 0, 0},
		struct {
			name                        string
			parent, child               []byte
			withLog, onlyTop            bool
			parentBalance, childBalance int64
		}{"selfdestruct-frame", parentCode, destruction, true, false, 0, 1},
	)
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var genesis gethcore.Genesis
			if err := json.Unmarshal(raw, &genesis); err != nil {
				t.Fatal(err)
			}
			pre := make(map[string]bridge.PreAccount)
			for address, account := range genesis.Alloc {
				storage := make(map[types.Hash]types.Hash)
				for key, value := range account.Storage {
					storage[types.Hash(key)] = types.Hash(value)
				}
				pre[address.Hex()] = bridge.PreAccount{Balance: account.Balance, Nonce: account.Nonce, Code: account.Code, Storage: storage}
			}
			for address, code := range map[types.Address][]byte{parent: test.parent, child: test.child} {
				account := pre[gethcommon.Address(address).Hex()]
				account.Code = code
				pre[gethcommon.Address(address).Hex()] = account
			}
			for address, balance := range map[types.Address]int64{parent: test.parentBalance, child: test.childBalance} {
				account := pre[gethcommon.Address(address).Hex()]
				account.Balance = big.NewInt(balance)
				pre[gethcommon.Address(address).Hex()] = account
			}
			reference, err := bridge.MakePreState(pre)
			if err != nil {
				t.Fatal(err)
			}
			defer reference.Close()
			options := map[string]any{"withLog": test.withLog, "onlyTopCall": test.onlyTop}
			configBytes, _ := json.Marshal(options)
			config := bridge.AmsterdamFixtureConfig()
			config.ChainID = big.NewInt(1337)
			observer, err := tracers.DefaultDirectory.New("callTracer", &tracers.Context{}, configBytes, config)
			if err != nil {
				t.Fatal(err)
			}
			header := genesis.ToBlock().Header()
			random := gethcommon.Hash{}
			context := gethvm.BlockContext{CanTransfer: gethcore.CanTransfer, Transfer: gethcore.Transfer, BlockNumber: header.Number, Time: header.Time, GasLimit: header.GasLimit, BaseFee: header.BaseFee, Difficulty: new(big.Int), Random: &random, CostPerStateByte: 1530}
			state := gethstate.NewHookedState(reference.StateDB, observer.Hooks)
			evm := gethvm.NewEVM(context, state, config, gethvm.Config{NoBaseFee: true, Tracer: observer.Hooks})
			defer evm.Release()
			to := gethcommon.Address(parent)
			tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Gas: 500000, GasFeeCap: new(big.Int), GasTipCap: new(big.Int), To: &to, Value: new(big.Int)})
			reference.StateDB.SetTxContext(tx.Hash(), 0, 1)
			observer.Hooks.OnTxStart(evm.GetVMContext(), tx, gethcommon.Address(sender))
			message := (&gethcore.Message{From: gethcommon.Address(sender), To: &to, GasLimit: 500000, Value: uint256.NewInt(0), GasPrice: uint256.NewInt(0), GasFeeCap: uint256.NewInt(0), GasTipCap: uint256.NewInt(0)}).SkipEOACheck().SkipNonceCheck().SkipGasLimitCapCheck().SkipExecutionGasCapCheck()
			result, err := gethcore.ApplyMessage(evm, message, gethcore.NewGasPool(500000))
			if err != nil {
				t.Fatal(err)
			}
			observer.Hooks.OnTxEnd(&gethtypes.Receipt{GasUsed: result.UsedGas}, nil)
			expected, err := observer.GetResult()
			if err != nil {
				t.Fatal(err)
			}
			var native any
			args := map[string]any{"from": sender.Hex(), "to": parent.Hex(), "gas": "0x7a120"}
			overrides := map[string]any{parent.Hex(): map[string]any{"code": hexutil.Encode(test.parent), "balance": hexutil.EncodeBig(big.NewInt(test.parentBalance))}, child.Hex(): map[string]any{"code": hexutil.Encode(test.child), "balance": hexutil.EncodeBig(big.NewInt(test.childBalance))}}
			rpc.call(t, "debug_traceCall", &native, args, "latest", map[string]any{"tracer": "callTracer", "tracerConfig": options, "stateOverrides": overrides})
			removeTraceDimensions(native)
			var gold any
			if err := json.Unmarshal(expected, &gold); err != nil {
				t.Fatal(err)
			}
			removeTraceDimensions(gold)
			if !reflect.DeepEqual(native, gold) {
				actual, _ := json.Marshal(native)
				t.Fatalf("native call tree %s differs from independent SDK %s", actual, expected)
			}
		})
	}
}
