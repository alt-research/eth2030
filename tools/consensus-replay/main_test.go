package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/heze"
	dynssz "github.com/pk910/dynamic-ssz"
)

func TestReleasedWireUsesReleasedCodec(t *testing.T) {
	// A codec-routing test, not evidence of an EIP-8015 CL's state transition.
	// Construct the released layout with its own SDK, then require the replay
	// tool to keep using that layout instead of the ETH2030 compatibility type.
	ds := dynssz.NewDynSsz(nil)
	raw, err := ds.MarshalSSZ(&heze.BeaconState{GenesisTime: 1, Slot: 2})
	if err != nil {
		t.Fatal(err)
	}
	decoded := new(heze.BeaconState)
	if err := ds.UnmarshalSSZ(decoded, raw); err != nil {
		t.Fatal(err)
	}
	root, err := ds.HashTreeRoot(decoded)
	if err != nil {
		t.Fatal(err)
	}
	result, err := replay(raw, manifest{StateRoot: "0x" + hex.EncodeToString(root[:]), Slot: 2})
	if err != nil || result.Codec != "upstream-v0.1.11" {
		t.Fatalf("released wire was not handled by the released codec: %+v %v", result, err)
	}
}

func fixture(t *testing.T) ([]byte, manifest) {
	t.Helper()
	raw, m, err := loadFixture("testdata/heze-a05a56de")
	if err != nil {
		t.Fatal(err)
	}
	return raw, m
}

func TestRealFinalizedHezeState(t *testing.T) {
	raw, m := fixture(t)
	// The published codec is the first candidate. Preserve its real failure,
	// rather than bypassing it based solely on a branch or version label.
	err := dynssz.NewDynSsz(m.Specs).UnmarshalSSZ(new(heze.BeaconState), raw)
	if err == nil || !strings.Contains(err.Error(), "incorrect offset") {
		t.Fatalf("released Heze layout boundary changed: %v", err)
	}
	result, err := replay(raw, m)
	if err != nil {
		t.Fatal(err)
	}
	if result.Codec != "eth2030-a05a56de" || result.Slot != 192 || result.Validators != 256 || result.PTCWindow != 96 || result.StateRoot != m.StateRoot {
		t.Fatalf("wrong finalized consensus state: %+v", result)
	}
}

func TestConsensusReplayRejectsChangedStateAndAuthority(t *testing.T) {
	t.Run("changed fixed field", func(t *testing.T) {
		raw, m := fixture(t)
		raw[0] ^= 1 // Valid SSZ with a different genesis time must fail the root.
		if _, err := replay(raw, m); err == nil || !strings.Contains(err.Error(), "state root") {
			t.Fatalf("accepted changed consensus state: %v", err)
		}
	})
	t.Run("unrelated header", func(t *testing.T) {
		raw, m := fixture(t)
		m.StateRoot = "0x" + strings.Repeat("00", 32)
		if _, err := replay(raw, m); err == nil {
			t.Fatal("accepted unrelated Beacon header")
		}
	})
	t.Run("truncated data", func(t *testing.T) {
		raw, m := fixture(t)
		if _, err := replay(raw[:len(raw)-1], m); err == nil {
			t.Fatal("accepted truncated consensus state")
		}
	})
	t.Run("different JSON balance", func(t *testing.T) {
		raw, m := fixture(t)
		m.Balances[255]++
		if _, err := replay(raw, m); err == nil || !strings.Contains(err.Error(), "balance 255") {
			t.Fatalf("accepted different epoch balance: %v", err)
		}
	})
	t.Run("different validator", func(t *testing.T) {
		raw, m := fixture(t)
		m.PublicKeys[255] = m.PublicKeys[0]
		if _, err := replay(raw, m); err == nil || !strings.Contains(err.Error(), "public key 255") {
			t.Fatalf("accepted different epoch validator: %v", err)
		}
	})
	t.Run("wrong PTC preset", func(t *testing.T) {
		raw, m := fixture(t)
		m.Specs["PTC_SIZE"] = uint64(511)
		if _, err := replay(raw, m); err == nil {
			t.Fatal("accepted a different SSZ preset")
		}
	})
}

func TestLegacyHezeFieldOrderIsCommitted(t *testing.T) {
	raw, m := fixture(t)
	ds := dynssz.NewDynSsz(m.Specs)
	state := new(LegacyHezeState)
	if err := ds.UnmarshalSSZ(state, raw); err != nil {
		t.Fatal(err)
	}
	state.LatestExecutionPayloadBid.ExecutionRequestsRoot[0] ^= 1
	changed, err := ds.MarshalSSZ(state)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(changed, raw) {
		t.Fatal("execution requests root omitted from the SSZ schema")
	}
	if _, err := replay(changed, m); err == nil {
		t.Fatal("accepted a changed execution requests root")
	}
}
