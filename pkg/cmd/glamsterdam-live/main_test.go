package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestRPCResponseIdentityAndFailures(t *testing.T) {
	for _, mode := range []string{"valid-null", "wrong-id", "missing-result", "rpc-error"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID uint64 `json:"id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				response := map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": nil}
				switch mode {
				case "wrong-id":
					response["id"] = request.ID + 1
				case "missing-result":
					delete(response, "result")
				case "rpc-error":
					delete(response, "result")
					response["error"] = map[string]any{"code": -32602, "message": "fixture error"}
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			var trace bytes.Buffer
			c := client{url: server.URL, http: server.Client(), record: &recorder{writer: &trace}}
			raw, err := c.call(context.Background(), "eth_getTransactionReceipt", common.Hash{})
			if mode == "valid-null" {
				if err != nil || !null(raw) {
					t.Fatalf("valid null: %s/%v", raw, err)
				}
			} else if err == nil {
				t.Fatal("invalid response accepted")
			}
			if trace.Len() == 0 || !strings.Contains(trace.String(), "eth_getTransactionReceipt") {
				t.Fatal("request/response evidence missing")
			}
		})
	}
}
func TestCompleteHeaderAndActualTrieCommitments(t *testing.T) {
	for _, fault := range []string{"none", "claim", "requested", "transactions", "receipts"} {
		t.Run(fault, func(t *testing.T) {
			h := &types.Header{Number: big.NewInt(1), Difficulty: new(big.Int), GasLimit: 30000000, Time: 12, BaseFee: big.NewInt(1), UncleHash: types.EmptyUncleHash, TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, Root: common.Hash{31: 1}}
			if fault == "transactions" {
				h.TxHash = common.Hash{31: 2}
			}
			if fault == "receipts" {
				h.ReceiptHash = common.Hash{31: 3}
			}
			raw, err := json.Marshal(h)
			if err != nil {
				t.Fatal(err)
			}
			var block map[string]any
			if err = json.Unmarshal(raw, &block); err != nil {
				t.Fatal(err)
			}
			block["transactions"] = []any{}
			if fault == "claim" {
				block["hash"] = common.Hash{31: 4}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     uint64
					Method string
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				var result any = block
				if req.Method == "eth_getBlockReceipts" {
					result = []any{}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
			}))
			defer server.Close()
			c := client{url: server.URL, http: server.Client(), record: &recorder{writer: io.Discard}}
			requested := h.Hash()
			if fault == "requested" {
				requested = common.Hash{31: 9}
			}
			_, err = verifyBlock(context.Background(), []client{c}, requested)
			if fault == "none" && err != nil {
				t.Fatal(err)
			}
			if fault != "none" && err == nil {
				t.Fatalf("tampered %s commitment accepted", fault)
			}
		})
	}
}
func TestPublishedSequenceResigningPreservesFullEnvelopes(t *testing.T) {
	c, err := loadCorpus()
	if err != nil {
		t.Fatal(err)
	}
	if c.ChainID != 1337 || c.QualifiedSource == "" {
		t.Fatal("unqualified fixture")
	}
	txs, queries := 0, 0
	for _, s := range c.Steps {
		if s.Kind == "query" {
			queries++
			continue
		}
		tx, err := resign(s)
		if err != nil {
			t.Fatal(err)
		}
		if tx.Nonce() != uint64(txs) {
			t.Fatal("missing or reordered signed transaction")
		}
		originalBytes, err := hexutil.Decode(s.Raw)
		if err != nil {
			t.Fatal(err)
		}
		var original types.Transaction
		if err = original.UnmarshalBinary(originalBytes); err != nil {
			t.Fatal(err)
		}
		if tx.Type() != original.Type() || tx.Gas() != original.Gas() || !reflect.DeepEqual(tx.To(), original.To()) || tx.Value().Cmp(original.Value()) != 0 || !bytes.Equal(tx.Data(), original.Data()) || !reflect.DeepEqual(tx.AccessList(), original.AccessList()) || !reflect.DeepEqual(tx.SetCodeAuthorizations(), original.SetCodeAuthorizations()) {
			t.Fatalf("%s complete application fields changed", s.Name)
		}
		sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(1337)), tx)
		if err != nil || sender != common.HexToAddress("0x7e5f4552091a69125d5dfcb7b8c2659029395bdf") {
			t.Fatalf("wrong re-signed sender %s/%v", sender, err)
		}
		txs++
	}
	if txs != 31 || queries != 17 {
		t.Fatalf("incomplete campaign: %d/%d", txs, queries)
	}
}
func TestLiveCampaignRefusesOtherChainsBeforePublishing(t *testing.T) {
	published := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     uint64
			Method string
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "eth_sendRawTransaction" {
			published = true
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x1"})
	}))
	defer server.Close()
	err := run(context.Background(), []string{server.URL, server.URL}, common.Hash{31: 1}, io.Discard)
	if err == nil || published {
		t.Fatalf("chain guard: %v published=%v", err, published)
	}
}
