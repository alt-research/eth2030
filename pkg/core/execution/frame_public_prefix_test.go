package execution

import (
	"encoding/binary"
	"fmt"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestFrameCAllSixteenPrefixesExecuteAndCaptureDependencies(t *testing.T) {
	for protocol := 0; protocol < 4; protocol++ {
		for base := 0; base < 4; base++ {
			t.Run(fmt.Sprintf("protocol_%d_base_%d", protocol, base), func(t *testing.T) {
				tx, db, cfg, header := frameExecutionFixture(t)
				slot := uint64(2)
				header.SlotNumber = &slot
				// The permissionless verifier approves exactly its current flags. There are
				// no signatures here: this exercises deployed code and prefix composition.
				verifier := []byte{0x60, 3, 0x60, 0x0a, 0xb0, 0xb3, 0x5f, 0x5f, 0xaa}
				factory, paymaster := types.Address{19: 0x90}, types.Address{19: 0x91}
				if base&1 != 0 {
					tx.Sender = types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
					db.AddBalance(tx.Sender, big.NewInt(1000000000))
					db.SetCode(factory, []byte{0x36, 0x5f, 0x5f, 0x37, 0x36, 0x5f, 0x5f, 0xf0, 0x50, 0})
				} else {
					db.SetCode(tx.Sender, verifier)
				}
				tx.Frames = nil
				if protocol&1 != 0 {
					deadline := make([]byte, 8)
					binary.BigEndian.PutUint64(deadline, header.Time)
					db.SetCode(types.FrameExpiryVerifier, types.FrameExpiryCode)
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Target: &types.FrameExpiryVerifier, Data: deadline, Limits: types.FrameGasLimits{Execution: 5000}})
				}
				if protocol&2 != 0 {
					ref := types.RecentRootReference{Slot: 1, Root: types.Hash{31: 7}}
					db.SetCode(types.RecentRootAddress, types.RecentRootCode)
					db.SetState(types.RecentRootAddress, ref.StorageKey(), ref.EntryHash())
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: ref.AppendTo(nil), Limits: types.FrameGasLimits{Execution: 20000}})
				}
				if base&1 != 0 {
					init := append([]byte{0x60, 9, 0x60, 10, 0x5f, 0x39, 0x60, 9, 0x5f, 0xf3}, verifier...)
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeDefault, Target: &factory, Data: init, Limits: types.FrameGasLimits{Execution: 30000, State: 13770}})
				}
				flags := uint8(3)
				payer := tx.Sender
				if base&2 != 0 {
					flags = 2
					payer = paymaster
					db.SetCode(paymaster, verifier)
					db.AddBalance(paymaster, big.NewInt(1000000000))
				}
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: flags, Limits: types.FrameGasLimits{Execution: 1000}})
				if base&2 != 0 {
					tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &paymaster, Limits: types.FrameGasLimits{Execution: 5000}})
				}
				// Executing the body here would fail; prefix admission must stop first.
				invalid := types.Address{19: 0x92}
				db.SetCode(invalid, []byte{0xfe})
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeSender, Target: &invalid, Limits: types.FrameGasLimits{Execution: 1000}})
				root := db.GetRoot()
				validation, err := ValidateFrame8141Dependencies(cfg, db, header, types.NewTransaction(tx), big.NewInt(1), false)
				if err != nil || validation.Payer != payer {
					t.Fatalf("prefix=%+v err=%v", validation, err)
				}
				if !validation.Unchanged(db) || db.GetRoot() != root {
					t.Fatal("prefix retained simulated writes or captured post-state values")
				}
				if _, found := validation.Dependencies[FrameDependency{Address: invalid, Kind: FrameCodeDependency}]; found {
					t.Fatal("body dependency leaked into prefix")
				}
			})
		}
	}
}
