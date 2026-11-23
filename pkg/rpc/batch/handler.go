// handler.go implements BatchHandler and ExtendedBatchHandler for parallel
// JSON-RPC batch request execution.
package rpcbatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eth2030/eth2030/log"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

var batchLog = log.Default().Module("rpc/batch")

// RequestHandler dispatches a single JSON-RPC request and returns a response.
// *ethapi.EthAPI satisfies this interface.
type RequestHandler interface {
	HandleRequest(req *rpctypes.Request) *rpctypes.Response
}

// AdminRequestHandler dispatches an admin_ JSON-RPC request.
// *rpc.AdminDispatchAPI satisfies this interface.
type AdminRequestHandler interface {
	HandleAdminRequest(req *rpctypes.Request) *rpctypes.Response
}

// NetRequestHandler dispatches net_ namespace JSON-RPC requests.
// *netapi.API satisfies this interface.
type NetRequestHandler interface {
	HandleNetRequest(req *rpctypes.Request) *rpctypes.Response
}

// BeaconRequestHandler dispatches beacon_ namespace JSON-RPC requests.
// *beaconapi.BeaconAPI satisfies this interface.
type BeaconRequestHandler interface {
	HandleBeaconRequest(req *rpctypes.Request) *rpctypes.Response
}

// BatchResponse represents a single response within a JSON-RPC batch.
type BatchResponse struct {
	JSONRPC string             `json:"jsonrpc"`
	Result  interface{}        `json:"result,omitempty"`
	Error   *rpctypes.RPCError `json:"error,omitempty"`
	ID      json.RawMessage    `json:"id"`
}

// BatchHandler processes JSON-RPC batch requests by dispatching each item
// to the underlying RequestHandler, optionally also serving admin_, net_,
// and beacon_ methods.
type BatchHandler struct {
	api            RequestHandler
	adminAPI       AdminRequestHandler
	netAPI         NetRequestHandler
	beaconAPI      BeaconRequestHandler
	allowedModules []string
	parallelism    int
	maxBatchSize   int
	reqSeq         atomic.Int64 // monotonic counter for batch item seq numbers
}

// NewBatchHandler creates a new batch handler for the given RequestHandler.
// It defaults to serving all namespaces ("*"): the HTTP boundary (ExtServer)
// always narrows this to the configured --http.api allowlist via
// SetAllowedModules. Standalone test usage keeps the permissive default.
func NewBatchHandler(api RequestHandler) *BatchHandler {
	return &BatchHandler{
		api:            api,
		allowedModules: []string{"*"},
		parallelism:    DefaultParallelism,
		maxBatchSize:   MaxBatchSize,
	}
}

// SetAdminHandler wires an AdminRequestHandler so admin_ methods are dispatched.
func (bh *BatchHandler) SetAdminHandler(h AdminRequestHandler) {
	bh.adminAPI = h
}

// SetAllowedModules sets the namespace allowlist applied to every batch item.
// nil denies all namespaces; the "*" entry allows all. Mirrors the single
// request path so batches cannot bypass --http.api.
func (bh *BatchHandler) SetAllowedModules(modules []string) {
	bh.allowedModules = modules
}

// SetNetHandler wires a NetRequestHandler so net_ methods are dispatched.
func (bh *BatchHandler) SetNetHandler(h NetRequestHandler) {
	bh.netAPI = h
}

// SetBeaconHandler wires a BeaconRequestHandler so beacon_ methods are dispatched.
func (bh *BatchHandler) SetBeaconHandler(h BeaconRequestHandler) {
	bh.beaconAPI = h
}

// SetMaxBatchSize sets the maximum number of items in a single batch.
func (bh *BatchHandler) SetMaxBatchSize(n int) {
	if n < 1 {
		n = 1
	}
	bh.maxBatchSize = n
}

// SetParallelism sets the number of goroutines for parallel execution.
func (bh *BatchHandler) SetParallelism(n int) {
	if n < 1 {
		n = 1
	}
	bh.parallelism = n
}

// Parallelism returns the current concurrency limit.
func (bh *BatchHandler) Parallelism() int {
	return bh.parallelism
}

// HandleBatch parses body as a JSON-RPC batch and executes it.
func (bh *BatchHandler) HandleBatch(body []byte) ([]BatchResponse, error) {
	return bh.HandleBatchContext(context.Background(), body)
}

// HandleBatchContext carries one HTTP lifetime into every admitted batch item.
func (bh *BatchHandler) HandleBatchContext(ctx context.Context, body []byte) ([]BatchResponse, error) {
	requests, err := ParseBatchRequests(body)
	if err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return nil, ErrBatchEmpty
	}
	if len(requests) > bh.maxBatchSize {
		if bh.maxBatchSize == MaxBatchSize {
			return nil, ErrBatchTooLarge
		}
		return nil, fmt.Errorf("rpc: batch exceeds maximum size of %d", bh.maxBatchSize)
	}
	return bh.ExecuteParallelContext(ctx, requests), nil
}

