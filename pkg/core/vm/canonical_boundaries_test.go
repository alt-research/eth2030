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
	programs := []struct {
		name    string
		code    []byte
		address types.Address
		input   []byte
	}{
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
				db := state.NewMemoryStateDB()
				db.CreateAccount(emptyAddress)
				if len(tc.code) > 0 {
					db.SetCode(address, tc.code)
					db.SetNonce(address, 1)
				}
				rules := ForkRules{IsEIP158: true, IsBerlin: true, IsShanghai: true, IsCancun: true, IsPrague: true, IsOsaka: true, IsAmsterdam: amsterdam}
				evm := NewEVMWithState(BlockContext{BlockNumber: big.NewInt(1), Time: 12, GasLimit: 1000000}, TxContext{}, Config{}, db)
				evm.SetForkRules(rules)
				evm.SetJumpTable(SelectJumpTable(rules))
				evm.SetPrecompiles(SelectPrecompiles(rules))
				got, left, nerr := evm.Call(caller, address, tc.input, 100000, new(big.Int))
				sdk, err := gethstate.New(gethtypes.EmptyRootHash, gethstate.NewDatabaseForTesting())
				if err != nil {
					t.Fatal(err)
				}
				sdk.CreateAccount(gethcommon.Address(emptyAddress))
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
				want, wantLeft, werr := runtime.Call(gethcommon.Address(address), tc.input, &runtime.Config{ChainConfig: &cfg, State: sdk, GasLimit: 100000, Origin: gethcommon.Address(caller), BlockNumber: big.NewInt(1), Time: 12})
				if (nerr == nil) != (werr == nil) || left != wantLeft || !bytes.Equal(got, want) {
					t.Fatalf("native output=%x left=%d err=%v; unchanged SDK output=%x left=%d err=%v", got, left, nerr, want, wantLeft, werr)
				}
			})
		}
	}
}
