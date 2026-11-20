package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSignatureProbeCanChangeOnlyBLSFieldAndRequiresRealRejection(t *testing.T) {
	input, err := os.ReadFile("testdata/actual-gloas-contents.ssz")
	if err != nil {
		t.Fatal(err)
	}
	offset := int(binary.LittleEndian.Uint32(input[:4]))
	headers := http.Header{"Content-Type": {"application/octet-stream"}, "Eth-Blob-Data-Included": {"true"}}
	for _, status := range []int{400, 200, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			dir := t.TempDir()
			signature := bytes.Repeat([]byte{0x42}, 96)
			if err := os.WriteFile(filepath.Join(dir, "probe-signature.bin"), signature, 0600); err != nil {
				t.Fatal(err)
			}
			observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if !bytes.Equal(body[:offset+4], input[:offset+4]) || !bytes.Equal(body[offset+100:], input[offset+100:]) || !bytes.Equal(body[offset+4:offset+100], signature) {
					t.Error("probe changed signed message or did not replace its exact signature")
				}
				w.WriteHeader(status)
			}))
			defer observer.Close()
			c := &controller{observer: observer.URL, signatureProbeDir: dir, client: observer.Client(), output: io.Discard}
			if err := c.signatureProbe(context.Background(), input, headers); (err == nil) != (status == 400) {
				t.Fatalf("probe status %d: %v", status, err)
			}
			captured, err := os.ReadFile(filepath.Join(dir, "probe-input.ssz"))
			if err != nil || !bytes.Equal(captured, input) {
				t.Fatal("original producer evidence was changed")
			}
		})
	}
}

func TestChosenEnvelopeKeepsExactSignedBytesAndRoutesOnlyFirstDelivery(t *testing.T) {
	input := []byte(`{"message":{"beacon_block_root":"original-root","payload":{"slot_number":"42","extra_signed_field":"keep this"}},"signature":"original-signature"}`)
	var mainBodies, observerBodies [][]byte
	main := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mainBodies = append(mainBodies, body)
		w.WriteHeader(201)
		_, _ = w.Write([]byte("main"))
	}))
	defer main.Close()
	observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(200) // Already has the authenticated original block.
			return
		}
		body, _ := io.ReadAll(r.Body)
		observerBodies = append(observerBodies, body)
		_, _ = w.Write([]byte("observer"))
	}))
	defer observer.Close()
	target, _ := url.Parse(main.URL)
	var evidence bytes.Buffer
	c := &controller{upstream: main.URL, observer: observer.URL, slot: 42, client: main.Client(), proxy: httputil.NewSingleHostReverseProxy(target), output: &evidence}
	for index, want := range []string{"observer", "main"} {
		req := httptest.NewRequest("POST", "/eth/v1/beacon/execution_payload_envelopes?broadcast_validation=consensus", bytes.NewReader(input))
		req.Header.Set("Authorization", "Bearer never-log-credentials")
		response := httptest.NewRecorder()
		c.ServeHTTP(response, req)
		if response.Body.String() != want {
			t.Fatalf("delivery %d = %d %s, want %s", index, response.Code, response.Body, want)
		}
	}
	if len(mainBodies) != 1 || len(observerBodies) != 1 || !bytes.Equal(mainBodies[0], input) || !bytes.Equal(observerBodies[0], input) {
		t.Fatal("routing dropped/re-encoded signed fields or delivered the first envelope twice")
	}
	if strings.Contains(evidence.String(), "never-log-credentials") {
		t.Fatal("capture leaked request headers")
	}
}

