package node

import (
	"bytes"
	"errors"
	"github.com/eth2030/eth2030/core/types"
	synchealer "github.com/eth2030/eth2030/sync/healer"
	syncstatesync "github.com/eth2030/eth2030/sync/statesync"
	"testing"
)

func TestUnwiredStateBackendsFailClosed(t *testing.T) {
	snap := &stubSnapStateBackend{}
	healer := &stubStateWriter{}
	syncer := &stubStateSyncWriter{}
	errs := []error{snap.AccountIterator(types.Hash{}, types.Hash{}, nil), snap.StorageIterator(types.Hash{}, types.Hash{}, nil, nil), healer.WriteAccount(types.Hash{}, synchealer.AccountData{}), healer.WriteStorage(types.Hash{}, types.Hash{}, nil), healer.WriteBytecode(types.Hash{}, nil), healer.WriteTrieNode(nil, nil), syncer.WriteAccount(types.Hash{}, syncstatesync.AccountData{}), syncer.WriteStorage(types.Hash{}, types.Hash{}, nil), syncer.WriteBytecode(types.Hash{}, nil), syncer.WriteTrieNode(nil, nil)}
	_, err := snap.AccountProof(types.Hash{}, types.Hash{})
	errs = append(errs, err)
	_, err = snap.TrieNode(types.Hash{}, nil)
	errs = append(errs, err)
	for i, err := range errs {
		if !errors.Is(err, errStateBackendUnsupported) {
			t.Fatalf("operation %d claimed success: %v", i, err)
		}
	}
}

func TestNativeRuntimePathAndUnsupportedBogotaAPI(t *testing.T) {
	cfg := makeTestConfig(t)
	n := newTestNode(t, &cfg)
	if n.stateHealer != nil || n.stateSyncSched != nil {
		t.Fatal("unwired state repair subsystem activated")
	}
	if n.engBackend == nil || n.engineServer == nil || n.Blockchain() == nil {
		t.Fatal("native path not wired")
	}
	// Inspect the actual node backend through its wire dispatcher; newer methods
	// must not silently route to an older payload schema.
	for _, method := range []string{"engine_newPayloadV6", "engine_forkchoiceUpdatedV5"} {
		raw := n.engineServer.HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":[]}`))
		if !bytes.Contains(raw, []byte(`-32601`)) {
			t.Fatalf("unsupported method %s accepted: %s", method, raw)
		}
	}
	t.Logf("runtime=native node=%T engine=%T processor=core/execution.StateProcessor", n, n.engBackend)
}
