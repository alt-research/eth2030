// Package rpcbackend defines the Backend interface and related service
// accessors used by the Ethereum JSON-RPC layer.
package rpcbackend

import (
	"errors"
	"math/big"

	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/eth2030/eth2030/trie"
)

// StateAtHeader opens the state of a specific block when the backend retains
// block-scoped state. A private overlay root alone does not identify its
// frozen pre-state or the conversion cursor.
func StateAtHeader(backend Backend, header *types.Header) (state.StateDB, error) {
	if byHeader, ok := backend.(interface {
		StateAtHeader(*types.Header) (state.StateDB, error)
	}); ok {
		return byHeader.StateAtHeader(header)
	}
	return backend.StateAt(header.Root)
}

// Backend provides access to chain data for the JSON-RPC API.
// This interface decouples the RPC layer from the chain implementation,
// following go-ethereum's ethapi.Backend pattern.
type Backend interface {
	// Chain data
	HeaderByNumber(number rpctypes.BlockNumber) *types.Header
	HeaderByHash(hash types.Hash) *types.Header
	BlockByNumber(number rpctypes.BlockNumber) *types.Block
	BlockByHash(hash types.Hash) *types.Block
	CurrentHeader() *types.Header
	ChainID() *big.Int

	// State access
	StateAt(root types.Hash) (state.StateDB, error)

	// Transaction pool
	SendTransaction(tx *types.Transaction) error
	GetTransaction(hash types.Hash) (*types.Transaction, uint64, uint64) // tx, blockNum, index

	// Gas estimation
	SuggestGasPrice() *big.Int

	// Receipts and logs
	GetReceipts(blockHash types.Hash) []*types.Receipt
	GetLogs(blockHash types.Hash) []*types.Log
	GetBlockReceipts(number uint64) []*types.Receipt

	// Proofs
	GetProof(addr types.Address, storageKeys []types.Hash, blockNumber rpctypes.BlockNumber) (*trie.AccountProof, error)

	// EVM execution
	EVMCall(from types.Address, to *types.Address, data []byte, gas uint64, value *big.Int, blockNumber rpctypes.BlockNumber) ([]byte, uint64, error)

	// Tracing
	TraceTransaction(txHash types.Hash) (*vm.StructLogTracer, error)

	// History availability (EIP-4444)
	// HistoryOldestBlock returns the oldest block number for which bodies
	// and receipts are still available. Returns 0 if no pruning occurred.
	HistoryOldestBlock() uint64

	// Blob schedule
	// BlobSchedule returns the target, max blobs per block and base fee update
	// fraction for the given block timestamp. Used by eth_blobBaseFee and
	// eth_feeHistory to compute correct per-fork blob fees and ratios.
	BlobSchedule(blockTime uint64) (target, max uint64, updateFraction uint64)
	// IsOsaka selects EIP-7918's excess update for blob fee forecasts.
	IsOsaka(blockTime uint64) bool
}

// AccessListResult is transaction simulation output, independent of wire encoding.
type AccessListResult struct {
	AccessList     types.AccessList
	GasUsed        uint64
	ExecutionError string
}

// AccessListBackend provides actual access discovery. Backends without this
// capability must report unsupported instead of claiming an empty discovered list.
type AccessListBackend interface {
	CreateAccessList(args rpctypes.CallArgs, blockNumber rpctypes.BlockNumber) (*AccessListResult, error)
}

// ErrInvalidCallArgs identifies malformed simulation input at the RPC boundary.
var ErrInvalidCallArgs = errors.New("invalid call arguments")

// CallBackend exposes actual method-specific simulation without narrowing input
// to the legacy EVMCall tuple. A transaction estimate always uses transaction rules.
type CallBackend interface {
	Call(args rpctypes.CallArgs, number rpctypes.BlockNumber, overrides ...rpctypes.StateOverride) ([]byte, uint64, error)
	EstimateGas(args rpctypes.CallArgs, number rpctypes.BlockNumber, overrides ...rpctypes.StateOverride) (uint64, error)
	TraceCall(args rpctypes.CallArgs, number rpctypes.BlockNumber, overrides ...rpctypes.StateOverride) (*vm.StructLogTracer, error)
}

// HeaderCallBackend keeps execution on one resolved header, including orphan branches.
type HeaderCallBackend interface {
	CallAtHeader(args rpctypes.CallArgs, header *types.Header, overrides ...rpctypes.StateOverride) ([]byte, uint64, error)
	EstimateGasAtHeader(args rpctypes.CallArgs, header *types.Header, overrides ...rpctypes.StateOverride) (uint64, error)
	TraceCallAtHeader(args rpctypes.CallArgs, header *types.Header, overrides ...rpctypes.StateOverride) (*vm.StructLogTracer, error)
}
type HeaderAccessListBackend interface {
	CreateAccessListAtHeader(args rpctypes.CallArgs, header *types.Header) (*AccessListResult, error)
}
type HeaderProofBackend interface {
	GetProofAtHeader(addr types.Address, storageKeys []types.Hash, header *types.Header) (*trie.AccountProof, error)
}

// BlockTraceBackend supplies canonical single-pass opcode traces.
type BlockTraceBackend interface {
	TraceBlock(hash types.Hash) ([]*vm.StructLogTracer, error)
}

// SimulateBackend executes isolated, sequential blocks using actual state.
type SimulateBackend interface {
	Simulate(options rpctypes.SimulateOptions, base *types.Header) ([]rpctypes.SimulateBlockResult, error)
}

// ExecutionTraceBackend returns observations from actual replay, with one pass
// for a block and complete prefix replay for a selected transaction.
type ExecutionTraceBackend interface {
	TraceCallDetailed(args rpctypes.CallArgs, header *types.Header, options rpctypes.TraceOptions) (vm.EVMLogger, error)
	TraceTransactionDetailed(hash types.Hash, options rpctypes.TraceOptions) (vm.EVMLogger, error)
	TraceBlockDetailed(hash types.Hash, options rpctypes.TraceOptions) ([]vm.EVMLogger, error)
}
