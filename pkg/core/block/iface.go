package block

import (
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// Validator validates block headers, bodies, and state.
type Validator interface {
	ValidateHeader(header, parent *types.Header) error
	ValidateBody(block *types.Block) error
	ValidateRequests(header *types.Header, requests types.Requests) error
	ValidateBlockAccessList(header *types.Header, computedBALHash *types.Hash) error
}

// BlockchainReader provides read-only access to the blockchain needed
// by block building. It avoids a circular dependency between block/ and chain/.
type BlockchainReader interface {
	Config() *config.ChainConfig
	CurrentBlock() *types.Block
	Genesis() *types.Block
	GetBlock(hash types.Hash) *types.Block
	StateAtBlock(block *types.Block) (state.StateDB, error)
	// GetHashFn returns a function that maps block numbers to canonical hashes,
	// used by the BLOCKHASH opcode so the builder matches the verifier.
	GetHashFn() func(uint64) types.Hash
}

// AncestorHashFn resolves BLOCKHASH against the parent branch of a block.
// The closure belongs to one execution and caches headers as it walks back.
func AncestorHashFn(parent types.Hash, number uint64, lookup func(types.Hash, uint64) *types.Header) func(uint64) types.Hash {
	head := number
	hashes := map[uint64]types.Hash{number: parent}
	return func(target uint64) types.Hash {
		if target > head || head-target >= 256 {
			return types.Hash{}
		}
		for number > target {
			header := lookup(parent, number)
			if header == nil || header.Hash() != parent {
				break
			}
			number--
			parent = header.ParentHash
			hashes[number] = parent
		}
		return hashes[target]
	}
}

// Compile-time checks.
var _ Validator = (*BlockValidator)(nil)
