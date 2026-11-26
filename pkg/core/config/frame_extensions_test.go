package config

import (
	"math/big"
	"testing"
)

func TestFrameExtensionActivationAndCompatibility(t *testing.T) {
	at := uint64(100)
	for _, name := range []string{"EIP8250", "EIP8151"} {
		t.Run(name, func(t *testing.T) {
			before := *TestConfigHogota
			after := before
			active := func(c *ChainConfig, ts uint64) bool {
				if name == "EIP8250" {
					return c.IsEIP8250(ts)
				}
				return c.IsEIP8151(ts)
			}
			if name == "EIP8250" {
				after.EIP8250Time = &at
			} else {
				after.EIP8151Time = &at
			}
			if name == "EIP8250" {
				after.ExperimentalLegacyFrames = true
				if after.IsLegacyFrameEnabled(100) {
					t.Fatal("historical prototype bypasses keyed schema")
				}
				after.ExperimentalLegacyFrames = false
			}
			if err := after.Validate(); err != nil {
				t.Fatal(err)
			}
			if active(&before, 100) || active(&after, 99) || !active(&after, 100) || !active(&after, 101) {
				t.Fatal("optional activation boundary")
			}
			if err := CheckConfigCompatible(&before, &after, 1, 99); err != nil {
				t.Fatal(err)
			}
			if err := CheckConfigCompatible(&before, &after, 1, 100); err == nil || err.ForkName != name {
				t.Fatal("retroactive activation accepted", err)
			}
			if err := CheckConfigCompatible(&after, &before, 1, 100); err == nil {
				t.Fatal("retroactive removal accepted")
			}
			if name == "EIP8250" {
				after.HogotaTime = nil
			} else {
				after.GlamsterdanTime = nil
			}
			if err := after.Validate(); err == nil {
				t.Fatal("missing prerequisite accepted")
			}
			after.TerminalTotalDifficulty = big.NewInt(0)
			if active(&after, 100) {
				t.Fatal("missing prerequisite activated extension")
			}
		})
	}
}
