//go:build canonicalproof_integration

package node

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/eth2030/eth2030/zkvm/canonicalproof"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestRealCanonicalProofThroughNativeNode(t *testing.T) {
	if !(canonicalproof.Prover{}).Available() {
		t.Fatal("real-proof integration requires the packaged prover")
	}
	cfg := makeTestConfig(t)
	n := newTestNode(t, &cfg)
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	wire := track4RPC{url: "http://" + n.rpcServer.Addr().String()}
	var program []byte
	for _, word := range []uint32{0x00200893, 0x73, 0x00100893, 0x73, 0x00000513, 0x00000893, 0x73} {
		program = binary.LittleEndian.AppendUint32(program, word)
	}
	machine := canonicalproof.Config{MaxCycles: 100, MemoryPages: 4, StackBase: 0x80000000}
	request := map[string]any{"program": hexutil.Encode(program), "input": "0x2a", "config": machine}
	var job canonicalproof.Job
	start := time.Now()
	wire.call(t, "zkvm_proveExecution", &job, request)
	if job.ID == "" || time.Since(start) > 3*time.Second {
		t.Fatal("proving submission blocked HTTP")
	}
	deadline := time.Now().Add(4 * time.Minute)
	for time.Now().Before(deadline) {
		var height string
		wire.call(t, "eth_blockNumber", &height)
		if height != "0x0" {
			t.Fatal("proof task changed execution chain")
		}
		wire.call(t, "zkvm_getProof", &job, job.ID)
		if job.Status == "complete" {
			break
		}
		if job.Status != "running" {
			t.Fatalf("real proving failed: %+v", job)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if job.Status != "complete" || job.Proof == nil {
		t.Fatal("real proof did not complete")
	}
	if job.Proof.Statement.Output != "2a" || job.Proof.Statement.Cycles != 7 {
		t.Fatal(job.Proof.Statement)
	}
	verify := map[string]any{"program": hexutil.Encode(program), "input": "0x2a", "config": machine, "proof": job.Proof}
	var result struct {
		Valid     bool
		Statement canonicalproof.Statement
	}
	wire.call(t, "zkvm_verifyExecution", &result, verify)
	if !result.Valid || result.Statement != job.Proof.Statement {
		t.Fatal("node did not independently verify its proof")
	}
	verify["input"] = "0x2b"
	wire.call(t, "zkvm_verifyExecution", &result, verify)
	if result.Valid {
		t.Fatal("node accepted changed input")
	}
	verify["input"] = "0x2a"
	copy := *job.Proof
	copy.Statement.StateHash = "0000000000000000000000000000000000000000000000000000000000000000"
	verify["proof"] = &copy
	wire.call(t, "zkvm_verifyExecution", &result, verify)
	if result.Valid {
		t.Fatal("node accepted changed final state")
	}
	verify["proof"] = job.Proof
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	n = newTestNode(t, &cfg)
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	wire = track4RPC{url: "http://" + n.rpcServer.Addr().String()}
	// Job handles belong to a node lifetime; the actual proof artifact remains
	// independently verifiable after restart with no producer cache or secret.
	if err := wire.invoke("zkvm_getProof", &canonicalproof.Job{}, job.ID); err == nil {
		t.Fatal("stale job handle survived as an invented result")
	}
	wire.call(t, "zkvm_verifyExecution", &result, verify)
	if !result.Valid {
		t.Fatal("proof artifact did not survive verifier restart")
	}
	t.Logf("actual local proof, native HTTP verification, tamper rejection and restart passed in %s", time.Since(start))
}
