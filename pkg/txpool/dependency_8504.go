package txpool

import "github.com/eth2030/eth2030/core/types"

const (
	MaxProxyDependents8298            = 8 // Local policy, not an EIP gas/consensus constant.
	MaxDependencyPoolBytes8504 uint64 = 64 * 1024 * 1024
)

// DependencyWrapper8504 serves only the single wrapper actually admitted for
// its transaction. Eviction, replacement and reorg removal share lookup.Remove.
func (pool *TxPool) DependencyWrapper8504(hash types.Hash) []byte {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	tx := pool.lookup.Get(pool.lookup.dependencyWrappers[hash])
	if tx == nil {
		return nil
	}
	raw, err := types.EncodeDependencyWrapper8504(tx)
	if err != nil {
		return nil
	}
	return raw
}
func (pool *TxPool) DependencyBytes8504() uint64 {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	return pool.lookup.dependencyBytes
}
