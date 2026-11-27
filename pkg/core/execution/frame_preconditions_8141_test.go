package execution

import (
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

type framePreconditionReadFault struct {
	state.StateDB
	err error
}

func (f *framePreconditionReadFault) Error() error { return f.err }

func TestGroupEFramePreconditionsAreReadOnlyAndFailClosed(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "keyed"}[keyed], func(t *testing.T) {
			transaction, _ := proof8504.CanonicalTransaction(keyed)
			frame := transaction.Inner().(*types.FrameTx8141)
			cfg := *config.TestConfigHogota
			cfg.ChainID = big.NewInt(1337)
			zero := uint64(0)
			cfg.EIP8504Time = &zero
			if keyed {
				cfg.EIP8250Time = &zero
			} else {
				cfg.EIP8250Time = nil
			}
			db := state.NewMemoryStateDB()
			balance := big.NewInt(1000000000000000000)
			db.AddBalance(frame.Sender, balance)
			// This code would mutate storage if a precondition check ran the EVM.
			db.SetCode(frame.Sender, []byte{0x60, 1, 0x5f, 0x55, 0})
			db.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			db.SetNonce(types.NonceManager8250, 1)
			h := &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 1000000, BaseFee: big.NewInt(1)}
			gp := new(gaspool.GasPool).AddGas(h.GasLimit)
			for i := 0; i < 3; i++ {
				if err := ValidateFramePreconditions8141(&cfg, db, h, frame, gp); err != nil {
					t.Fatal(err)
				}
			}
			if gp.Gas() != h.GasLimit || gp.StateGas() != h.GasLimit || db.GetNonce(frame.Sender) != 0 || db.GetBalance(frame.Sender).Cmp(balance) != 0 || db.GetState(frame.Sender, types.Hash{}) != (types.Hash{}) || db.AddressInAccessList(frame.Sender) {
				t.Fatal("preconditions mutated state, warmth or gas capacity")
			}
			if p := frame.NonceProfile8250(); p != nil {
				for _, k := range p.NonceKeys {
					if db.GetState(types.NonceManager8250, types.NonceSlot8250(frame.Sender, k)) != (types.Hash{}) {
						t.Fatal("preconditions consumed nonce")
					}
				}
			}
			tiny := new(gaspool.GasPool).AddGas(1)
			if err := ValidateFramePreconditions8141(&cfg, db, h, frame, tiny); !errors.Is(err, gaspool.ErrGasPoolExhausted) || tiny.Gas() != 1 || tiny.StateGas() != 1 {
				t.Fatal("reservation check changed capacity", err)
			}
			unavailable := errors.New("authenticated account/slot unavailable")
			fault := &framePreconditionReadFault{StateDB: db, err: unavailable}
			// The read error must survive even if a fabricated zero would look like
			// an invalid nonce. It cannot become an omission excuse.
			bad := *frame
			bad.Nonce = 1
			if err := ValidateFramePreconditions8141(&cfg, fault, h, &bad, gp); !errors.Is(err, unavailable) {
				t.Fatal("read failure was hidden by validity result", err)
			}
		})
	}
}
