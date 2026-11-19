// Command eth2030-state-history builds verified immutable history for an existing
// local database. It never starts network, validator or RPC services.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/node"
)

func build(ctx context.Context, datadir, genesis string) (chain.StateHistoryResult, error) {
	if err := ctx.Err(); err != nil {
		return chain.StateHistoryResult{}, err
	}
	if datadir == "" || genesis == "" {
		return chain.StateHistoryResult{}, fmt.Errorf("existing datadir and original genesis path are required")
	}
	info, err := os.Stat(filepath.Join(datadir, "chaindata", "LOCK"))
	if err != nil || !info.Mode().IsRegular() {
		return chain.StateHistoryResult{}, fmt.Errorf("existing native database lock file is unavailable")
	}
	if info, err := os.Stat(genesis); err != nil || !info.Mode().IsRegular() {
		return chain.StateHistoryResult{}, fmt.Errorf("original genesis file is unavailable")
	}
	return node.BuildStateHistory(ctx, datadir, genesis)
}
func main() {
	datadir := flag.String("datadir", "", "existing exclusive native data directory")
	genesis := flag.String("genesis", "", "original chain genesis JSON")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	result, err := build(ctx, *datadir, *genesis)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
