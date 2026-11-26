package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

// Reuse the key already written by core.CommitGenesis. It stores the actual
// resolved runtime config, including CLI overrides, rather than the input file.
var persistedChainConfigKey = []byte("ETH2030-chain-config")

var scheduledConfigFields = []string{
	"HomesteadBlock", "EIP150Block", "EIP155Block", "EIP158Block", "ByzantiumBlock",
	"ConstantinopleBlock", "PetersburgBlock", "IstanbulBlock", "MuirGlacierBlock",
	"BerlinBlock", "LondonBlock", "ArrowGlacierBlock", "GrayGlacierBlock",
	"ShanghaiTime", "CancunTime", "PragueTime", "OsakaTime", "AmsterdamTime",
	"GlamsterdanTime", "HogotaTime", "EIP8250Time", "EIP8151Time", "IPlusTime", "SSZBlockTime", "BPO1Time", "BPO2Time",
	"EIP8298Time", "EIP7906Time", "EIP8504Time",
}

func checkPersistedChainConfig(db rawdb.Database, incoming *config.ChainConfig) error {
	if err := incoming.Validate(); err != nil {
		return err
	}
	resolved := *incoming
	resolved.BlobScheduleConfig = incoming.ResolvedBlobSchedule()
	incoming = &resolved
	stored, configErr := db.Get(persistedChainConfigKey)
	if configErr != nil && !errors.Is(configErr, rawdb.ErrNotFound) {
		return fmt.Errorf("read persisted chain config: %w", configErr)
	}
	headHash, err := rawdb.ReadHeadBlockHash(db)
	if errors.Is(err, rawdb.ErrNotFound) {
		if _, genesisErr := rawdb.ReadCanonicalHash(db, 0); genesisErr == nil {
			return errors.New("stored genesis exists without a readable chain head")
		} else if !errors.Is(genesisErr, rawdb.ErrNotFound) {
			return fmt.Errorf("read stored genesis before configuration adoption: %w", genesisErr)
		}
		if configErr == nil {
			return errors.New("persisted chain config exists without a chain head")
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read chain head before configuration adoption: %w", err)
	}
	if _, err := rawdb.ReadCanonicalHash(db, 0); err != nil {
		return fmt.Errorf("read stored genesis for configuration comparison: %w", err)
	}
	number, err := rawdb.ReadHeaderNumber(db, headHash)
	if err != nil {
		return fmt.Errorf("read persisted head number: %w", err)
	}
	encoded, err := rawdb.ReadHeader(db, number, headHash)
	if err != nil {
		return fmt.Errorf("read persisted head header: %w", err)
	}
	header, err := types.DecodeHeaderRLP(encoded)
	if err != nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != number || header.Hash() != types.Hash(headHash) {
		return errors.New("invalid persisted head header for configuration comparison")
	}
	if errors.Is(configErr, rawdb.ErrNotFound) {
		if number == 0 {
			return nil
		} // No executed history; NewBlockchain still checks genesis identity.
		return errors.New("executed history has no persisted chain config; use a separately initialized data directory with a verified genesis and replay")
	}
	var previous config.ChainConfig
	decoder := json.NewDecoder(bytes.NewReader(stored))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&previous); err != nil {
		return fmt.Errorf("decode persisted chain config: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("trailing data in persisted chain config")
	}
	if err := previous.Validate(); err != nil {
		return fmt.Errorf("invalid persisted chain config: %w", err)
	}
	if previous.BlobScheduleConfig == nil && number > 0 {
		return errors.New("executed history has no resolved blobSchedule authority; use a separately initialized verified replay")
	}
	if previous.ChainID.Cmp(incoming.ChainID) != 0 {
		return errors.New("incompatible persisted chain ID")
	}
	if err := config.CheckConfigCompatible(&previous, incoming, number, header.Time); err != nil {
		return err
	}
	if number == 0 {
		return nil
	}
	if err := config.CheckBlobScheduleCompatible(&previous, incoming, header.Time); err != nil {
		return err
	}
	// Fields outside the registered fork schedule cannot be changed after
	// execution without an explicit migration rule. Never silently replay old
	// history under different contract addresses or research parameters.
	oldFields, err := nonScheduleConfigFields(&previous)
	if err != nil {
		return err
	}
	newFields, err := nonScheduleConfigFields(incoming)
	if err != nil {
		return err
	}
	for field, oldValue := range oldFields {
		if !bytes.Equal(oldValue, newFields[field]) {
			return fmt.Errorf("incompatible persisted chain parameter %s", field)
		}
	}
	for field, newValue := range newFields {
		if !bytes.Equal(newValue, oldFields[field]) {
			return fmt.Errorf("incompatible persisted chain parameter %s", field)
		}
	}
	return nil
}

func nonScheduleConfigFields(cfg *config.ChainConfig) (map[string]json.RawMessage, error) {
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	for _, field := range scheduledConfigFields {
		delete(fields, field)
	}
	delete(fields, "blobSchedule")
	return fields, nil
}

func writePersistedChainConfig(db rawdb.Database, cfg *config.ChainConfig) error {
	resolved := *cfg
	resolved.BlobScheduleConfig = cfg.ResolvedBlobSchedule()
	encoded, err := json.Marshal(&resolved)
	if err != nil {
		return err
	}
	if err := db.Put(persistedChainConfigKey, encoded); err != nil {
		return fmt.Errorf("persist resolved chain config: %w", err)
	}
	return nil
}
