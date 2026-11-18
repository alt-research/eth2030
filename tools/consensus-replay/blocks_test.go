package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"testing"

	dynssz "github.com/pk910/dynamic-ssz"
)

func blockFixture(t *testing.T, slot uint64) []byte {
	t.Helper()
	f, err := os.Open(fmt.Sprintf("testdata/heze-a05a56de/beacon-block-%d.ssz.gz", slot))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	raw, err := io.ReadAll(io.LimitReader(gz, 16*1024*1024))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestOriginalSponsoredAndBlobBeaconBlocks(t *testing.T) {
	rawState, m := fixture(t)
	if _, err := replay(rawState, m); err != nil {
		t.Fatal(err)
	}
	ds := dynssz.NewDynSsz(m.Specs)
	state := new(LegacyHezeState)
	if err := ds.UnmarshalSSZ(state, rawState); err != nil {
		t.Fatal(err)
	}
	if len(m.Blocks) != 2 {
		t.Fatal("missing original workload block fixtures")
	}
	if err := replayHistoricalBlocks("testdata/heze-a05a56de", rawState, m); err != nil {
		t.Fatal(err)
	}
	for _, expected := range m.Blocks {
		t.Run(fmt.Sprint(expected.Slot), func(t *testing.T) {
			raw := blockFixture(t, expected.Slot)
			if err := replayBlock(raw, state, ds, expected); err != nil {
				t.Fatal(err)
			}
			block := new(LegacyHezeSignedBlock)
			if err := ds.UnmarshalSSZ(block, raw); err != nil {
				t.Fatal(err)
			}
			block.Signature[0] ^= 1
			changedSignature, err := ds.MarshalSSZ(block)
			if err != nil {
				t.Fatal(err)
			}
			if err := replayBlock(changedSignature, state, ds, expected); err == nil {
				t.Fatal("changed signature escaped the captured signed-byte comparison")
			}
			block.Signature[0] ^= 1
			block.Message.Body.SignedExecutionPayloadBid.Message.InclusionListBits[0] ^= 1
			changed, err := ds.MarshalSSZ(block)
			if err != nil {
				t.Fatal(err)
			}
			if err := replayBlock(changed, state, ds, expected); err == nil {
				t.Fatal("changed inclusion-list bits escaped the finalized root commitment")
			}
			if err := ds.UnmarshalSSZ(block, raw); err != nil {
				t.Fatal(err)
			}
			bid := block.Message.Body.SignedExecutionPayloadBid.Message
			if len(bid.BlobKZGCommitments) > 0 {
				bid.BlobKZGCommitments[0][0] ^= 1
				changed, err := ds.MarshalSSZ(block)
				if err != nil {
					t.Fatal(err)
				}
				if err := replayBlock(changed, state, ds, expected); err == nil {
					t.Fatal("changed blob commitment escaped the finalized root commitment")
				}
			}
			expected.ExecutionHash = "0x00"
			if err := replayBlock(raw, state, ds, expected); err == nil {
				t.Fatal("accepted unrelated EL receipt identity")
			}
		})
	}
}
