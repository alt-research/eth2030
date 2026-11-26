package statesync

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

type BALStep struct {
	Header  *types.Header
	Encoded []byte
}

// PivotUpdate is resolved against the CL-selected ancestry before state use.
// Peer must be a fresh proof-verified adapter bound to Header.Root.
type PivotUpdate struct {
	Common   *types.Header
	Header   *types.Header
	Rollback []BALStep
	Forward  []BALStep
	Peer     SnapPeer
}

type PivotResolver func(context.Context, *types.Header) (*PivotUpdate, error)

// PivotRestart requests a fresh bulk attempt when authenticated orphan/BAL data
// required for a safe rebase is unavailable. It does not authorize old state reuse.
type PivotRestart struct{ Header *types.Header }

func (e *PivotRestart) Error() string { return "snap/2 pivot requires a fresh state download" }

func decodeBALPath(common, end *types.Header, steps []BALStep) ([][]bal.AccountChanges, error) {
	if common == nil || end == nil || common.Number == nil || end.Number == nil {
		return nil, fmt.Errorf("snap/2 invalid pivot path")
	}
	previous := common
	decoded := make([][]bal.AccountChanges, 0, len(steps))
	for _, step := range steps {
		if step.Header == nil || step.Header.Number == nil || !step.Header.Number.IsUint64() || !previous.Number.IsUint64() || step.Header.Number.Uint64() == 0 || step.Header.Number.Uint64()-1 != previous.Number.Uint64() || step.Header.ParentHash != previous.Hash() {
			return nil, fmt.Errorf("snap/2 non-contiguous BAL path")
		}
		accounts, err := bal.DecodeForHeader(step.Encoded, step.Header)
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, accounts)
		previous = step.Header
	}
	if previous.Hash() != end.Hash() {
		return nil, fmt.Errorf("snap/2 BAL path does not reach pivot")
	}
	return decoded, nil
}

func snapshotAccountBytes(account AccountData) ([]byte, error) {
	if account.Balance == nil || account.Balance.Sign() < 0 || account.Balance.BitLen() > 256 {
		return nil, fmt.Errorf("snap/2 invalid verified account")
	}
	return rlp.EncodeToBytes([]any{account.Nonce, account.Balance, account.Root, account.CodeHash})
}

// rebase authenticates all BALs before touching the staged snapshot. Account
// point proofs restore fields absent from diffs, including account nonexistence.
// Old-only storage keys are re-fetched rather than guessed or left on the fork.
func (w *SnapshotWriter) rebase(old *types.Header, update *PivotUpdate) error {
	if update == nil || update.Peer == nil {
		return fmt.Errorf("snap/2 missing pivot update peer")
	}
	rollback, err := decodeBALPath(update.Common, old, update.Rollback)
	if err != nil {
		return err
	}
	forward, err := decodeBALPath(update.Common, update.Header, update.Forward)
	if err != nil {
		return err
	}
	staged := w.snapshot.Copy()
	touched := make(map[types.Hash]bool)
	oldSlots := make(map[types.Hash]map[types.Hash]bool)
	writes := make(map[types.Hash]map[types.Hash][]byte)
	collect := func(path [][]bal.AccountChanges, isOld bool) error {
		for _, accounts := range path {
			for _, account := range accounts {
				if len(account.StorageChanges)+len(account.BalanceChanges)+len(account.NonceChanges)+len(account.CodeChanges) == 0 {
					continue
				}
				key := crypto.Keccak256Hash(account.Address[:])
				touched[key] = true
				for _, slot := range account.StorageChanges {
					plain := types.BytesToHash(slot.Slot.Bytes())
					hashed := crypto.Keccak256Hash(plain[:])
					if isOld {
						if oldSlots[key] == nil {
							oldSlots[key] = make(map[types.Hash]bool)
						}
						oldSlots[key][hashed] = true
						continue
					}
					if writes[key] == nil {
						writes[key] = make(map[types.Hash][]byte)
					}
					value := slot.Changes[len(slot.Changes)-1].Value
					if value.Sign() == 0 {
						writes[key][hashed] = nil
					} else {
						encoded, err := rlp.EncodeToBytes(value.Bytes())
						if err != nil {
							return err
						}
						writes[key][hashed] = encoded
					}
				}
				if !isOld {
					for _, change := range account.CodeChanges {
						if _, adopted := account.AdoptedCodeHash(change.Index); adopted {
							continue
						} // Hash-only adoption never installs empty bytecode. Missing code is proof-fetched at the final pivot below.
						staged.Codes[crypto.Keccak256Hash(change.Value)] = bytes.Clone(change.Value)
					}
				}
			}
		}
		return nil
	}
	if err := collect(rollback, true); err != nil {
		return err
	}
	if err := collect(forward, false); err != nil {
		return err
	}
	keys := make([]types.Hash, 0, len(touched))
	for key := range touched {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
	for _, key := range keys {
		response, err := update.Peer.RequestAccountRange(AccountRangeRequest{Root: update.Header.Root, Origin: key, Limit: key, Bytes: SnapSyncSoftByteLimit})
		if err != nil {
			return err
		}
		if response == nil {
			return fmt.Errorf("snap/2 missing account point proof")
		}
		var found *AccountData
		for i := range response.Accounts {
			if response.Accounts[i].Hash == key {
				found = &response.Accounts[i]
				break
			}
		}
		if found == nil {
			delete(staged.Accounts, key)
			delete(staged.Storage, key)
			continue
		}
		body, err := snapshotAccountBytes(*found)
		if err != nil {
			return err
		}
		staged.Accounts[key] = body
		if found.Root == types.EmptyRootHash {
			delete(staged.Storage, key)
			continue
		}
		if staged.Storage[key] == nil {
			staged.Storage[key] = make(map[types.Hash][]byte)
		}
		for slot := range oldSlots[key] {
			if _, overwritten := writes[key][slot]; overwritten {
				continue
			}
			response, err := update.Peer.RequestStorageRange(StorageRangeRequest{Root: update.Header.Root, Accounts: []types.Hash{key}, Origin: slot, Limit: slot, Bytes: SnapSyncSoftByteLimit})
			if err != nil {
				return err
			}
			if response == nil {
				return fmt.Errorf("snap/2 missing storage point proof")
			}
			delete(staged.Storage[key], slot)
			for _, value := range response.Slots {
				if value.AccountHash == key && value.SlotHash == slot {
					staged.Storage[key][slot] = bytes.Clone(value.Value)
				}
			}
		}
		for slot, value := range writes[key] {
			if value == nil {
				delete(staged.Storage[key], slot)
			} else {
				staged.Storage[key][slot] = bytes.Clone(value)
			}
		}
	}
	w.snapshot = staged
	return nil
}

func (s *StateSyncScheduler) SetPivotResolver(resolver PivotResolver) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running.Load() {
		return ErrStateSyncRunning
	}
	s.pivotResolver = resolver
	return nil
}

