package node

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestMetricsServer_StartsAndResponds(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.DataDir = dir
	cfg.P2PPort = 0
	cfg.RPCPort = 0
	cfg.EnginePort = 0
	cfg.Metrics = true
	cfg.MetricsAddr = "127.0.0.1"
	cfg.MetricsPort = 0 // ephemeral — we'll probe directly

	n, err := New(&cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := n.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer func() { _ = n.Stop() }()

	// Start must reserve a real metrics listener before it returns. A fixed
	// high test port can be allocated to the node's ephemeral P2P listener.
	_, port, err := net.SplitHostPort(n.metricsServer.Addr)
	if err != nil || port == "0" {
		t.Fatalf("metrics listener not bound after Start: %q (%v)", n.metricsServer.Addr, err)
	}
	url := "http://" + n.metricsServer.Addr + "/debug/vars"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s error: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics status = %d, want 200", resp.StatusCode)
	}
	var vars map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&vars); err != nil {
		t.Fatalf("decode expvar response: %v", err)
	}
	if len(vars["memstats"]) == 0 {
		t.Fatal("response did not come from the expvar metrics handler")
	}
	if err := n.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
	reopened, err := net.Listen("tcp", n.metricsServer.Addr)
	if err != nil {
		t.Fatalf("metrics listener not released by Stop: %v", err)
	}
	reopened.Close()
}

func TestMetricsServer_DisabledByDefault(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.DataDir = dir
	cfg.P2PPort = 0
	cfg.RPCPort = 0
	cfg.EnginePort = 0
	// Metrics disabled by default.

	n, err := New(&cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := n.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer func() { _ = n.Stop() }()

	if n.metricsServer != nil {
		t.Error("metricsServer should be nil when metrics disabled")
	}
}
