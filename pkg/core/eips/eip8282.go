package eips

import (
	_ "embed"
	"encoding/hex"
	"strings"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// Canonical sys-asm assets: 83f9801245ff56878a450b5625801101b9a225a1.
//
//go:embed contracts/eip8282/deposit_runtime.hex
var builderDepositRuntimeHex string

//go:embed contracts/eip8282/deposit_init.hex
var builderDepositInitHex string

//go:embed contracts/eip8282/exit_runtime.hex
var builderExitRuntimeHex string

//go:embed contracts/eip8282/exit_init.hex
var builderExitInitHex string

func builderContractBytes(value string) []byte {
	code, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		panic("invalid embedded EIP-8282 bytecode: " + err.Error())
	}
	return code
}
func BuilderDepositRuntime() []byte  { return builderContractBytes(builderDepositRuntimeHex) }
func BuilderDepositInitCode() []byte { return builderContractBytes(builderDepositInitHex) }
func BuilderExitRuntime() []byte     { return builderContractBytes(builderExitRuntimeHex) }
func BuilderExitInitCode() []byte    { return builderContractBytes(builderExitInitHex) }

// ApplyBuilderRequestGenesis explicitly allocates canonical contracts when
// constructing an owned genesis/test state. It is never a fork-time transition.
// Callers must apply it before computing or publishing the genesis state root.
func ApplyBuilderRequestGenesis(db state.StateDB) {
	depositBalance := db.GetBalance(types.BuilderDepositRequestAddress)
	db.CreateAccount(types.BuilderDepositRequestAddress)
	db.AddBalance(types.BuilderDepositRequestAddress, depositBalance)
	db.SetNonce(types.BuilderDepositRequestAddress, 1)
	db.SetCode(types.BuilderDepositRequestAddress, BuilderDepositRuntime())
	exitBalance := db.GetBalance(types.BuilderExitRequestAddress)
	db.CreateAccount(types.BuilderExitRequestAddress)
	db.AddBalance(types.BuilderExitRequestAddress, exitBalance)
	db.SetNonce(types.BuilderExitRequestAddress, 1)
	db.SetCode(types.BuilderExitRequestAddress, BuilderExitRuntime())
	inhibitor := types.Hash{}
	for i := range inhibitor {
		inhibitor[i] = 255
	}
	db.SetState(types.BuilderExitRequestAddress, types.Hash{}, inhibitor)
}