// updatePivot runs only on the scheduler goroutine between verified responses.
func (s *StateSyncScheduler) updatePivot() error {
	if s.pivotResolver == nil {
		return nil
	}
	update, err := s.pivotResolver(s.runContext, s.pivotHeader)
	if err != nil {
		return err
	}
	if update == nil {
		return nil
	}
	writer, ok := s.writer.(*SnapshotWriter)
	if !ok {
		return fmt.Errorf("snap/2 rebase requires snapshot writer")
	}
	original := writer.snapshot
	committed := false
	defer func() {
		if !committed {
			writer.snapshot = original
		}
	}()
	if err := writer.rebase(s.pivotHeader, update); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldTasks := make(map[types.Hash]*storageTask)
	for _, task := range s.storageTasks {
		oldTasks[task.accountHash] = task
	}
	var storageTasks []*storageTask
	codeTasks := make(map[types.Hash]*codeTask)
	for key, body := range writer.snapshot.Accounts {
		var account gethtypes.StateAccount
		if err := gethrlp.DecodeBytes(body, &account); err != nil {
			return err
		}
		root := types.Hash(account.Root)
		if root != types.EmptyRootHash {
			task := &storageTask{accountHash: key}
			if previous := oldTasks[key]; previous != nil {
				*task = *previous
			}
			actual, err := state.SnapshotTrieRoot(writer.snapshot.Storage[key])
			if err != nil {
				return err
			}
			if actual == root {
				task.done = true
			} else {
				if task.done {
					return fmt.Errorf("snap/2 BAL diffs contradict completed storage root")
				}
				task.done = false
			}
			storageTasks = append(storageTasks, task)
		}
		code := types.BytesToHash(account.CodeHash)
		if code != types.EmptyCodeHash {
			codeTasks[code] = &codeTask{hash: code}
		}
	}
	s.storageTasks, s.codeTasks = storageTasks, codeTasks
	s.storageIndex = make(map[types.Hash]*storageTask, len(storageTasks))
	for _, task := range storageTasks {
		s.storageIndex[task.accountHash] = task
	}
	committed = true
	s.pivotHeader, s.pivotRoot, s.pivotBlock = update.Header, update.Header.Root, update.Header.Number.Uint64()
	s.activePeer = update.Peer
	s.generation++
	s.progress.PivotBlock, s.progress.PivotRoot = s.pivotBlock, s.pivotRoot
	s.progress.StorageAccountsTotal = len(s.storageTasks)
	s.progress.StorageAccountsDone = 0
	for _, task := range s.storageTasks {
		if task.done {
			s.progress.StorageAccountsDone++
		}
	}
	s.progress.CodeHashesDone = 0
	return nil
}
