package rpcserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	rpcbatch "github.com/eth2030/eth2030/rpc/batch"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

// stubAdminHandler answers every admin_ request with success; used to prove
// the module allowlist gates privileged namespaces before dispatch.
type stubAdminHandler struct{}

func (s *stubAdminHandler) HandleAdminRequest(req *rpctypes.Request) *rpctypes.Response {
	return rpctypes.NewSuccessResponse(req.ID, "admin-ok")
}

// postMethod sends a single JSON-RPC request to ts and decodes the response.
func postMethod(t *testing.T, ts *httptest.Server, method string) *rpctypes.Response {
	t.Helper()
	body := `{"jsonrpc":"2.0","method":"` + method + `","params":[],"id":1}`
	resp, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", method, err)
	}
	defer resp.Body.Close()
	var out rpctypes.Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response for %s: %v", method, err)
	}
	return &out
}

// --- methodAllowed / hostAllowed unit tests ---

func TestMethodAllowed(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		modules []string
		want    bool
	}{
		{"nil denies all", "eth_chainId", nil, false},
		{"eth listed", "eth_chainId", []string{"eth", "net"}, true},
		{"net listed", "net_version", []string{"eth", "net"}, true},
		{"admin not listed", "admin_peers", []string{"eth", "net"}, false},
		{"debug not listed", "debug_traceTransaction", []string{"eth", "net"}, false},
		{"wildcard allows admin", "admin_addPeer", []string{"*"}, true},
		{"wildcard allows debug", "debug_setHead", []string{"*"}, true},
		{"case-insensitive", "ETH_chainId", []string{"eth"}, true},
		{"no underscore is own namespace", "rpc_modules", []string{"rpc"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := methodAllowed(tt.method, tt.modules); got != tt.want {
				t.Fatalf("methodAllowed(%q, %v) = %v, want %v", tt.method, tt.modules, got, tt.want)
			}
		})
	}
}

func TestHostAllowed(t *testing.T) {
	mkReq := func(host string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Host = host
		return r
	}
	tests := []struct {
		name   string
		host   string
		vhosts []string
		want   bool
	}{
		{"empty vhosts allows all", "evil.example", nil, true},
		{"star allows all", "evil.example", []string{"*"}, true},
		{"localhost listed", "localhost:8545", []string{"localhost"}, true},
		{"loopback listed", "127.0.0.1:8545", []string{"localhost", "127.0.0.1"}, true},
		{"foreign host rejected", "evil.example", []string{"localhost"}, false},
		{"rebind host rejected", "attacker.rebind.example:8545", []string{"localhost"}, false},
		{"case-insensitive", "LOCALHOST:8545", []string{"localhost"}, true},
		{"ipv6 bracketed", "[::1]:8545", []string{"::1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostAllowed(mkReq(tt.host), tt.vhosts); got != tt.want {
				t.Fatalf("hostAllowed(%q, %v) = %v, want %v", tt.host, tt.vhosts, got, tt.want)
			}
		})
	}
}

// --- plain Server module enforcement ---

func TestServer_AllowedModules_GatesAdmin(t *testing.T) {
	srv := NewServer(newStubHandler())
	srv.SetAdminHandler(&stubAdminHandler{})

	// nil modules denies everything, including admin with a wired handler.
	ts := httptest.NewServer(srv.Handler())
	resp := postMethod(t, ts, "admin_nodeInfo")
	if resp.Error == nil || resp.Error.Code != rpctypes.ErrCodeMethodNotFound {
		t.Fatalf("nil modules should deny admin_nodeInfo, got %+v", resp.Error)
	}
	ts.Close()

	// Listing eth serves eth but still denies admin.
	srv.SetAllowedModules([]string{"eth"})
	ts = httptest.NewServer(srv.Handler())
	defer ts.Close()
	if resp := postMethod(t, ts, "eth_chainId"); resp.Error != nil {
		t.Fatalf("eth_chainId should be served: %s", resp.Error.Message)
	}
	if resp := postMethod(t, ts, "admin_nodeInfo"); resp.Error == nil || resp.Error.Code != rpctypes.ErrCodeMethodNotFound {
		t.Fatalf("eth-only modules should deny admin_nodeInfo, got %+v", resp.Error)
	}
}

// --- ExtServer module + vhost enforcement ---

func TestExtServer_AllowedModules_GateDispatchAndBatch(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.AllowedModules = []string{"eth"}
	srv := NewExtServer(newStubHandler(), cfg)
	srv.SetAdminHandler(&stubAdminHandler{})

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	if resp := postMethod(t, ts, "eth_chainId"); resp.Error != nil {
		t.Fatalf("eth_chainId should be served: %s", resp.Error.Message)
	}
	resp := postMethod(t, ts, "admin_nodeInfo")
	if resp.Error == nil || resp.Error.Code != rpctypes.ErrCodeMethodNotFound {
		t.Fatalf("admin_nodeInfo should be denied by allowlist, got %+v", resp.Error)
	}

	// Batch items go through the same allowlist.
	body := `[{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1},
	          {"jsonrpc":"2.0","method":"admin_peers","params":[],"id":2}]`
	httpResp, err := http.Post(ts.URL, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("batch POST: %v", err)
	}
	defer httpResp.Body.Close()
	var batch []rpctypes.Response
	if err := json.NewDecoder(httpResp.Body).Decode(&batch); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if len(batch) != 2 {
		t.Fatalf("want 2 batch responses, got %d", len(batch))
	}
	if batch[0].Error != nil {
		t.Fatalf("eth item should be served: %s", batch[0].Error.Message)
	}
	if batch[1].Error == nil || batch[1].Error.Code != rpctypes.ErrCodeMethodNotFound {
		t.Fatalf("admin item should be denied, got %+v", batch[1].Error)
	}
}

func TestExtServer_AllowedVhosts_RejectForeignHost(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.AllowedModules = []string{"*"}
	cfg.AllowedVhosts = []string{"localhost"}
	srv := NewExtServer(newStubHandler(), cfg)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`
	req, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Host = "attacker.rebind.example"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign Host should be 403, got %d", resp.StatusCode)
	}

	// A Host matching the vhost (with port) passes.
	req2, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req2.Host = "localhost:8545"
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("vhost-matching request should be served, got %d", resp2.StatusCode)
	}
}

// --- batch handler module enforcement ---

func TestBatchHandler_AllowedModules(t *testing.T) {
	bh := rpcbatch.NewBatchHandler(newStubHandler())
	bh.SetAdminHandler(&stubAdminHandler{})
	bh.SetAllowedModules([]string{"eth"})

	body := []byte(`[{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}]`)
	responses, err := bh.HandleBatch(body)
	if err != nil {
		t.Fatalf("HandleBatch: %v", err)
	}
	if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != rpctypes.ErrCodeMethodNotFound {
		t.Fatalf("batch admin item should be denied, got %+v", responses)
	}
}
