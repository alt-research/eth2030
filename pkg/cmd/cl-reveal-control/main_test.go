package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"
)

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
