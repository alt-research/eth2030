package node

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFrameBlobSidecarThroughActualHTTPAdmission(t *testing.T) {
	const token = "frame-blob-rpc"
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	fields := genesis["config"].(map[string]any)
	fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"] = 0, 0, 0
	genesis["alloc"].(map[string]any)[sender.Hex()] = map[string]string{"balance": "1000000000000000000"}
	rawGenesis, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeTestConfig(t)
	cfg.GenesisPath, cfg.EngineAuthToken = filepath.Join(cfg.DataDir, "genesis.json"), token
	if err := os.WriteFile(cfg.GenesisPath, rawGenesis, 0600); err != nil {
		t.Fatal(err)
	}
	node := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, node, token)
	blob := make([]byte, types.FrameBytesPerBlob)
	blob[31] = 1
	backend := bls.DefaultKZGBackend()
	commitment, err := backend.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	_, generated, err := backend.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	versioned := sha256.Sum256(commitment[:])
	versioned[0] = 1
	frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender,
		Fees:       types.FrameFeeBudget{MaxFee: big.NewInt(100000000000000000), MaxPriorityFeePerGas: big.NewInt(1000000000)},
		Frames:     []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}},
		Signatures: []types.FrameSignature{{Scheme: 1, Signature: make([]byte, 65)}}, BlobVersionedHashes: []types.Hash{types.Hash(versioned)}}
	hash, err := frame.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	signature, err := gethcrypto.Sign(hash[:], key)
	if err != nil {
		t.Fatal(err)
	}
	frame.Signatures[0].Signature = append([]byte{signature[64]}, signature[:64]...)
	tx := types.NewTransaction(frame)
	sidecar := &types.BlobSidecarData{Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
	for _, proof := range generated {
		sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(proof[:]))
	}
	tx.SetBlobSidecar(sidecar)
	sidecar.Proofs[127] = bytes.Clone(sidecar.Proofs[0])
	bad, err := types.EncodePooledTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	var submitted types.Hash
	if err := rpc.invoke("eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(bad)); err == nil {
		t.Fatal("HTTP accepted incorrect final-cell proof")
	}
	sidecar.Proofs[127] = bytes.Clone(generated[127][:])
	raw, err := types.EncodePooledTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	rpc.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(raw))
	if submitted != tx.Hash() {
		t.Fatal("HTTP changed canonical transaction identity")
	}
	stored := node.TxPool().Get(submitted)
	if stored == nil || stored.BlobSidecar() == nil || len(stored.BlobSidecar().Proofs) != 128 {
		t.Fatal("HTTP admission dropped the blob sidecar")
	}
	var blobs []map[string]any
	rpc.call(t, "engine_getBlobsV2", &blobs, []types.Hash{types.Hash(versioned)})
	if len(blobs) != 1 || blobs[0] == nil || blobs[0]["blob"] != "0x"+hex.EncodeToString(blob) {
		t.Fatal("Engine could not retrieve the admitted Frame blob")
	}
	proofs, ok := blobs[0]["proofs"].([]any)
	if !ok || len(proofs) != len(generated) {
		t.Fatal("Engine did not return the ordered cell proof array")
	}
	for i, proof := range generated {
		if proofs[i] != "0x"+hex.EncodeToString(proof[:]) {
			t.Fatalf("Engine changed cell proof %d", i)
		}
	}
}
