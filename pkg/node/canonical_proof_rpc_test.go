package node

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/zkvm/canonicalproof"
)

func TestCanonicalProofRPCBoundsAndUnavailableBackend(t *testing.T) {
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = "canonical-rpc"
	n := newTestNode(t, &cfg)
	n.canonicalProver.Close()
	n.canonicalProver = canonicalproof.NewService(context.Background(), canonicalproof.Prover{Binary: filepath.Join(t.TempDir(), "missing-prover")})
	wire := track4NodeRPC(t, n, cfg.EngineAuthToken)
	for _, tc := range []struct {
		method string
		param  any
		code   int
	}{
		{"zkvm_proveExecution", map[string]any{"program": "0x", "input": "0x"}, -32602},
		{"zkvm_proveExecution", map[string]any{"program": "0x73000000", "input": "0x", "extra": true}, -32602},
		{"zkvm_proveExecution", map[string]any{"program": "0x73000000", "input": "0x"}, -32001},
		{"zkvm_verifyExecution", map[string]any{"program": "0x73000000", "input": "0x"}, -32602},
		{"zkvm_getProof", "bad-job-id", -32602},
		{"zkvm_cancelProof", "00000000000000000000000000000000", -32000},
	} {
		var result any
		err := wire.invoke(tc.method, &result, tc.param)
		if detail, ok := err.(*track4RPCError); !ok || detail.Code != tc.code {
			t.Fatalf("%s: result=%v error=%v", tc.method, result, err)
		}
	}
}
