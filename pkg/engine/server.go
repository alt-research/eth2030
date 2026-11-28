package engine

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/backendapi"
	"github.com/eth2030/eth2030/engine/blobval"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/metrics"
	"github.com/golang-jwt/jwt/v4"
)

// Backend is a type alias — canonical definition in engine/backendapi.
type Backend = backendapi.Backend

// EngineAPI implements the Engine API JSON-RPC methods.
type EngineAPI struct {
	backend         Backend
	blobValidator   *blobval.BlobValidator
	server          *http.Server
	listener        net.Listener
	ethHandler      http.Handler // optional: handles non-engine_ methods (eth_, web3_, net_)
	authSecret      string
	jwtSecret       *[32]byte
	authVhosts      []string // Host header allowlist (--authrpc.vhosts); empty or ["*"] allows all
	maxRequestSize  int64
	requestTimeout  time.Duration    // P0: configurable request timeout (default 8s)
	priorityHandler *PriorityHandler // P2: tracks request priorities
	mu              sync.Mutex
	serveDone       chan struct{}
	serveErr        error
	stopped         bool
}

// NewEngineAPI creates a new Engine API instance with the given backend.
func NewEngineAPI(backend Backend) *EngineAPI {
	return &EngineAPI{
		backend:         backend,
		blobValidator:   blobval.NewBlobValidator(),
		maxRequestSize:  5 * 1024 * 1024,
		requestTimeout:  8 * time.Second,      // P0: default 8s timeout
		priorityHandler: NewPriorityHandler(), // P2: priority tracking
	}
}

// SetAuthSecret sets the bearer token required for Engine API authentication.
// An empty token disables auth checks.
func (api *EngineAPI) SetAuthSecret(secret string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.authSecret = secret
	api.jwtSecret = nil
}

// SetAuthVhosts sets the Host header allowlist (--authrpc.vhosts). An empty
// list or ["*"] allows all hosts; otherwise the Host must match an entry
// (case-insensitive, port stripped).
func (api *EngineAPI) SetAuthVhosts(vhosts []string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.authVhosts = vhosts
}

// vhostAllowed reports whether the request Host passes the configured
// vhost allowlist.
func vhostAllowed(r *http.Request, vhosts []string) bool {
	if len(vhosts) == 0 || (len(vhosts) == 1 && vhosts[0] == "*") {
		return true
	}
	host := r.Host
	if strings.HasPrefix(host, "[") {
		if idx := strings.LastIndex(host, "]"); idx >= 0 {
			host = host[1:idx]
		}
	} else if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	for _, v := range vhosts {
		if strings.EqualFold(v, host) {
			return true
		}
	}
	return false
}

// SetJWTSecret configures Engine API HS256 authentication.
func (api *EngineAPI) SetJWTSecret(secret [32]byte) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.authSecret = ""
	api.jwtSecret = &secret
}

// SetMaxRequestSize updates the maximum allowed request body size.
// A non-positive value restores the default of 5 MiB.
func (api *EngineAPI) SetMaxRequestSize(bytes int64) {
	if bytes <= 0 {
		bytes = 5 * 1024 * 1024
	}
	api.mu.Lock()
	defer api.mu.Unlock()
	api.maxRequestSize = bytes
}

// SetRequestTimeout sets the timeout for Engine API requests.
// A non-positive value disables timeout (not recommended for production).
func (api *EngineAPI) SetRequestTimeout(timeout time.Duration) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.requestTimeout = timeout
}

