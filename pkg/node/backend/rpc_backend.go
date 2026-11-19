package backend

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/eth2030/eth2030/core/execution"
	coregas "github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rpc"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/eth2030/eth2030/trie"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RPCBackend adapts NodeDeps to the rpc.Backend interface.
type RPCBackend struct {
	ctx    context.Context
	node   NodeDeps
	engine *EngineBackend
}

// NewRPCBackend creates a new RPCBackend.
func NewRPCBackend(node NodeDeps, engine *EngineBackend) rpc.Backend {
	return &RPCBackend{node: node, engine: engine}
}

func (b *RPCBackend) SyncProgress() (bool, uint64, uint64, uint64) {
	if source, ok := b.node.(interface {
		SyncProgress() (bool, uint64, uint64, uint64)
	}); ok {
		return source.SyncProgress()
	}
	return false, 0, 0, 0
}

func (b *RPCBackend) HeaderByNumber(number rpc.BlockNumber) *types.Header {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}

	switch number {
	case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
		blk := bc.CurrentBlock()
		if blk != nil {
			return blk.Header()
		}
		return nil
	case rpc.EarliestBlockNumber:
		blk := bc.GetBlockByNumber(0)
		if blk != nil {
			return blk.Header()
		}
		return nil
	case rpc.FinalizedBlockNumber:
		if blk := bc.CurrentFinalBlock(); blk != nil {
			return blk.Header()
		}
		return nil
	case rpc.SafeBlockNumber:
		if blk := bc.CurrentSafeBlock(); blk != nil {
			return blk.Header()
		}
		return nil
	default:
		blk := bc.GetBlockByNumber(uint64(number))
		if blk != nil {
			return blk.Header()
		}
		return nil
	}
}

func (b *RPCBackend) HeaderByHash(hash types.Hash) *types.Header {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	blk := bc.GetBlock(hash)
	if blk != nil {
		return blk.Header()
	}
	return nil
}

func (b *RPCBackend) BlockByNumber(number rpc.BlockNumber) *types.Block {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}

	switch number {
	case rpc.LatestBlockNumber, rpc.PendingBlockNumber:
		return bc.CurrentBlock()
	case rpc.EarliestBlockNumber:
		return bc.GetBlockByNumber(0)
	case rpc.FinalizedBlockNumber:
		if b.engine != nil {
			if h := b.engine.GetFinalizedHash(); h != (types.Hash{}) {
				return bc.GetBlock(h)
			}
		}
		return nil
	case rpc.SafeBlockNumber:
		if b.engine != nil {
			if h := b.engine.GetSafeHash(); h != (types.Hash{}) {
				return bc.GetBlock(h)
			}
		}
		return nil
	default:
		return bc.GetBlockByNumber(uint64(number))
	}
}

func (b *RPCBackend) BlockByHash(hash types.Hash) *types.Block {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	return bc.GetBlock(hash)
}

func (b *RPCBackend) CurrentHeader() *types.Header {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	blk := bc.CurrentBlock()
	if blk != nil {
		return blk.Header()
	}
	return nil
}

func (b *RPCBackend) ChainID() *big.Int {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	return bc.Config().ChainID
}

func (b *RPCBackend) StateAt(root types.Hash) (state.StateDB, error) {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil, fmt.Errorf("blockchain not available")
	}
	return bc.StateAtRoot(root)
}

func (b *RPCBackend) StateAtHeader(header *types.Header) (state.StateDB, error) {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil, fmt.Errorf("blockchain not available")
	}
	if header == nil {
		return nil, fmt.Errorf("block header not available")
	}
	hash := header.Hash()
	blk := bc.GetBlock(hash)
	if blk == nil {
		return nil, fmt.Errorf("block %s not found", hash)
	}
	ctx, cancel := context.WithTimeout(b.executionContext(), 5*time.Second)
	defer cancel()
	return bc.StateAtBlockContext(ctx, blk)
}

