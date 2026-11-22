package node

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

func startupConfig(t *testing.T) Config {
	t.Helper()
	c := DefaultConfig()
	c.DataDir = t.TempDir()
	c.P2PPort = 0
	c.RPCPort = 0
	c.EnginePort = 0
	c.HTTPAddr = "127.0.0.1"
	c.AuthAddr = "127.0.0.1"
	c.WSEnabled = true
	c.WSAddr = "127.0.0.1"
	c.WSPort = 0
	c.Metrics = true
	c.MetricsAddr = "127.0.0.1"
	c.MetricsPort = 0
	return c
}

func requireStartupRolledBack(t *testing.T, n *Node) {
	t.Helper()
	if n.Running() {
		t.Error("failed node reports running")
	}
	// Rebind every actually reserved address, including sockets never passed
	// to an HTTP serving goroutine when a later startup stage failed.
	for _, ln := range n.listeners {
		reopened, err := net.Listen("tcp", ln.Addr().String())
		if err != nil {
			t.Errorf("reserved listener leaked: %v", err)
		} else {
			reopened.Close()
		}
	}
	if addr := n.p2pServer.ListenAddr(); addr != nil {
		reopened, err := net.Listen("tcp", addr.String())
		if err != nil {
			t.Errorf("P2P listener leaked: %v", err)
		} else {
			reopened.Close()
		}
	}
	if n.db != nil {
		t.Error("failed startup retains its database lock")
	}
	if n.dasNetMgr.IsStarted() || n.lightClient.IsRunning() || n.starkAgg.IsRunning() {
		t.Error("failed startup retains an active subsystem")
	}
	select {
	case <-n.stop:
	default:
		t.Error("Wait was not released after failed startup")
	}
	if err := n.Stop(); err != nil {
		t.Errorf("idempotent Stop: %v", err)
	}
	if err := n.Start(); err == nil {
		t.Error("terminal failed instance accepted Start")
	}
}

func TestStartupRequiredBindingsRollbackAndFreshRetry(t *testing.T) {
	for _, role := range []string{"p2p", "rpc", "engine", "ws", "metrics"} {
		t.Run(role, func(t *testing.T) {
			held, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer held.Close()
			c := startupConfig(t)
			port := held.Addr().(*net.TCPAddr).Port
			switch role {
			case "p2p":
				c.P2PPort = port
			case "rpc":
				c.RPCPort = port
			case "engine":
				c.EnginePort = port
			case "ws":
				c.WSPort = port
			case "metrics":
				c.MetricsPort = port
			}
			n, err := New(&c)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = n.Stop() })
			err = n.Start()
			if err == nil {
				t.Fatal("occupied required " + role + " listener accepted")
			}
			if !errors.Is(err, syscall.EADDRINUSE) {
				t.Errorf("bind cause not preserved: %v", err)
			}
			if !strings.Contains(strings.ToLower(err.Error()), role) {
				t.Errorf("error does not identify %s: %v", role, err)
			}
			requireStartupRolledBack(t, n)
			if err := held.Close(); err != nil {
				t.Fatal(err)
			}
			retry, err := New(&c)
			if err != nil {
				t.Fatalf("fresh New on original datadir: %v", err)
			}
			t.Cleanup(func() { _ = retry.Stop() })
			if err := retry.Start(); err != nil {
				t.Fatalf("ordinary retry: %v", err)
			}
			for _, address := range []net.Addr{retry.rpcServer.Addr(), retry.engineServer.Addr(), retry.p2pServer.ListenAddr()} {
				if address == nil {
					t.Fatal("successful startup has no bound required address")
				}
			}
			client := &http.Client{Timeout: 2 * time.Second}
			response, err := client.Get("http://" + retry.metricsServer.Addr + "/debug/vars")
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatalf("metrics readiness: %d", response.StatusCode)
			}
			if err := retry.Stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStartupWorkerFailureClosesConstructorResources(t *testing.T) {
	c := startupConfig(t)
	n, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Stop() })
	// A real subsystem error after earlier starts, without a fake socket error.
	if err := n.starkAgg.Start(); err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err == nil {
		t.Fatal("already-running aggregation worker accepted")
	}
	requireStartupRolledBack(t, n)
	fresh, err := New(&c)
	if err != nil {
		t.Fatalf("constructor lock leaked: %v", err)
	}
	fresh.Stop()
}

func TestStartupDisabledEndpointsDoNotBind(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	c := startupConfig(t)
	c.WSEnabled = false
	c.Metrics = false
	c.WSPort = held.Addr().(*net.TCPAddr).Port
	c.MetricsPort = c.WSPort
	n, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Stop()
	if err := n.Start(); err != nil {
		t.Fatalf("disabled optional endpoints affected readiness: %v", err)
	}
	if n.wsServer != nil || n.metricsServer != nil {
		t.Fatal("disabled endpoints were started")
	}
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(n.lifecycleCtx.Err(), context.Canceled) {
		t.Fatal("node-owned context not canceled")
	}
}

func TestNodeDNSResolverHonorsLifecycleCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (nodeDNSResolver{ctx}).LookupTXT("example.invalid"); !errors.Is(err, context.Canceled) {
		t.Fatalf("resolver ignored shutdown: %v", err)
	}
}

func TestStartupLateServerFailureReleasesAllOwners(t *testing.T) {
	c := startupConfig(t)
	n, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Stop() })
	// Real server lifecycle failure after P2P, workers and RPC have started.
	if err := n.engineServer.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err == nil {
		t.Fatal("closed Engine server accepted startup")
	}
	requireStartupRolledBack(t, n)
	fresh, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Stop()
	if err := fresh.Start(); err != nil {
		t.Fatalf("fresh retry after late rollback: %v", err)
	}
}
