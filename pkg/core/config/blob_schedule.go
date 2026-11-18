package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"

	"github.com/eth2030/eth2030/core/types"
)

// BlobConfig is one EIP-7840 parameter set. Named forks after Prague inherit
// the latest BPO configuration instead of implicitly changing these values.
type BlobConfig struct {
	Target         uint64 `json:"target"`
	Max            uint64 `json:"max"`
	UpdateFraction uint64 `json:"baseFeeUpdateFraction"`
}

type BlobScheduleConfig struct {
	Cancun *BlobConfig `json:"cancun,omitempty"`
	Prague *BlobConfig `json:"prague,omitempty"`
	BPO1   *BlobConfig `json:"bpo1,omitempty"`
	BPO2   *BlobConfig `json:"bpo2,omitempty"`
	// Older owned genesis generators repeat inherited values under these names.
	// They are accepted only as no-ops when scheduled and removed on resolution.
	Osaka     *BlobConfig `json:"osaka,omitempty"`
	Amsterdam *BlobConfig `json:"amsterdam,omitempty"`
	Bogota    *BlobConfig `json:"bogota,omitempty"`
}

func (s *BlobScheduleConfig) UnmarshalJSON(data []byte) error {
	type plain BlobScheduleConfig
	var decoded plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*s = BlobScheduleConfig(decoded)
	return nil
}

func defaultBlobConfig(name string) BlobConfig {
	switch name {
	case "prague":
		return BlobConfig{6, 9, 5007716}
	case "bpo1":
		return BlobConfig{10, 15, 8346193}
	case "bpo2":
		return BlobConfig{14, 21, 11684671}
	default:
		return BlobConfig{3, 6, 3338477}
	}
}

func (s *BlobScheduleConfig) entry(name string) *BlobConfig {
	if s == nil {
		return nil
	}
	switch name {
	case "cancun":
		return s.Cancun
	case "prague":
		return s.Prague
	case "bpo1":
		return s.BPO1
	case "bpo2":
		return s.BPO2
	}
	return nil
}

func (c *ChainConfig) blobConfigForFork(name string) BlobConfig {
	if c != nil {
		if configured := c.BlobScheduleConfig.entry(name); configured != nil {
			return *configured
		}
	}
	return defaultBlobConfig(name)
}

// BlobConfigAt selects parameters using the current block's timestamp.
func (c *ChainConfig) BlobConfigAt(time uint64) BlobConfig {
	name := "cancun"
	if c != nil {
		switch {
		case c.IsBPO2(time):
			name = "bpo2"
		case c.IsBPO1(time):
			name = "bpo1"
		case c.IsPrague(time):
			name = "prague"
		}
	}
	return c.blobConfigForFork(name)
}

// ResolvedBlobSchedule returns a private complete copy, applying documented
// defaults only to missing entries. Explicit invalid entries never fall back.
func (c *ChainConfig) ResolvedBlobSchedule() *BlobScheduleConfig {
	cancun, prague, bpo1, bpo2 := c.blobConfigForFork("cancun"), c.blobConfigForFork("prague"), c.blobConfigForFork("bpo1"), c.blobConfigForFork("bpo2")
	return &BlobScheduleConfig{Cancun: &cancun, Prague: &prague, BPO1: &bpo1, BPO2: &bpo2}
}

func (c *ChainConfig) ValidateBlobSchedule() error {
	if c == nil {
		return fmt.Errorf("nil chain config")
	}
	s := c.BlobScheduleConfig
	if s == nil {
		return nil
	}
	for name, entry := range map[string]*BlobConfig{"cancun": s.Cancun, "prague": s.Prague, "bpo1": s.BPO1, "bpo2": s.BPO2, "osaka": s.Osaka, "amsterdam": s.Amsterdam, "bogota": s.Bogota} {
		if entry == nil {
			continue
		}
		if entry.Target == 0 || entry.Max == 0 || entry.Target > entry.Max || entry.Max > math.MaxUint64/types.BlobTxBlobGasPerBlob || entry.UpdateFraction == 0 {
			return fmt.Errorf("invalid blobSchedule %s parameters", name)
		}
	}
	for _, alias := range []struct {
		name  string
		time  *uint64
		value *BlobConfig
	}{{"osaka", c.OsakaTime, s.Osaka}, {"amsterdam", c.AmsterdamTime, s.Amsterdam}, {"bogota", c.HogotaTime, s.Bogota}} {
		if alias.time != nil && alias.value != nil && *alias.value != c.BlobConfigAt(*alias.time) {
			return fmt.Errorf("blobSchedule %s must repeat inherited Prague/BPO parameters", alias.name)
		}
	}
	return nil
}

// CheckBlobScheduleCompatible protects every already-activated parameter set,
// including superseded forks needed by historical state/receipt replay.
func CheckBlobScheduleCompatible(old, newCfg *ChainConfig, headTime uint64) error {
	for _, fork := range []struct {
		name             string
		oldTime, newTime *uint64
	}{{"cancun", old.CancunTime, newCfg.CancunTime}, {"prague", old.PragueTime, newCfg.PragueTime}, {"bpo1", old.BPO1Time, newCfg.BPO1Time}, {"bpo2", old.BPO2Time, newCfg.BPO2Time}} {
		if isTimestampForked(fork.oldTime, headTime) || isTimestampForked(fork.newTime, headTime) {
			if old.blobConfigForFork(fork.name) != newCfg.blobConfigForFork(fork.name) {
				return fmt.Errorf("incompatible historical blobSchedule %s", fork.name)
			}
		}
	}
	return nil
}
