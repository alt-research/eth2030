package node

import (
	"bytes"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/focil/heze"
)

type hezeOneClockView struct {
	*nativeHezeView
	calls int
}

func (v *hezeOneClockView) Clock() (heze.Clock, error) {
	v.calls++
	if v.calls > 1 {
		return heze.Clock{}, errors.New("unnecessary second clock lookup")
	}
	return v.clock, nil
}

func TestGroupEHezeIngressCommitsAfterPersistence(t *testing.T) {
	fixture := func() (*nativeHezeView, func(uint64) []byte) {
		secret := big.NewInt(1)
		v := &nativeHezeView{clock: heze.Clock{Slot: 1, IntoSlotMS: 1, SlotDurationMS: 12000}, state: heze.DependentState{KnownBlock: true, StateAvailable: true, PossibleDependent: true, ValidatorPubkeys: map[uint64][48]byte{0: crypto.BLSPubkeyFromSecret(secret)}}}
		signed := func(slot uint64) []byte {
			il := heze.SignedInclusionList{Message: heze.InclusionList{Slot: slot, Transactions: [][]byte{{0xff}}}}
			root := il.Message.SigningRoot(v.state.ForkVersion, v.state.GenesisValidatorsRoot)
			il.Signature = crypto.BLSSign(secret, root[:])
			raw, err := il.MarshalSSZ()
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}
		return v, signed
	}
	t.Run("one_validated_clock", func(t *testing.T) {
		base, signed := fixture()
		v := &hezeOneClockView{nativeHezeView: base}
		n := &Node{config: &Config{}, hezeView: v, hezeLists: heze.NewStore()}
		if err := n.HandleHezeInclusionList(signed(1)); err != nil {
			t.Fatal("valid ingress failed after committing its clock snapshot", err)
		}
		if v.calls != 1 {
			t.Fatal("ingress used multiple clock snapshots", v.calls)
		}
	})
	for _, stage := range []string{"create", "rename"} {
		t.Run(stage, func(t *testing.T) {
			v, signed := fixture()
			directory := t.TempDir()
			n := &Node{config: &Config{DataDir: directory}}
			if err := n.ConfigureHezeConsensus(v); err != nil {
				t.Fatal(err)
			}
			if err := n.HandleHezeInclusionList(signed(1)); err != nil {
				t.Fatal(err)
			}
			before, err := n.hezeLists.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, "heze-inclusion-lists.json")
			if stage == "create" {
				blocked := filepath.Join(directory, "not-a-directory")
				if err := os.WriteFile(blocked, []byte("blocked"), 0600); err != nil {
					t.Fatal(err)
				}
				n.config.DataDir = blocked
			} else {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "preserve"), []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			v.clock.Slot = 4 // successful commit must also prune the expired first list
			raw := signed(4)
			for i := 0; i < 3; i++ {
				if err := n.HandleHezeInclusionList(raw); err == nil {
					t.Fatal("failed persistence was reported as success")
				}
				after, err := n.hezeLists.MarshalBinary()
				if err != nil || !bytes.Equal(before, after) {
					t.Error("failed ingress changed bytes, timeliness, pruning or quota", err)
				}
			}
			n.config.DataDir = directory
			if stage == "rename" {
				value, err := os.ReadFile(filepath.Join(path, "preserve"))
				if err != nil || string(value) != "original" {
					t.Fatal("rename failure changed prior data", err)
				}
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, before, 0600); err != nil {
					t.Fatal(err)
				}
			}
			// Retries retain their full two-message allowance, and disk and memory
			// publish the same current/pruned snapshot only after successful persistence.
			for i := 0; i < 2; i++ {
				if err := n.HandleHezeInclusionList(raw); err != nil {
					t.Fatal("failed write consumed a gossip allowance", err)
				}
			}
			var validation *heze.ValidationError
			if err := n.HandleHezeInclusionList(raw); !errors.As(err, &validation) || validation.Disposition != heze.Ignore {
				t.Fatal("successful ingress lost its quota", err)
			}
			current, err := n.hezeLists.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			disk, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, disk) {
				t.Fatal("memory/disk commit disagreement", err)
			}
			restored, err := heze.Restore(disk, v)
			if err != nil {
				t.Fatal(err)
			}
			if len(restored.Transactions(heze.Key{Slot: 1}, true)) != 0 || len(restored.Transactions(heze.Key{Slot: 4}, true)) != 1 {
				t.Fatal("commit did not preserve pruning and timeliness")
			}
			entries, err := os.ReadDir(directory)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if len(e.Name()) >= 9 && e.Name()[:9] == ".heze-il-" {
					t.Fatal("temporary file leaked")
				}
			}
		})
	}
}
