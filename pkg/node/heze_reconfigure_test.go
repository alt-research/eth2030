package node

import (
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/focil/heze"
)

// Replacing the authenticated CL adapter must enforce the same signature and
// dependent-state boundary as restoring from disk, including memory-only nodes.
func TestGroupEHezeReconfigureReauthenticatesMemoryAtomically(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "memory"
		if persistent {
			name = "disk"
		}
		t.Run(name, func(t *testing.T) {
			cfg := &Config{}
			if persistent {
				cfg.DataDir = t.TempDir()
			}
			n := &Node{config: cfg}
			secret := big.NewInt(1)
			view := &nativeHezeView{
				clock: heze.Clock{Slot: 1, IntoSlotMS: 1, SlotDurationMS: 12000},
				state: heze.DependentState{KnownBlock: true, StateAvailable: true, PossibleDependent: true, ValidatorPubkeys: map[uint64][48]byte{0: crypto.BLSPubkeyFromSecret(secret)}},
			}
			if err := n.ConfigureHezeConsensus(view); err != nil {
				t.Fatal(err)
			}
			il := heze.SignedInclusionList{Message: heze.InclusionList{Slot: 1, Transactions: [][]byte{{0xff}}}}
			root := il.Message.SigningRoot(view.state.ForkVersion, view.state.GenesisValidatorsRoot)
			il.Signature = crypto.BLSSign(secret, root[:])
			raw, err := il.MarshalSSZ()
			if err != nil {
				t.Fatal(err)
			}
			if err := n.HandleHezeInclusionList(raw); err != nil {
				t.Fatal(err)
			}
			// Missing ancestry and changed validator keys cannot bless an existing cache.
			for _, unavailable := range []bool{true, false} {
				replacement := &nativeHezeView{clock: view.clock, state: view.state}
				if unavailable {
					replacement.state.StateAvailable = false
				} else {
					replacement.state.ValidatorPubkeys = map[uint64][48]byte{0: crypto.BLSPubkeyFromSecret(big.NewInt(2))}
				}
				if err := n.ConfigureHezeConsensus(replacement); err == nil {
					t.Fatal("replacement adapter accepted an unauthenticated retained message")
				}
				if n.hezeView != view {
					t.Fatal("failed replacement changed the active adapter")
				}
			}
			if err := n.ConfigureHezeConsensus(view); err != nil {
				t.Fatal(err)
			}
			txs, err := n.HezeInclusionListTransactions(heze.Key{Slot: 1, DependentRoot: types.Hash{}}, true)
			if err != nil || len(txs) != 1 || len(txs[0]) != 1 || txs[0][0] != 0xff {
				t.Fatal("reconfiguration lost the timely view", txs, err)
			}
			if err := n.HandleHezeInclusionList(raw); err != nil {
				t.Fatal(err)
			}
			var validation *heze.ValidationError
			if err := n.HandleHezeInclusionList(raw); !errors.As(err, &validation) || validation.Disposition != heze.Ignore {
				t.Fatal("reconfiguration reset the gossip quota", err)
			}
		})
	}
}
