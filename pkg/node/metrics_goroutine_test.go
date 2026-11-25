package node

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

//go:noinline
func waitEngineDiagnosticMarker(ready chan<- struct{}, stop <-chan struct{}) {
	close(ready)
	<-stop
}

func TestMetricsExposeLiveBlockedGoroutine(t *testing.T) {
	c := startupConfig(t)
	n, err := New(&c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Stop() })
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	ready, stop, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() { defer close(done); waitEngineDiagnosticMarker(ready, stop) }()
	<-ready
	defer func() { close(stop); <-done }()
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + n.metricsServer.Addr + "/debug/pprof/goroutine?debug=2")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("live goroutine profile status: %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "waitEngineDiagnosticMarker") || !strings.Contains(text, "chan receive") {
		t.Fatal("profile did not capture the actual blocked marker goroutine")
	}
	other, err := client.Get("http://" + n.metricsServer.Addr + "/debug/pprof/profile")
	if err != nil {
		t.Fatal(err)
	}
	other.Body.Close()
	if other.StatusCode != http.StatusNotFound {
		t.Fatalf("unconfigured CPU profiler status: %d", other.StatusCode)
	}
}
