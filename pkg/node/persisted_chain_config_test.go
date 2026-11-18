package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func configRecoveryDB(t *testing.T, number, timestamp uint64) rawdb.Database {
	t.Helper()
	db := rawdb.NewMemoryDB()
	header := &types.Header{Number: new(big.Int).SetUint64(number), Time: timestamp, GasLimit: 30_000_000, Difficulty: new(big.Int), BaseFee: big.NewInt(1)}
	genesis := &types.Header{Number: new(big.Int), GasLimit: 30_000_000, Difficulty: new(big.Int), BaseFee: big.NewInt(1)}
	genesisData, err := genesis.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteHeader(db, 0, genesis.Hash(), genesisData); err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteCanonicalHash(db, 0, genesis.Hash()); err != nil {
		t.Fatal(err)
	}
	if number != 0 {
		header.ParentHash = genesis.Hash()
	}
	encoded, err := header.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteHeader(db, number, header.Hash(), encoded); err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteHeadBlockHash(db, header.Hash()); err != nil {
		t.Fatal(err)
	}
	if err := rawdb.WriteCanonicalHash(db, number, header.Hash()); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestEveryMutableScheduleFieldHasCompatibilityAuthority(t *testing.T) {
	for _, field := range scheduledConfigFields {
		t.Run(field, func(t *testing.T) {
			old, newCfg := *config.TestConfig, *config.TestConfig
			left := reflect.ValueOf(&old).Elem().FieldByName(field)
			right := reflect.ValueOf(&newCfg).Elem().FieldByName(field)
			left.Set(reflect.Zero(left.Type()))
			if strings.HasSuffix(field, "Block") {
				right.Set(reflect.ValueOf(big.NewInt(0)))
			} else {
				value := uint64(0)
				right.Set(reflect.ValueOf(&value))
			}
			if config.CheckConfigCompatible(&old, &newCfg, 1, 1) == nil || config.CheckConfigCompatible(&newCfg, &old, 1, 1) == nil {
				t.Fatal("mutable field is absent from activated-fork authority")
			}
		})
	}
}

type failingConfigDB struct {
	rawdb.Database
	readError, writeError bool
}

var errConfigStorage = errors.New("injected config storage error")

func (db failingConfigDB) Get(key []byte) ([]byte, error) {
	if db.readError && bytes.Equal(key, persistedChainConfigKey) {
		return nil, errConfigStorage
	}
	return db.Database.Get(key)
}
func (db failingConfigDB) Put(key, value []byte) error {
	if db.writeError && bytes.Equal(key, persistedChainConfigKey) {
		return errConfigStorage
	}
	return db.Database.Put(key, value)
}

func TestPersistedConfigAuthorityAndStorageFailures(t *testing.T) {
	cfg := *config.TestConfig
	if err := checkPersistedChainConfig(rawdb.NewMemoryDB(), &cfg); err != nil {
		t.Fatal(err)
	}
	if err := checkPersistedChainConfig(configRecoveryDB(t, 0, 0), &cfg); err != nil {
		t.Fatal("genesis-only legacy recovery:", err)
	}
	db := configRecoveryDB(t, 1, 12)
	if err := checkPersistedChainConfig(db, &cfg); err == nil {
		t.Fatal("unknown executed-history config was silently adopted")
	}
	if err := writePersistedChainConfig(db, &cfg); err != nil {
		t.Fatal(err)
	}
	valid, _ := db.Get(persistedChainConfigKey)
	for _, corrupt := range [][]byte{[]byte("null"), []byte("{"), append(bytes.Clone(valid), []byte(" {}")...), []byte(`{"ChainID":1337,"unsupportedFutureRule":true}`)} {
		if err := db.Put(persistedChainConfigKey, corrupt); err != nil {
			t.Fatal(err)
		}
		if err := checkPersistedChainConfig(db, &cfg); err == nil {
			t.Fatalf("corrupt metadata accepted: %s", corrupt)
		}
		after, _ := db.Get(persistedChainConfigKey)
		if !bytes.Equal(after, corrupt) {
			t.Fatal("rejected metadata overwritten")
		}
	}
	if err := db.Put(persistedChainConfigKey, valid); err != nil {
		t.Fatal(err)
	}
	if err := checkPersistedChainConfig(failingConfigDB{Database: db, readError: true}, &cfg); !errors.Is(err, errConfigStorage) {
		t.Fatalf("read failure hidden: %v", err)
	}
	if err := writePersistedChainConfig(failingConfigDB{Database: db, writeError: true}, &cfg); !errors.Is(err, errConfigStorage) {
		t.Fatalf("write failure hidden: %v", err)
	}
	after, _ := db.Get(persistedChainConfigKey)
	if !bytes.Equal(after, valid) {
		t.Fatal("failed write lost prior authority")
	}
	if err := checkPersistedChainConfig(db, &cfg); err != nil {
		t.Fatal(err)
	}
	orphan := rawdb.NewMemoryDB()
	if err := orphan.Put(persistedChainConfigKey, valid); err != nil {
		t.Fatal(err)
	}
	if err := checkPersistedChainConfig(orphan, &cfg); err == nil {
		t.Fatal("orphan metadata accepted as a fresh chain")
	}
	missingHead := rawdb.NewMemoryDB()
	if err := rawdb.WriteCanonicalHash(missingHead, 0, types.Hash{1}); err != nil {
		t.Fatal(err)
	}
	if err := checkPersistedChainConfig(missingHead, &cfg); err == nil {
		t.Fatal("stored genesis without head accepted as fresh")
	}
}

func TestPersistedConfigUsesEitherSideActivationAndRegisteredBPOs(t *testing.T) {
	db := configRecoveryDB(t, 1, 20)
	previous := *config.TestConfig
	zero, future := uint64(0), uint64(1000)
	previous.OsakaTime, previous.AmsterdamTime, previous.BPO1Time = &zero, &future, &future
	if err := writePersistedChainConfig(db, &previous); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*config.ChainConfig)
		reject bool
	}{
		{"future Amsterdam", func(c *config.ChainConfig) { at := uint64(2000); c.AmsterdamTime = &at }, false},
		{"future BPO", func(c *config.ChainConfig) { at := uint64(2000); c.BPO1Time = &at }, false},
		{"incoming active Amsterdam", func(c *config.ChainConfig) { at := uint64(20); c.AmsterdamTime = &at }, true},
		{"incoming active BPO", func(c *config.ChainConfig) { at := uint64(20); c.BPO1Time = &at }, true},
		{"removed active Osaka", func(c *config.ChainConfig) { c.OsakaTime = nil; c.AmsterdamTime = nil }, true},
		{"new unregistered historical parameter", func(c *config.ChainConfig) { at := uint64(2000); c.LegacyNativeBLSUntil = &at }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			incoming := previous
			test.change(&incoming)
			err := checkPersistedChainConfig(db, &incoming)
			if (err != nil) != test.reject {
				t.Fatalf("rejection=%v error=%v", test.reject, err)
			}
		})
	}
}

