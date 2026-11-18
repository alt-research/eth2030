package config

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/params"
	"math"
	"testing"
)

func TestPublicBPOInputsMatchPinnedOfficialChainSpecs(t *testing.T) {
	for _, pair := range []struct {
		native    *ChainConfig
		reference *params.ChainConfig
	}{{MainnetConfig, params.MainnetChainConfig}, {SepoliaConfig, params.SepoliaChainConfig}} {
		if pair.native.BPO1Time == nil || pair.native.BPO2Time == nil || *pair.native.BPO1Time != *pair.reference.BPO1Time || *pair.native.BPO2Time != *pair.reference.BPO2Time {
			t.Fatal("public BPO inputs differ from pinned official chain spec")
		}
		if pair.native.BlobConfigAt(*pair.native.BPO2Time).Max != 21 {
			t.Fatal("configured public BPO2 not active at boundary")
		}
	}
}

func TestBlobScheduleDefaultsAliasesAndValidation(t *testing.T) {
	cfg := *TestConfig
	osaka, bpo, amsterdam := uint64(0), uint64(24), uint64(24)
	cfg.OsakaTime, cfg.BPO1Time, cfg.AmsterdamTime = &osaka, &bpo, &amsterdam
	for _, raw := range []string{
		`{"prague":{"target":4,"max":7,"baseFeeUpdateFraction":12345},"osaka":{"target":4,"max":7,"baseFeeUpdateFraction":12345},"amsterdam":{"target":10,"max":15,"baseFeeUpdateFraction":8346193}}`,
		`{}`,
	} {
		var schedule BlobScheduleConfig
		if err := json.Unmarshal([]byte(raw), &schedule); err != nil {
			t.Fatal(err)
		}
		cfg.BlobScheduleConfig = &schedule
		if err := cfg.ValidateBlobSchedule(); err != nil {
			t.Fatal(err)
		}
		resolved := cfg.ResolvedBlobSchedule()
		if resolved.Osaka != nil || resolved.Amsterdam != nil || resolved.Cancun == nil || resolved.Prague == nil || resolved.BPO1 == nil || resolved.BPO2 == nil {
			t.Fatal("resolution retained aliases or omitted defaults")
		}
		resolved.Prague.Max = 1
		if cfg.BlobConfigAt(0).Max == 1 {
			t.Fatal("resolved copy mutated input")
		}
	}
	for _, invalid := range []BlobConfig{{0, 1, 1}, {2, 1, 1}, {1, 1, 0}, {1, math.MaxUint64, 1}} {
		cfg.BlobScheduleConfig = &BlobScheduleConfig{Prague: &invalid}
		if cfg.ValidateBlobSchedule() == nil {
			t.Fatalf("invalid explicit parameters fell back: %+v", invalid)
		}
	}
	cfg.BlobScheduleConfig = &BlobScheduleConfig{Osaka: &BlobConfig{4, 7, 123}}
	if cfg.ValidateBlobSchedule() == nil {
		t.Fatal("named fork silently changed BPO parameters")
	}
	var unknown BlobScheduleConfig
	if json.Unmarshal([]byte(`{"unknownFork":{"target":1,"max":2,"baseFeeUpdateFraction":1}}`), &unknown) == nil {
		t.Fatal("unknown fork silently ignored")
	}
}

func TestBlobHistoryProtectsSupersededForkAndAllowsFutureParameters(t *testing.T) {
	old := *TestConfig
	at := uint64(20)
	old.BPO1Time = &at
	old.BlobScheduleConfig = old.ResolvedBlobSchedule()
	updated := old
	updated.BlobScheduleConfig = old.ResolvedBlobSchedule()
	updated.BlobScheduleConfig.BPO1.Max++
	if err := CheckBlobScheduleCompatible(&old, &updated, 19); err != nil {
		t.Fatal("future params rejected:", err)
	}
	if CheckBlobScheduleCompatible(&old, &updated, 20) == nil {
		t.Fatal("active BPO params changed")
	}
	updated.BlobScheduleConfig = old.ResolvedBlobSchedule()
	updated.BlobScheduleConfig.Prague.UpdateFraction++
	if CheckBlobScheduleCompatible(&old, &updated, 30) == nil {
		t.Fatal("superseded Prague history lost its parameters")
	}
}
