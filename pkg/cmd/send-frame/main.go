// send-frame signs and submits the seven-field EIP-8141 envelope pinned by
// ETH2030. A codeless sender uses protocol signature validation; an optional
// second key sponsors the transaction through a separate payment frame.
package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type options struct {
	rpc, key, sponsor, sender, to, data, value, blob, paymaster string
	execution, state                                            uint64
	dry                                                         bool
}

func main() {
	var opts options
	flag.StringVar(&opts.rpc, "rpc", "http://localhost:8545", "JSON-RPC URL")
	flag.StringVar(&opts.key, "key", "", "Sender private key")
	flag.StringVar(&opts.sponsor, "sponsor-key", "", "Optional funded sponsor or paymaster owner private key")
	flag.StringVar(&opts.paymaster, "paymaster", "", "ETH2030 v1 paymaster instance (requires sponsor-key)")
	flag.StringVar(&opts.sender, "sender", "", "Expected sender address (checked against key)")
	flag.StringVar(&opts.to, "to", "", "Application target (defaults to sender)")
	flag.StringVar(&opts.data, "data", "0x", "Application calldata")
	flag.StringVar(&opts.value, "value", "0", "Application value in wei")
	flag.StringVar(&opts.blob, "blob", "", "Optional 131072-byte blob file")
	flag.Uint64Var(&opts.execution, "gas", 100000, "Application execution gas limit")
	flag.Uint64Var(&opts.state, "state-gas", 200000, "Application state gas limit")
	flag.BoolVar(&opts.dry, "dry-run", false, "Print the signed envelope without submitting it")
	flag.Parse()
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type rpcClient struct {
	url    string
	client http.Client
}

func (rpc *rpcClient) call(ctx context.Context, method string, out any, params ...any) error {
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpc.url, bytes.NewReader(request))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := rpc.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int
			Message string
		} `json:"error"`
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s HTTP status %d", method, response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("%s: %d: %s", method, envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 {
		return errors.New("RPC result is missing")
	}
	return json.Unmarshal(envelope.Result, out)
}

func quantity(text string) (*big.Int, error) {
	value, ok := new(big.Int).SetString(text, 0)
	if !ok || value.Sign() < 0 {
		return nil, fmt.Errorf("invalid unsigned quantity %q", text)
	}
	return value, nil
}

func keyFromHex(text string) (*ecdsa.PrivateKey, error) {
	return gethcrypto.HexToECDSA(strings.TrimPrefix(text, "0x"))
}

func makeFrame(opts options, chainID *big.Int, nonce uint64, baseFee, blobFee *big.Int) (*types.Transaction, error) {
	if opts.paymaster != "" && opts.sponsor == "" {
		return nil, errors.New("paymaster requires sponsor-key")
	}
	senderKey, err := keyFromHex(opts.key)
	if err != nil {
		return nil, fmt.Errorf("sender key: %w", err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(senderKey.PublicKey))
	if opts.sender != "" && !strings.EqualFold(opts.sender, sender.Hex()) {
		return nil, errors.New("sender does not match the signing key")
	}
	target := sender
	if opts.to != "" {
		targetBytes, err := hex.DecodeString(strings.TrimPrefix(opts.to, "0x"))
		if err != nil || len(targetBytes) != 20 {
			return nil, errors.New("target must be a 20-byte address")
		}
		copy(target[:], targetBytes)
	}
	data, err := hex.DecodeString(strings.TrimPrefix(opts.data, "0x"))
	if err != nil {
		return nil, err
	}
	value, err := quantity(opts.value)
	if err != nil {
		return nil, err
	}
	frame := &types.FrameTx8141{ChainID: chainID, Nonce: nonce, Sender: sender,
		Fees:   types.FrameFeeBudget{MaxFee: new(big.Int), MaxPriorityFeePerGas: big.NewInt(1_000_000_000)},
		Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}}
	keys := []*ecdsa.PrivateKey{senderKey}
	if opts.sponsor != "" {
		sponsorKey, err := keyFromHex(opts.sponsor)
		if err != nil {
			return nil, fmt.Errorf("sponsor key: %w", err)
		}
		sponsor := types.Address(gethcrypto.PubkeyToAddress(sponsorKey.PublicKey))
		paymentData := []byte(nil)
		paymentGas := uint64(4000)
		if opts.paymaster != "" {
			address, err := hex.DecodeString(strings.TrimPrefix(opts.paymaster, "0x"))
			if err != nil || len(address) != 20 {
				return nil, errors.New("paymaster must be a 20-byte address")
			}
			copy(sponsor[:], address)
			paymentData = framepay.ValidationData()
			paymentGas = 10000
		}
		frame.Frames[0].Flags = 2
		frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &sponsor, Data: paymentData, Limits: types.FrameGasLimits{Execution: paymentGas, State: 183600}})
		keys = append(keys, sponsorKey)
	}
	frame.Frames = append(frame.Frames, types.Frame8141{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: opts.execution, State: opts.state}, Value: value, Data: data})
	var sidecar *types.BlobSidecarData
	if opts.blob != "" {
		blob, err := os.ReadFile(opts.blob)
		if err != nil {
			return nil, err
		}
		if len(blob) != types.FrameBytesPerBlob {
			return nil, errors.New("blob must be exactly 131072 bytes")
		}
		kzg := bls.DefaultKZGBackend()
		commitment, err := kzg.BlobToCommitment(blob)
		if err != nil {
			return nil, err
		}
		_, proofs, err := kzg.ComputeCellsAndProofs(blob)
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256(commitment[:])
		hash[0] = 1
		frame.BlobVersionedHashes = []types.Hash{types.Hash(hash)}
		sidecar = &types.BlobSidecarData{Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
		for _, proof := range proofs {
			sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(proof[:]))
		}
	}
	frame.Signatures = make([]types.FrameSignature, len(keys))
	for i, key := range keys {
		frame.Signatures[i] = types.FrameSignature{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}
		address := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
		if address != sender {
			frame.Signatures[i].Signer = &address
		}
	}
	budget, err := frame.GasBudget()
	if err != nil {
		return nil, err
	}
	price := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), frame.Fees.MaxPriorityFeePerGas)
	frame.Fees.MaxFee.Mul(new(big.Int).SetUint64(budget.MaxGas), price)
	frame.Fees.MaxFee.Add(frame.Fees.MaxFee, new(big.Int).Mul(new(big.Int).SetUint64(uint64(len(frame.BlobVersionedHashes))*131072*2), blobFee))
	hash, err := frame.SigningHash()
	if err != nil {
		return nil, err
	}
	for i, key := range keys {
		sig, err := gethcrypto.Sign(hash[:], key)
		if err != nil {
			return nil, err
		}
		frame.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
	}
	if err := frame.Validate(); err != nil {
		return nil, err
	}
	tx := types.NewTransaction(frame)
	tx.SetBlobSidecar(sidecar)
	return tx, nil
}

