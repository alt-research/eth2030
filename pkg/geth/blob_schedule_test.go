package geth

import (
	"github.com/eth2030/eth2030/core/config"
	geth4844 "github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"testing"
)

func TestBlobScheduleBridgePreservesConfiguredParameters(t *testing.T) {
	cfg := *config.TestConfig
	cfg.AmsterdamTime = nil
	at12, at24 := uint64(12), uint64(24)
	cfg.BPO1Time, cfg.BPO2Time = &at12, &at24
	cfg.BlobScheduleConfig = cfg.ResolvedBlobSchedule()
	cfg.BlobScheduleConfig.BPO1 = &config.BlobConfig{Target: 4, Max: 8, UpdateFraction: 100_000}
	converted := ToGethChainConfig(&cfg)
	for _, time := range []uint64{11, 12, 23, 24} {
		if got, want := geth4844.MaxBlobsPerBlock(converted, time), int(cfg.BlobConfigAt(time).Max); got != want {
			t.Fatalf("time %d bridge capacity %d want %d", time, got, want)
		}
	}
	converted.BlobScheduleConfig.BPO1.Max++
	if cfg.BlobScheduleConfig.BPO1.Max != 8 {
		t.Fatal("reference config mutated native input")
	}
}