// NewPayloadV3 validates and executes a new Cancun/Deneb payload.
// Per EIP-4844: blob versioned hashes must match commitments in transactions.
// The wire handler requires a parentBeaconBlockRoot DATA value, including zero.
func (api *EngineAPI) NewPayloadV3(
	payload ExecutionPayloadV3,
	expectedBlobVersionedHashes []types.Hash,
	parentBeaconBlockRoot types.Hash,
) (PayloadStatusV1, error) {
	// Validate that the payload timestamp falls within the Cancun fork.
	if !api.backend.IsCancun(payload.Timestamp) || api.backend.IsPrague(payload.Timestamp) {
		return PayloadStatusV1{}, ErrUnsupportedFork
	}

	// EIP-4844: validate blob versioned hashes match transaction contents.
	if err := validateBlobHashes(&payload, expectedBlobVersionedHashes); err != nil {
		errMsg := err.Error()
		return PayloadStatusV1{
			Status:          StatusInvalid,
			ValidationError: &errMsg,
		}, nil
	}

	return api.backend.ProcessBlock(&payload, expectedBlobVersionedHashes, parentBeaconBlockRoot)
}

// ForkchoiceUpdatedV3 processes a forkchoice update with optional V3 payload attributes.
// Validates withdrawals and parentBeaconBlockRoot in payload attributes.
// Validates timestamp progression when attributes are provided.
func (api *EngineAPI) ForkchoiceUpdatedV3(
	state ForkchoiceStateV1,
	payloadAttributes *PayloadAttributesV3,
) (ForkchoiceUpdatedResult, error) {
	if payloadAttributes != nil {
		headTimestamp := api.backend.GetBlockTimestamp(state.HeadBlockHash)
		if payloadAttributes.Timestamp == 0 ||
			payloadAttributes.Timestamp <= headTimestamp {
			result, err := api.backend.ForkchoiceUpdated(state, nil)
			return forkchoiceAttributeError(result, err, ErrInvalidPayloadAttributes)
		}
		if !api.backend.IsCancun(payloadAttributes.Timestamp) || api.backend.IsAmsterdam(payloadAttributes.Timestamp) {
			result, err := api.backend.ForkchoiceUpdated(state, nil)
			return forkchoiceAttributeError(result, err, ErrUnsupportedFork)
		}
	}
	return api.backend.ForkchoiceUpdated(state, payloadAttributes)
}

// GetPayloadV3 retrieves a previously built payload by ID.
// Returns ExecutionPayloadV3 + blockValue + blobsBundle (no executionRequests).
func (api *EngineAPI) GetPayloadV3(payloadID PayloadID) (*GetPayloadV3Response, error) {
	resp, err := api.backend.GetPayloadByID(payloadID)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.ExecutionPayload == nil {
		return nil, ErrUnknownPayload
	}
	if !api.backend.IsCancun(resp.ExecutionPayload.Timestamp) || api.backend.IsPrague(resp.ExecutionPayload.Timestamp) {
		return nil, ErrUnsupportedFork
	}
	if err := api.validateBlobsBundle(resp.BlobsBundle); err != nil {
		return nil, fmt.Errorf("invalid blobs bundle: %w", err)
	}
	// Build the V3 response (no executionRequests in V3).
	v3Resp := &GetPayloadV3Response{
		ExecutionPayload: &resp.ExecutionPayload.ExecutionPayloadV3,
		BlockValue:       resp.BlockValue,
		BlobsBundle:      resp.BlobsBundle,
		Override:         resp.Override,
	}
	return v3Resp, nil
}

// validateBlobsBundle validates the KZG commitments and blob counts in a blobs bundle.
// It converts the dynamic [][]byte slices to fixed-size arrays required by blobval.
func (api *EngineAPI) validateBlobsBundle(bundle *payload.BlobsBundleV1) error {
	if bundle == nil || len(bundle.Blobs) == 0 {
		return nil
	}
	commitments := make([][48]byte, len(bundle.Commitments))
	for i, c := range bundle.Commitments {
		if len(c) != 48 {
			return fmt.Errorf("commitment[%d]: expected 48 bytes, got %d", i, len(c))
		}
		copy(commitments[i][:], c)
	}
	blobs := make([][131072]byte, len(bundle.Blobs))
	for i, b := range bundle.Blobs {
		if len(b) != 131072 {
			return fmt.Errorf("blob[%d]: expected 131072 bytes, got %d", i, len(b))
		}
		copy(blobs[i][:], b)
	}
	return api.blobValidator.ValidateKZGCommitments(commitments, blobs)
}

