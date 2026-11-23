package backend

import (
	"errors"
	"os"
	"testing"

	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
)

func TestEngineFencedDatabaseCannotBecomeSyncingOrCachedValid(t *testing.T) {
	dir := t.TempDir()
	db, err := rawdb.NewFileDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	node := newTestBackendNode(t)
	genesis := node.blockchain.Genesis()
	node.blockchain, err = chain.NewBlockchain(config.TestConfig, genesis, state.NewMemoryStateDB(), db)
	if err != nil {
		t.Fatal(err)
	}
	b := NewEngineBackend(node)
	t.Cleanup(b.Close)
	unknown := payload.ForkchoiceStateV1{HeadBlockHash: types.Hash{31: 99}}
	status, err := b.ForkchoiceUpdated(unknown, nil)
	if err != nil || status.PayloadStatus.Status != "SYNCING" {
		t.Fatalf("healthy unknown head = %+v, %v", status, err)
	}
	// Real OS materialization failure after FileDB has appended/synced intent.
	if err := os.Mkdir(rawdb.FileDBValuePath(dir, []byte("fault"))+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	if err := db.Put([]byte("fault"), []byte("durable intent")); err == nil {
		t.Fatal("expected actual materialization error")
	}
	for _, head := range []types.Hash{unknown.HeadBlockHash, genesis.Hash()} {
		status, err := b.ForkchoiceUpdated(payload.ForkchoiceStateV1{HeadBlockHash: head}, nil)
		if !errors.Is(err, chain.ErrStorageFailure) || status.PayloadStatus.Status != "" {
			t.Fatalf("fenced forkchoice %s = %+v, %v", head, status, err)
		}
		if err := node.blockchain.ApplyForkchoice(head, types.Hash{}, types.Hash{}); !errors.Is(err, chain.ErrStorageFailure) {
			t.Fatalf("fenced direct forkchoice %s = %v", head, err)
		}
	}
	statusPayload, err := b.ProcessBlock(&payload.ExecutionPayloadV3{}, nil, types.Hash{})
	if !errors.Is(err, chain.ErrStorageFailure) || statusPayload.Status != "" {
		t.Fatalf("fenced payload = %+v, %v", statusPayload, err)
	}
}
