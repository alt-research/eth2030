package node

import (
	"bytes"
	"github.com/eth2030/eth2030/core/types"
	"testing"
)

func TestNativeSnapRejectsUnknownStateRoots(t *testing.T) {
	cfg := makeTestConfig(t)
	n := newTestNode(t, &cfg)
	backend := newSnapStateBackend(n.Blockchain(), n.db)
	root := types.Hash{0xff}
	if err := backend.AccountIterator(root, types.Hash{}, func(types.Hash, []byte) bool { return true }); err == nil {
		t.Fatal("unknown root accepted")
	}
	if _, err := backend.AccountProof(root, types.Hash{}); err == nil {
		t.Fatal("unknown root proof claimed success")
	}
	if _, err := backend.TrieNode(root, nil); err == nil {
		t.Fatal("unknown trie node claimed success")
	}
}

func TestNativeRuntimePathAndVersionedBogotaAPI(t *testing.T) {
	cfg := makeTestConfig(t)
	n := newTestNode(t, &cfg)
	if n.engBackend == nil || n.engineServer == nil || n.Blockchain() == nil {
		t.Fatal("native path not wired")
	}
	// Exercise the real native dispatcher. Master now implements Bogotá;
	// malformed calls must be rejected by the versioned schema, not fall through.
	for _, method := range []string{"engine_newPayloadV6", "engine_forkchoiceUpdatedV5"} {
		raw := n.engineServer.HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":[]}`))
		if !bytes.Contains(raw, []byte(`-32602`)) {
			t.Fatalf("versioned schema %s: %s", method, raw)
		}
	}
	raw := n.engineServer.HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"engine_newPayloadV99","params":[]}`))
	if !bytes.Contains(raw, []byte(`-32601`)) {
		t.Fatalf("unknown schema accepted: %s", raw)
	}
	t.Logf("runtime=native node=%T engine=%T processor=core/execution.StateProcessor", n, n.engBackend)
}
