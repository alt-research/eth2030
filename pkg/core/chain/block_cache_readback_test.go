package chain

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

func TestBlockCacheSeedingKeepsBoundedEvictionIndex(t *testing.T) {
	bc, _ := testChain(t)
	bc.opts.BlockCacheSize = 1
	for number := int64(1); number <= 16; number++ {
		block := types.NewBlock(&types.Header{Number: big.NewInt(number), Difficulty: new(big.Int), GasLimit: 30000000}, &types.Body{})
		bc.StoreBlockInCache(block)
		bc.StoreBlockInCache(block)
		if len(bc.blockCache) != 1 || len(bc.blockCacheOrder) != 1 || bc.blockCacheOrder[0] != block.Hash() || bc.blockCache[block.Hash()] != block {
			t.Fatalf("unbounded or unindexed cached bodies at %d: map=%d/order=%d", number, len(bc.blockCache), len(bc.blockCacheOrder))
		}
	}
}
