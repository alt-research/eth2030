package node

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
)

var nativeCommitStallConfig = flag.String("native-commit-stall-config", "", "explicit owned diagnostic fixture manifest; never a production node option")

type commitStallDB struct {
	rawdb.Database
	ctx         context.Context
	arm, status string
	duration    time.Duration
	once        sync.Once
	started     chan struct{}
	release     <-chan struct{}
}
type commitStallBatch struct {
	rawdb.Batch
	db        *commitStallDB
	header    *types.Header
	validated bool
}

func (db *commitStallDB) NewBatch() rawdb.Batch {
	return &commitStallBatch{Batch: db.Database.NewBatch(), db: db}
}
func (b *commitStallBatch) Put(key, value []byte) error {
	if len(key) == 41 && key[0] == 'h' {
		header, err := types.DecodeHeaderRLP(value)
		if err != nil {
			return err
		}
		b.header = header
	}
	if len(key) == 48 && bytes.HasPrefix(key, []byte("validated-block/")) && bytes.Equal(value, []byte{1}) {
		b.validated = true
	}
	return b.Batch.Put(key, value)
}
func (b *commitStallBatch) Write() error {
	h := b.header
	if b.validated && h != nil && h.Number.Uint64() > 0 && h.SlotNumber != nil && *h.SlotNumber%32 == 0 {
		wire, err := os.ReadFile(b.db.arm)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			var arm struct {
				MinimumSlot uint64 `json:"minimum_slot"`
			}
			if err := json.Unmarshal(wire, &arm); err != nil {
				return err
			}
			if *h.SlotNumber >= arm.MinimumSlot {
				claim := false
				b.db.once.Do(func() { claim = true })
				if claim {
					record := map[string]any{"phase": "blocked", "at": time.Now().UTC().Format(time.RFC3339Nano), "number": h.Number.Uint64(), "slot": *h.SlotNumber, "hash": h.Hash().Hex(), "hold_seconds": b.db.duration.Seconds(), "scope": "Deliberately held native checked batch before atomic commit; not the original unknown holder or operating acceptance"}
					save := func() error {
						wire, err := json.MarshalIndent(record, "", "  ")
						if err != nil {
							return err
						}
						tmp := b.db.status + ".tmp"
						if err := os.WriteFile(tmp, wire, 0600); err != nil {
							return err
						}
						return os.Rename(tmp, b.db.status)
					}
					if err := save(); err != nil {
						return err
					}
					close(b.db.started)
					select {
					case <-time.After(b.db.duration):
						record["phase"] = "released"
					case <-b.db.release:
						record["phase"] = "released"
					case <-b.db.ctx.Done():
						record["phase"] = "aborted_by_owner"
					}
					record["released_at"] = time.Now().UTC().Format(time.RFC3339Nano)
					if err := save(); err != nil {
						return err
					}
				}
			}
		}
	}
	return b.Batch.Write()
}

func TestNewWithDatabasePreservesOpenFailure(t *testing.T) {
	c := startupConfig(t)
	sentinel := errors.New("owned database open control")
	calls := 0
	n, err := newWithDatabase(&c, func(path string) (rawdb.Database, error) {
		calls++
		if path != c.ResolvePath("chaindata") {
			t.Fatal("wrong database path")
		}
		return nil, sentinel
	})
	if n != nil || !errors.Is(err, sentinel) || calls != 1 {
		t.Fatalf("database opener failure lost: node=%v err=%v calls=%d", n, err, calls)
	}
}

