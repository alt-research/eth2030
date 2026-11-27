package node

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/eth2030/eth2030/focil/heze"
)

// ConfigureHezeConsensus installs an authenticated CL state adapter. The EL
// never substitutes its execution head or timestamp for beacon shuffling data.
// Restored messages must reauthenticate against that adapter before use.
func (n *Node) ConfigureHezeConsensus(view heze.ConsensusView) error {
	if view == nil {
		return heze.ErrContextUnavailable
	}
	clock, err := view.Clock()
	if err != nil {
		return err
	}
	if clock.SlotDurationMS == 0 || clock.IntoSlotMS >= clock.SlotDurationMS {
		return heze.ErrContextUnavailable
	}
	n.hezeMu.Lock()
	defer n.hezeMu.Unlock()
	var store *heze.Store
	if n.hezeView != nil && n.hezeLists != nil {
		// Replacing the CL adapter changes the authentication boundary even
		// without a restart. Reverify the live snapshot before installing it;
		// a failure must preserve the prior adapter and store atomically.
		data, err := n.hezeLists.MarshalBinary()
		if err != nil {
			return err
		}
		store, err = heze.Restore(data, view)
		if err != nil {
			return err
		}
	} else if n.config.DataDir != "" {
		data, err := os.ReadFile(filepath.Join(n.config.DataDir, "heze-inclusion-lists.json"))
		if err == nil {
			store, err = heze.Restore(data, view)
			if err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if store == nil {
		store = heze.NewStore()
	}
	store.Prune(clock.Slot)
	n.hezeView, n.hezeLists = view, store
	return nil
}

// HandleHezeInclusionList is the native topic adapter's authenticated SSZ
// ingress. It is also usable by an external CL transport adapter. The native
// topic manager is not claimed as a full beacon gossipsub implementation.
func (n *Node) HandleHezeInclusionList(data []byte) error {
	n.hezeMu.Lock()
	defer n.hezeMu.Unlock()
	if n.hezeView == nil {
		return heze.ErrContextUnavailable
	}
	// Publish memory only after authentication, pruning and persistence succeed.
	// A failed write must not consume a gossip allowance or prune the live view.
	store := n.hezeLists.Clone()
	clock, err := store.GossipWithClock(data, n.hezeView)
	if err != nil {
		return err
	}
	store.Prune(clock.Slot)
	if n.config.DataDir != "" {
		snapshot, err := store.MarshalBinary()
		if err != nil {
			return err
		}
		if err := n.persistHezeInclusionLists(snapshot); err != nil {
			return err
		}
	}
	n.hezeLists = store
	return nil
}

func (n *Node) persistHezeInclusionLists(snapshot []byte) error {
	path := filepath.Join(n.config.DataDir, "heze-inclusion-lists.json")
	file, err := os.CreateTemp(n.config.DataDir, ".heze-il-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err = file.Write(snapshot); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	directory, err := os.Open(n.config.DataDir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func (n *Node) HezeInclusionListTransactions(key heze.Key, onlyTimely bool) ([][]byte, error) {
	n.hezeMu.Lock()
	defer n.hezeMu.Unlock()
	if n.hezeView == nil {
		return nil, heze.ErrContextUnavailable
	}
	return n.hezeLists.Transactions(key, onlyTimely), nil
}

// HezeInclusionListsByIndices serves canonical successful response chunks.
// Framing, fork-digest negotiation and beacon transport ownership remain CL
// responsibilities; no unsupported Engine method is invented for this request.
func (n *Node) HezeInclusionListsByIndices(data []byte, forkSlot uint64) ([][]byte, error) {
	var request heze.Request
	if err := request.UnmarshalSSZ(data); err != nil {
		return nil, err
	}
	n.hezeMu.Lock()
	defer n.hezeMu.Unlock()
	if n.hezeView == nil {
		return nil, heze.ErrContextUnavailable
	}
	clock, err := n.hezeView.Clock()
	if err != nil {
		return nil, err
	}
	view, err := n.hezeView.DependentState(request.Slot, request.DependentRoot)
	if err != nil {
		return nil, err
	}
	if !view.KnownBlock || !view.StateAvailable || !view.PossibleDependent || view.BlockSlot > view.ShufflingDependentSlot {
		return nil, heze.ErrContextUnavailable
	}
	lists, err := n.hezeLists.ByIndices(request, view.Committee, clock.Slot, forkSlot)
	if err != nil {
		return nil, err
	}
	out := make([][]byte, 0, len(lists))
	for _, list := range lists {
		raw, err := list.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}
