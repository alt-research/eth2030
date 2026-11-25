// Replay the actual consensus SSZ before promoting an explorer codec.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	dynssz "github.com/pk910/dynamic-ssz"
)

type manifest struct {
	StateRoot  string         `json:"state_root"`
	Specs      map[string]any `json:"specs"`
	Slot       uint64         `json:"slot"`
	Validators int            `json:"validators"`
	Balances   []uint64       `json:"balances"`
	PublicKeys []string       `json:"public_keys"`
	Blocks     []blockRecord  `json:"blocks"`
}

type report struct {
	Codec      string `json:"codec"`
	Slot       uint64 `json:"slot"`
	Validators int    `json:"validators"`
	PTCWindow  int    `json:"ptc_window"`
	StateRoot  string `json:"state_root"`
}

func checkState(ds *dynssz.DynSsz, state any, raw []byte, expectedRoot string) error {
	encoded, err := ds.MarshalSSZ(state)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, raw) {
		return errors.New("state SSZ byte roundtrip differs")
	}
	root, err := ds.HashTreeRoot(state)
	if err != nil {
		return err
	}
	if "0x"+hex.EncodeToString(root[:]) != expectedRoot {
		return errors.New("state root differs from the captured Beacon header")
	}
	return nil
}

// replay always tries the released schema first. The ETH2030 compatibility
// path is accepted only with an exact SSZ roundtrip and the original state root.
func replay(raw []byte, m manifest) (report, error) {
	ds := dynssz.NewDynSsz(m.Specs)
	released := new(heze.BeaconState)
	releasedErr := ds.UnmarshalSSZ(released, raw)
	if releasedErr == nil {
		releasedErr = checkState(ds, released, raw, m.StateRoot)
	}
	if releasedErr == nil {
		if err := checkEpochFields(uint64(released.Slot), released.Validators, released.Balances, m); err != nil {
			return report{}, err
		}
		return report{"upstream-v0.1.11", uint64(released.Slot), len(released.Validators), len(released.PTCWindow), m.StateRoot}, nil
	}
	legacy := new(LegacyHezeState)
	if err := ds.UnmarshalSSZ(legacy, raw); err != nil {
		return report{}, fmt.Errorf("upstream: %v; ETH2030 compatibility: %w", releasedErr, err)
	}
	if err := checkState(ds, legacy, raw, m.StateRoot); err != nil {
		return report{}, err
	}
	if err := checkEpochFields(uint64(legacy.Slot), legacy.Validators, legacy.Balances, m); err != nil {
		return report{}, err
	}
	return report{"eth2030-a05a56de", uint64(legacy.Slot), len(legacy.Validators), len(legacy.PTCWindow), m.StateRoot}, nil
}

func checkEpochFields(slot uint64, validators []*phase0.Validator, balances []phase0.Gwei, m manifest) error {
	if slot != m.Slot || len(validators) != m.Validators || len(balances) != len(m.Balances) || len(validators) != len(m.PublicKeys) {
		return errors.New("SSZ epoch fields differ from captured Beacon JSON")
	}
	for i, balance := range balances {
		if uint64(balance) != m.Balances[i] {
			return fmt.Errorf("validator balance %d differs from captured Beacon JSON", i)
		}
	}
	for i, validator := range validators {
		if validator == nil || "0x"+hex.EncodeToString(validator.PublicKey[:]) != m.PublicKeys[i] {
			return fmt.Errorf("validator public key %d differs from captured Beacon JSON", i)
		}
	}
	return nil
}

func loadFixture(dir string) ([]byte, manifest, error) {
	var m manifest
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, m, err
	}
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, m, err
	}
	raw, err := readCompressed(filepath.Join(dir, "heze-finalized.ssz.gz"))
	return raw, m, err
}

func readCompressed(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	raw, err := io.ReadAll(io.LimitReader(gz, 16*1024*1024+1))
	if len(raw) > 16*1024*1024 {
		return nil, errors.New("oversized fixture")
	}
	return raw, err
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: consensus-replay fixture-directory")
		os.Exit(2)
	}
	raw, m, err := loadFixture(os.Args[1])
	var result report
	if err == nil {
		result, err = replay(raw, m)
	}
	if err == nil {
		err = replayHistoricalBlocks(os.Args[1], raw, m)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output := struct {
		report
		HistoricalBlocks int `json:"historical_blocks_verified"`
	}{result, len(m.Blocks)}
	if err = json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