// NewPayloadV4 validates and executes a Prague/Electra payload with execution requests.
// Per EIP-7685: executionRequests must be provided.
func (api *EngineAPI) NewPayloadV4(
	payload ExecutionPayloadV3,
	expectedBlobVersionedHashes []types.Hash,
	parentBeaconBlockRoot types.Hash,
	executionRequests [][]byte,
) (PayloadStatusV1, error) {
	// Validate that the payload timestamp falls within the Prague fork.
	if !api.backend.IsPrague(payload.Timestamp) || api.backend.IsAmsterdam(payload.Timestamp) {
		return PayloadStatusV1{}, ErrUnsupportedFork
	}

	// EIP-7685: executionRequests must be provided (can be empty list, not nil).
	if err := types.ValidateExecutionRequestsWire(executionRequests); err != nil {
		return PayloadStatusV1{}, ErrInvalidParams
	}

	// EIP-4844: validate blob versioned hashes match transaction contents.
	if err := validateBlobHashes(&payload, expectedBlobVersionedHashes); err != nil {
		errMsg := err.Error()
		return PayloadStatusV1{
			Status:          StatusInvalid,
			ValidationError: &errMsg,
		}, nil
	}

	return api.backend.ProcessBlockV4(&payload, expectedBlobVersionedHashes, parentBeaconBlockRoot, executionRequests)
}

// GetPayloadV4 retrieves a previously built payload for Prague.
// Returns ExecutionPayloadV3 + blockValue + blobsBundle + executionRequests.
func (api *EngineAPI) GetPayloadV4(payloadID PayloadID) (*GetPayloadV4Response, error) {
	resp, err := api.backend.GetPayloadV4ByID(payloadID)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.ExecutionPayload == nil {
		return nil, ErrUnknownPayload
	}
	if !api.backend.IsPrague(resp.ExecutionPayload.Timestamp) || api.backend.IsOsaka(resp.ExecutionPayload.Timestamp) {
		return nil, ErrUnsupportedFork
	}
	return resp, nil
}

// GetPayloadV5 serves the Osaka shape: V3 payload and cell-proof bundle.
func (api *EngineAPI) GetPayloadV5(payloadID PayloadID) (*payload.GetPayloadV5Response, error) {
	resp, err := api.backend.GetPayloadV6ByID(payloadID)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.ExecutionPayload == nil {
		return nil, ErrUnknownPayload
	}
	timestamp := resp.ExecutionPayload.Timestamp
	if !api.backend.IsOsaka(timestamp) || api.backend.IsAmsterdam(timestamp) {
		return nil, ErrUnsupportedFork
	}
	return &payload.GetPayloadV5Response{
		ExecutionPayload: &resp.ExecutionPayload.ExecutionPayloadV3,
		BlockValue:       resp.BlockValue, BlobsBundle: resp.BlobsBundle,
		Override: resp.Override, ExecutionRequests: resp.ExecutionRequests,
	}, nil
}

// NewPayloadV5 validates and executes an Amsterdam payload with BAL.
func (api *EngineAPI) NewPayloadV5(
	payload ExecutionPayloadV5,
	expectedBlobVersionedHashes []types.Hash,
	parentBeaconBlockRoot types.Hash,
	executionRequests [][]byte,
) (PayloadStatusV1, error) {
	// Check that the timestamp falls within the Amsterdam fork.
	if !api.backend.IsAmsterdam(payload.Timestamp) || api.isBogota(payload.Timestamp) {
		return PayloadStatusV1{}, ErrUnsupportedFork
	}
	return api.processAmsterdamPayload(payload, expectedBlobVersionedHashes, parentBeaconBlockRoot, executionRequests)
}

func (api *EngineAPI) processAmsterdamPayload(
	payload ExecutionPayloadV5,
	expectedBlobVersionedHashes []types.Hash,
	parentBeaconBlockRoot types.Hash,
	executionRequests [][]byte,
) (PayloadStatusV1, error) {
	return api.processAmsterdamPayloadWith(payload, expectedBlobVersionedHashes,
		parentBeaconBlockRoot, executionRequests, api.backend.ProcessBlockV5)
}

