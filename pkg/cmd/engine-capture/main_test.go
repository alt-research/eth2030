package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCapturePreservesCompleteEngineBodyAndAuthorization(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":7,"method":"engine_newPayloadV5","params":[{"transactions":[],"withdrawals":[]}]} `
	output := `{"jsonrpc":"2.0","id":7,"result":{"status":"VALID","latestValidHash":null}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != input || r.Header.Get("Authorization") != "Bearer private-test-token" {
			t.Error("Engine input or auth header changed")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, output)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	var evidence bytes.Buffer
	proxy := &capture{upstream: target, transport: upstream.Client().Transport, output: &evidence, session: "test"}
	req := httptest.NewRequest(http.MethodPost, "http://el:8551/", strings.NewReader(input))
	req.Header.Set("Authorization", "Bearer private-test-token")
	response := httptest.NewRecorder()
	proxy.ServeHTTP(response, req)
	if response.Code != 200 || response.Body.String() != output {
		t.Fatal("Engine response bytes changed")
	}
	var record struct {
		RequestBody  []byte `json:"request_body"`
		ResponseBody []byte `json:"response_body"`
		Complete     bool   `json:"complete"`
	}
	if err := json.Unmarshal(evidence.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if string(record.RequestBody) != input || string(record.ResponseBody) != output || !record.Complete {
		t.Fatal("incomplete evidence")
	}
	if strings.Contains(evidence.String(), "private-test-token") {
		t.Fatal("authorization header entered evidence")
	}
}
func TestCaptureRetainsMalformedPayloadForActualEngineRejection(t *testing.T) {
	input := `{malformed payload`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != input {
			t.Error("malformed input was rewritten")
		}
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":"actual Engine rejection"}`)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	var evidence bytes.Buffer
	handler := &capture{upstream: target, transport: upstream.Client().Transport, output: &evidence}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://el/", strings.NewReader(input)))
	var record struct {
		RequestBody []byte `json:"request_body"`
	}
	if err := json.Unmarshal(evidence.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if response.Code != 400 || string(record.RequestBody) != input {
		t.Fatal("actual rejection or malformed evidence lost")
	}
}

func TestCapturePreservesNonJSONBytesWithoutUTF8Replacement(t *testing.T) {
	input, output := []byte{0xff, 0, 1}, []byte{0xfe, 0, 2}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if !bytes.Equal(raw, input) {
			t.Error("binary request changed")
		}
		w.WriteHeader(400)
		_, _ = w.Write(output)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	var evidence bytes.Buffer
	handler := &capture{upstream: target, transport: upstream.Client().Transport, output: &evidence}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "http://el/", bytes.NewReader(input)))
	var record struct {
		RequestBody  []byte `json:"request_body"`
		ResponseBody []byte `json:"response_body"`
	}
	if err := json.Unmarshal(evidence.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response.Body.Bytes(), output) || !bytes.Equal(record.RequestBody, input) || !bytes.Equal(record.ResponseBody, output) {
		t.Fatal("malformed byte evidence changed")
	}
}
