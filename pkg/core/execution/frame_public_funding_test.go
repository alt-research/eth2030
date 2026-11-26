package execution

import (
	"fmt"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

// EIP-8141 Policy Summary permits the factory's code, not an unaccounted
// shared factory balance. CREATE endowment must not evade this dependency
// rule merely because it uses no BALANCE or value-bearing CALL opcode.
func TestFrameCPublicFactoryFundingDependencyScope(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		for _, factoryPayer := range []bool{false, true} {
			t.Run(fmt.Sprintf("create2_%t_factory_payer_%t", create2, factoryPayer), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				factory := types.Address{19: 0x90}
				verifier := []byte{0x60, 3, 0x60, 0x0a, 0xb0, 0xb3, 0x5f, 0x5f, 0xaa}
				init := append([]byte{0x60, 9, 0x60, 10, 0x5f, 0x39, 0x60, 9, 0x5f, 0xf3}, verifier...)
				deploy := []byte{0x36, 0x5f, 0x5f, 0x37}
				if create2 {
					deploy = append(deploy, 0x5f)
				} // salt 0, beneath size/offset/endowment
				deploy = append(deploy, 0x36, 0x5f, 0x60, 1)
				op := byte(0xf0)
				if create2 {
					op = 0xf5
				}
				deploy = append(deploy, op, 0x50, 0)
				// Same existing contract serves as a generic PAY target in the positive
				// exception case, so an approved payer balance remains a valid dependency.
				code := []byte{0x60, 3, 0x60, 0x0a, 0xb0, 0xb3, 0x60, 1, 0x14, 0x60, byte(12 + len(deploy)), 0x57}
				code = append(code, deploy...)
				code = append(code, 0x5b, 0x60, 1, 0x5f, 0x5f, 0xaa)
				db.SetCode(factory, code)
				funding := int64(1)
				if factoryPayer {
					funding = 1000000000
				}
				db.AddBalance(factory, big.NewInt(funding))
				sender := types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
				if create2 {
					sender = types.Address(gethcrypto.CreateAddress2([20]byte(factory), [32]byte{}, gethcrypto.Keccak256(init)))
				}
				tx.Sender = sender
				db.AddBalance(sender, big.NewInt(1000000000))
				flags := uint8(3)
				payer := sender
				if factoryPayer {
					flags = 2
					payer = factory
				}
				tx.Frames = []types.Frame8141{{Mode: types.ModeDefault, Target: &factory, Data: init, Limits: types.FrameGasLimits{Execution: 30000, State: 13770}}, {Mode: types.ModeVerify, Flags: flags, Limits: types.FrameGasLimits{Execution: 1000}}}
				if factoryPayer {
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &factory, Limits: types.FrameGasLimits{Execution: 1000}})
				}
				transaction := types.NewTransaction(tx)
				root := db.GetRoot()
				validation, err := ValidateFrame8141Dependencies(cfg, db, header, transaction, big.NewInt(1), false)
				if (err == nil) != factoryPayer {
					t.Fatalf("unaccounted factory funding admitted=%t want=%t: %v", err == nil, factoryPayer, err)
				}
				if err == nil && validation.Payer != payer {
					t.Fatal("approved factory payer lost")
				}
				if db.GetRoot() != root {
					t.Fatal("public funding validation changed live state")
				}
				receipt, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit))
				if err != nil || receipt.Payer != payer || db.GetNonce(sender) != 2 {
					t.Fatalf("private consensus creation differs: receipt=%+v err=%v", receipt, err)
				}
			})
		}
	}
}