func (b *RPCBackend) GetProof(addr types.Address, storageKeys []types.Hash, blockNumber rpc.BlockNumber) (*trie.AccountProof, error) {
	header := b.HeaderByNumber(blockNumber)
	if header == nil {
		return nil, fmt.Errorf("block not found")
	}

	return b.GetProofAtHeader(addr, storageKeys, header)
}
func (b *RPCBackend) GetProofAtHeader(addr types.Address, storageKeys []types.Hash, header *types.Header) (*trie.AccountProof, error) {
	statedb, err := b.StateAtHeader(header)
	if err != nil {
		return nil, err
	}

	snapshot, err := state.ExportStateSnapshot(statedb)
	if err != nil {
		return nil, err
	}
	if snapshot.Root() != header.Root {
		return nil, fmt.Errorf("proof state root differs from requested header")
	}
	stateTrie, storageTrie := trie.New(), trie.New()
	for key, value := range snapshot.Accounts {
		if err := stateTrie.Put(key[:], value); err != nil {
			return nil, err
		}
	}
	accountHash := crypto.Keccak256Hash(addr[:])
	for key, value := range snapshot.Storage[accountHash] {
		if err := storageTrie.Put(key[:], value); err != nil {
			return nil, err
		}
	}

	return trie.ProveAccountWithStorage(stateTrie, addr, storageTrie, storageKeys)
}

func (b *RPCBackend) SendTransaction(tx *types.Transaction) error {
	pool := b.node.TxPool()
	if pool == nil {
		return fmt.Errorf("transaction pool not available")
	}

	if err := pool.AddLocal(tx); err != nil {
		return err
	}

	// Broadcast via eth handler
	if h := b.node.EthHandler(); h != nil {
		h.BroadcastTransactions([]*types.Transaction{tx})
	}

	// Journal transaction
	if journal := b.node.TxJournal(); journal != nil {
		if jerr := journal.Insert(tx, true); jerr != nil {
			slog.Debug("tx journal insert failed", "hash", tx.Hash(), "err", jerr)
		}
	}

	slog.Debug("transaction sent", "hash", tx.Hash())
	return nil
}

func (b *RPCBackend) GetTransaction(hash types.Hash) (*types.Transaction, uint64, uint64) {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil, 0, 0
	}

	// Check blockchain first
	if entry, found := bc.GetTxLookupEntry(hash); found {
		blk := bc.GetBlock(entry.BlockHash)
		if blk != nil {
			txs := blk.Transactions()
			if int(entry.TxIndex) < len(txs) {
				return txs[entry.TxIndex], entry.BlockNumber, entry.TxIndex
			}
		}
	}

	// Check tx pool
	pool := b.node.TxPool()
	if pool != nil {
		tx := pool.Get(hash)
		if tx != nil {
			return tx, 0, 0
		}
	}

	return nil, 0, 0
}

func (b *RPCBackend) SuggestGasPrice() *big.Int {
	if oracle := b.node.GasOracle(); oracle != nil {
		if oracle.BaseFee().Sign() > 0 {
			return oracle.SuggestGasPrice()
		}
	}

	bc := b.node.Blockchain()
	if bc != nil {
		blk := bc.CurrentBlock()
		if blk != nil && blk.Header().BaseFee != nil {
			return new(big.Int).Set(blk.Header().BaseFee)
		}
	}
	return big.NewInt(1_000_000_000)
}

func (b *RPCBackend) GetReceipts(blockHash types.Hash) []*types.Receipt {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	return bc.GetReceipts(blockHash)
}

func (b *RPCBackend) GetLogs(blockHash types.Hash) []*types.Log {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	return bc.GetLogs(blockHash)
}

func (b *RPCBackend) GetBlockReceipts(number uint64) []*types.Receipt {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil
	}
	return bc.GetBlockReceipts(number)
}

func (b *RPCBackend) EVMCall(from types.Address, to *types.Address, data []byte, gas uint64, value *big.Int, number rpc.BlockNumber) ([]byte, uint64, error) {
	fromHex := from.Hex()
	input := hexutil.Encode(data)
	args := rpctypes.CallArgs{From: &fromHex, Input: &input}
	if to != nil {
		v := to.Hex()
		args.To = &v
	}
	if gas == 0 {
		gas = 50_000_000
	}
	limit := hexutil.EncodeUint64(gas)
	args.Gas = &limit
	if value != nil {
		v := hexutil.EncodeBig(value)
		args.Value = &v
	}
	output, used, err := b.Call(args, number)
	if used > gas {
		return output, 0, fmt.Errorf("call exceeded supplied gas")
	}
	return output, gas - used, err
}

