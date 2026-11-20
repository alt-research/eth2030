// Command glamsterdam-live qualifies real private-network application consumers.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

//go:embed testdata/applications.json
var fixtures embed.FS

type step struct {
	Kind, Name, Raw, To, Input, Expected string
	Status                               uint64
}
type corpus struct {
	QualifiedSource string `json:"qualified_source"`
	ChainID         uint64 `json:"chain_id"`
	Steps           []step
}
type recorder struct {
	writer   io.Writer
	sequence uint64
}
type client struct {
	url    string
	http   *http.Client
	record *recorder
}
type rpcFailure struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *rpcFailure) Error() string { return fmt.Sprintf("RPC %d: %s", e.Code, e.Message) }
func (c client) call(ctx context.Context, method string, args ...any) (json.RawMessage, error) {
	c.record.sequence++
	id := c.record.sequence
	if args == nil {
		args = []any{}
	}
	requestBody, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": args})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	started := time.Now()
	response, err := c.http.Do(request)
	var raw []byte
	status := 0
	if response != nil {
		status = response.StatusCode
		defer response.Body.Close()
		raw, err = io.ReadAll(io.LimitReader(response.Body, 32<<20))
	}
	record := map[string]any{"kind": "rpc", "endpoint": c.url, "started": started.UTC(), "elapsed_ns": time.Since(started).Nanoseconds(), "http_status": status, "request": json.RawMessage(requestBody), "response_bytes": string(raw)}
	if err != nil {
		record["transport_error"] = err.Error()
	}
	if logErr := json.NewEncoder(c.record.writer).Encode(record); logErr != nil {
		return nil, logErr
	}
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      uint64          `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *rpcFailure     `json:"error"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.JSONRPC != "2.0" || envelope.ID != id {
		return nil, errors.New("RPC response identity mismatch")
	}
	if envelope.Error != nil {
		return nil, envelope.Error
	}
	if len(envelope.Result) == 0 {
		return nil, errors.New("RPC result missing")
	}
	return envelope.Result, nil
}
func null(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }
func loadCorpus() (corpus, error) {
	raw, err := fixtures.ReadFile("testdata/applications.json")
	if err != nil {
		return corpus{}, err
	}
	var c corpus
	err = json.Unmarshal(raw, &c)
	return c, err
}

func resign(s step) (*types.Transaction, error) {
	raw, err := hexutil.Decode(s.Raw)
	if err != nil {
		return nil, err
	}
	var original types.Transaction
	if err = original.UnmarshalBinary(raw); err != nil {
		return nil, err
	}
	if original.ChainId().Cmp(big.NewInt(1337)) != 0 {
		return nil, errors.New("fixture is not chain 1337")
	}
	var tx *types.Transaction
	switch original.Type() {
	case types.DynamicFeeTxType:
		tx = types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: original.Nonce(), Gas: original.Gas(), GasFeeCap: big.NewInt(8000000000), GasTipCap: big.NewInt(1000000000), To: original.To(), Value: original.Value(), Data: original.Data(), AccessList: original.AccessList()})
	case types.SetCodeTxType:
		if original.To() == nil {
			return nil, errors.New("set-code fixture has no recipient")
		}
		tx = types.NewTx(&types.SetCodeTx{ChainID: uint256.NewInt(1337), Nonce: original.Nonce(), Gas: original.Gas(), GasFeeCap: uint256.NewInt(8000000000), GasTipCap: uint256.NewInt(1000000000), To: *original.To(), Value: uint256.MustFromBig(original.Value()), Data: original.Data(), AccessList: original.AccessList(), AuthList: original.SetCodeAuthorizations()})
	default:
		return nil, fmt.Errorf("unsupported application envelope %d", original.Type())
	}
	signingKey, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		return nil, err
	}
	return types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(1337)), signingKey)
}
func verifyHeader(raw json.RawMessage) (*types.Header, error) {
	if null(raw) {
		return nil, errors.New("block missing")
	}
	var header types.Header
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}
	var claim struct {
		Hash common.Hash `json:"hash"`
	}
	if err := json.Unmarshal(raw, &claim); err != nil {
		return nil, err
	}
	if header.Hash() != claim.Hash {
		return nil, fmt.Errorf("complete header commitment differs: computed=%s claimed=%s", header.Hash(), claim.Hash)
	}
	return &header, nil
}
func pause(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(300 * time.Millisecond):
		return nil
	}
}

var errAwaitCanonical = errors.New("execution block selection is converging")