// ExecuteParallel executes batch items in parallel and returns results in order.
func (bh *BatchHandler) ExecuteParallel(requests []BatchRequest) []BatchResponse {
	return bh.ExecuteParallelContext(context.Background(), requests)
}

// ExecuteParallelContext bounds admission and stops queued work after cancellation.
func (bh *BatchHandler) ExecuteParallelContext(ctx context.Context, requests []BatchRequest) []BatchResponse {
	responses := make([]BatchResponse, len(requests))
	sem := make(chan struct{}, bh.parallelism)
	var wg sync.WaitGroup
	for i, req := range requests {
		if ctx.Err() == nil {
			select {
			case sem <- struct{}{}:
				if ctx.Err() == nil {
					wg.Add(1)
					go func(idx int, request BatchRequest) {
						defer wg.Done()
						defer func() { <-sem }()
						responses[idx] = bh.executeOneContext(ctx, request)
					}(i, req)
					continue
				}
				<-sem
			case <-ctx.Done():
			}
		}
		responses[i] = BatchResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpctypes.RPCError{Code: -32000, Message: ctx.Err().Error()}}
	}
	wg.Wait()
	return responses
}

func isAdminMethod(method string) bool {
	return len(method) > 6 && method[:6] == "admin_"
}

// methodNamespace returns the JSON-RPC namespace of a method, i.e. the part
// before the first underscore. These helpers mirror rpcserver's (rpcbatch
// cannot import rpcserver without an import cycle).
func methodNamespace(method string) string {
	if idx := strings.Index(method, "_"); idx >= 0 {
		return method[:idx]
	}
	return method
}

// methodAllowed reports whether method passes the namespace allowlist.
// A nil modules list denies everything; the "*" entry allows everything.
func methodAllowed(method string, modules []string) bool {
	ns := strings.ToLower(methodNamespace(method))
	for _, m := range modules {
		if m == "*" || strings.EqualFold(m, ns) {
			return true
		}
	}
	return false
}

// methodUnavailableResponse builds the geth-style denial used when a method
// is not served under the configured module allowlist.
func methodUnavailableResponse(method string) *rpctypes.Response {
	return &rpctypes.Response{
		JSONRPC: "2.0",
		Error: &rpctypes.RPCError{
			Code:    rpctypes.ErrCodeMethodNotFound,
			Message: fmt.Sprintf("the method %s does not exist/is not available", method),
		},
	}
}

func isNetMethod(method string) bool {
	return len(method) > 4 && method[:4] == "net_"
}

func isBeaconMethod(method string) bool {
	return len(method) > 7 && method[:7] == "beacon_"
}

func (bh *BatchHandler) executeOne(req BatchRequest) BatchResponse {
	return bh.executeOneContext(context.Background(), req)
}

func (bh *BatchHandler) executeOneContext(ctx context.Context, req BatchRequest) BatchResponse {
	if req.JSONRPC != "2.0" {
		return BatchResponse{
			JSONRPC: "2.0",
			Error:   &rpctypes.RPCError{Code: rpctypes.ErrCodeInvalidRequest, Message: "invalid jsonrpc version"},
			ID:      req.ID,
		}
	}
	if req.Method == "" {
		return BatchResponse{
			JSONRPC: "2.0",
			Error:   &rpctypes.RPCError{Code: rpctypes.ErrCodeInvalidRequest, Message: "method is required"},
			ID:      req.ID,
		}
	}
	if !methodAllowed(req.Method, bh.allowedModules) {
		resp := methodUnavailableResponse(req.Method)
		return BatchResponse{JSONRPC: "2.0", Error: resp.Error, ID: req.ID}
	}
	seq := bh.reqSeq.Add(1)
	start := time.Now()
	batchLog.Debug("rpc_batch_start",
		"event", "rpc_batch_start",
		"seq", seq,
		"method", req.Method,
	)

	apiReq := &rpctypes.Request{
		JSONRPC: req.JSONRPC,
		Method:  req.Method,
		Params:  req.Params,
		ID:      req.ID,
	}
	apiReq = apiReq.WithContext(ctx)
	var resp *rpctypes.Response
	switch {
	case isAdminMethod(req.Method) && bh.adminAPI != nil:
		resp = bh.adminAPI.HandleAdminRequest(apiReq)
	case isNetMethod(req.Method) && bh.netAPI != nil:
		resp = bh.netAPI.HandleNetRequest(apiReq)
	case isBeaconMethod(req.Method) && bh.beaconAPI != nil:
		resp = bh.beaconAPI.HandleBeaconRequest(apiReq)
	default:
		resp = bh.api.HandleRequest(apiReq)
	}

	latency := time.Since(start)
	hasErr := resp != nil && resp.Error != nil
	slow := latency > 500*time.Millisecond
	if hasErr || slow {
		batchLog.Info("rpc_batch_done",
			"event", "rpc_batch_done",
			"seq", seq,
			"method", req.Method,
			"latency_ms", latency.Milliseconds(),
			"error", hasErr,
			"slow", slow,
		)
	} else {
		batchLog.Info("rpc_batch_done",
			"event", "rpc_batch_done",
			"seq", seq,
			"method", req.Method,
			"latency_ms", latency.Milliseconds(),
		)
	}

	return BatchResponse{
		JSONRPC: resp.JSONRPC,
		Result:  resp.Result,
		Error:   resp.Error,
		ID:      resp.ID,
	}
}

