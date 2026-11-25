package engine

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
)

func TestOwnedListenerImmediateStopAndConcurrentStart(t *testing.T) {
	for iteration := 0; iteration < 30; iteration++ {
		server := NewEngineAPI(nil)
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		address := ln.Addr().String()
		var wg sync.WaitGroup
		wg.Add(2)
		var startErr, stopErr error
		go func() {
			defer wg.Done()
			startErr = server.ServeListener(ln)
			if startErr != nil {
				ln.Close()
			}
		}()
		go func() { defer wg.Done(); stopErr = server.Stop() }()
		wg.Wait()
		if startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			t.Fatal(startErr)
		}
		if stopErr != nil {
			t.Fatal(stopErr)
		}
		if err := server.Stop(); err != nil {
			t.Fatalf("second stop: %v", err)
		}
		reopened, err := net.Listen("tcp", address)
		if err != nil {
			t.Fatalf("listener leaked: %v", err)
		}
		reopened.Close()
	}
}