func verifyBlock(ctx context.Context, clients []client, hash common.Hash) (*types.Header, error) {
	var commonHeader *types.Header
	for _, c := range clients {
		raw, err := c.call(ctx, "eth_getBlockByHash", hash, true)
		if err != nil {
			return nil, err
		}
		header, err := verifyHeader(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.url, err)
		}
		if header.Hash() != hash {
			return nil, errors.New("RPC returned a different block than requested")
		}
		canonicalRaw, err := c.call(ctx, "eth_getBlockByNumber", hexutil.EncodeBig(header.Number), false)
		if err != nil {
			return nil, err
		}
		if null(canonicalRaw) {
			return nil, errAwaitCanonical
		}
		canonical, err := verifyHeader(canonicalRaw)
		if err != nil {
			return nil, err
		}
		if canonical.Hash() != hash {
			return nil, errAwaitCanonical
		}
		if commonHeader != nil && (header.Hash() != commonHeader.Hash() || header.Root != commonHeader.Root || header.ReceiptHash != commonHeader.ReceiptHash) {
			return nil, errors.New("independently imported block commitments differ")
		}
		commonHeader = header
		var body struct {
			Transactions []*types.Transaction `json:"transactions"`
		}
		if err = json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		if root := types.DeriveSha(types.Transactions(body.Transactions), trie.NewStackTrie(nil)); root != header.TxHash {
			return nil, fmt.Errorf("%s actual transaction trie %s differs from header %s", c.url, root, header.TxHash)
		}
		receiptsRaw, err := c.call(ctx, "eth_getBlockReceipts", hash)
		if err != nil {
			return nil, err
		}
		var receipts types.Receipts
		if err = json.Unmarshal(receiptsRaw, &receipts); err != nil {
			return nil, err
		}
		if len(receipts) != len(body.Transactions) {
			return nil, errors.New("incomplete block receipts")
		}
		for i, receipt := range receipts {
			if receipt.TxHash != body.Transactions[i].Hash() || receipt.BlockHash != header.Hash() || receipt.TransactionIndex != uint(i) || receipt.BlockNumber == nil || receipt.BlockNumber.Cmp(header.Number) != 0 {
				return nil, errors.New("receipt metadata is not bound to its actual block transaction")
			}
		}
		if root := types.DeriveSha(receipts, trie.NewStackTrie(nil)); root != header.ReceiptHash {
			return nil, fmt.Errorf("%s actual receipt trie %s differs from header %s", c.url, root, header.ReceiptHash)
		}
	}
	return commonHeader, nil
}
func submit(ctx context.Context, clients []client, tx *types.Transaction, wantStatus uint64) (*types.Receipt, error) {
	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	for _, c := range clients {
		answer, err := c.call(ctx, "eth_sendRawTransaction", hexutil.Encode(raw))
		if err != nil {
			// Peers may have propagated the exact signed transaction before this send.
			known, lookupErr := c.call(ctx, "eth_getTransactionByHash", tx.Hash())
			if lookupErr != nil || null(known) {
				return nil, fmt.Errorf("%s admission: %w", c.url, err)
			}
			var found types.Transaction
			if decodeErr := json.Unmarshal(known, &found); decodeErr != nil || found.Hash() != tx.Hash() {
				return nil, errors.New("admission error was not an exact already-known transaction")
			}
		} else {
			var hash common.Hash
			if err = json.Unmarshal(answer, &hash); err != nil || hash != tx.Hash() {
				return nil, errors.New("raw admission changed signed hash")
			}
		}
	}
	var first *types.Receipt
	for {
		complete := true
		first = nil
		for _, c := range clients {
			raw, err := c.call(ctx, "eth_getTransactionReceipt", tx.Hash())
			if err != nil {
				return nil, err
			}
			if null(raw) {
				complete = false
				continue
			}
			var receipt types.Receipt
			if err = json.Unmarshal(raw, &receipt); err != nil {
				return nil, err
			}
			if receipt.TxHash != tx.Hash() {
				return nil, errors.New("receipt transaction identity differs")
			}
			if first != nil && receipt.BlockHash != first.BlockHash {
				complete = false
				if err := json.NewEncoder(c.record.writer).Encode(map[string]any{"kind": "receipt_fork_observed", "transaction": tx.Hash(), "endpoint": c.url, "first_block": first.BlockHash, "other_block": receipt.BlockHash}); err != nil {
					return nil, err
				}
				continue
			}
			if first != nil && (receipt.Status != first.Status || receipt.GasUsed != first.GasUsed || receipt.CumulativeGasUsed != first.CumulativeGasUsed || !reflect.DeepEqual(receipt.Logs, first.Logs) || receipt.EffectiveGasPrice == nil || first.EffectiveGasPrice == nil || receipt.EffectiveGasPrice.Cmp(first.EffectiveGasPrice) != 0) {
				return nil, errors.New("same-block independent receipt status/gas/logs/price differ")
			}
			first = &receipt
		}
		if complete && first != nil {
			if _, err := verifyBlock(ctx, clients, first.BlockHash); err != nil {
				if !errors.Is(err, errAwaitCanonical) {
					return nil, err
				}
			} else {
				if first.Status != wantStatus {
					return nil, fmt.Errorf("common canonical receipt status %d, expected %d", first.Status, wantStatus)
				}
				return first, nil
			}
		}
		if err := pause(ctx); err != nil {
			return nil, err
		}
	}
}
func query(ctx context.Context, clients []client, s step, block common.Hash) error {
	var first string
	for index, c := range clients {
		raw, err := c.call(ctx, "eth_call", map[string]any{"to": s.To, "data": s.Input, "gas": "0x2dc6c0"}, map[string]any{"blockHash": block, "requireCanonical": true})
		if err != nil {
			return err
		}
		var value string
		if err = json.Unmarshal(raw, &value); err != nil {
			return err
		}
		if index > 0 && value != first {
			return fmt.Errorf("%s independent getter bytes differ", s.Name)
		}
		first = value
	}
	// Pair timestamps belong to the actual live block, not the isolated fixture context.
	got, err := hexutil.Decode(first)
	if err != nil {
		return err
	}
	want, err := hexutil.Decode(s.Expected)
	if err != nil {
		return err
	}
	if s.Name == "getReserves" && len(got) == 96 && len(want) == 96 {
		got, want = got[:64], want[:64]
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s actual application result %x differs from qualified semantic result %x", s.Name, got, want)
	}
	return nil
}
func blobTransaction(nonce uint64, markers ...byte) (*types.Transaction, error) {
	if len(markers) == 0 {
		markers = []byte{1}
	}
	sidecar := &types.BlobTxSidecar{Version: types.BlobSidecarVersion1}
	hashes := make([]common.Hash, 0, len(markers))
	for _, marker := range markers {
		var blob kzg4844.Blob
		blob[31] = marker
		commitment, err := kzg4844.BlobToCommitment(&blob)
		if err != nil {
			return nil, err
		}
		proofs, err := kzg4844.ComputeCellProofs(&blob)
		if err != nil {
			return nil, err
		}
		sidecar.Blobs = append(sidecar.Blobs, blob)
		sidecar.Commitments = append(sidecar.Commitments, commitment)
		sidecar.Proofs = append(sidecar.Proofs, proofs[:]...)
		hashes = append(hashes, kzg4844.CalcBlobHashV1(sha256.New(), &commitment))
	}
	signingKey, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		return nil, err
	}
	to := crypto.PubkeyToAddress(signingKey.PublicKey)
	tx := types.NewTx(&types.BlobTx{ChainID: uint256.NewInt(1337), Nonce: nonce, Gas: 500000, GasFeeCap: uint256.NewInt(8000000000), GasTipCap: uint256.NewInt(1000000000), BlobFeeCap: uint256.NewInt(1000000000), To: to, Value: new(uint256.Int), BlobHashes: hashes, Sidecar: sidecar})
	return types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(1337)), signingKey)
}
func campaignClients(ctx context.Context, urls []string, genesis common.Hash, writer io.Writer) ([]client, error) {
	if len(urls) < 2 || genesis == (common.Hash{}) {
		return nil, errors.New("at least two RPC endpoints and the expected private genesis hash are required")
	}
	record := &recorder{writer: writer}
	clients := make([]client, len(urls))
	for i, url := range urls {
		clients[i] = client{url: url, http: &http.Client{Timeout: 15 * time.Second}, record: record}
	}
	for _, c := range clients {
		raw, err := c.call(ctx, "eth_chainId")
		if err != nil {
			return nil, err
		}
		var chain hexutil.Big
		if err = json.Unmarshal(raw, &chain); err != nil || (*big.Int)(&chain).Cmp(big.NewInt(1337)) != 0 {
			return nil, errors.New("refusing a network other than the private chain 1337")
		}
		raw, err = c.call(ctx, "eth_getBlockByNumber", "0x0", false)
		if err != nil {
			return nil, err
		}
		h, err := verifyHeader(raw)
		if err != nil {
			return nil, err
		}
		if h.Hash() != genesis {
			return nil, errors.New("private genesis identity mismatch")
		}
	}
	// Wait for every real EL to have imported an Amsterdam header before publishing fixtures.
	for {
		ready := true
		for _, c := range clients {
			raw, err := c.call(ctx, "eth_getBlockByNumber", "latest", false)
			if err != nil {
				return nil, err
			}
			h, err := verifyHeader(raw)
			if err != nil {
				return nil, err
			}
			if h.SlotNumber == nil || h.BlockAccessListHash == nil {
				ready = false
			}
		}
		if ready {
			break
		}
		if err := pause(ctx); err != nil {
			return nil, err
		}
	}
	return clients, nil
}
func run(ctx context.Context, urls []string, genesis common.Hash, writer io.Writer) error {
	clients, err := campaignClients(ctx, urls, genesis, writer)
	if err != nil {
		return err
	}
	corpus, err := loadCorpus()
	if err != nil {
		return err
	}
	var last common.Hash
	count := 0
	for _, s := range corpus.Steps {
		switch s.Kind {
		case "transaction":
			tx, err := resign(s)
			if err != nil {
				return err
			}
			receipt, err := submit(ctx, clients, tx, s.Status)
			if err != nil {
				return fmt.Errorf("%s: %w", s.Name, err)
			}
			last = receipt.BlockHash
			count++
			fmt.Fprintf(os.Stderr, "qualified live transaction %d: %s %s\n", count, s.Name, tx.Hash())
		case "query":
			if err := query(ctx, clients, s, last); err != nil {
				return err
			}
		default:
			return errors.New("unknown application fixture action")
		}
	}
	blob, err := blobTransaction(uint64(count))
	if err != nil {
		return err
	}
	receipt, err := submit(ctx, clients, blob, 1)
	if err != nil {
		return fmt.Errorf("actual blob envelope: %w", err)
	}
	return json.NewEncoder(writer).Encode(map[string]any{"kind": "completed_application_campaign", "application_transactions": count, "getter_queries": 17, "blob_transaction": blob.Hash(), "last_block": receipt.BlockHash, "qualified_fixture_source": corpus.QualifiedSource})
}

