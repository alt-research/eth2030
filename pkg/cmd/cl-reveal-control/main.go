// cl-reveal-control is private-devnet transport instrumentation. It preserves
// signed bytes and every client validation; only the first chosen slot's
// envelope is delivered to the observer before the normal network sees it.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type controller struct {
	upstream, observer string
	slot               uint64
	slotFile           string
	signatureProbeDir  string
	client             *http.Client
	proxy              *httputil.ReverseProxy
	mu                 sync.Mutex
	routed             bool
	logMu              sync.Mutex
	output             io.Writer
}

// signatureProbe permits an offline signer to replace only the fixed BLS
// signature of the chosen original SSZ message. The real observer must reject
// it before the untouched producer bytes are delivered normally.
func (c *controller) signatureProbe(ctx context.Context, input []byte, headers http.Header) error {
	if c.signatureProbeDir == "" {
		return nil
	}
	if headers.Get("Content-Type") != "application/octet-stream" || headers.Get("Eth-Blob-Data-Included") != "true" || len(input) < 12 {
		return fmt.Errorf("signature probe requires original stateless SSZ Contents")
	}
	offset := uint64(binary.LittleEndian.Uint32(input[:4]))
	if offset < 12 || offset+100 > uint64(len(input)) {
		return fmt.Errorf("signature probe Contents offset is invalid")
	}
	file, err := os.OpenFile(filepath.Join(c.signatureProbeDir, "probe-input.ssz"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(input)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	deadline, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	for {
		signature, err := os.ReadFile(filepath.Join(c.signatureProbeDir, "probe-signature.bin"))
		if err == nil {
			if len(signature) != 96 || bytes.Equal(signature, input[offset+4:offset+100]) {
				return fmt.Errorf("probe needs one distinct 96-byte BLS signature")
			}
			altered := bytes.Clone(input)
			copy(altered[offset+4:offset+100], signature)
			body, status, err := c.requestHeaders(deadline, "POST", c.observer+"/eth/v1/beacon/execution_payload_envelopes?broadcast_validation=consensus", altered, headers)
			c.record("replaced-key-signature-probe", altered, body, status, err)
			if err != nil || status < 400 || status >= 500 {
				return fmt.Errorf("observer did not reject replaced-key signature: %d %v", status, err)
			}
			root, _, err := envelopeIdentity(input, headers)
			if err != nil {
				return err
			}
			body, status, err = c.request(deadline, "GET", c.observer+"/eth/v1/beacon/execution_payload_envelopes/"+root, nil)
			c.record("replaced-key-availability-after-rejection", nil, body, status, err)
			if err != nil || status != http.StatusNotFound {
				return fmt.Errorf("rejected signature became available: %d %v", status, err)
			}
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		select {
		case <-deadline.Done():
			return deadline.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// The producer rebuilds its signed-preference cache after reconnecting. Keep
// transport forwarding normal during warm-up, then select one actual future
// duty without restarting the producer and losing that cache again.
func (c *controller) controlSlot() uint64 {
	if c.slotFile != "" {
		if data, err := os.ReadFile(c.slotFile); err == nil {
			if slot, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil && slot > 0 {
				return slot
			}
		}
	}
	return c.slot
}

func (c *controller) record(action string, input, output []byte, status int, err error) {
	record := map[string]any{"at": time.Now().UTC(), "action": action, "slot": c.controlSlot(), "request_body": input, "response_body": output, "http_status": status, "complete": err == nil}
	if err != nil {
		record["error"] = err.Error()
	}
	c.logMu.Lock()
	defer c.logMu.Unlock()
	if err := json.NewEncoder(c.output).Encode(record); err != nil {
		log.Printf("control evidence write failed: %v", err)
	}
}

func (c *controller) request(ctx context.Context, method, target string, body []byte) ([]byte, int, error) {
	return c.requestHeaders(ctx, method, target, body, nil)
}

func (c *controller) requestHeaders(ctx context.Context, method, target string, body []byte, headers http.Header) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	if headers != nil {
		req.Header = headers.Clone()
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Eth-Consensus-Version", "gloas")
		req.Header.Set("Eth-Blob-Data-Included", "false")
	}
	response, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<20)+1))
	if len(data) > 64<<20 {
		err = fmt.Errorf("control response exceeds capture limit")
	}
	return data, response.StatusCode, err
}

type signedBlock struct {
	Message struct {
		Slot       string `json:"slot"`
		ParentRoot string `json:"parent_root"`
		Body       struct {
			Bid struct {
				Message struct {
					BlockHash  string `json:"block_hash"`
					ParentHash string `json:"parent_block_hash"`
				} `json:"message"`
			} `json:"signed_execution_payload_bid"`
		} `json:"body"`
	} `json:"message"`
}

func (c *controller) block(ctx context.Context, root string) (json.RawMessage, signedBlock, error) {
	data, status, err := c.request(ctx, "GET", c.upstream+"/eth/v2/beacon/blocks/"+root, nil)
	var wrapper struct{ Data json.RawMessage }
	var block signedBlock
	if err != nil || status != 200 {
		return nil, block, fmt.Errorf("read original signed block %s: status %d: %v", root, status, err)
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, block, err
	}
	if err := json.Unmarshal(wrapper.Data, &block); err != nil {
		return nil, block, err
	}
	return wrapper.Data, block, nil
}

func (c *controller) mirrorEnvelope(ctx context.Context, root string) error {
	// Re-importing an already available envelope is unnecessary and API
	// versions may classify that duplicate differently. This does not set
	// availability: the observer still validates every subsequent child.
	_, known, _ := c.request(ctx, "GET", c.observer+"/eth/v1/beacon/execution_payload_envelopes/"+root, nil)
	if known == 200 {
		return nil
	}
	data, status, err := c.request(ctx, "GET", c.upstream+"/eth/v1/beacon/execution_payload_envelopes/"+root, nil)
	if err != nil || status != 200 {
		return fmt.Errorf("original envelope %s unavailable: %d %v", root, status, err)
	}
	var wrapper struct{ Data json.RawMessage }
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	body, status, err := c.request(ctx, "POST", c.observer+"/eth/v1/beacon/execution_payload_envelopes?broadcast_validation=consensus", wrapper.Data)
	c.record("mirror-original-envelope", wrapper.Data, body, status, err)
	if err != nil || status < 200 || status >= 300 {
		return fmt.Errorf("observer envelope: %d %v", status, err)
	}
	return nil
}

func (c *controller) ensureBlock(ctx context.Context, root string, depth int) error {
	_, status, err := c.request(ctx, "GET", c.observer+"/eth/v1/beacon/headers/"+root, nil)
	if err == nil && status == 200 {
		return nil
	}
	if depth >= 64 {
		return fmt.Errorf("observer needs more than the declared 64-block catch-up")
	}
	raw, block, err := c.block(ctx, root)
	if err != nil {
		return err
	}
	if err := c.ensureBlock(ctx, block.Message.ParentRoot, depth+1); err != nil {
		return err
	}
	// A FULL parent must be available before its child is processed. An EMPTY
	// child must use the original pre-payload branch even if the observer has
	// already executed the parent's valid envelope.
	_, parent, err := c.block(ctx, block.Message.ParentRoot)
	if err != nil {
		return err
	}
	if block.Message.Body.Bid.Message.ParentHash == parent.Message.Body.Bid.Message.BlockHash {
		if err := c.mirrorEnvelope(ctx, block.Message.ParentRoot); err != nil {
			return err
		}
	}
	body, status, err := c.request(ctx, "POST", c.observer+"/eth/v2/beacon/blocks?broadcast_validation=consensus", raw)
	c.record("mirror-original-signed-block", raw, body, status, err)
	if err != nil || status < 200 || status >= 300 {
		return fmt.Errorf("observer block: %d %v: %s", status, err, body)
	}
	return nil
}

// envelopeIdentity projects only routing metadata. The receiving real client
// still validates the complete SSZ/JSON, signature, state and execution hash.
func envelopeIdentity(input []byte, headers http.Header) (root string, slot uint64, err error) {
	if strings.HasPrefix(headers.Get("Content-Type"), "application/octet-stream") {
		signed := input
		if headers.Get("Eth-Blob-Data-Included") == "true" {
			if len(input) < 12 {
				return "", 0, fmt.Errorf("incomplete Contents offsets")
			}
			offset := uint64(binary.LittleEndian.Uint32(input[:4]))
			if offset < 12 || offset > uint64(len(input)) {
				return "", 0, fmt.Errorf("invalid signed-envelope offset")
			}
			signed = input[offset:]
		}
		if len(signed) < 100 {
			return "", 0, fmt.Errorf("incomplete signed envelope")
		}
		offset := uint64(binary.LittleEndian.Uint32(signed[:4]))
		if offset < 100 || offset+80 > uint64(len(signed)) {
			return "", 0, fmt.Errorf("invalid message offset")
		}
		// Gloas message: payload offset, requests offset, builder index,
		// beacon root, parent beacon root. Do not re-encode the signed bytes.
		return "0x" + hex.EncodeToString(signed[offset+16:offset+48]), 0, nil
	}
	var contents struct {
		Envelope json.RawMessage `json:"signed_execution_payload_envelope"`
	}
	if headers.Get("Eth-Blob-Data-Included") == "true" {
		if err := json.Unmarshal(input, &contents); err != nil {
			return "", 0, err
		}
		input = contents.Envelope
	}
	var envelope struct {
		Message struct {
			Root    string `json:"beacon_block_root"`
			Payload struct {
				Slot string `json:"slot_number"`
			} `json:"payload"`
		} `json:"message"`
	}
	if err := json.Unmarshal(input, &envelope); err != nil {
		return "", 0, err
	}
	slot, err = strconv.ParseUint(envelope.Message.Payload.Slot, 10, 64)
	return envelope.Message.Root, slot, err
}

func (c *controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.URL.Path != "/eth/v1/beacon/execution_payload_envelopes" {
		c.proxy.ServeHTTP(w, r)
		return
	}
	input, err := io.ReadAll(io.LimitReader(r.Body, (64<<20)+1))
	if err != nil || len(input) > 64<<20 {
		http.Error(w, "control body unavailable", 413)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(input))
	root, slot, decodeErr := envelopeIdentity(input, r.Header)
	if decodeErr == nil && slot == 0 {
		// SSZ identifies the authenticated beacon block; its public signed
		// header supplies the slot without assuming payload field offsets.
		_, block, err := c.block(r.Context(), root)
		if err != nil {
			decodeErr = err
		} else {
			slot, decodeErr = strconv.ParseUint(block.Message.Slot, 10, 64)
		}
	}
	c.mu.Lock()
	chosen := decodeErr == nil && slot == c.controlSlot() && !c.routed
	if chosen {
		c.routed = true
	}
	c.mu.Unlock()
	if !chosen {
		c.proxy.ServeHTTP(w, r)
		return
	}
	if err := c.ensureBlock(r.Context(), root, 0); err != nil {
		c.mu.Lock()
		c.routed = false
		c.mu.Unlock()
		c.record("first-envelope-observer-block-failure", input, nil, 502, err)
		http.Error(w, err.Error(), 502)
		return
	}
	if err := c.signatureProbe(r.Context(), input, r.Header); err != nil {
		c.record("signature-probe-failed", input, nil, 502, err)
		http.Error(w, err.Error(), 502)
		return
	}
	body, status, err := c.requestHeaders(r.Context(), "POST", c.observer+r.URL.RequestURI(), input, r.Header)
	c.record("first-envelope-to-observer", input, body, status, err)
	if err != nil || status < 200 || status >= 300 {
		c.mu.Lock()
		c.routed = false
		c.mu.Unlock()
	}
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (c *controller) mirrorHeads(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		func() {
			requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			body, status, err := c.request(requestCtx, "GET", c.upstream+"/eth/v1/beacon/headers/head", nil)
			if err != nil || status != 200 {
				return
			}
			var header struct {
				Data struct{ Root string } `json:"data"`
			}
			if json.Unmarshal(body, &header) != nil {
				return
			}
			if err := c.ensureBlock(requestCtx, header.Data.Root, 0); err != nil {
				c.record("mirror-head-paused", nil, nil, 0, err)
				return
			}
			_, block, err := c.block(requestCtx, header.Data.Root)
			if err == nil && block.Message.Slot != strconv.FormatUint(c.controlSlot(), 10) {
				_ = c.mirrorEnvelope(requestCtx, header.Data.Root)
			}
		}()
	}
}

func main() {
	listen := flag.String("listen", "0.0.0.0:4000", "owned private control listener")
	upstream := flag.String("upstream", "", "original private CL REST endpoint")
	observer := flag.String("observer", "", "isolated observer CL REST endpoint")
	slot := flag.Uint64("slot", 0, "the single selected development slot")
	slotFile := flag.String("slot-file", "", "owned target selected after actual signed-preference warm-up")
	probeDir := flag.String("signature-probe-dir", "", "optional owned directory for one replaced-key signature probe")
	output := flag.String("output", "", "new exact signed-body evidence file")
	flag.Parse()
	if *upstream == "" || *observer == "" || *slot == 0 || *output == "" {
		log.Fatal("explicit upstream, observer, slot and new evidence path are required")
	}
	target, err := url.Parse(*upstream)
	if err != nil || target.Scheme != "http" {
		log.Fatal("private HTTP upstream required")
	}
	file, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	c := &controller{upstream: *upstream, observer: *observer, slot: *slot, slotFile: *slotFile, signatureProbeDir: *probeDir, output: file,
		client: &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}},
		proxy:  httputil.NewSingleHostReverseProxy(target)}
	c.proxy.Transport = c.client.Transport
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	mirrorDone := make(chan struct{})
	go func() {
		defer close(mirrorDone)
		c.mirrorHeads(ctx)
	}()
	server := &http.Server{Addr: *listen, Handler: c, ReadHeaderTimeout: 5 * time.Second}
	go func() { <-ctx.Done(); _ = server.Close() }()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	stop()
	<-mirrorDone
}