func run(opts options) error {
	key, err := keyFromHex(opts.key)
	if err != nil {
		return err
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	rpc := rpcClient{url: opts.rpc, client: http.Client{Timeout: 10 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	var chainText, nonceText string
	if err := rpc.call(ctx, "eth_chainId", &chainText); err != nil {
		return err
	}
	if err := rpc.call(ctx, "eth_getTransactionCount", &nonceText, sender.Hex(), "latest"); err != nil {
		return err
	}
	chainID, err := quantity(chainText)
	if err != nil {
		return err
	}
	nonce, err := quantity(nonceText)
	if err != nil || !nonce.IsUint64() {
		return errors.New("invalid RPC nonce")
	}
	var head struct {
		BaseFee       string `json:"baseFeePerGas"`
		ExcessBlobGas string `json:"excessBlobGas"`
	}
	if err := rpc.call(ctx, "eth_getBlockByNumber", &head, "latest", false); err != nil {
		return err
	}
	base, err := quantity(head.BaseFee)
	if err != nil {
		return err
	}
	blobFee := big.NewInt(1)
	if head.ExcessBlobGas != "" {
		var feeText string
		if err := rpc.call(ctx, "eth_blobBaseFee", &feeText); err != nil {
			return err
		}
		blobFee, err = quantity(feeText)
		if err != nil || blobFee.Sign() <= 0 {
			return errors.New("invalid RPC blob base fee")
		}
	}
	tx, err := makeFrame(opts, chainID, nonce.Uint64(), base, blobFee)
	if err != nil {
		return err
	}
	if opts.paymaster != "" {
		var code string
		if err := rpc.call(ctx, "eth_getCode", &code, opts.paymaster, "latest"); err != nil {
			return err
		}
		decoded, err := hex.DecodeString(strings.TrimPrefix(code, "0x"))
		if err != nil || !framepay.IsV1(decoded) {
			return errors.New("paymaster does not match the frozen ETH2030 v1 runtime")
		}
	}
	raw, err := types.EncodePooledTransaction(tx)
	if err != nil {
		return err
	}
	report := map[string]any{"transaction": tx.Hash().Hex(), "sender": sender.Hex(), "raw": "0x" + hex.EncodeToString(raw)}
	if opts.dry {
		return json.NewEncoder(os.Stdout).Encode(report)
	}
	var hash string
	if err := rpc.call(ctx, "eth_sendRawTransaction", &hash, report["raw"]); err != nil {
		return err
	}
	if !strings.EqualFold(hash, tx.Hash().Hex()) {
		return errors.New("RPC returned a different transaction hash")
	}
	for {
		var receipt map[string]any
		if err := rpc.call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
			return err
		}
		if receipt != nil {
			if receipt["type"] != "0x6" || receipt["status"] != "0x1" {
				return fmt.Errorf("unexpected Frame receipt: %v", receipt)
			}
			report["receipt"] = receipt
			return json.NewEncoder(os.Stdout).Encode(report)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
