package node

import "testing"

// TestDefaultConfig_SecureRPCDefaults pins the secure-by-default RPC posture:
// loopback binds and public namespaces only (CWE-862 regression guard).
func TestDefaultConfig_SecureRPCDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !isLoopbackHost(cfg.HTTPAddr) {
		t.Errorf("HTTPAddr default %q must be loopback", cfg.HTTPAddr)
	}
	if !isLoopbackHost(cfg.AuthAddr) {
		t.Errorf("AuthAddr default %q must be loopback", cfg.AuthAddr)
	}
	if !isLoopbackHost(cfg.WSAddr) {
		t.Errorf("WSAddr default %q must be loopback", cfg.WSAddr)
	}
	if !isLoopbackHost(cfg.MetricsAddr) {
		t.Errorf("MetricsAddr default %q must be loopback", cfg.MetricsAddr)
	}
	for _, m := range []string{"admin", "debug", "engine"} {
		for _, set := range [][]string{cfg.HTTPModules, cfg.WSModules} {
			for _, got := range set {
				if got == m {
					t.Errorf("privileged module %q must not be in default module set %v", m, set)
				}
			}
		}
	}
	if cfg.RPCAuthSecret != "" {
		t.Error("RPCAuthSecret must default to empty (opt-in)")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"", false},
		{"localhost", true},
		{"127.0.0.1", true},
		{"127.9.9.9", true},
		{"::1", true},
		{"localhost:8545", true},
		{"127.0.0.1:8545", true},
		{"0.0.0.0", false},
		{"::", false},
		{"192.168.1.5", false},
		{"[::1]:8545", false}, // bracketed form is not a config bind address
		{"node-1.example.com", false},
	}
	for _, tt := range tests {
		if got := isLoopbackHost(tt.addr); got != tt.want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

// TestConfigValidation_RemotePrivilegedRPC covers the guardrail: admin/debug
// on a non-loopback bind require --rpc.auth_secret or --rpc.allow-insecure.
func TestConfigValidation_RemotePrivilegedRPC(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:   "loopback with admin needs no secret",
			modify: func(c *Config) { c.HTTPModules = append(c.HTTPModules, "admin", "debug") },
		},
		{
			name: "wildcard bind with admin denied",
			modify: func(c *Config) {
				c.HTTPAddr = "0.0.0.0"
				c.HTTPModules = append(c.HTTPModules, "admin")
			},
			wantErr: true,
		},
		{
			name: "wildcard bind with debug denied",
			modify: func(c *Config) {
				c.HTTPAddr = "0.0.0.0"
				c.HTTPModules = append(c.HTTPModules, "debug")
			},
			wantErr: true,
		},
		{
			name: "lan ip with admin denied",
			modify: func(c *Config) {
				c.HTTPAddr = "192.168.1.10"
				c.HTTPModules = append(c.HTTPModules, "admin", "debug")
			},
			wantErr: true,
		},
		{
			name: "wildcard bind with admin plus secret allowed",
			modify: func(c *Config) {
				c.HTTPAddr = "0.0.0.0"
				c.HTTPModules = append(c.HTTPModules, "admin", "debug")
				c.RPCAuthSecret = "s3cret"
			},
		},
		{
			name: "wildcard bind with admin plus allow-insecure allowed",
			modify: func(c *Config) {
				c.HTTPAddr = "0.0.0.0"
				c.HTTPModules = append(c.HTTPModules, "admin", "debug")
				c.RPCAllowInsecureRemoteAdmin = true
			},
		},
		{
			name:   "wildcard bind public modules only allowed",
			modify: func(c *Config) { c.HTTPAddr = "0.0.0.0" },
		},
		{
			name: "ws disabled wildcard admin ignored",
			modify: func(c *Config) {
				c.WSAddr = "0.0.0.0"
				c.WSModules = []string{"admin"}
			},
		},
		{
			name: "ws enabled wildcard admin denied",
			modify: func(c *Config) {
				c.WSEnabled = true
				c.WSAddr = "0.0.0.0"
				c.WSModules = []string{"admin"}
			},
			wantErr: true,
		},
		{
			name: "ws enabled wildcard public modules allowed",
			modify: func(c *Config) {
				c.WSEnabled = true
				c.WSAddr = "0.0.0.0"
			},
		},
		{
			name: "ws enabled wildcard admin with allow-insecure allowed",
			modify: func(c *Config) {
				c.WSEnabled = true
				c.WSAddr = "0.0.0.0"
				c.WSModules = []string{"admin"}
				c.RPCAllowInsecureRemoteAdmin = true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRPCCORSAllowedOrigins(t *testing.T) {
	// Empty default means CORS off (nil), not allow-all.
	cfg := DefaultConfig()
	if got := cfg.RPCCORSAllowedOrigins(); got != nil {
		t.Errorf("empty RPCCORSOrigins should yield nil (no CORS headers), got %v", got)
	}
	cfg.RPCCORSOrigins = "*"
	got := cfg.RPCCORSAllowedOrigins()
	if len(got) != 1 || got[0] != "*" {
		t.Errorf("RPCCORSOrigins=* should yield [\"*\"], got %v", got)
	}
	cfg.RPCCORSOrigins = " http://a.com, http://b.com ,http://a.com"
	got = cfg.RPCCORSAllowedOrigins()
	if len(got) != 2 || got[0] != "http://a.com" || got[1] != "http://b.com" {
		t.Errorf("expected deduped trimmed origins, got %v", got)
	}
}
