package eips_test

import (
	"bytes"
	"encoding/hex"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
	"math/big"
	"testing"
)

func TestCanonicalFactoryGenesisAndExecution(t *testing.T) {
	address := types.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")
	runtime, _ := hex.DecodeString("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3")
	db := state.NewMemoryStateDB()
	eips.ApplyEIP7997(db)
	if eips.FactoryAddress != address || db.GetNonce(address) != 1 || !bytes.Equal(db.GetCode(address), runtime) {
		t.Fatal("genesis factory does not match the canonical address/runtime/nonce")
	}
	sender := types.HexToAddress("0x1234")
	db.AddBalance(sender, big.NewInt(1000))
	evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, db)
	evm.SetJumpTable(vm.NewCancunJumpTable())
	init := []byte{0x60, 1, 0x60, 0, 0xf3}
	salt := make([]byte, 32)
	salt[31] = 7
	input := append(append([]byte(nil), salt...), init...)
	codehash := crypto.Keccak256Hash(init)
	preimage := append([]byte{0xff}, address[:]...)
	preimage = append(preimage, salt...)
	preimage = append(preimage, codehash[:]...)
	h := crypto.Keccak256Hash(preimage)
	want := types.BytesToAddress(h[12:])
	output, _, err := evm.Call(sender, address, input, 500000, big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 20 || !bytes.Equal(output, want[:]) || !bytes.Equal(db.GetCode(want), []byte{0}) || db.GetBalance(want).Cmp(big.NewInt(7)) != 0 {
		t.Fatalf("incorrect factory result: %x", output)
	}
	if _, _, err := evm.Call(sender, address, input, 500000, big.NewInt(0)); err == nil {
		t.Fatal("factory must revert on CREATE2 collision")
	}
	if _, _, err := evm.Call(sender, address, []byte{1}, 500000, big.NewInt(0)); err == nil {
		t.Fatal("factory accepted input shorter than salt")
	}
}
