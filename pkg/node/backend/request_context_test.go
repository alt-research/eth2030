package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/rpc"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

func cancellationBackend(t *testing.T) (*RPCBackend, *state.MemoryStateDB, types.Address) {
	t.Helper()
	db := state.NewMemoryStateDB()
	target := types.Address{18: 0x42, 19: 1}
	// Write storage on the request-owned copy, then loop until execution is stopped.
	db.SetCode(target, []byte{0x60, 1, 0x60, 0, 0x55, 0x5b, 0x60, 5, 0x56})
	header := &types.Header{Number: new(big.Int), GasLimit: 100_000_000, Root: db.GetRoot(), BaseFee: big.NewInt(1)}
	bc, err := chain.NewBlockchain(config.TestConfig, types.NewBlock(header, &types.Body{}), db, rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	b := NewRPCBackend(&mockNodeDeps{blockchain: bc, config: &Config{}}, nil).(*RPCBackend)
	return b, db, target
}

func TestRequestContextStopsNativeExecutionWithoutCrossRequestMutation(t *testing.T) {
	for _, method := range []string{"call", "estimate", "access-list", "trace", "simulate"} {
		t.Run(method, func(t *testing.T) {
			backend, db, target := cancellationBackend(t)
			root := db.GetRoot()
			gas, to := "0x5f5e100", target.Hex()
			args := rpctypes.CallArgs{To: &to, Gas: &gas}
			header := backend.HeaderByNumber(rpc.LatestBlockNumber)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			scoped := backend.WithContext(ctx).(*RPCBackend)
			done := make(chan error, 1)
			go func() {
				var err error
				switch method {
				case "call":
					_, _, err = scoped.CallAtHeader(args, header)
				case "estimate":
					_, err = scoped.EstimateGasAtHeader(args, header)
				case "access-list":
					_, err = scoped.CreateAccessListAtHeader(args, header)
				case "trace":
					_, err = scoped.TraceCallDetailed(args, header, rpctypes.TraceOptions{Tracer: "callTracer", OnlyTopCall: true})
				case "simulate":
					var options rpctypes.SimulateOptions
					if err = json.Unmarshal([]byte(`{"blockStateCalls":[{"calls":[{"to":"`+to+`","gas":"0x2faf080"}]}]}`), &options); err == nil {
						_, err = scoped.Simulate(options, header)
					}
				}
				done <- err
			}()
			select {
			case err := <-done:
				t.Fatalf("execution returned before cancellation: %v", err)
			case <-time.After(50 * time.Millisecond):
			}
			cancel()
			select {
			case err := <-done:
				var wire *rpctypes.RPCError
				if !errors.Is(err, context.Canceled) && !(errors.As(err, &wire) && wire.Code == -32016) {
					t.Fatalf("cancellation error = %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("native execution did not release after cancellation")
			}
			if backend.executionContext().Err() != nil {
				t.Fatal("request cancelled the shared backend")
			}
			if db.GetRoot() != root || db.GetState(target, types.Hash{}) != (types.Hash{}) {
				t.Fatal("cancelled execution changed canonical state")
			}
			empty, little := types.Address{18: 0x43, 19: 1}.Hex(), "0x5208"
			result, _, err := backend.CallAtHeader(rpctypes.CallArgs{To: &empty, Gas: &little}, header)
			if err != nil || len(result) != 0 {
				t.Fatalf("independent request after cancellation: %x %v", result, err)
			}
		})
	}
}

func TestHTTPDisconnectCancelsActualNativeSingleAndBatchCalls(t *testing.T) {
	for _, mode := range []string{"simple", "extended", "batch"} {
		t.Run(mode, func(t *testing.T) {
			backend, db, target := cancellationBackend(t)
			root := db.GetRoot()
			var handler http.Handler
			if mode == "simple" {
				handler = rpc.NewServer(backend).Handler()
			} else {
				handler = rpc.NewExtServer(backend, rpc.ServerConfig{MaxBatchSize: 100}).Handler()
			}
			entered, finished := make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				defer close(finished)
				handler.ServeHTTP(w, r)
			}))
			defer server.Close()
			payload := `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":"` + target.Hex() + `","gas":"0x5f5e100"},{"blockHash":"` + backend.HeaderByNumber(rpc.LatestBlockNumber).Hash().Hex() + `","requireCanonical":true}]}`
			if mode == "batch" {
				payload = "[" + payload + "," + strings.Replace(payload, `"id":1`, `"id":2`, 1) + "]"
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewBufferString(payload))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/json")
			clientDone := make(chan error, 1)
			go func() {
				response, err := server.Client().Do(request)
				if response != nil {
					response.Body.Close()
				}
				clientDone <- err
			}()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("HTTP server never accepted request")
			}
			select {
			case <-finished:
				t.Fatal("real execution finished before disconnect")
			case <-time.After(50 * time.Millisecond):
			}
			cancel()
			select {
			case err := <-clientDone:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("client error = %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("HTTP client did not cancel")
			}
			select {
			case <-finished:
			case <-time.After(time.Second):
				t.Fatal("HTTP handler retained native EVM work after disconnect")
			}
			if db.GetRoot() != root || db.GetState(target, types.Hash{}) != (types.Hash{}) {
				t.Fatal("HTTP cancellation changed committed state")
			}
		})
	}
}

func TestPreCancelledExecutionAvoidsAllNodeStateAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scoped := (&RPCBackend{}).WithContext(ctx).(*RPCBackend)
	calls := []func() error{
		func() error { _, _, err := scoped.CallAtHeader(rpctypes.CallArgs{}, nil); return err },
		func() error { _, err := scoped.EstimateGasAtHeader(rpctypes.CallArgs{}, nil); return err },
		func() error { _, err := scoped.CreateAccessListAtHeader(rpctypes.CallArgs{}, nil); return err },
		func() error {
			_, err := scoped.TraceCallDetailed(rpctypes.CallArgs{}, nil, rpctypes.TraceOptions{Tracer: "callTracer"})
			return err
		},
		func() error { _, err := scoped.Simulate(rpctypes.SimulateOptions{}, nil); return err },
	}
	for index, call := range calls {
		err := call()
		var wire *rpctypes.RPCError
		if !errors.Is(err, context.Canceled) && !(errors.As(err, &wire) && wire.Code == -32016) {
			t.Fatalf("pre-cancelled consumer %d: %v", index, err)
		}
	}
}
