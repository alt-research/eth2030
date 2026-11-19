package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/node"
)

func TestHistoryMaintenanceRejectsMissingExistingInputsWithoutCreatingData(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "missing")
	if _, err := build(context.Background(), directory, filepath.Join(root, "genesis.json")); err == nil {
		t.Fatal("accepted missing original database")
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("maintenance created a new database instead of requiring existing data")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := build(ctx, directory, "unused"); err != context.Canceled {
		t.Fatalf("pre-cancelled maintenance continued: %v", err)
	}
}

func TestHistoryMaintenanceUsesExclusiveOriginalDatabaseAndReleasesLock(t *testing.T) {
	directory := t.TempDir()
	genesis := filepath.Join(directory, "genesis.json")
	input, err := json.Marshal(map[string]any{
		"config": map[string]any{"chainId": 1337, "terminalTotalDifficulty": "0x0", "homesteadBlock": 0, "eip155Block": 0, "eip158Block": 0, "byzantiumBlock": 0, "constantinopleBlock": 0, "petersburgBlock": 0, "istanbulBlock": 0, "berlinBlock": 0, "londonBlock": 0, "shanghaiTime": 0, "cancunTime": 0, "pragueTime": 0, "osakaTime": 0},
		"nonce":  "0x0", "timestamp": "0x0", "extraData": "0x", "gasLimit": "0x1c9c380", "difficulty": "0x0", "alloc": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(genesis, input, 0600); err != nil {
		t.Fatal(err)
	}
	config := node.DefaultConfig()
	config.DataDir, config.GenesisPath = directory, genesis
	instance, err := node.New(&config)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Stop()
	head := instance.Blockchain().CurrentBlock().Hash()
	if _, err := build(context.Background(), directory, genesis); err == nil {
		t.Fatal("maintenance opened the live node database concurrently")
	}
	if instance.Blockchain().CurrentBlock().Hash() != head {
		t.Fatal("failed maintenance changed live head")
	}
	if err := instance.Stop(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		result, err := build(context.Background(), directory, genesis)
		if err != nil || result.Head != head || result.Number != 0 || result.Indexed != 0 {
			t.Fatalf("exclusive maintenance/reopen changed original state: %+v %v", result, err)
		}
	}
}
