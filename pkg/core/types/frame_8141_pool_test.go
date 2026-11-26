package types

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestFrame8141PublicSixteenPrefixShapes(t *testing.T) {
	for protocol := 0; protocol < 4; protocol++ {
		for base := 0; base < 4; base++ {
			t.Run(fmt.Sprintf("protocol_%d_base_%d", protocol, base), func(t *testing.T) {
				tx := &FrameTx8141{Sender: Address{19: 42}}
				if protocol&1 != 0 {
					data := make([]byte, 8)
					binary.BigEndian.PutUint64(data, 100)
					tx.Frames = append(tx.Frames, Frame8141{Mode: ModeVerify, Target: &FrameExpiryVerifier, Data: data, Limits: FrameGasLimits{Execution: 1000}})
				}
				if protocol&2 != 0 {
					ref := RecentRootReference{Slot: 1}
					tx.Frames = append(tx.Frames, Frame8141{Mode: ModeVerify, Target: &RecentRootAddress, Data: ref.AppendTo(nil), Limits: FrameGasLimits{Execution: 10000}})
				}
				deploy := -1
				if base&1 != 0 {
					deploy = len(tx.Frames)
					tx.Frames = append(tx.Frames, Frame8141{Mode: ModeDefault, Limits: FrameGasLimits{Execution: 1000, State: 500000}})
				}
				flags := uint8(3)
				if base&2 != 0 {
					flags = 2
				}
				tx.Frames = append(tx.Frames, Frame8141{Mode: ModeVerify, Flags: flags, Limits: FrameGasLimits{Execution: 1000}})
				if base&2 != 0 {
					tx.Frames = append(tx.Frames, Frame8141{Mode: ModeVerify, Flags: 1, Limits: FrameGasLimits{Execution: 1000}})
				}
				end := len(tx.Frames) - 1
				tx.Frames = append(tx.Frames, Frame8141{Mode: ModeSender})
				if got, err := tx.ValidationPrefix8141(100); err != nil || got != end {
					t.Fatalf("end=%d err=%v", got, err)
				}
				if got := tx.ValidationDeployIndex8141(); got != deploy {
					t.Fatalf("deploy=%d want=%d", got, deploy)
				}
				tx.Frames[0].Limits.Execution += 100000
				if _, err := tx.ValidationPrefix8141(100); err == nil {
					t.Fatal("oversized prefix accepted")
				}
			})
		}
	}
}

func TestFrame8141PublicRootGrammar(t *testing.T) {
	ref := RecentRootReference{Slot: 1}
	root := Frame8141{Mode: ModeVerify, Target: &RecentRootAddress, Data: ref.AppendTo(nil)}
	account := Frame8141{Mode: ModeVerify, Flags: 3}
	expiry := Frame8141{Mode: ModeVerify, Target: &FrameExpiryVerifier, Data: make([]byte, 8)}
	for name, frames := range map[string][]Frame8141{
		"duplicate": {root, root, account}, "after_account": {account, root}, "reversed": {root, expiry, account},
		"no_account": {root}, "malformed": {{Mode: ModeVerify, Target: &RecentRootAddress, Data: make([]byte, 73)}, account},
	} {
		t.Run(name, func(t *testing.T) {
			tx := &FrameTx8141{Frames: frames}
			if _, err := tx.ValidationPrefix8141(0); err == nil {
				t.Fatal("invalid root grammar admitted")
			}
		})
	}
}