func TestMirroredEmptyChildKeepsOriginalBlockAndDoesNotFetchItsParentsEnvelope(t *testing.T) {
	original := []byte(`{"message":{"slot":"2","parent_root":"parent","body":{"signed_execution_payload_bid":{"message":{"block_hash":"child-hash","parent_block_hash":"older-executed-hash"}},"extra_field":"preserve"}},"signature":"real-signed-bytes"}`)
	var received []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/eth/v2/beacon/blocks/child":
			_, _ = w.Write(append(append([]byte(`{"data":`), original...), '}'))
		case "/eth/v2/beacon/blocks/parent":
			_, _ = w.Write([]byte(`{"data":{"message":{"slot":"1","body":{"signed_execution_payload_bid":{"message":{"block_hash":"unselected-parent-payload"}}}}}}`))
		default:
			t.Errorf("unexpected fetch of unselected parent payload: %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer upstream.Close()
	observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if strings.HasSuffix(r.URL.Path, "/parent") {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(404)
			}
			return
		}
		received, _ = io.ReadAll(r.Body)
	}))
	defer observer.Close()
	c := &controller{upstream: upstream.URL, observer: observer.URL, client: upstream.Client(), output: io.Discard}
	if err := c.ensureBlock(context.Background(), "child", 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(received, original) {
		t.Fatal("mirroring rewrote the original signed block")
	}
}

func TestControlRequestHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	c := &controller{client: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, _, err := c.request(ctx, "GET", server.URL, nil); err == nil {
		t.Fatal("cancelled upstream request succeeded")
	}
}

func TestActualStatelessSSZContentsRoutesWithoutReencodingOrLosingDiscriminator(t *testing.T) {
	input, err := os.ReadFile("testdata/actual-gloas-contents.ssz")
	if err != nil {
		t.Fatal(err)
	}
	const wantRoot = "0x7ba11c7c7fb0b5a8e099586871102982c3578873edafcc8357ec48811f7d0690"
	headers := http.Header{"Content-Type": {"application/octet-stream"}, "Eth-Blob-Data-Included": {"true"}, "Eth-Consensus-Version": {"gloas"}}
	root, slot, err := envelopeIdentity(input, headers)
	if err != nil || root != wantRoot || slot != 0 {
		t.Fatalf("actual SSZ identity: %s %d %v", root, slot, err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, wantRoot) {
			t.Errorf("unexpected upstream request: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"message":{"slot":"20696"}}}`))
	}))
	defer upstream.Close()
	var received []byte
	observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			return
		}
		if r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Eth-Blob-Data-Included") != "true" || r.Header.Get("Eth-Consensus-Version") != "gloas" {
			t.Error("lost the actual stateless submission headers")
		}
		received, _ = io.ReadAll(r.Body)
	}))
	defer observer.Close()
	c := &controller{upstream: upstream.URL, observer: observer.URL, slot: 20696, client: upstream.Client(), output: io.Discard}
	req := httptest.NewRequest("POST", "/eth/v1/beacon/execution_payload_envelopes", bytes.NewReader(input))
	req.Header = headers
	response := httptest.NewRecorder()
	c.ServeHTTP(response, req)
	if response.Code != 200 || !bytes.Equal(received, input) {
		t.Fatalf("actual signed SSZ forwarding changed bytes: %d", response.Code)
	}
	broken := bytes.Clone(input)
	for i := 0; i < 4; i++ {
		broken[i] = 0xff
	}
	if _, _, err := envelopeIdentity(broken, headers); err == nil {
		t.Fatal("accepted an out-of-bounds Contents offset")
	}
}

func TestRejectedObserverDeliveryCannotLeakAProducerRetryToMain(t *testing.T) {
	var observerPosts, mainPosts int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mainPosts++
	}))
	defer upstream.Close()
	observer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			return
		}
		observerPosts++
		if observerPosts == 1 {
			w.WriteHeader(503)
		}
	}))
	defer observer.Close()
	target, _ := url.Parse(upstream.URL)
	c := &controller{upstream: upstream.URL, observer: observer.URL, slot: 42, client: upstream.Client(), proxy: httputil.NewSingleHostReverseProxy(target), output: io.Discard}
	for index, want := range []int{503, 200, 200} {
		response := httptest.NewRecorder()
		c.ServeHTTP(response, httptest.NewRequest("POST", "/eth/v1/beacon/execution_payload_envelopes", strings.NewReader(`{"message":{"beacon_block_root":"root","payload":{"slot_number":"42"}}}`)))
		if response.Code != want {
			t.Fatalf("delivery %d: status %d want %d", index, response.Code, want)
		}
	}
	if observerPosts != 2 || mainPosts != 1 {
		t.Fatalf("retry routing: observer=%d main=%d", observerPosts, mainPosts)
	}
}