func (api *EngineAPI) processAmsterdamPayloadWith(
	payload ExecutionPayloadV5,
	expectedBlobVersionedHashes []types.Hash,
	parentBeaconBlockRoot types.Hash,
	executionRequests [][]byte,
	process func(*ExecutionPayloadV5, []types.Hash, types.Hash, [][]byte) (PayloadStatusV1, error),
) (PayloadStatusV1, error) {
	// EIP-7685: executionRequests must be provided.
	if err := types.ValidateExecutionRequestsWire(executionRequests); err != nil {
		return PayloadStatusV1{}, ErrInvalidParams
	}

	// The blockAccessList field must be present.
	if payload.BlockAccessList == nil {
		return PayloadStatusV1{}, ErrInvalidParams
	}

	// EIP-4844: validate blob versioned hashes match transaction contents.
	if err := validateBlobHashes(&payload.ExecutionPayloadV3, expectedBlobVersionedHashes); err != nil {
		errMsg := err.Error()
		return PayloadStatusV1{
			Status:          StatusInvalid,
			ValidationError: &errMsg,
		}, nil
	}

	return process(&payload, expectedBlobVersionedHashes, parentBeaconBlockRoot, executionRequests)
}

// GetPayloadV6 retrieves a previously built payload for Amsterdam.
// Returns ExecutionPayloadV4 (which includes BAL) with execution requests.
func (api *EngineAPI) GetPayloadV6(payloadID PayloadID) (*GetPayloadV6Response, error) {
	resp, err := api.backend.GetPayloadV6ByID(payloadID)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.ExecutionPayload == nil {
		return nil, ErrUnknownPayload
	}
	if !api.backend.IsAmsterdam(resp.ExecutionPayload.Timestamp) {
		return nil, ErrUnsupportedFork
	}
	return resp, nil
}

// payloadV7Backend is implemented by backends that support the EIP-7807
// SSZ payload surface.
type payloadV7Backend interface {
	GetPayloadV7ByID(PayloadID) (*GetPayloadV6Response, error)
}

// GetPayloadV7 retrieves a previously built payload with the EIP-7807 SSZ
// activation fields (sszPayload + sszBlockHash) when SSZ blocks are active.
func (api *EngineAPI) GetPayloadV7(payloadID PayloadID) (*GetPayloadV6Response, error) {
	v7, ok := api.backend.(payloadV7Backend)
	if !ok {
		return nil, ErrUnsupportedFork
	}
	return v7.GetPayloadV7ByID(payloadID)
}

// NewPayloadV7Request carries a V5 payload plus the EIP-7807 SSZ binding:
// the SSZ payload encoding and its hash tree root (the SSZ execution block
// hash). The EL verifies the SSZ binding before executing the payload.
type NewPayloadV7Request struct {
	ExecutionPayload      ExecutionPayloadV5 `json:"executionPayload"`
	BlobVersionedHashes   []types.Hash       `json:"expectedBlobVersionedHashes"`
	ParentBeaconBlockRoot types.Hash         `json:"parentBeaconBlockRoot"`
	SSZPayload            json.RawMessage    `json:"sszPayload"`
	SSZBlockHash          types.Hash         `json:"sszBlockHash"`
}

