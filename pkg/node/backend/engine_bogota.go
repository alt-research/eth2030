package backend

import (
	"errors"
	"fmt"
	"log/slog"

	blockpkg "github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/focil"
)

func (b *EngineBackend) IsHogota(timestamp uint64) bool {
	return b.node.Blockchain().Config().IsHogota(timestamp)
}

// ProcessBlockV6 runs the shared Amsterdam checks and retains the exact list
// for a known off-canonical payload before returning ACCEPTED.
func (b *EngineBackend) ProcessBlockV6(
	p *payload.ExecutionPayloadV5, root types.Hash, requests, list [][]byte,
) (payload.PayloadStatusV1, bool, error) {
	if err := b.beginWork(); err != nil {
		return payload.PayloadStatusV1{}, false, err
	}
	defer b.workers.Done()
	b.bogotaMu.Lock()
	defer b.bogotaMu.Unlock()

	bc := b.node.Blockchain()
	record := &chain.BogotaPayload{Number: p.BlockNumber, InclusionList: list, SatisfactionVersion: chain.BogotaSatisfactionVersion}
	status, err := b.processBlockV5(p, root, requests, func(block *types.Block) (bool, error) {
		parent := bc.GetBlock(block.ParentHash())
		if parent == nil || parent.NumberU64()+1 != block.NumberU64() {
			return false, nil
		}
		if _, err := bc.StateAtBlock(parent); err != nil {
			if errors.Is(err, chain.ErrStateNotFound) {
				return false, nil
			}
			return false, err
		}
		if finalized := bc.CurrentFinalBlock(); finalized != nil && parent.NumberU64() < finalized.NumberU64() {
			return false, nil
		}
		encoded, err := block.EncodeRLP()
		if err != nil {
			return false, err
		}
		record.Block = encoded
		return true, bc.WriteBogotaPayload(block.Hash(), record)
	})
	if err != nil || status.Status != engine.StatusValid {
		return status, false, err
	}
	record.Satisfied, err = b.checkPayloadInclusionList(p.BlockHash, list)
	if err != nil {
		return payload.PayloadStatusV1{}, false, err
	}
	return status, record.Satisfied, bc.WriteBogotaPayload(p.BlockHash, record)
}

// ForkchoiceUpdatedBogota completes an accepted payload before ordinary
// fork-choice processing and uses its retained view for this reply.
func (b *EngineBackend) ForkchoiceUpdatedBogota(
	state payload.ForkchoiceStateV1, attrs *payload.PayloadAttributesV4,
) (payload.ForkchoiceUpdatedResult, bool, error) {
	if err := b.beginWork(); err != nil {
		return payload.ForkchoiceUpdatedResult{}, false, err
	}
	defer b.workers.Done()
	b.bogotaMu.Lock()
	defer b.bogotaMu.Unlock()

	bc := b.node.Blockchain()
	head := bc.GetBlock(state.HeadBlockHash)
	if head != nil && !b.IsHogota(head.Time()) {
		result, err := b.ForkchoiceUpdatedV4(state, attrs)
		return result, true, err
	}
	// Genesis is configured locally and has no submitted inclusion-list view.
	// It must remain a usable head when Bogotá is active from genesis.
	if head != nil && head.NumberU64() == 0 && head.Hash() == bc.Genesis().Hash() {
		result, err := b.ForkchoiceUpdatedV4(state, attrs)
		return result, true, err
	}
	record, err := bc.ReadBogotaPayload(state.HeadBlockHash)
	if err != nil {
		return payload.ForkchoiceUpdatedResult{}, false, err
	}
	if record == nil {
		if head == nil || !bc.HasValidatedBlock(state.HeadBlockHash) {
			return b.requestForkchoiceSync(state), false, nil
		}
		return payload.ForkchoiceUpdatedResult{
			PayloadStatus: payload.PayloadStatusV1{Status: engine.StatusSyncing},
		}, false, nil
	}
	if record.SatisfactionVersion > chain.BogotaSatisfactionVersion {
		return payload.ForkchoiceUpdatedResult{}, false, fmt.Errorf("unsupported Bogotá satisfaction cache version %d", record.SatisfactionVersion)
	}
	if len(record.Block) > 0 {
		status, err := b.completeBogotaPayload(state.HeadBlockHash, record)
		if err != nil || status.Status != engine.StatusValid {
			if status.Status == engine.StatusSyncing {
				return b.requestForkchoiceSync(state), false, nil
			}
			return payload.ForkchoiceUpdatedResult{PayloadStatus: status}, false, err
		}
	}

	if record.SatisfactionVersion != chain.BogotaSatisfactionVersion {
		// Legacy verdicts used final-block state. A live view can be corrected;
		// a legacy compact verdict needs the CL to resubmit the exact list.
		if !record.HasInclusionListView() {
			return payload.ForkchoiceUpdatedResult{PayloadStatus: payload.PayloadStatusV1{Status: engine.StatusSyncing}}, false, nil
		}
		if head == nil || !bc.HasValidatedBlock(state.HeadBlockHash) {
			return b.requestForkchoiceSync(state), false, nil
		}
		satisfied, err := b.checkPayloadInclusionList(state.HeadBlockHash, record.InclusionList)
		if err != nil {
			return payload.ForkchoiceUpdatedResult{}, false, err
		}
		record.Satisfied, record.SatisfactionVersion = satisfied, chain.BogotaSatisfactionVersion
		if err := bc.WriteBogotaPayload(state.HeadBlockHash, record); err != nil {
			return payload.ForkchoiceUpdatedResult{}, false, err
		}
	}
	result, err := b.ForkchoiceUpdatedV4(state, attrs)
	if err == nil && result.PayloadStatus.Status == engine.StatusValid {
		if finalized := bc.CurrentFinalBlock(); finalized != nil && finalized.NumberU64() > b.bogotaFinalized {
			if err := bc.PruneBogotaPayloads(finalized.NumberU64()); err != nil {
				slog.Warn("cannot prune finalized Bogotá views", "error", err)
			} else {
				b.bogotaFinalized = finalized.NumberU64()
			}
		}
	}
	return result, record.Satisfied, err
}

