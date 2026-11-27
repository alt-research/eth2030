package vm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// Exercise actual recover-and-compare authorization kernels, not a mocked
// ecrecover return. These are compatibility controls, not an assertion that
// complete deployed permit/multisig/dispute applications have been audited.
func TestMigration8151PermitMultisigAndFaultProofAuthorization(t *testing.T) {
	for _, application := range []string{"permit", "multisig-two-of-two", "fault-proof"} {
		for _, restricted := range []bool{false, true} {
			evm, db, _ := adoptionEVM8298()
			rules := evm.GetForkRules()
			rules.IsEIP8151 = restricted
			evm.SetForkRules(rules)
			var inputs []byte
			var addresses []types.Address
			count := 1
			if application == "multisig-two-of-two" {
				count = 2
			}
			for i := 0; i < count; i++ {
				key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", i+2))
				message := gethcrypto.Keccak256([]byte(application))
				sig, _ := gethcrypto.Sign(message, key)
				input := make([]byte, 128)
				copy(input, message)
				input[63] = sig[64] + 27
				copy(input[64:], sig[:64])
				inputs = append(inputs, input...)
				addresses = append(addresses, types.Address(gethcrypto.PubkeyToAddress(key.PublicKey)))
			}
			// Only the first authority migrated. A second unchanged EOA does
			// not make a two-of-two policy safe to keep using the retired key.
			db.SetCode(addresses[0], []byte{0})
			code := []byte{byte(PUSH2), byte(len(inputs) >> 8), byte(len(inputs)), byte(PUSH0), byte(PUSH0), byte(CALLDATACOPY)}
			for i, a := range addresses {
				code = append(code, byte(PUSH1), 32, byte(PUSH2), 1, 0, byte(PUSH1), 128, byte(PUSH1), byte(i*128), byte(PUSH1), 1, byte(PUSH2), 0xff, 0xff, byte(STATICCALL), byte(POP), byte(PUSH2), 1, 0, byte(MLOAD), byte(PUSH20))
				code = append(code, a[:]...)
				code = append(code, byte(EQ), byte(PUSH2), 0, 0)
				jumpAt := len(code) - 2
				code = append(code, byte(JUMPI), byte(PUSH0), byte(PUSH0), byte(REVERT))
				destination := len(code)
				code[jumpAt], code[jumpAt+1] = byte(destination>>8), byte(destination)
				code = append(code, byte(JUMPDEST))
			}
			code = append(code, byte(PUSH1), 1, byte(PUSH0), byte(SSTORE), byte(STOP))
			contract := types.Address{19: 0x91}
			db.SetCode(contract, code)
			_, _, err := evm.Call(types.Address{19: 0x92}, contract, inputs, 1_000_000, new(big.Int))
			if (err != nil) != restricted || db.GetState(contract, types.Hash{}).IsZero() != restricted {
				t.Fatal(application, "restricted", restricted, "authorization/state", err)
			}
		}
	}
}