// NewPayloadV7 validates an SSZ-bound payload (EIP-7807) and executes it
// through the Amsterdam path. The SSZ payload must decode and its hash tree
// root must match sszBlockHash; otherwise the payload is INVALID.
func (api *EngineAPI) NewPayloadV7(req NewPayloadV7Request) (PayloadStatusV1, error) {
	sszBytes, err := decodeHexParam(req.SSZPayload)
	if err != nil {
		errMsg := "invalid sszPayload encoding: " + err.Error()
		return PayloadStatusV1{Status: StatusInvalid, ValidationError: &errMsg}, nil
	}
	var decoded types.ExecutionPayloadSSZ
	if err := decoded.UnmarshalSSZ(sszBytes); err != nil {
		errMsg := "invalid sszPayload encoding: " + err.Error()
		return PayloadStatusV1{Status: StatusInvalid, ValidationError: &errMsg}, nil
	}
	root, err := decoded.HashTreeRoot()
	if err != nil {
		errMsg := "ssz payload hash error: " + err.Error()
		return PayloadStatusV1{Status: StatusInvalid, ValidationError: &errMsg}, nil
	}
	if root != req.SSZBlockHash {
		errMsg := "ssz block hash mismatch: header=" + req.SSZBlockHash.Hex() + " computed=" + root.Hex()
		return PayloadStatusV1{Status: StatusInvalid, ValidationError: &errMsg}, nil
	}
	// Cross-check the SSZ container against the JSON payload it binds.
	if decoded.ExecutionSSZFields.StateRoot != req.ExecutionPayload.StateRoot ||
		decoded.ExecutionSSZFields.Number != req.ExecutionPayload.BlockNumber ||
		decoded.ExecutionSSZFields.Timestamp != req.ExecutionPayload.Timestamp {
		errMsg := "ssz payload does not match the submitted payload fields"
		return PayloadStatusV1{Status: StatusInvalid, ValidationError: &errMsg}, nil
	}
	if !api.backend.IsAmsterdam(req.ExecutionPayload.Timestamp) {
		return PayloadStatusV1{}, ErrUnsupportedFork
	}
	return api.processAmsterdamPayload(
		req.ExecutionPayload,
		req.BlobVersionedHashes,
		req.ParentBeaconBlockRoot,
		req.ExecutionPayload.ExecutionRequests,
	)
}

// ForkchoiceUpdatedV4 processes a forkchoice update with V4 payload attributes (Amsterdam).
func (api *EngineAPI) ForkchoiceUpdatedV4(
	state ForkchoiceStateV1,
	payloadAttributes *PayloadAttributesV4,
) (ForkchoiceUpdatedResult, error) {
	if payloadAttributes != nil && (!api.backend.IsAmsterdam(payloadAttributes.Timestamp) || api.isBogota(payloadAttributes.Timestamp)) {
		result, err := api.backend.ForkchoiceUpdatedV4(state, nil)
		return forkchoiceAttributeError(result, err, ErrUnsupportedFork)
	}
	return api.backend.ForkchoiceUpdatedV4(state, payloadAttributes)
}

// Invalid attributes may prevent building, after the valid forkchoice is applied.
func forkchoiceAttributeError(result ForkchoiceUpdatedResult, err, attributeErr error) (ForkchoiceUpdatedResult, error) {
	if err != nil || result.PayloadStatus.Status != StatusValid {
		return result, err
	}
	return ForkchoiceUpdatedResult{}, attributeErr
}

// ExchangeCapabilities returns the list of Engine API methods this node supports.
func (api *EngineAPI) ExchangeCapabilities(requested []string) []string {
	supported := []string{
		"engine_newPayloadV3",
		"engine_newPayloadV4",
		"engine_newPayloadV5",
		"engine_forkchoiceUpdatedV3",
		"engine_forkchoiceUpdatedV4",
		"engine_getPayloadV3",
		"engine_getPayloadV4",
		"engine_getPayloadV5",
		"engine_getPayloadV6",
		"engine_getPayloadV7",
		"engine_newPayloadV7",
		"engine_getClientVersionV1",
		"engine_getBlobsV1",
		"engine_getBlobsV2",
		"engine_getBlobsV3",
		"engine_getBlobsV4",
		"engine_getPayloadBodiesByHashV1",
		"engine_getPayloadBodiesByRangeV1",
		"engine_getPayloadBodiesByHashV2",
		"engine_getPayloadBodiesByRangeV2",
	}
	if _, ok := api.backend.(bogotaBackend); ok {
		supported = append(supported, "engine_newPayloadV6", "engine_forkchoiceUpdatedV5")
	}
	if _, ok := api.backend.(InclusionListBackend); ok {
		supported = append(supported, "engine_getInclusionListV1")
	}
	return supported
}