func (b *EngineBackend) completeBogotaPayload(hash types.Hash, record *chain.BogotaPayload) (payload.PayloadStatusV1, error) {
	block, err := types.DecodeBlockRLP(record.Block)
	if err != nil {
		return payload.PayloadStatusV1{}, err
	}
	if block.Hash() != hash || block.NumberU64() != record.Number {
		return payload.PayloadStatusV1{}, fmt.Errorf("retained Bogotá payload does not match %s", hash)
	}
	bc := b.node.Blockchain()
	if !bc.HasValidatedBlock(hash) {
		if !bc.HasValidatedBlock(block.ParentHash()) {
			return payload.PayloadStatusV1{Status: engine.StatusSyncing}, nil
		}
		if err := bc.InsertSyncedBlock(block); err != nil {
			switch {
			case errors.Is(err, chain.ErrStateNotFound), errors.Is(err, blockpkg.ErrUnknownParent):
				return payload.PayloadStatusV1{Status: engine.StatusSyncing}, nil
			case errors.Is(err, chain.ErrStorageFailure):
				return payload.PayloadStatusV1{}, err
			default:
				parent, message := block.ParentHash(), err.Error()
				return payload.PayloadStatusV1{Status: engine.StatusInvalid, LatestValidHash: &parent, ValidationError: &message}, nil
			}
		}
		b.cacheBlobsFromBlock(block)
		var staged payload.ExecutionPayloadV3
		staged.BlockHash, staged.ParentHash, staged.BlockNumber = hash, block.ParentHash(), block.NumberU64()
		b.registerBlockInFCState(&staged)
	}
	satisfied, err := b.checkPayloadInclusionList(hash, record.InclusionList)
	if err != nil {
		return payload.PayloadStatusV1{}, err
	}
	record.Block = nil
	record.Satisfied = satisfied
	record.SatisfactionVersion = chain.BogotaSatisfactionVersion
	if err := bc.WriteBogotaPayload(hash, record); err != nil {
		return payload.PayloadStatusV1{}, err
	}
	return payload.PayloadStatusV1{Status: engine.StatusValid, LatestValidHash: &hash}, nil
}

