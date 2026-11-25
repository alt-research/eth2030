package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// runBinaryMigration converts committed flat state while holding the source
// and destination FileDB locks. Each batch publishes its own checkpoint.
func runBinaryMigration(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("binary-migrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	source := flags.String("source", "", "existing FileDB with committed TrieStateDB tables")
	dest := flags.String("destination", "", "FileDB for binary nodes and checkpoints")
	root := flags.String("source-root", "", "trusted 32-byte MPT state root in hex")
	size := flags.Int("batch-size", 1000, "maximum accounts per checkpoint")
	maxBatches := flags.Int("max-batches", 0, "stop after this many batches (0 runs to completion)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(*root, "0x"))
	if *source == "" || *dest == "" || *size <= 0 || *maxBatches < 0 || flags.NArg() != 0 || err != nil || len(decoded) != 32 {
		fmt.Fprintln(stderr, "binary-migrate requires source, destination, a 32-byte source-root, a positive batch-size and a nonnegative max-batches")
		return 2
	}
	progress, err := migrateBinaryFiles(*source, *dest, types.BytesToHash(decoded), *size, *maxBatches)
	if err != nil {
		fmt.Fprintf(stderr, "binary-migrate: %v\n", err)
		return 1
	}
	result := struct {
		SourceRoot string `json:"source_root"`
		BinaryRoot string `json:"binary_root"`
		Accounts   uint64 `json:"accounts"`
		Complete   bool   `json:"complete"`
	}{progress.SourceRoot.Hex(), progress.Root.Hex(), progress.Accounts, progress.Complete}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintf(stderr, "binary-migrate: write result: %v\n", err)
		return 1
	}
	return 0
}

func migrateBinaryFiles(source, dest string, root types.Hash, size, maxBatches int) (progress state.BinaryMigrationProgress, err error) {
	var sourceDirectories []os.FileInfo
	for _, name := range []string{"data", "data-v2"} {
		info, statErr := os.Stat(filepath.Join(source, name))
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return progress, fmt.Errorf("source FileDB: %w", statErr)
		}
		if !info.IsDir() {
			return progress, errors.New("source FileDB data path is not a directory")
		}
		sourceDirectories = append(sourceDirectories, info)
	}
	if len(sourceDirectories) == 0 {
		return progress, errors.New("source FileDB has no value directory")
	}
	// Both legacy and current value directories must be independent, including
	// cross-layout symlink aliases, before opening either database.
	for _, name := range []string{"data", "data-v2"} {
		info, statErr := os.Stat(filepath.Join(dest, name))
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return progress, fmt.Errorf("destination FileDB: %w", statErr)
		}
		for _, sourceInfo := range sourceDirectories {
			if os.SameFile(sourceInfo, info) {
				return progress, errors.New("source and destination must be separate databases")
			}
		}
	}

	src, err := rawdb.NewFileDB(source)
	if err != nil {
		return progress, fmt.Errorf("open source: %w", err)
	}
	defer func() { err = errors.Join(err, src.Close()) }()
	dst, err := rawdb.NewFileDB(dest)
	if err != nil {
		return progress, fmt.Errorf("open destination: %w", err)
	}
	defer func() { err = errors.Join(err, dst.Close()) }()
	for batch := 0; maxBatches == 0 || batch < maxBatches; batch++ {
		progress, err = state.MigrateCommittedBinaryBatch(src, root, dst, size)
		if err != nil || progress.Complete {
			return progress, err
		}
	}
	return progress, nil
}
