package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// EIP-8141 permits only sender SSTOREs and sender deployment as prefix state
// writes. EIP-1153 defines TSTORE as a transient storage write; it receives no
// additional public exception. Private execution retains ordinary semantics.
func TestFrameCPublicTransientStorageScope(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		factory []byte
		init    []byte
		verify  []byte
		caught  bool
		accept  bool
	}{
		{name: "factory_write", factory: []byte{0x60, 1, 0x5f, 0x5d}},
		{name: "factory_read", factory: []byte{0x5f, 0x5c, 0x50}},
		{name: "constructor_write", init: []byte{0x60, 1, 0x5f, 0x5d}},
		{name: "caught_helper_write", caught: true},
		{name: "sender_read", verify: []byte{0x5f, 0x5c, 0x50}, accept: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			tx, db, cfg, header := frameExecutionFixture(t)
			factory := types.Address{19: 0x90}
			if scenario.caught {
				helper := types.Address{19: 0x91}
				db.SetCode(helper, []byte{0x60, 1, 0x5f, 0x5d, 0})
				scenario.factory = []byte{0x5f, 0x5f, 0x5f, 0x5f, 0x5f, 0x73}
				scenario.factory = append(scenario.factory, helper[:]...)
				scenario.factory = append(scenario.factory, 0x5a, 0xf1, 0x50)
			}
			verifier := append(append([]byte(nil), scenario.verify...), 0x60, 3, 0x5f, 0x5f, 0xaa)
			init := append([]byte(nil), scenario.init...)
			init = append(init, 0x60, byte(len(verifier)), 0x60, byte(len(init)+10), 0x5f, 0x39, 0x60, byte(len(verifier)), 0x5f, 0xf3)
			init = append(init, verifier...)
			factoryCode := append(append([]byte(nil), scenario.factory...), 0x36, 0x5f, 0x5f, 0x37, 0x36, 0x5f, 0x5f, 0xf0, 0x50, 0)
			db.SetCode(factory, factoryCode)
			tx.Sender = types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
			db.AddBalance(tx.Sender, big.NewInt(1000000000))
			tx.Frames = []types.Frame8141{{Mode: types.ModeDefault, Target: &factory, Data: init, Limits: types.FrameGasLimits{Execution: 30000, State: 13770}}, {Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}
			transaction := types.NewTransaction(tx)
			root := db.GetRoot()
			_, err := ValidateFrame8141Dependencies(cfg, db, header, transaction, big.NewInt(1), false)
			if (err == nil) != scenario.accept {
				t.Fatalf("public transient scope accepted=%t want=%t: %v", err == nil, scenario.accept, err)
			}
			if db.GetRoot() != root {
				t.Fatal("public transient validation changed live state")
			}
			if _, _, err := ApplyTransaction(cfg, db, header, transaction, new(gaspool.GasPool).AddGas(header.GasLimit)); err != nil {
				t.Fatalf("private transient execution was rejected: %v", err)
			}
		})
	}
}
