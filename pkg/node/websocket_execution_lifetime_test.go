package node

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/rpc"
	"github.com/gorilla/websocket"
)

type wsExecutionProbe struct {
	node     *Node
	target   types.Address
	server   *httptest.Server
	entered  chan int
	finished chan int
	mu       sync.Mutex
	calls    map[int]int
	root     types.Hash
}

func newWSExecutionProbe(t *testing.T) *wsExecutionProbe {
	t.Helper()
	raw, _, _, _ := simulationV1Genesis(t, 0)
	var genesis map[string]any
	if err := json.Unmarshal(raw, &genesis); err != nil {
		t.Fatal(err)
	}
	target := types.Address{18: 0x42, 19: 1}
	genesis["alloc"].(map[string]any)[target.Hex()] = map[string]string{"balance": "1", "nonce": "0x1", "code": "0x60016000555b600556"}
	raw, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err = os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	probe := &wsExecutionProbe{node: n, target: target, entered: make(chan int, 64), finished: make(chan int, 64), calls: make(map[int]int), root: n.blockchain.Genesis().Header().Root}
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		raw, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
			return
		}
		request.Body = io.NopCloser(strings.NewReader(string(raw)))
		var decoded struct {
			ID int `json:"id"`
		}
		if err = json.Unmarshal(raw, &decoded); err != nil {
			t.Error(err)
			return
		}
		probe.mu.Lock()
		probe.calls[decoded.ID]++
		probe.mu.Unlock()
		probe.entered <- decoded.ID
		defer func() { probe.finished <- decoded.ID }()
		n.rpcHandler.Handler().ServeHTTP(w, request)
	})
	probe.server = httptest.NewUnstartedServer(nil)
	probe.server.Config = newWSServer(wrapped, "", nil)
	probe.server.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = probe.server.Config.Shutdown(ctx)
	})
	t.Cleanup(probe.server.Close)
	return probe
}
func (p *wsExecutionProbe) dial(t *testing.T) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(p.server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
func (p *wsExecutionProbe) startLoop(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	payload := `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":"` + p.target.Hex() + `","gas":"0x3b9aca00"},"latest"]}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	select {
	case id := <-p.entered:
		if id != 1 {
			t.Fatal(id)
		}
	case <-time.After(time.Second):
		t.Fatal("actual EVM request never entered HTTP dispatch")
	}
	select {
	case <-p.finished:
		t.Fatal("high-gas EVM control completed before cancellation")
	case <-time.After(50 * time.Millisecond):
	}
}
func (p *wsExecutionProbe) assertCancelledAndIsolated(t *testing.T) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		select {
		case id := <-p.finished:
			if id != 1 {
				continue
			}
			state, err := p.node.blockchain.StateAtBlock(p.node.blockchain.Genesis())
			if err != nil {
				t.Fatal(err)
			}
			if state.GetRoot() != p.root || state.GetState(p.target, types.Hash{}) != (types.Hash{}) {
				t.Fatal("WebSocket cancellation changed canonical state")
			}
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.calls[2] != 0 {
				t.Fatal("queued request dispatched after connection cancellation")
			}
			return
		case <-deadline.C:
			t.Fatal("WebSocket retained actual native EVM work after connection close")
			return
		}
	}
}
func TestActualWebSocketDisconnectCancelsExecutionWithQueuedFrame(t *testing.T) {
	p := newWSExecutionProbe(t)
	conn := p.dial(t)
	p.startLoop(t, conn)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":2,"method":"eth_chainId","params":[]}`)); err != nil {
		t.Fatal(err)
	}
	other := p.dial(t)
	if err := other.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":9,"method":"eth_chainId","params":[]}`)); err != nil {
		t.Fatal(err)
	}
	other.SetReadDeadline(time.Now().Add(time.Second))
	var result struct {
		ID     int    `json:"id"`
		Result string `json:"result"`
		Error  any    `json:"error"`
	}
	if err := other.ReadJSON(&result); err != nil || result.ID != 9 || result.Error != nil || result.Result == "" {
		t.Fatalf("concurrent independent WebSocket: %+v %v", result, err)
	}
	conn.Close()
	p.assertCancelledAndIsolated(t)
}
func TestActualWebSocketQueueOverflowCancelsActiveAndQueuedWork(t *testing.T) {
	p := newWSExecutionProbe(t)
	conn := p.dial(t)
	p.startLoop(t, conn)
	for i := 0; i <= maxPendingWSRequests; i++ {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":2,"method":"eth_chainId","params":[]}`)); err != nil {
			break
		}
	}
	p.assertCancelledAndIsolated(t)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("overflowed connection remained usable")
	}
}
func TestActualWebSocketOversizedFrameCancelsActiveExecution(t *testing.T) {
	p := newWSExecutionProbe(t)
	conn := p.dial(t)
	p.startLoop(t, conn)
	_ = conn.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("x", rpc.WSMaxMessageSize+1)))
	p.assertCancelledAndIsolated(t)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("oversized frame did not close connection")
	}
}
func TestActualWebSocketQueuedNormalRequestsRetainOrderAndIDs(t *testing.T) {
	p := newWSExecutionProbe(t)
	conn := p.dial(t)
	for i := 3; i < 8; i++ {
		if err := conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": i, "method": "eth_chainId", "params": []any{}}); err != nil {
			t.Fatal(err)
		}
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	for i := 3; i < 8; i++ {
		var result struct {
			ID     int    `json:"id"`
			Result string `json:"result"`
			Error  any    `json:"error"`
		}
		if err := conn.ReadJSON(&result); err != nil || result.ID != i || result.Result == "" || result.Error != nil {
			t.Fatalf("ordered response %d: %+v %v", i, result, err)
		}
	}
}

func TestActualWebSocketServerShutdownCancelsHijackedExecution(t *testing.T) {
	p := newWSExecutionProbe(t)
	conn := p.dial(t)
	p.startLoop(t, conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.server.Config.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	p.assertCancelledAndIsolated(t)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("shutdown left a hijacked WebSocket usable")
	}
}