func TestPersistedBlobParameterAuthorityAndUnknownLegacyHistory(t *testing.T) {
	db := configRecoveryDB(t, 1, 12)
	cfg := *config.TestConfig
	at := uint64(24)
	cfg.BPO1Time = &at
	cfg.BlobScheduleConfig = cfg.ResolvedBlobSchedule()
	if err := writePersistedChainConfig(db, &cfg); err != nil {
		t.Fatal(err)
	}
	future := cfg
	future.BlobScheduleConfig = cfg.ResolvedBlobSchedule()
	future.BlobScheduleConfig.BPO1.UpdateFraction++
	if err := checkPersistedChainConfig(db, &future); err != nil {
		t.Fatal("genuinely future blob update rejected:", err)
	}
	legacy := cfg
	legacy.BlobScheduleConfig = nil
	encoded, err := json.Marshal(&legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Put(persistedChainConfigKey, encoded); err != nil {
		t.Fatal(err)
	}
	if err := checkPersistedChainConfig(db, &cfg); err == nil || !strings.Contains(err.Error(), "no resolved blobSchedule authority") {
		t.Fatalf("unknown historical defaults guessed: %v", err)
	}
}

func TestNodeForkOverridesDoNotMutateNetworkDefaults(t *testing.T) {
	before := *config.MainnetConfig
	beforeJSON, err := json.Marshal(&before)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	cfg.Network = "mainnet"
	activation := uint64(4_000_000_000)
	cfg.GlamsterdamOverride = &activation
	n := newTestNode(t, &cfg)
	if at := n.Blockchain().Config().GlamsterdanTime; at == nil || *at != activation {
		t.Fatal("resolved override was not adopted")
	}
	afterJSON, err := json.Marshal(config.MainnetConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeJSON, afterJSON) {
		*config.MainnetConfig = before
		t.Fatal("one node's override changed network defaults")
	}
}

func TestActualNodeRestartRejectsHistoricalConfigChanges(t *testing.T) {
	const token = "config-recovery-test"
	var genesis map[string]any
	if err := json.Unmarshal(minimalGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	forks := genesis["config"].(map[string]any)
	forks["amsterdamTime"], forks["osakaTime"] = uint64(100), uint64(0)
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	genesis["alloc"].(map[string]any)[sender.Hex()] = map[string]string{"balance": "100000000000000000000"}
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	writeInput := func() {
		t.Helper()
		data, err := json.Marshal(genesis)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cfg.GenesisPath, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeInput()
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	recipient := types.Address{19: 0x44}
	to := gethcommon.Address(recipient)
	tx, err := gethtypes.SignTx(gethtypes.NewTransaction(0, to, big.NewInt(123456), 25_000, big.NewInt(3_000_000_000), nil), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var txHash types.Hash
	rpc.call(t, "eth_sendRawTransaction", &txHash, "0x"+hex.EncodeToString(raw))
	root := types.Hash{1}
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV3", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}, payload.PayloadAttributesV3{PayloadAttributesV2: payload.PayloadAttributesV2{PayloadAttributesV1: payload.PayloadAttributesV1{Timestamp: 12}, Withdrawals: []*payload.Withdrawal{}}, ParentBeaconBlockRoot: root})
	if fc.PayloadID == nil {
		t.Fatal("no production payload")
	}
	var built struct {
		ExecutionPayload  *payload.ExecutionPayloadV3
		ExecutionRequests []string
	}
	rpc.call(t, "engine_getPayloadV5", &built, fc.PayloadID)
	if built.ExecutionPayload == nil || len(built.ExecutionPayload.Transactions) != 1 {
		t.Fatal("signed transaction not included")
	}
	var status payload.PayloadStatusV1
	rpc.call(t, "engine_newPayloadV4", &status, built.ExecutionPayload, []types.Hash{}, root, built.ExecutionRequests)
	if status.Status != "VALID" {
		t.Fatalf("initial import: %+v", status)
	}
	head, stateRoot := built.ExecutionPayload.BlockHash, built.ExecutionPayload.StateRoot
	rpc.call(t, "engine_forkchoiceUpdatedV3", &fc, payload.ForkchoiceStateV1{HeadBlockHash: head}, nil)
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	db, err := rawdb.NewFileDB(filepath.Join(cfg.DataDir, "chaindata"))
	if err != nil {
		t.Fatal(err)
	}
	authority, err := db.Get(persistedChainConfigKey)
	if err != nil {
		t.Fatal("actual Node did not persist resolved config:", err)
	}
	_ = db.Close()
	// Changes that otherwise share the same genesis hash must fail before replay.
	for _, change := range []struct {
		field   string
		value   any
		message string
	}{
		{"amsterdamTime", uint64(12), "incompatible fork"},
		{"chainId", uint64(1338), "chain ID"},
		{"depositContractAddress", (types.Address{19: 0x55}).Hex(), "chain parameter"},
	} {
		old, exists := forks[change.field]
		forks[change.field] = change.value
		writeInput()
		rejected, err := New(&cfg)
		if rejected != nil {
			_ = rejected.Stop()
		}
		if err == nil || !strings.Contains(err.Error(), change.message) {
			t.Fatalf("%s changed historical authority: %v", change.field, err)
		}
		if exists {
			forks[change.field] = old
		} else {
			delete(forks, change.field)
		}
		check, err := rawdb.NewFileDB(filepath.Join(cfg.DataDir, "chaindata"))
		if err != nil {
			t.Fatal(err)
		}
		stored, _ := check.Get(persistedChainConfigKey)
		storedHead, headErr := rawdb.ReadHeadBlockHash(check)
		_ = check.Close()
		if !bytes.Equal(stored, authority) || headErr != nil || types.Hash(storedHead) != head {
			t.Fatal("failed restart changed config or canonical head")
		}
	}
	// A genuinely future update is adopted and survives another fresh Node.
	forks["amsterdamTime"] = uint64(200)
	writeInput()
	restarted := newTestNode(t, &cfg)
	if restarted.Blockchain().CurrentBlock().Hash() != head || restarted.Blockchain().CurrentBlock().Root() != stateRoot {
		t.Fatal("future update changed history")
	}
	if restarted.Blockchain().State().GetBalance(recipient).Cmp(big.NewInt(123456)) != 0 {
		t.Fatal("signed historical state did not survive replay")
	}
	if err := restarted.Stop(); err != nil {
		t.Fatal(err)
	}
	again := newTestNode(t, &cfg)
	if at := again.Blockchain().Config().AmsterdamTime; at == nil || *at != 200 || again.Blockchain().CurrentBlock().Hash() != head {
		t.Fatal("future authority was not retained")
	}
}
