package vm

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestNativeVMIndependentCanonicalBoundaries(t *testing.T) {
	huge := append([]byte{byte(PUSH32)}, bytes.Repeat([]byte{255}, 32)...)
	overflow := make([]byte, 9)
	overflow[0] = byte(PUSH8)
	binary.BigEndian.PutUint64(overflow[1:], math.MaxUint64-16)
	emptyAddress := types.Address{19: 0x99}
	hashEmpty := append([]byte{byte(PUSH20)}, emptyAddress[:]...)
	hashEmpty = append(hashEmpty, byte(EXTCODEHASH), byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN))
	wideOffset := append([]byte{byte(PUSH32)}, new(big.Int).Lsh(big.NewInt(1), 64).FillBytes(make([]byte, 32))...)
	finishWord := []byte{byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN)}
	initReturn := []byte{byte(PUSH1), 42, byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN)}
	createProgram := append([]byte{byte(PUSH8)}, initReturn...)
	createProgram = append(createProgram, byte(PUSH0), byte(MSTORE), byte(PUSH1), 8, byte(PUSH1), 24, byte(PUSH0), byte(CREATE), byte(POP), byte(RETURNDATASIZE))
	createProgram = append(createProgram, finishWord...)
	collisionProgram := []byte{byte(PUSH0), byte(PUSH0), byte(PUSH0), byte(CREATE), byte(POP), byte(GAS)}
	collisionProgram = append(collisionProgram, finishWord...)

	wideCopy := func(op OpCode) []byte {
		code := append([]byte{byte(PUSH1), 1}, wideOffset...)
		return append(code, byte(PUSH0), byte(op), byte(PUSH1), 32, byte(PUSH0), byte(RETURN))
	}

	programs := []struct {
		name      string
		code      []byte
		address   types.Address
		input     []byte
		gas       uint64
		create    bool
		collision bool
	}{
		{name: "calldata_load_wide_source", code: append(append(bytes.Clone(wideOffset), byte(CALLDATALOAD)), finishWord...), input: []byte{255}},
		{name: "calldata_copy_wide_source", code: wideCopy(CALLDATACOPY), input: []byte{255}},
		{name: "code_copy_wide_source", code: wideCopy(CODECOPY)},
		{name: "zero_returndata_copy_checks_source", code: []byte{byte(PUSH0), byte(PUSH1), 1, byte(PUSH0), byte(RETURNDATACOPY), byte(STOP)}},
		{name: "sstore_stipend_sentry", code: []byte{byte(PUSH1), 0, byte(PUSH1), 0, byte(SSTORE), byte(STOP)}, gas: 2306},
		{name: "create_success_clears_return_buffer", code: createProgram, gas: 500000},
		{name: "create_collision_retains_parent_gas", code: collisionProgram, gas: 500000, collision: true},
		{name: "top_create_exception_burns_child_budget", code: []byte{byte(INVALID)}, gas: 500000, create: true},
		{name: "oversized_initcode_halts_parent", code: []byte{byte(PUSH3), 2, 0, 1, byte(PUSH0), byte(PUSH0), byte(CREATE), byte(STOP)}, gas: 500000},
		{name: "cold_selfdestruct_access", code: append(append([]byte{byte(PUSH20)}, types.Address{19: 0x98}.Bytes()...), byte(SELFDESTRUCT))},

		{name: "multiplication_gas", code: []byte{byte(PUSH1), 2, byte(PUSH1), 3, byte(MUL), byte(PUSH0), byte(MSTORE), byte(PUSH1), 32, byte(PUSH0), byte(RETURN)}},
		{name: "zero_length_return_ignores_offset", code: append(append([]byte{byte(PUSH0)}, huge...), byte(RETURN))},
		{name: "zero_length_keccak_ignores_offset", code: append(append([]byte{byte(PUSH0)}, huge...), byte(KECCAK256), byte(POP), byte(STOP))},
		{name: "zero_length_mcopy_ignores_offsets", code: append(append(append([]byte{byte(PUSH0)}, huge...), huge...), byte(MCOPY), byte(STOP))},
		{name: "memory_word_rounding_overflow", code: append(append([]byte{byte(PUSH1), 1}, overflow...), byte(RETURN))},
		{name: "extcodehash_empty_account", code: hashEmpty},
		{name: "failed_ecadd_burns_all_gas", address: types.Address{19: 6}, input: bytes.Repeat([]byte{255}, 128)},
	}
	for _, amsterdam := range []bool{false, true} {
		for _, tc := range programs {
			t.Run(tc.name+map[bool]string{false: "/Osaka", true: "/Amsterdam"}[amsterdam], func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("native panic on consensus input: %v", r)
					}
				}()
				address := tc.address
				if address == (types.Address{}) {
					address = types.Address{19: 0x42}
				}
				caller := types.Address{19: 0xe0}
				budget := tc.gas
				if budget == 0 {
					budget = 100000
				}
				collisionAddress := types.Address(gethcrypto.CreateAddress(gethcommon.Address(address), 1))
				db := state.NewMemoryStateDB()
				db.CreateAccount(emptyAddress)
				if tc.collision {
					db.SetNonce(collisionAddress, 1)
					db.SetCode(collisionAddress, []byte{0})
				}
				if len(tc.code) > 0 {
					db.SetCode(address, tc.code)
					db.SetNonce(address, 1)
				}
				rules := ForkRules{IsEIP158: true, IsLondon: true, IsBerlin: true, IsShanghai: true, IsCancun: true, IsPrague: true, IsOsaka: true, IsAmsterdam: amsterdam}
				evm := NewEVMWithState(BlockContext{BlockNumber: big.NewInt(1), Time: 12, GasLimit: 1000000}, TxContext{}, Config{}, db)
				evm.SetForkRules(rules)
				evm.SetJumpTable(SelectJumpTable(rules))
				evm.SetPrecompiles(SelectPrecompiles(rules))
				db.AddAddressToAccessList(caller)
				db.AddAddressToAccessList(address)
				for address := range SelectPrecompiles(rules) {
					db.AddAddressToAccessList(address)
				}
				var got []byte
				var left uint64
				var nerr error
				if tc.create {
					got, _, left, nerr = evm.Create(caller, tc.code, budget, new(big.Int))
				} else {
					got, left, nerr = evm.Call(caller, address, tc.input, budget, new(big.Int))
				}
				sdk, err := gethstate.New(gethtypes.EmptyRootHash, gethstate.NewDatabaseForTesting())
				if err != nil {
					t.Fatal(err)
				}
				sdk.CreateAccount(gethcommon.Address(emptyAddress))
				if tc.collision {
					sdk.SetNonce(gethcommon.Address(collisionAddress), 1, tracing.NonceChangeUnspecified)
					sdk.SetCode(gethcommon.Address(collisionAddress), []byte{0}, tracing.CodeChangeUnspecified)
				}
				if len(tc.code) > 0 {
					sdk.SetCode(gethcommon.Address(address), tc.code, tracing.CodeChangeUnspecified)
					sdk.SetNonce(gethcommon.Address(address), 1, tracing.NonceChangeUnspecified)
				}
				cfg := *params.AllDevChainProtocolChanges
				cfg.AmsterdamTime = nil
				cfg.BogotaTime = nil
				if amsterdam {
					cfg.AmsterdamTime = new(uint64)
				}
				runtimeConfig := &runtime.Config{ChainConfig: &cfg, State: sdk, GasLimit: budget, Origin: gethcommon.Address(caller), BlockNumber: big.NewInt(1), Time: 12, Value: new(big.Int)}
				var want []byte
				var wantLeft uint64
				var werr error
				if tc.create {
					want, _, wantLeft, werr = runtime.Create(tc.code, runtimeConfig)
				} else {
					want, wantLeft, werr = runtime.Call(gethcommon.Address(address), tc.input, runtimeConfig)
				}
				if (nerr == nil) != (werr == nil) || left != wantLeft || !bytes.Equal(got, want) {
					t.Fatalf("native output=%x left=%d err=%v; unchanged SDK output=%x left=%d err=%v", got, left, nerr, want, wantLeft, werr)
				}
			})
		}
	}
}
