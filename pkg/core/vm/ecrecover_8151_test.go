package vm

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func recoveryInput8151(t *testing.T) ([]byte, types.Address) {
	t.Helper()
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000002")
	if err != nil {
		t.Fatal(err)
	}
	hash := gethcrypto.Keccak256([]byte("8151 raw-code class"))
	sig, err := gethcrypto.Sign(hash, key)
	if err != nil {
		t.Fatal(err)
	}
	input := make([]byte, 128)
	copy(input, hash)
	input[63] = sig[64] + 27
	copy(input[64:], sig[:64])
	return input, types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
}

func TestEIP8151RecoveryCodeClassGasAndUnpaidAccess(t *testing.T) {
	input, address := recoveryInput8151(t)
	delegation := append([]byte{0xef, 1, 0}, bytes.Repeat([]byte{0x42}, 20)...)
	for _, code := range [][]byte{nil, {0}, delegation, delegation[:22], append(bytes.Clone(delegation), 0)} {
		for _, warm := range []bool{false, true} {
			for _, short := range []bool{false, true} {
				name := hex.EncodeToString(code)
				t.Run(name+map[bool]string{false: "/cold", true: "/warm"}[warm]+map[bool]string{false: "/paid", true: "/short"}[short], func(t *testing.T) {
					evm, tracker := newTestEVMWithBAL()
					evm.StateDB = state.NewMemoryStateDB()
					evm.SetForkRules(ForkRules{IsEIP8151: true})
					evm.StateDB.SetCode(address, code)
					if warm {
						evm.StateDB.AddAddressToAccessList(address)
					}
					gas := uint64(6100)
					if warm {
						gas = 3200
					}
					if short {
						gas--
					}
					output, left, err := evm.StaticCall(types.Address{19: 0x44}, types.Address{19: 1}, input, gas)
					touched := false
					for _, a := range tracker.addressTouches {
						if a == address {
							touched = true
						}
					}
					if short {
						if !errors.Is(err, ErrOutOfGas) || left != 0 || touched || evm.StateDB.AddressInAccessList(address) != warm {
							t.Fatal("unpaid code check accessed/warmed signer", err)
						}
						return
					}
					if err != nil || left != 0 || !touched || !evm.StateDB.AddressInAccessList(address) {
						t.Fatal("paid code access/gas differs", err, left)
					}
					if PermittedRecoveryCode8151(code) {
						if len(output) != 32 || !bytes.Equal(output[12:], address[:]) {
							t.Fatal("permitted recovery rejected")
						}
					} else if len(output) != 0 {
						t.Fatal("regular code recovery returned address")
					}
				})
			}
		}
	}
	evm, tracker := newTestEVMWithBAL()
	evm.StateDB = state.NewMemoryStateDB()
	evm.SetForkRules(ForkRules{IsEIP8151: true})
	if out, left, err := evm.StaticCall(types.Address{}, types.Address{19: 1}, nil, 3000); err != nil || len(out) != 0 || left != 0 {
		t.Fatal("invalid recovery changed base behavior")
	}
	for _, a := range tracker.addressTouches {
		if a == address {
			t.Fatal("failed recovery read state")
		}
	}
	evm.SetForkRules(ForkRules{})
	evm.StateDB.SetCode(address, []byte{0})
	if out, _, err := evm.StaticCall(types.Address{}, types.Address{19: 1}, input, 3000); err != nil || len(out) != 32 {
		t.Fatal("pre-fork ecRecover changed")
	}
}

func TestEIP8151CallFamiliesAndOuterRevert(t *testing.T) {
	input, address := recoveryInput8151(t)
	for _, kind := range []string{"call", "callcode", "delegatecall", "staticcall"} {
		t.Run(kind, func(t *testing.T) {
			evm, tracker := newTestEVMWithBAL()
			evm.StateDB = state.NewMemoryStateDB()
			evm.SetForkRules(ForkRules{IsEIP8151: true})
			caller := types.Address{19: 0x44}
			var out []byte
			var left uint64
			var err error
			switch kind {
			case "call":
				out, left, err = evm.Call(caller, types.Address{19: 1}, input, 6100, new(big.Int))
			case "callcode":
				out, left, err = evm.CallCode(caller, types.Address{19: 1}, input, 6100, new(big.Int))
			case "delegatecall":
				out, left, err = evm.DelegateCall(caller, types.Address{19: 1}, input, 6100)
			case "staticcall":
				out, left, err = evm.StaticCall(caller, types.Address{19: 1}, input, 6100)
			}
			if err != nil || left != 0 || len(out) != 32 || !evm.StateDB.AddressInAccessList(address) {
				t.Fatal("call family bypassed restricted recovery", err, left)
			}
			found := false
			for _, a := range tracker.addressTouches {
				if a == address {
					found = true
				}
			}
			if !found {
				t.Fatal("missing signer BAL read")
			}
		})
	}
	for _, prewarm := range []bool{false, true} {
		evm, tracker := newTestEVMWithBAL()
		evm.StateDB = state.NewMemoryStateDB()
		evm.SetForkRules(ForkRules{IsEIP8151: true, IsShanghai: true})
		evm.SetJumpTable(SelectJumpTable(evm.forkRules))
		contract := types.Address{19: 0x44}
		code, _ := hex.DecodeString("60805f5f375f5f60805f600161fffffa505f5ffd")
		evm.StateDB.SetCode(contract, code)
		if prewarm {
			evm.StateDB.AddAddressToAccessList(address)
		}
		_, _, err := evm.Call(types.Address{}, contract, input, 100000, new(big.Int))
		if !errors.Is(err, ErrExecutionReverted) {
			t.Fatal(err)
		}
		if evm.StateDB.AddressInAccessList(address) != prewarm {
			t.Fatal("outer revert retained new warming or lost old warming")
		}
		found := false
		for _, a := range tracker.addressTouches {
			if a == address {
				found = true
			}
		}
		if !found {
			t.Fatal("outer revert erased paid signer read")
		}
	}
}
