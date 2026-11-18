package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"path/filepath"

	dynssz "github.com/pk910/dynamic-ssz"
)

type blockRecord struct {
	Slot          uint64   `json:"slot"`
	Root          string   `json:"root"`
	ExecutionHash string   `json:"execution_hash"`
	Commitments   []string `json:"commitments"`
	InclusionBits string   `json:"inclusion_list_bits"`
	Signature     string   `json:"signature"`
}

// The caller has already checked the state against its original Beacon header.
func replayHistoricalBlocks(dir string, rawState []byte, m manifest) error {
	if len(m.Blocks) == 0 {
		return nil
	}
	ds := dynssz.NewDynSsz(m.Specs)
	state := new(LegacyHezeState)
	if err := ds.UnmarshalSSZ(state, rawState); err != nil {
		return err
	}
	for _, expected := range m.Blocks {
		raw, err := readCompressed(filepath.Join(dir, fmt.Sprintf("beacon-block-%d.ssz.gz", expected.Slot)))
		if err != nil {
			return err
		}
		if err := replayBlock(raw, state, ds, expected); err != nil {
			return err
		}
	}
	return nil
}

func replayBlock(raw []byte, state *LegacyHezeState, ds *dynssz.DynSsz, expected blockRecord) error {
	block := new(LegacyHezeSignedBlock)
	if err := ds.UnmarshalSSZ(block, raw); err != nil {
		return err
	}
	encoded, err := ds.MarshalSSZ(block)
	if err != nil || !bytes.Equal(encoded, raw) {
		return fmt.Errorf("historical block SSZ roundtrip differs: %v", err)
	}
	slot := uint64(block.Message.Slot)
	if slot != expected.Slot || slot >= uint64(state.Slot) || uint64(state.Slot)-slot > uint64(len(state.BlockRoots)) {
		return fmt.Errorf("historical block is outside the finalized root window")
	}
	root, err := ds.HashTreeRoot(block.Message)
	if err != nil {
		return err
	}
	if root != state.BlockRoots[slot%uint64(len(state.BlockRoots))] || "0x"+hex.EncodeToString(root[:]) != expected.Root {
		return fmt.Errorf("historical block root differs from the finalized state commitment")
	}
	if "0x"+hex.EncodeToString(block.Signature[:]) != expected.Signature {
		return fmt.Errorf("historical block signature differs from captured Beacon JSON")
	}
	bid := block.Message.Body.SignedExecutionPayloadBid.Message
	if "0x"+hex.EncodeToString(bid.BlockHash[:]) != expected.ExecutionHash || len(bid.BlobKZGCommitments) != len(expected.Commitments) {
		return fmt.Errorf("historical bid differs from captured Beacon/EL evidence")
	}
	for i, commitment := range bid.BlobKZGCommitments {
		if "0x"+hex.EncodeToString(commitment[:]) != expected.Commitments[i] {
			return fmt.Errorf("historical blob commitment %d changed", i)
		}
	}
	if "0x"+hex.EncodeToString(bid.InclusionListBits) != expected.InclusionBits {
		return fmt.Errorf("historical inclusion-list bits changed")
	}
	return nil
}