// ClientVersionV1 represents the client version information.
type ClientVersionV1 struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// GetClientVersionV1 returns the client version information.
func (api *EngineAPI) GetClientVersionV1(peerVersion ClientVersionV1) []ClientVersionV1 {
	return []ClientVersionV1{
		{
			Code:    "ET",
			Name:    "ETH2030",
			Version: "v0.1.0",
			Commit:  "00000000",
		},
	}
}

// SetEthHandler registers an HTTP handler for non-engine_ JSON-RPC methods.
// The authenticated engine endpoint must also serve eth_ namespace methods
// (e.g. eth_syncing, eth_getBlockByNumber) per the Engine API specification.
func (api *EngineAPI) SetEthHandler(h http.Handler) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.ethHandler = h
}

// Start binds and serves until Stop, preserving the blocking standalone API.
func (api *EngineAPI) Start(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if err := api.ServeListener(ln); err != nil {
		ln.Close()
		return err
	}
	api.mu.Lock()
	done := api.serveDone
	api.mu.Unlock()
	<-done
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.serveErr
}

// ServeListener takes ownership of an already bound listener after successful
// initialization. The address and shutdown owner are published before return.
func (api *EngineAPI) ServeListener(ln net.Listener) error {
	if ln == nil {
		return fmt.Errorf("nil HTTP listener")
	}
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.stopped {
		return http.ErrServerClosed
	}
	if api.server != nil {
		return fmt.Errorf("HTTP server already started")
	}
	srv := &http.Server{Handler: api.Handler()}
	api.server = srv
	api.listener = ln
	done := make(chan struct{})
	api.serveDone = done
	go func() {
		err := srv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
			err = nil
		}
		api.mu.Lock()
		api.serveErr = err
		api.mu.Unlock()
		close(done)
	}()
	return nil
}

// Addr returns the actual owned listener address, including port-zero binds.
func (api *EngineAPI) Addr() net.Addr {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.listener == nil {
		return nil
	}
	return api.listener.Addr()
}

// Stop has a bounded graceful period, then forcibly releases listeners and
// connections. It also joins the serving goroutine if Stop wins the startup race.
func (api *EngineAPI) Stop() error {
	api.mu.Lock()
	api.stopped = true
	srv, ln, done := api.server, api.listener, api.serveDone
	api.mu.Unlock()
	if srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		_ = srv.Close()
	}
	if ln != nil {
		_ = ln.Close()
	}
	if done != nil {
		<-done
	}
	return err
}

// Handler exposes the same HTTP handler used by Start.
func (api *EngineAPI) Handler() http.Handler { return http.HandlerFunc(api.httpHandler) }

