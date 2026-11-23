package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEngineVhostCheck covers the --authrpc.vhosts enforcement on the live
// handler: foreign Host headers are rejected before auth is even evaluated.
func TestEngineVhostCheck(t *testing.T) {
	tests := []struct {
		name   string
		vhosts []string
		host   string
		want   int
	}{
		// Passing vhosts fall through to the dispatcher (GET => 405 since no
		// JWT is configured and auth is disabled without one).
		{"empty vhosts allows all", nil, "attacker.example", http.StatusMethodNotAllowed},
		{"star allows all", []string{"*"}, "cl-1-lighthouse", http.StatusMethodNotAllowed},
		{"localhost listed accepts", []string{"localhost"}, "localhost:8551", http.StatusMethodNotAllowed},
		{"localhost listed rejects foreign", []string{"localhost"}, "attacker.example", http.StatusForbidden},
		{"ipv6 bracketed accepted", []string{"::1"}, "[::1]:8551", http.StatusMethodNotAllowed},
		{"case-insensitive", []string{"localhost"}, "LOCALHOST", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := NewEngineAPI(nil)
			api.SetAuthVhosts(tt.vhosts)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host
			res := httptest.NewRecorder()
			api.httpHandler(res, req)
			if res.Code != tt.want {
				t.Fatalf("Host %q with vhosts %v: HTTP %d, want %d", tt.host, tt.vhosts, res.Code, tt.want)
			}
		})
	}
}