// MarshalBatchResponse serializes a batch of responses to JSON.
func MarshalBatchResponse(responses []BatchResponse) ([]byte, error) {
	return json.Marshal(responses)
}

// ExtendedBatchHandler adds validation, statistics, and parallel execution
// on top of basic batch handling.
type ExtendedBatchHandler struct {
	api            RequestHandler
	allowedModules []string
	parallelism    int
	stats          BatchStats
	validator      *BatchValidator
}

// NewExtendedBatchHandler creates an extended batch handler. Like
// NewBatchHandler it defaults to all namespaces; the HTTP boundary narrows it.
func NewExtendedBatchHandler(api RequestHandler) *ExtendedBatchHandler {
	return &ExtendedBatchHandler{
		api:            api,
		allowedModules: []string{"*"},
		parallelism:    DefaultParallelism,
		validator:      NewBatchValidator(MaxBatchSize),
	}
}

// SetAllowedModules sets the namespace allowlist applied to every batch item.
// nil denies all namespaces; the "*" entry allows all.
func (bh *ExtendedBatchHandler) SetAllowedModules(modules []string) {
	bh.allowedModules = modules
}

// SetParallelism sets the concurrency limit.
func (bh *ExtendedBatchHandler) SetParallelism(n int) {
	if n < 1 {
		n = 1
	}
	bh.parallelism = n
}

// Parallelism returns the current concurrency limit.
func (bh *ExtendedBatchHandler) Parallelism() int {
	return bh.parallelism
}

// HandleBatchValidated parses, validates, and executes a batch.
func (bh *ExtendedBatchHandler) HandleBatchValidated(body []byte) ([]BatchResponse, error) {
	requests, err := ParseBatchRequests(body)
	if err != nil {
		return nil, err
	}
	if err := bh.validator.ValidateBatchSize(len(requests)); err != nil {
		return nil, err
	}
	itemErrors := bh.validator.Validate(requests)
	bh.stats.TotalBatches.Add(1)
	bh.stats.TotalRequests.Add(uint64(len(requests)))
	current := uint64(len(requests))
	for {
		largest := bh.stats.LargestBatch.Load()
		if current <= largest {
			break
		}
		if bh.stats.LargestBatch.CompareAndSwap(largest, current) {
			break
		}
	}
	return bh.executeWithValidation(requests, itemErrors), nil
}

func (bh *ExtendedBatchHandler) executeWithValidation(requests []BatchRequest, itemErrors []error) []BatchResponse {
	n := len(requests)
	responses := make([]BatchResponse, n)
	sem := make(chan struct{}, bh.parallelism)
	var wg sync.WaitGroup
	for i, req := range requests {
		if itemErrors[i] != nil {
			responses[i] = BatchResponse{
				JSONRPC: "2.0",
				Error:   &rpctypes.RPCError{Code: rpctypes.ErrCodeInvalidRequest, Message: itemErrors[i].Error()},
				ID:      req.ID,
			}
			bh.stats.TotalErrors.Add(1)
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, r BatchRequest) {
			defer wg.Done()
			defer func() { <-sem }()
			if !methodAllowed(r.Method, bh.allowedModules) {
				resp := methodUnavailableResponse(r.Method)
				responses[idx] = BatchResponse{JSONRPC: "2.0", Error: resp.Error, ID: r.ID}
				bh.stats.TotalErrors.Add(1)
				return
			}
			apiReq := &rpctypes.Request{
				JSONRPC: r.JSONRPC,
				Method:  r.Method,
				Params:  r.Params,
				ID:      r.ID,
			}
			resp := bh.api.HandleRequest(apiReq)
			responses[idx] = BatchResponse{
				JSONRPC: resp.JSONRPC,
				Result:  resp.Result,
				Error:   resp.Error,
				ID:      resp.ID,
			}
			if resp.Error != nil {
				bh.stats.TotalErrors.Add(1)
			}
		}(i, req)
	}
	wg.Wait()
	bh.stats.ParallelBatches.Add(1)
	return responses
}

// Stats returns a snapshot of current batch statistics.
func (bh *ExtendedBatchHandler) Stats() BatchStatsSnapshot {
	return bh.stats.Snapshot()
}
