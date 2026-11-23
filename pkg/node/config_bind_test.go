package node

import (
	"net"
	"testing"
)

func TestPrivilegedRPCBindUsesActualLoopbackAddress(t *testing.T) {
	for _, host := range []string{"", "127.node.invalid", "127.0.0.999", "127.0.0.1.example"} {
		t.Run(host, func(t *testing.T) {
			c := DefaultConfig()
			c.HTTPAddr = host
			c.HTTPModules = []string{"admin"}
			if err := c.Validate(); err == nil {
				t.Fatalf("non-loopback bind%q accepted as local", host)
			}
		})
	}
	// An empty host is a real wildcard listener, not a local-only alias.
	c := DefaultConfig()
	c.HTTPAddr = ""
	c.RPCPort = 0
	l, err := net.Listen("tcp", c.RPCAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if !l.Addr().(*net.TCPAddr).IP.IsUnspecified() {
		t.Fatalf("unexpected bind%v", l.Addr())
	}
}
func TestIPv6ConfiguredListenAddresses(t *testing.T) {
	c := DefaultConfig()
	c.HTTPAddr, c.AuthAddr, c.WSAddr, c.MetricsAddr = "::1", "::1", "::1", "::1"
	for _, address := range []string{c.HTTPListenAddr(), c.AuthListenAddr(), c.WSListenAddr(), c.MetricsListenAddr()} {
		host, _, err := net.SplitHostPort(address)
		if err != nil || host != "::1" {
			t.Fatalf("invalid configured IPv6 endpoint%q: %v", address, err)
		}
	}
}
