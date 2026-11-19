package eips

import (
	"encoding/hex"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// EIP-7997 makes the established deterministic CREATE2 factory a chain
// requirement. Chains deploy it with an ordinary transaction or genesis alloc;
// block execution must not inject or check it at a fork boundary.

// FactoryAddress is the predeploy address for the EIP-7997 CREATE2 factory.
var FactoryAddress = types.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")

// factoryCodeHex is the bytecode from the EIP-7997 specification.
// It implements a minimal CREATE2 factory that takes salt(32 bytes) || initcode.
const factoryCodeHex = "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3"

// FactoryCode is the decoded bytecode for the EIP-7997 CREATE2 factory.
var FactoryCode []byte

func init() {
	var err error
	FactoryCode, err = hex.DecodeString(factoryCodeHex)
	if err != nil {
		panic("eip7997: invalid factory bytecode hex: " + err.Error())
	}
}

// ApplyEIP7997 deploys the deterministic CREATE2 factory at FactoryAddress.
// This helper is for explicitly constructing genesis/test state only.
// It must not be called during block processing or fork activation.
// If the account already has code, it is a no-op.
func ApplyEIP7997(statedb state.StateDB) {
	if statedb.GetCodeSize(FactoryAddress) > 0 {
		return
	}
	if !statedb.Exist(FactoryAddress) {
		statedb.CreateAccount(FactoryAddress)
	}
	statedb.SetNonce(FactoryAddress, 1)
	statedb.SetCode(FactoryAddress, FactoryCode)
}