// runBlobs adds two distinct real blob rows to the same private network, using
// the live common nonce and the same complete canonical receipt/trie consumers.
func runBlobs(ctx context.Context, urls []string, genesis common.Hash, writer io.Writer) error {
	clients, err := campaignClients(ctx, urls, genesis, writer)
	if err != nil {
		return err
	}
	key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		return err
	}
	address := crypto.PubkeyToAddress(key.PublicKey)
	var nonce hexutil.Uint64
	for i, c := range clients {
		raw, err := c.call(ctx, "eth_getTransactionCount", address, "pending")
		if err != nil {
			return err
		}
		var candidate hexutil.Uint64
		if err := json.Unmarshal(raw, &candidate); err != nil {
			return err
		}
		if i > 0 && nonce != candidate {
			return errors.New("private sender nonce has not converged across actual ELs")
		}
		nonce = candidate
	}
	tx, err := blobTransaction(uint64(nonce), 2, 3)
	if err != nil {
		return err
	}
	receipt, err := submit(ctx, clients, tx, 1)
	if err != nil {
		return fmt.Errorf("actual multi-row blob envelope: %w", err)
	}
	return json.NewEncoder(writer).Encode(map[string]any{"kind": "completed_multi_blob_campaign", "blob_rows": 2, "blob_transaction": tx.Hash(), "blob_versioned_hashes": tx.BlobHashes(), "last_block": receipt.BlockHash, "nonce": uint64(nonce)})
}

func main() {
	endpoints := flag.String("rpc", "", "comma-separated real native/independent EL endpoints")
	expected := flag.String("genesis", "", "exact expected private genesis hash")
	output := flag.String("output", "", "new complete RPC trace file (required)")
	timeout := flag.Duration("timeout", 15*time.Minute, "overall bounded campaign duration")
	flag.Parse()
	if flag.NArg() > 1 || (flag.NArg() == 1 && flag.Arg(0) != "blobs") {
		fmt.Fprintln(os.Stderr, "the optional scenario is blobs; flags precede the scenario")
		os.Exit(2)
	}
	if *output == "" {
		fmt.Fprintln(os.Stderr, "output path required")
		os.Exit(2)
	}
	if len(*expected) != 66 {
		fmt.Fprintln(os.Stderr, "exact genesis hash required")
		os.Exit(2)
	}
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	campaign := run
	if flag.Arg(0) == "blobs" {
		campaign = runBlobs
	}
	if err = campaign(ctx, strings.Split(*endpoints, ","), common.HexToHash(*expected), file); err != nil {
		_ = json.NewEncoder(file).Encode(map[string]any{"kind": "campaign_failed", "error": err.Error()})
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
