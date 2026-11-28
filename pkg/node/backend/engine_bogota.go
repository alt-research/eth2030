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
// the validated payload's transaction-end state using pre-execution checks only.
// Candidate execution and unmerged byte-size exemptions are not part of it.
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
	block := bc.GetBlock(hash)
	if block == nil || !bc.HasValidatedBlock(hash) {
		return false, fmt.Errorf("inclusion list check requires validated payload %s", hash)
	}
	included := make(map[string]bool, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		raw, err := tx.EncodeRLP()
		if err != nil {
			return false, err
		}
		included[string(raw)] = true
	}
	seen := make(map[string]bool)
	var end *inclusionEndState
	var capacity *gaspool.GasPool
	for _, raw := range list {
		if included[string(raw)] || seen[string(raw)] {
			continue
		}
		seen[string(raw)] = true
		tx, err := types.DecodeBlockTransaction(raw)
		// The merged EIP-7805 explicitly ignores type 3, regardless of witnesses.
		if err != nil || tx.Type() == types.BlobTxType {
			continue
		}
		frame, canonical := tx.Inner().(*types.FrameTx8141)
		if canonical && execution.ValidateFrameContext8141(bc.Config(), block.Header(), frame) != nil {
			continue
		}
		if canonical && frame.ValidateWithExtensions(bc.Config().FrameExtensions(block.Time())) != nil {
			continue
		}
		if len(tx.BlobHashes()) > 0 {
			maximum := gas.MaxBlobsForBlock(bc.Config(), block.Time()) * gas.GasPerBlob
			used := uint64(0)
			if value := block.Header().BlobGasUsed; value != nil {
				used = *value
			}
			if used > maximum || tx.BlobGas() > maximum-used {
				continue
			}
		}
		if end == nil {
			end, err = newInclusionEndState(bc, block)
			if err != nil {
				return false, err
			}
		}
		maximum := new(gaspool.GasPool).AddGas(block.GasLimit())
		var ordinary focil.Profile1Result
		if canonical {
			// Reconstruct the accepted payload prefix when needed; never execute T.
			if end.needsReplay(types.Address{}, true) {
				if err := end.replay(); err != nil {
					return false, err
				}
			}
			preconditionErr := execution.ValidateFramePreconditions8141(bc.Config(), end.state, block.Header(), frame, maximum)
			if err := end.readError(end.state); err != nil {
				return false, err
			}
			if preconditionErr != nil {
				continue
			}
		} else {
			ordinary, err = focil.EvaluateProfile1WithCapacity(raw, bc.Config(), block.Header(), end.state, maximum)
			if err != nil {
				return false, err
			}
			if end.needsReplay(ordinary.Sender, false) {
				if err := end.replay(); err != nil {
					return false, err
				}
				ordinary, err = focil.EvaluateProfile1WithCapacity(raw, bc.Config(), block.Header(), end.state, maximum)
				if err != nil {
					return false, err
				}
			}
			if ordinary.Kind == focil.Profile1Outside {
				// Unknown future research envelopes cannot become an omission excuse.
				if !bc.Config().Rules(block.Header().Number, bc.Config().IsMerge(), block.Time()).IsIPlus {
					continue
				}
				return false, fmt.Errorf("pre-execution IL validation unavailable for transaction type %d", tx.Type())
			}
			if ordinary.Kind != focil.Profile1Enforceable {
				continue
			}
		}
		if capacity == nil {
			capacity, err = end.frameGasPool()
			if err != nil {
				return false, err
			}
		}
		executionReservation, stateReservation := ordinary.ExecutionReservation, ordinary.StateReservation
		if canonical {
			budget, err := frame.GasBudget()
			if err != nil {
				return false, err
			}
			executionReservation, stateReservation = budget.ExecutionReservation, budget.StateReservation
		}
		if !capacity.FitsFrame(executionReservation, stateReservation) {
			continue
		}
		// EIP-8504 remains a separately selected draft with no omitted-IL witness
		// composition. This gate does not execute a candidate or excuse its omission.
		if canonical && tx.HasDependencies8504() {
			return false, fmt.Errorf("EIP-8504 omitted inclusion-list dependency validity unavailable: witness-bearing IL composition is unspecified")
		}
		// Do not run candidate code, consume nonce/gas, or add a byte-size exemption.
		// EIPs PR #12723 proposes a different size rule and remains unmerged.
		return false, nil
	}
	return true, nil
}
