package backend

import (
	"fmt"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
)

// inclusionEndState separates EIP-7805's transaction-end state from the final
// block state. Only execution-validated BAL metadata can prove an account's
// fields were untouched by the later withdrawals/request phase. Missing BAL
// data requires reconstruction, never an omission excuse.
type inclusionEndState struct {
	bc                       *chain.Blockchain
	block                    *types.Block
	state                    state.StateDB
	getHash                  vm.GetHashFunc
	known, changed, replayed bool
	accounts                 map[types.Address]bool
	receipts                 []*types.Receipt
	ancestryErr              error
}

func newInclusionEndState(bc *chain.Blockchain, block *types.Block) (*inclusionEndState, error) {
	db, err := bc.StateAtBlock(block)
	if err != nil {
		return nil, fmt.Errorf("read payload post-state: %w", err)
	}
	e := &inclusionEndState{bc: bc, block: block, state: db, accounts: make(map[types.Address]bool)}
	// BLOCKHASH must follow the submitted payload, including off-canonical branches.
	// Each link authenticates its parent's hash; the queried header itself is
	// unnecessary. Cache the bounded window so repeated opcodes do not reread it.
	hashes := make(map[uint64]types.Hash)
	oldest := uint64(0)
	if block.NumberU64() > 0 {
		oldest = block.NumberU64() - 1
		hashes[oldest] = block.ParentHash()
	}
	e.getHash = func(number uint64) types.Hash {
		if number >= block.NumberU64() || block.NumberU64()-number > 256 {
			return types.Hash{}
		}
		if hash, ok := hashes[number]; ok {
			return hash
		}
		for oldest > number {
			hash := hashes[oldest]
			header := bc.GetHeaderByHash(hash)
			if header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != oldest {
				e.ancestryErr = fmt.Errorf("authenticated BLOCKHASH ancestor %s at %d unavailable", hash, oldest)
				return types.Hash{}
			}
			oldest--
			hashes[oldest] = header.ParentHash
		}
		return hashes[number]
	}
	encoded, err := bc.BlockAccessList(block.Hash())
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 {
		return e, nil
	}
	accounts, err := bal.DecodeForHeader(encoded, block.Header())
	if err != nil {
		return nil, err
	}
	e.known = true
	phase := uint64(len(block.Transactions())) + 1
	for _, account := range accounts {
		if hasInclusionPhase(account.BalanceChanges, phase) || hasInclusionPhase(account.NonceChanges, phase) || hasInclusionPhase(account.CodeChanges, phase) {
			e.accounts[account.Address], e.changed = true, true
		}
		for _, slot := range account.StorageChanges {
			if hasInclusionPhase(slot.Changes, phase) {
				e.changed = true
			}
		}
	}
	return e, nil
}
func hasInclusionPhase[T any](changes []bal.IndexedChange[T], phase uint64) bool {
	for _, change := range changes {
		if change.Index == phase {
			return true
		}
	}
	return false
}
func (e *inclusionEndState) needsReplay(sender types.Address, fullState bool) bool {
	return !e.replayed && (!e.known || e.accounts[sender] || fullState && e.changed)
}
func (e *inclusionEndState) replay() error {
	if e.replayed {
		return nil
	}
	parent := e.bc.GetBlock(e.block.ParentHash())
	if parent == nil {
		return fmt.Errorf("missing parent for IL transaction-end state")
	}
	before, err := e.bc.StateAtBlock(parent)
	if err != nil {
		return err
	}
	before, err = execution.PrepareBlockState(e.bc.Config(), e.block.Time(), before)
	if err != nil {
		return err
	}
	processor := execution.NewStateProcessor(e.bc.Config()).WithGetHash(e.getHash)
	result, err := processor.ProcessTransactionsWithBAL(e.block, before)
	if readErr := e.readError(before); readErr != nil {
		return readErr
	}
	if err != nil {
		return fmt.Errorf("reconstruct IL transaction-end state: %w", err)
	}
	e.state, e.receipts, e.replayed = before, result.Receipts, true
	return nil
}

// A missing authenticated header must not turn a real BLOCKHASH into zero and
// make a Frame look invalid. EVM's callback cannot return errors directly.
func (e *inclusionEndState) readError(db state.StateDB) error {
	if err := state.ReadError(db); err != nil {
		return err
	}
	return e.ancestryErr
}
