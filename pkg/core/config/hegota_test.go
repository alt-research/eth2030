package config

import (
	"math/big"
	"testing"
)

func TestLegacyFramesRequireExplicitForkScope(t *testing.T) {
	activation := uint64(100)
	for _, tt := range []struct {
		name                 string
		time                 uint64
		opt, scheduled, want bool
	}{
		{"before", 99, true, true, false}, {"at", 100, true, true, true}, {"after", 101, true, true, true}, {"no schedule", 100, true, false, false}, {"canonical does not enable prototype", 100, false, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := *TestConfigHogota
			c.HogotaTime = nil
			if tt.scheduled {
				c.HogotaTime = &activation
			}
			c.ExperimentalLegacyFrames = tt.opt
			if got := c.IsLegacyFrameEnabled(tt.time); got != tt.want {
				t.Fatalf("got %t want %t", got, tt.want)
			}
		})
	}
	var absent *ChainConfig
	if absent.IsLegacyFrameEnabled(100) {
		t.Fatal("nil config enabled frames")
	}
}

func TestLegacyFrameConfigChangeAfterActivationIsIncompatible(t *testing.T) {
	before := *TestConfigHogota
	after := before
	ts := uint64(100)
	before.HogotaTime = &ts
	after.HogotaTime = &ts
	after.ExperimentalLegacyFrames = true
	if err := CheckConfigCompatible(&before, &after, 1, 99); err != nil {
		t.Fatal("future opt-in cannot change unexecuted state", err)
	}
	if err := CheckConfigCompatible(&before, &after, 1, 100); err == nil || err.ForkName != "ExperimentalLegacyFrames" {
		t.Fatal("retroactive prototype activation accepted")
	}
	if err := CheckConfigCompatible(&after, &before, 1, 101); err == nil {
		t.Fatal("prototype deactivation accepted for executed history")
	}
}

func TestConfigCompatibilityRejectsRetroactiveRemoteActivation(t *testing.T) {
	for _, name := range []string{"Hogota", "IPlus"} {
		t.Run(name, func(t *testing.T) {
			local := *TestConfig
			remote := local
			at := uint64(100)
			if name == "Hogota" {
				remote.HogotaTime = &at
			} else {
				remote.IPlusTime = &at
			}
			if err := CheckConfigCompatible(&local, &remote, 1, 99); err != nil {
				t.Fatal(err)
			}
			if err := CheckConfigCompatible(&local, &remote, 1, 100); err == nil || err.ForkName != name {
				t.Fatal("retroactive remote activation accepted", err)
			}
			if err := CheckConfigCompatible(&remote, &local, 1, 100); err == nil {
				t.Fatal("retroactive deactivation accepted")
			}
		})
	}
}
func TestLegacyFrameGateIncludesBlockPrerequisites(t *testing.T) {
	c := *TestConfigHogota
	c.ExperimentalLegacyFrames = true
	c.LondonBlock = big.NewInt(10)
	if c.IsLegacyFrameEnabledAt(big.NewInt(9), 100) {
		t.Fatal("enabled before London")
	}
	if !c.IsLegacyFrameEnabledAt(big.NewInt(10), 100) {
		t.Fatal("explicit profile disabled at prerequisites")
	}
	c.TerminalTotalDifficulty = nil
	if c.IsLegacyFrameEnabledAt(big.NewInt(10), 100) {
		t.Fatal("enabled before Merge")
	}
}