// httpHandler handles incoming HTTP requests and dispatches them as JSON-RPC.
// P0: Implements request timeout protection to prevent CL-EL cascade failures.
// P2: Tracks request priorities for monitoring and potential scheduling.
func (api *EngineAPI) httpHandler(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	vhosts := api.authVhosts
	api.mu.Unlock()
	if !vhostAllowed(r, vhosts) {
		http.Error(w, "invalid host", http.StatusForbidden)
		return
	}

	if !api.authorizedRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"unauthorized"},"id":null}`))
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	api.mu.Lock()
	maxRequestSize := api.maxRequestSize
	requestTimeout := api.requestTimeout
	api.mu.Unlock()
	if r.ContentLength > maxRequestSize {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestSize+1))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if int64(len(body)) > maxRequestSize {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	// P2: Determine request priority from method name.
	var peek struct {
		Method string `json:"method"`
	}
	priority := PriorityNormal
	if json.Unmarshal(body, &peek) == nil {
		priority = MethodPriority(peek.Method)
	}
	api.priorityHandler.StartRequest(priority)
	defer api.priorityHandler.EndRequest(priority)

	// Forward non-engine_ methods (eth_, web3_, net_, admin_) to the eth handler
	// if one has been registered. Per the Engine API spec, the authenticated
	// endpoint must also serve eth_syncing, eth_getBlockByNumber, etc.
	api.mu.Lock()
	ethH := api.ethHandler
	api.mu.Unlock()

	if ethH != nil {
		if peek.Method != "" && !strings.HasPrefix(peek.Method, "engine_") {
			r2 := r.Clone(r.Context())
			r2.Body = io.NopCloser(bytes.NewReader(body))
			ethH.ServeHTTP(w, r2)
			return
		}
	}

	// P0: Handle request with timeout protection.
	if requestTimeout <= 0 {
		// No timeout - direct handling (not recommended for production).
		resp := api.HandleRequest(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
		return
	}

	// Use channel-based timeout for async handling.
	type result struct {
		data []byte
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		defer close(resultCh)
		resp := api.HandleRequest(body)
		resultCh <- result{data: resp}
	}()

	select {
	case res := <-resultCh:
		w.Header().Set("Content-Type", "application/json")
		w.Write(res.data)
	case <-time.After(requestTimeout):
		// P0: Return timeout error response.
		// Per Engine API spec, CL should retry on timeout.
		metrics.EngineRequestTimeoutTotal.Inc()
		engineLog.Warn("engine_request_timeout",
			"event", "engine_request_timeout",
			"timeout_ms", requestTimeout.Milliseconds(),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		resp := []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"request timeout"},"id":null}`)
		w.Write(resp)
	case <-r.Context().Done():
		// Client disconnected.
		engineLog.Debug("engine_request_cancelled",
			"event", "engine_request_cancelled",
		)
	}
}

func (api *EngineAPI) authorizedRequest(r *http.Request) bool {
	api.mu.Lock()
	secret := api.authSecret
	jwtSecret := api.jwtSecret
	api.mu.Unlock()

	if secret == "" && jwtSecret == nil {
		return true
	}

	auth := r.Header.Get("Authorization")
	if len(auth) < 7 || !strings.EqualFold(auth[:7], "Bearer ") {
		return false
	}
	if jwtSecret == nil {
		return subtle.ConstantTimeCompare([]byte(auth[7:]), []byte(secret)) == 1
	}

	var claims jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(auth[7:], &claims, func(*jwt.Token) (any, error) {
		return jwtSecret[:], nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil || !token.Valid || claims.IssuedAt == nil {
		return false
	}
	now := time.Now()
	age := now.Sub(claims.IssuedAt.Time)
	return age >= -time.Minute && age <= time.Minute && claims.VerifyExpiresAt(now, false)
}

// validateBlobHashes checks that the blob versioned hashes from the CL
// match the versioned hashes found in blob transactions within the payload.
// Per EIP-4844, each blob transaction carries versioned hashes that must
// appear in the same order in the expectedBlobVersionedHashes list.
func validateBlobHashes(payload *ExecutionPayloadV3, expected []types.Hash) error {
	// Collect all blob versioned hashes from transactions in the payload.
	var actual []types.Hash
	for _, txBytes := range payload.Transactions {
		tx, err := types.DecodeBlockTransaction(txBytes)
		if err != nil {
			// A malformed envelope cannot supply a trustworthy blob-hash list.
			return fmt.Errorf("invalid transaction encoding: %w", err)
		}
		hashes := tx.BlobHashes()
		if len(hashes) > 0 {
			actual = append(actual, hashes...)
		}
	}

	// The expected list must match the actual list exactly (same length, same order).
	if len(expected) != len(actual) {
		return fmt.Errorf("%w: expected %d blob hashes, got %d from transactions",
			ErrInvalidBlobHashes, len(expected), len(actual))
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return fmt.Errorf("%w: hash mismatch at index %d", ErrInvalidBlobHashes, i)
		}
	}
	return nil
}

// decodeHexParam decodes a 0x-hex JSON string parameter into raw bytes.
func decodeHexParam(raw json.RawMessage) ([]byte, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}
