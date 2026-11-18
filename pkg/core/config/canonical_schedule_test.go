package config

import (
	"math/big"
	"testing"
)

func TestCanonicalForkScheduleDependencies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*ChainConfig)
	}{
		{"missing Shanghai", func(c *ChainConfig) { c.ShanghaiTime = nil }},
		{"missing Cancun", func(c *ChainConfig) { c.CancunTime = nil }},
		{"missing Prague", func(c *ChainConfig) { c.PragueTime = nil }},
		{"missing Osaka", func(c *ChainConfig) { c.OsakaTime = nil }},
		{"Osaka after Amsterdam", func(c *ChainConfig) { c.OsakaTime = newUint64(2) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := *TestConfig
			cfg.OsakaTime = newUint64(0)
			tc.change(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("incoherent canonical schedule accepted")
			}
		})
	}
	for _, cfg := range []*ChainConfig{MainnetConfig, SepoliaConfig, HoleskyConfig, TestConfig, TestConfigGlamsterdan, TestnetConfig(), DevConfig(), {ChainID: big.NewInt(1337)}} {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("valid configured prefix rejected: %v", err)
		}
	}
}

func TestConfigCompatibilityIsSymmetricAtEitherActivation(t *testing.T) {
	before := *TestConfig
	after := before
	before.AmsterdamTime = nil
	after.AmsterdamTime = newUint64(100)
	for _, head := range []uint64{99, 100, 101} {
		left := CheckConfigCompatible(&before, &after, 1, head)
		right := CheckConfigCompatible(&after, &before, 1, head)
		if (left != nil) != (head >= 100) || (right != nil) != (head >= 100) {
			t.Fatalf("head %d asymmetric activated-fork compatibility: %v / %v", head, left, right)
		}
	}
	before.OsakaTime = newUint64(50)
	after.OsakaTime = newUint64(60)
	if len(ConfigDiff(&before, &after)) != 2 {
		t.Fatal("Osaka fork missing from configuration identity")
	}
}
