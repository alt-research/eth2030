package backend

import (
	"fmt"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/types"
)

func (b *EngineBackend) IsHogota(timestamp uint64) bool {
	return b.node.Blockchain().Config().IsHogota(timestamp)
}

// CheckPayloadInclusionList examines each omitted transaction independently in
// the validated payload's own post-state. An EVM revert still constitutes
// a valid transaction if the transaction can be accepted and charged.
func (b *EngineBackend) CheckPayloadInclusionList(hash types.Hash, list [][]byte) (bool, error) {
	if err := b.beginWork(); err != nil {
		return false, err
	}
	defer b.workers.Done()
	satisfied, err := b.checkPayloadInclusionList(hash, list)
	if err != nil {
		return false, err
	}
	b.fcMu.Lock()
	defer b.fcMu.Unlock()
	if _, present := b.inclusionResults[hash]; !present {
		b.inclusionOrder = append(b.inclusionOrder, hash)
		if len(b.inclusionOrder) > 256 {
			delete(b.inclusionResults, b.inclusionOrder[0])
			b.inclusionOrder = b.inclusionOrder[1:]
		}
	}
	b.inclusionResults[hash] = satisfied
	return satisfied, nil
}

func (b *EngineBackend) PayloadInclusionListSatisfied(hash types.Hash) bool {
	b.fcMu.RLock()
	defer b.fcMu.RUnlock()
	satisfied, known := b.inclusionResults[hash]
	// A payload imported without a client-supplied view has no known list constraint.
	return !known || satisfied
}

func (b *EngineBackend) checkPayloadInclusionList(hash types.Hash, list [][]byte) (bool, error) {
	if len(list) == 0 {
		return true, nil
	}
	bc := b.node.Blockchain()
	payload := bc.GetBlock(hash)
	if payload == nil || !bc.HasValidatedBlock(hash) {
		return false, fmt.Errorf("inclusion list check requires validated payload %s", hash)
	}
	remaining := payload.GasLimit() - payload.GasUsed()
	included := make(map[types.Hash]struct{}, len(payload.Transactions()))
	for _, tx := range payload.Transactions() {
		included[tx.Hash()] = struct{}{}
	}
	var omitted []*types.Transaction
	for _, raw := range list {
		tx, err := types.DecodeTxRLP(raw)
		if err != nil || tx.Gas() > remaining {
			continue
		}
		if _, present := included[tx.Hash()]; !present {
			omitted = append(omitted, tx)
		}
	}
	if len(omitted) == 0 {
		return true, nil
	}
	postState, err := bc.StateAtBlock(payload)
	if err != nil {
		return false, fmt.Errorf("read payload post-state: %w", err)
	}
	// BLOCKHASH for an omitted transaction must read this payload's ancestry.
	// The canonical head can point to a competing branch.
	getHash := func(number uint64) types.Hash {
		if number >= payload.NumberU64() || payload.NumberU64()-number > 256 {
			return types.Hash{}
		}
		for parent := bc.GetBlock(payload.ParentHash()); parent != nil; parent = bc.GetBlock(parent.ParentHash()) {
			if parent.NumberU64() == number {
				return parent.Hash()
			}
		}
		return types.Hash{}
	}
	for _, tx := range omitted {
		trial := postState.Dup()
		trial.SetTxContext(tx.Hash(), len(payload.Transactions()))
		gas := new(gaspool.GasPool).AddGas(remaining)
		if _, _, err := execution.ApplyTransactionWithBALGetHash(
			bc.Config(), getHash, trial, payload.Header(), tx, gas, nil,
		); err == nil {
			return false, nil
		}
	}
	return true, nil
}