// CheckPayloadInclusionList examines each omitted transaction independently in
// the validated payload's own transaction-end state. An EVM revert still constitutes
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
	included := make(map[string]struct{}, len(payload.Transactions()))
	for _, tx := range payload.Transactions() {
		raw, err := tx.EncodeRLP()
		if err != nil {
			return false, err
		}
		included[string(raw)] = struct{}{}
	}
	var omitted []*types.Transaction
	rawEnvelopes := make(map[*types.Transaction][]byte)
	seen := make(map[string]bool)
	for _, raw := range list {
		if _, present := included[string(raw)]; present {
			continue
		}
		if seen[string(raw)] {
			continue
		}
		seen[string(raw)] = true
		tx, err := types.DecodeBlockTransaction(raw)
		if err != nil {
			continue
		}
		if _, frame := tx.Inner().(*types.FrameTx8141); !frame && tx.Gas() > remaining {
			continue
		}
		if frame, ok := tx.Inner().(*types.FrameTx8141); ok && frame.ValidateWithExtensions(bc.Config().FrameExtensions(payload.Time())) != nil {
			continue
		}
		if len(tx.BlobHashes()) > 0 {
			maximum := gas.MaxBlobsForBlock(bc.Config(), payload.Time()) * gas.GasPerBlob
			used := uint64(0)
			if value := payload.Header().BlobGasUsed; value != nil {
				used = *value
			}
			if used > maximum || tx.BlobGas() > maximum-used {
				continue
			}
		}
		if _, present := included[string(raw)]; !present {
			// The IL API carries canonical transaction bytes, not witnesses.
			// EIP-8504 only defines witnesses for the included dependency union;
			// adding an omitted candidate's witness there would make that block
			// invalid. Neither local pool contents nor PROOF-frame execution may
			// supply a consensus validity verdict for this unspecified composition.
			if tx.HasDependencies8504() {
				return false, fmt.Errorf("EIP-8504 omitted inclusion-list dependency validity unavailable: witness-bearing IL composition is unspecified")
			}
			omitted = append(omitted, tx)
			rawEnvelopes[tx] = raw
		}
	}
	if len(omitted) == 0 {
		return true, nil
	}
	endState, err := newInclusionEndState(bc, payload)
	if err != nil {
		return false, err
	}
	getHash := endState.getHash
	sizer, err := types.NewBlockAppendSizer(payload)
	if err != nil {
		return false, err
	}
	// Once transaction-end account fields are established, Profile 1 eligibility
	// executes no candidate code. An RLP header-width boundary needs an exact probe.
	var trials []*types.Transaction
	for _, tx := range omitted {
		if _, frame := tx.Inner().(*types.FrameTx8141); frame {
			if endState.needsReplay(types.Address{}, true) {
				if err := endState.replay(); err != nil {
					return false, err
				}
			}
			trials = append(trials, tx)
			continue
		}
		result, err := focil.EvaluateProfile1(rawEnvelopes[tx], bc.Config(), payload.Header(), endState.state)
		if err != nil {
			return false, err
		}
		if endState.needsReplay(result.Sender, result.Kind == focil.Profile1Outside) {
			if err := endState.replay(); err != nil {
				return false, err
			}
			result, err = focil.EvaluateProfile1(rawEnvelopes[tx], bc.Config(), payload.Header(), endState.state)
			if err != nil {
				return false, err
			}
		}
		if result.Kind != focil.Profile1Enforceable {
			// Informational EIP-8369 cannot excuse a base EIP-7805 omission.
			// Other envelope types retain independent full-state validation.
			if result.Kind == focil.Profile1Outside {
				trials = append(trials, tx)
			}
			continue
		}
		minimum, err := sizer.Size(tx, payload.GasUsed()+result.MinimumGas)
		if err != nil {
			return false, err
		}
		if bc.Config().IsOsaka(payload.Time()) && minimum > blockpkg.MaxRLPBlockSize {
			continue
		}
		maximum, err := sizer.Size(tx, payload.GasUsed()+tx.Gas())
		if err != nil {
			return false, err
		}
		if !bc.Config().IsOsaka(payload.Time()) || maximum <= blockpkg.MaxRLPBlockSize {
			return false, nil
		}
		trials = append(trials, tx)
	}
	omitted = trials
	// A byte-boundary probe executes even ordinary sender code. It needs the
	// complete transaction-end state, not just unchanged sender account fields.
	if len(omitted) > 0 && endState.needsReplay(types.Address{}, true) {
		if err := endState.replay(); err != nil {
			return false, err
		}
	}
	var framePool *gaspool.GasPool
	for _, tx := range omitted {
		if _, canonical := tx.Inner().(*types.FrameTx8141); !canonical {
			continue
		}
		receipts := bc.GetReceipts(hash)
		if endState.replayed {
			receipts = endState.receipts
		}
		replay := len(receipts) != len(payload.Transactions())
		for _, receipt := range receipts {
			if (receipt.FrameResults8141 != nil || bc.Config().IsGlamsterdan(payload.Time())) && receipt.BlockGasUsed == 0 {
				replay = true
			}
		}
		if replay {
			if err := endState.replay(); err != nil {
				return false, err
			}
			receipts = endState.receipts
		}
		var executionUsed, stateUsed uint64
		for _, receipt := range receipts {
			if (receipt.FrameResults8141 != nil || bc.Config().IsGlamsterdan(payload.Time())) && receipt.BlockGasUsed != 0 {
				executionUsed += receipt.BlockGasUsed
			} else {
				executionUsed += receipt.GasUsed
			}
			for _, frame := range receipt.FrameResults8141 {
				stateUsed += frame.GasUsed.State
			}
		}
		framePool = new(gaspool.GasPool).AddGas(payload.GasLimit())
		if err := framePool.ReserveFrame(executionUsed, stateUsed); err != nil {
			return false, err
		}
		break
	}
	for _, tx := range omitted {
		trial := endState.state.Dup()
		trial.SetTxContext(tx.Hash(), len(payload.Transactions()))
		gas := new(gaspool.GasPool).AddGas(remaining)
		if _, canonical := tx.Inner().(*types.FrameTx8141); canonical {
			copied := *framePool
			gas = &copied
		}
		receipt, _, applyErr := execution.ApplyTransactionWithBALGetHash(
			bc.Config(), getHash, trial, payload.Header(), tx, gas, nil,
		)
		if err := state.ReadError(trial); err != nil {
			return false, err
		}
		if applyErr == nil {
			used := receipt.GasUsed
			if receipt.BlockGasUsed > 0 {
				used = receipt.BlockGasUsed
			}
			size, err := sizer.Size(tx, payload.GasUsed()+used)
			if err != nil {
				return false, err
			}
			if !bc.Config().IsOsaka(payload.Time()) || size <= blockpkg.MaxRLPBlockSize {
				return false, nil
			}
		}
	}
	return true, nil
}