func (b *RPCBackend) HistoryOldestBlock() uint64 {
	bc := b.node.Blockchain()
	if bc == nil {
		return 0
	}
	return bc.HistoryOldestBlock()
}

func (b *RPCBackend) BlobSchedule(blockTime uint64) (target, max, updateFraction uint64) {
	bc := b.node.Blockchain()
	if bc == nil {
		return 0, 0, 0
	}
	chainCfg := bc.Config()
	sched := coregas.GetBlobSchedule(chainCfg, blockTime)
	return sched.Target, sched.Max, sched.UpdateFraction
}

func (b *RPCBackend) TraceTransaction(hash types.Hash) (*vm.StructLogTracer, error) {
	bc := b.node.Blockchain()
	if bc == nil {
		return nil, fmt.Errorf("blockchain not available")
	}
	entry, ok := bc.GetTxLookupEntry(hash)
	if !ok {
		return nil, fmt.Errorf("transaction %s not found", hash)
	}
	block := bc.GetBlock(entry.BlockHash)
	if block == nil {
		return nil, fmt.Errorf("block not found")
	}
	traces, err := b.traceTransactions(block, &entry.TxIndex)
	if err != nil {
		return nil, err
	}
	return traces[0], nil
}

// TraceBlock replays its prefix once instead of independently rebuilding it for
// every target. Each tracer observes its own complete transaction transition.
func (b *RPCBackend) TraceBlock(hash types.Hash) ([]*vm.StructLogTracer, error) {
	block := b.node.Blockchain().GetBlock(hash)
	if block == nil {
		return nil, fmt.Errorf("block not found")
	}
	return b.traceTransactions(block, nil)
}
func (b *RPCBackend) traceTransactions(block *types.Block, target *uint64) ([]*vm.StructLogTracer, error) {
	bc := b.node.Blockchain()
	txs := block.Transactions()
	if len(txs) == 0 {
		return []*vm.StructLogTracer{}, nil
	}
	last := uint64(len(txs) - 1)
	if target != nil {
		last = *target
	}
	if last >= uint64(len(txs)) {
		return nil, fmt.Errorf("transaction index out of range")
	}
	header := block.Header()
	parent := bc.GetBlock(header.ParentHash)
	if parent == nil {
		return nil, fmt.Errorf("parent block not found")
	}
	base, err := b.StateAtHeader(parent.Header())
	if err != nil {
		return nil, err
	}
	db := base.Dup()
	if err := execution.ApplyPreExecutionSystems(bc.Config(), bc.GetHashFn(), db, header, nil); err != nil {
		return nil, err
	}
	gp := new(gaspool.GasPool).AddGas(header.GasLimit)
	var traces []*vm.StructLogTracer
	for i := uint64(0); i <= last; i++ {
		tx := txs[i]
		db.SetTxContext(tx.Hash(), int(i))
		db.ClearTransientStorage()
		db.SubRefund(db.GetRefund())
		tracing := vm.Config{}
		var tracer *vm.StructLogTracer
		if target == nil || i == last {
			if tx.Type() > types.SetCodeTxType {
				return nil, fmt.Errorf("transaction envelope has no qualified opcode tracer")
			}
			tracer = vm.NewStructLogTracer()
			tracing = vm.Config{Debug: true, Tracer: tracer}
		}
		receipt, used, err := execution.ApplyTransactionWithVMConfig(bc.Config(), bc.GetHashFn(), db, header, tx, gp, tracing)
		if err != nil {
			return nil, fmt.Errorf("replay tx %d: %w", i, err)
		}
		execution.ApplyAAPostExecution(db, tx, receipt, used, nil)
		if tracer != nil {
			traces = append(traces, tracer)
		}
	}
	return traces, nil
}

// IsOsaka supplies the fork rule used by blob fee forecasts.
func (b *RPCBackend) IsOsaka(timestamp uint64) bool {
	return b.node.Blockchain().Config().IsOsaka(timestamp)
}

// WithContext returns a request-scoped backend; Node and Engine ownership remain shared.
func (b *RPCBackend) WithContext(ctx context.Context) rpc.Backend {
	return &RPCBackend{node: b.node, engine: b.engine, ctx: ctx}
}

func (b *RPCBackend) executionContext() context.Context {
	if b.ctx != nil {
		return b.ctx
	}
	return context.Background()
}