func TestNativeCommitBarrierKeepsReadsAndValidationReal(t *testing.T) {
	const token = "real-native-commit-barrier"
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	forks := genesis["config"].(map[string]any)
	forks["terminalTotalDifficulty"], forks["amsterdamTime"] = 0, 0
	wire, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	c := startupConfig(t)
	c.GenesisPath = filepath.Join(c.DataDir, "genesis.json")
	c.EngineAuthToken = token
	if err := os.WriteFile(c.GenesisPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	db := &commitStallDB{ctx: context.Background(), arm: filepath.Join(c.DataDir, "arm.json"), status: filepath.Join(c.DataDir, "commit.json"), duration: 10 * time.Second, started: make(chan struct{}), release: release}
	n, err := newWithDatabase(&c, func(path string) (rawdb.Database, error) {
		d, err := rawdb.NewFileDB(path)
		db.Database = d
		return db, err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		unblock()
		if err := n.Stop(); err != nil {
			t.Error(err)
		}
	}()
	rpc := track4NodeRPC(t, n, token)
	parent := n.blockchain.Genesis().Hash()
	beacon := types.Hash{31: 32}
	attrs := map[string]any{"timestamp": "0x180", "slotNumber": "0x20", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380"}
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
	if fc.PayloadID == nil {
		t.Fatal("build did not start")
	}
	var built struct {
		ExecutionPayload  payload.ExecutionPayloadV5
		ExecutionRequests []string
	}
	rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	// The real builder prevalidates its candidate. Arm only after building and
	// reset this private test marker so the incoming Engine call must execute.
	if err := n.db.Delete(append([]byte("validated-block/"), built.ExecutionPayload.BlockHash.Bytes()...)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(db.arm, []byte(`{"minimum_slot":32}`), 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		var status payload.PayloadStatusV1
		err := rpc.invoke("engine_newPayloadV5", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests)
		if err == nil && status.Status != "VALID" {
			err = fmt.Errorf("payload status%+v", status)
		}
		done <- err
	}()
	select {
	case <-db.started:
	case <-time.After(time.Second):
		t.Fatal("actual checked native batch did not reach barrier")
	}
	if n.blockchain.HasValidatedBlock(built.ExecutionPayload.BlockHash) || n.blockchain.CurrentBlock().Hash() != parent {
		t.Fatal("uncommitted block acquired validation/canonical authority")
	}
	var syncing any
	rpc.call(t, "eth_syncing", &syncing)
	if syncing != false {
		t.Fatalf("unrelated upcheck did not remain readable:%v", syncing)
	}
	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("real native commit did not finish after release")
	}
	if !n.blockchain.HasValidatedBlock(built.ExecutionPayload.BlockHash) {
		t.Fatal("native validation marker was not atomically published")
	}
}

func TestNativeCommitStallDiagnostic(t *testing.T) {
	if *nativeCommitStallConfig == "" {
		t.Skip("explicit owned commit-stall diagnostic not requested; not liveness coverage")
	}
	var input struct {
		DataDir     string `json:"datadir"`
		Genesis     string `json:"genesis"`
		JWT         string `json:"jwt"`
		Bootnodes   string `json:"bootnodes"`
		Arm         string `json:"arm"`
		Status      string `json:"status"`
		HoldSeconds int    `json:"hold_seconds"`
		Disposable  bool   `json:"disposable"`
	}
	wire, err := os.ReadFile(*nativeCommitStallConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wire, &input); err != nil {
		t.Fatal(err)
	}
	if !input.Disposable || input.HoldSeconds < 1 || input.HoldSeconds > 180 || !filepath.IsAbs(input.DataDir) || !filepath.IsAbs(input.Arm) || !filepath.IsAbs(input.Status) {
		t.Fatal("bounded explicit owned diagnostic input required")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	c := DefaultConfig()
	c.DataDir, c.GenesisPath, c.JWTSecret = input.DataDir, input.Genesis, input.JWT
	c.NetworkID = 1337
	c.Bootnodes = input.Bootnodes
	c.HTTPAddr, c.AuthAddr, c.MetricsAddr = "0.0.0.0", "0.0.0.0", "0.0.0.0"
	c.RPCPort, c.EnginePort, c.MetricsPort = 8545, 8551, 9001
	c.Metrics = true
	c.RPCRateLimitPerSec = 0
	c.NAT = "none"
	db := &commitStallDB{ctx: ctx, arm: input.Arm, status: input.Status, duration: time.Duration(input.HoldSeconds) * time.Second, started: make(chan struct{})}
	n, err := newWithDatabase(&c, func(path string) (rawdb.Database, error) {
		d, err := rawdb.NewFileDB(path)
		db.Database = d
		return db, err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		if err := n.Stop(); err != nil {
			t.Error(err)
		}
	}()
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	// Test-only read-only profiling on the existing listener. No production
	// flag or listener is added by the fixture.
	mux, ok := n.metricsServer.Handler.(*http.ServeMux)
	if !ok {
		t.Fatal("unexpected native metrics handler")
	}
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	<-ctx.Done()
}
