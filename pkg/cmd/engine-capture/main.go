// Command engine-capture records exact private-devnet Engine exchanges.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const captureLimit = 64 << 20

type capture struct {
	upstream  *url.URL
	transport http.RoundTripper
	output    io.Writer
	sequence  atomic.Uint64
	mu        sync.Mutex
	session   string
}

func (c *capture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	sequence := c.sequence.Add(1)
	input, err := io.ReadAll(io.LimitReader(r.Body, captureLimit+1))
	if err != nil || len(input) > captureLimit {
		http.Error(w, "capture input unavailable or over campaign limit", http.StatusRequestEntityTooLarge)
		return
	}
	target := *c.upstream
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery
	request, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), bytes.NewReader(input))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	request.Header = r.Header.Clone()
	request.Host = r.Host
	response, exchangeErr := c.transport.RoundTrip(request)
	var body []byte
	status := http.StatusBadGateway
	if response != nil {
		defer response.Body.Close()
		status = response.StatusCode
		body, err = io.ReadAll(io.LimitReader(response.Body, captureLimit+1))
		if err != nil || len(body) > captureLimit {
			exchangeErr = fmt.Errorf("capture response unavailable or over campaign limit: %v", err)
			status = http.StatusBadGateway
		}
	}
	record := map[string]any{"session": c.session, "sequence": sequence, "started": started.UTC(), "elapsed_ns": time.Since(started).Nanoseconds(), "method": r.Method, "path": r.URL.RequestURI(), "body_encoding": "base64", "request_body": input, "response_body": body, "http_status": status, "complete": exchangeErr == nil}
	if exchangeErr != nil {
		record["transport_error"] = exchangeErr.Error()
	}
	// Headers are forwarded verbatim and excluded from the evidence file.
	c.mu.Lock()
	logErr := json.NewEncoder(c.output).Encode(record)
	c.mu.Unlock()
	if logErr != nil {
		log.Printf("capture evidence write failed: %v", logErr)
		http.Error(w, "capture evidence write failed", http.StatusInsufficientStorage)
		return
	}
	if exchangeErr != nil {
		http.Error(w, "Engine upstream exchange incomplete", http.StatusBadGateway)
		return
	}
	for name, values := range response.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
func main() {
	listen := flag.String("listen", "0.0.0.0:8551", "private campaign capture address")
	upstream := flag.String("upstream", "http://127.0.0.1:18551", "actual Engine server inside this container")
	path := flag.String("output", "/data/engine-exchanges.jsonl", "append-only campaign evidence file")
	flag.Parse()
	target, err := url.Parse(*upstream)
	if err != nil || target.Scheme != "http" || target.Hostname() != "127.0.0.1" {
		log.Fatal("an explicit loopback HTTP Engine target is required")
	}
	file, err := os.OpenFile(*path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	handler := &capture{upstream: target, transport: &http.Transport{Proxy: nil, DisableCompression: true, MaxIdleConns: 32, MaxIdleConnsPerHost: 16, IdleConnTimeout: time.Minute}, output: file, session: time.Now().UTC().Format(time.RFC3339Nano)}
	server := http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
