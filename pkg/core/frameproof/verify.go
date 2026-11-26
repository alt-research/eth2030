// Package frameproof verifies EIP-8504 dependencies outside EVM execution.
package frameproof

import (
	"errors"
	"sync"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/crypto/leansphincs"
)

// Cache stores only successful stateless verification identities. Its fixed
// capacity bounds memory and eviction never changes correctness.
type Cache struct {
	mu       sync.Mutex
	entries  map[types.Hash]struct{}
	order    []types.Hash
	capacity int
}

func NewCache(capacity int) *Cache {
	if capacity < 1 {
		capacity = 1
	}
	return &Cache{entries: make(map[types.Hash]struct{}), capacity: capacity}
}

var publicCache = NewCache(1024)

func (c *Cache) Verify(deps []types.Dependency8504, witnesses [][]byte) error {
	if err := types.ValidateDependencyWitnessShape8504(deps, witnesses, false); err != nil {
		return err
	}
	for i, d := range deps {
		key := crypto.Keccak256Hash(d.Bytes(), witnesses[i])
		c.mu.Lock()
		_, known := c.entries[key]
		c.mu.Unlock()
		if known {
			continue
		}
		if !leansphincs.Verify(d.VerificationKeyHash[:], d.DataHash[:], witnesses[i]) {
			return errors.New("invalid EIP-8504 leanSPHINCS witness")
		}
		c.mu.Lock()
		if _, exists := c.entries[key]; !exists {
			if len(c.order) == c.capacity {
				delete(c.entries, c.order[0])
				c.order = c.order[1:]
			}
			c.entries[key] = struct{}{}
			c.order = append(c.order, key)
		}
		c.mu.Unlock()
	}
	return nil
}

func VerifyTransaction(tx *types.Transaction, extensions types.FrameExtensions, public bool) error {
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok {
		return nil
	}
	if err := frame.ValidateWithExtensions(extensions); err != nil {
		return err
	}
	deps, err := frame.Dependencies8504()
	if err != nil {
		return err
	}
	if len(deps) == 0 {
		if len(tx.DependencyWitnesses8504()) != 0 {
			return errors.New("unexpected dependency witnesses")
		}
		return nil
	}
	w := tx.DependencyWitnesses8504()
	if err := types.ValidateDependencyWitnessShape8504(deps, w, public); err != nil {
		return err
	}
	return publicCache.Verify(deps, w)
}

// VerifyBlock authenticates the exact canonical union and witness commitment
// before execution or system calls. Proof-frame success alone proves nothing.
func VerifyBlock(block *types.Block, active bool, extensions types.FrameExtensions) error {
	h := block.Header()
	w := block.DependencyWitnesses8504()
	if !active {
		if h.DependencyWitnessesHash != nil || w != nil {
			return errors.New("dependency block fields before EIP-8504")
		}
		return nil
	}
	if h.DependencyWitnessesHash == nil || w == nil {
		return errors.New("missing active EIP-8504 block fields")
	}
	if uint64(len(w)) > h.GasLimit/types.DependencyWitnessGas8504 {
		return errors.New("dependency witnesses exceed block gas bound")
	}
	for _, tx := range block.Transactions() {
		if f, ok := tx.Inner().(*types.FrameTx8141); ok {
			if err := f.ValidateWithExtensions(extensions); err != nil {
				return err
			}
		}
	}
	if types.DependencyWitnessesHash8504(w) != *h.DependencyWitnessesHash {
		return errors.New("dependency witness commitment mismatch")
	}
	deps, err := types.BlockDependencies8504(block.Transactions())
	if err != nil {
		return err
	}
	return publicCache.Verify(deps, w)
}

func Collect(transactions []*types.Transaction) ([][]byte, error) {
	byDependency := make(map[types.Dependency8504][]byte)
	for _, tx := range transactions {
		if !tx.HasDependencies8504() {
			continue
		}
		deps, err := tx.Inner().(*types.FrameTx8141).Dependencies8504()
		if err != nil {
			return nil, err
		}
		w := tx.DependencyWitnesses8504()
		if err := types.ValidateDependencyWitnessShape8504(deps, w, false); err != nil {
			return nil, err
		}
		for i, d := range deps {
			if _, known := byDependency[d]; !known {
				byDependency[d] = w[i]
			}
		}
	}
	deps, err := types.BlockDependencies8504(transactions)
	if err != nil {
		return nil, err
	}
	w := make([][]byte, len(deps))
	for i, d := range deps {
		w[i] = byDependency[d]
	}
	if err := publicCache.Verify(deps, w); err != nil {
		return nil, err
	}
	return w, nil
}
