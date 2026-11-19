package node

import (
	"context"
	"fmt"
	"os"

	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
)

// BuildStateHistory opens the original database exclusively for cancellable
// maintenance. It deliberately avoids Node.New: normal node startup replays
// persisted blocks before returning and has no request cancellation lifetime.
func BuildStateHistory(ctx context.Context, datadir, genesisPath string) (chain.StateHistoryResult, error) {
	if err := ctx.Err(); err != nil {
		return chain.StateHistoryResult{}, err
	}
	config := DefaultConfig()
	config.DataDir, config.GenesisPath = datadir, genesisPath
	genesis, err := loadGenesisFile(&config)
	if err != nil {
		return chain.StateHistoryResult{}, err
	}
	if genesis.Config.BinaryOverlayTime != nil {
		return chain.StateHistoryResult{}, fmt.Errorf("binary overlay history uses its existing checkpoint format")
	}
	db, err := rawdb.NewFileDB(config.ResolvePath("chaindata"))
	if err != nil {
		return chain.StateHistoryResult{}, err
	}
	defer db.Close()
	if err := checkPersistedChainConfig(db, genesis.Config); err != nil {
		return chain.StateHistoryResult{}, err
	}
	initial := state.NewMemoryStateDB()
	block := genesis.SetupGenesisBlock(initial)
	bc, err := chain.NewBlockchain(genesis.Config, block, initial, db)
	if err != nil {
		return chain.StateHistoryResult{}, err
	}
	ancientPath := config.ResolvePath("ancient")
	if _, err := os.Stat(ancientPath); err == nil {
		ancient, err := rawdb.NewAncientStore(rawdb.AncientStoreConfig{DataDir: ancientPath})
		if err != nil {
			return chain.StateHistoryResult{}, err
		}
		defer ancient.Close()
		bc.SetAncientStore(ancient)
	} else if !os.IsNotExist(err) {
		return chain.StateHistoryResult{}, err
	}
	return bc.BuildTrieHistory(ctx)
}
