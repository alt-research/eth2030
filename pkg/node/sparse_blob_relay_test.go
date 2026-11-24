package node

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethengine "github.com/ethereum/go-ethereum/beacon/engine"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethkzg "github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
)

func TestEth72ActualNodeFullPromotionAndEngineCells(t *testing.T) {
	const token = "eth72-node-test"
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	cfgMap := genesis["config"].(map[string]any)
	cfgMap["osakaTime"], cfgMap["amsterdamTime"], cfgMap["terminalTotalDifficulty"] = 0, 0, 0
	genesis["alloc"].(map[string]any)[sender.Hex()] = map[string]string{"balance": "100000000000000000000"}
	input, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	makeCfg := func() *Config {
		cfg := makeTestConfig(t)
		cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
		cfg.EngineAuthToken = token
		if err := os.WriteFile(cfg.GenesisPath, input, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	sourceCfg, targetCfg := makeCfg(), makeCfg()
	source, target := newTestNode(t, sourceCfg), newTestNode(t, targetCfg)
	proposer, receiver := track4NodeRPC(t, source, token), track4NodeRPC(t, target, token)
	// A real builder chooses permanent eager fetching. The sampler role is
	// qualified separately with deterministic native-pool encrypted peers.
	target.ethHandler.PrepareBlobBuilding()
	t.Cleanup(source.p2pServer.Stop)
	t.Cleanup(target.p2pServer.Stop)
	if err := source.p2pServer.Start(); err != nil {
		t.Fatal(err)
	}
	if err := target.p2pServer.Start(); err != nil {
		t.Fatal(err)
	}
	connectSyncNodes(t, source, target)
	backend := bls.DefaultKZGBackend()
	blob := make([]byte, 131072)
	blob[31] = 7
	commitment, err := backend.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	all, proofs, err := backend.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	side := &gethtypes.BlobTxSidecar{Version: 1, Blobs: []gethkzg.Blob{gethkzg.Blob(blob)}, Commitments: []gethkzg.Commitment{gethkzg.Commitment(commitment)}}
	for _, p := range proofs {
		side.Proofs = append(side.Proofs, gethkzg.Proof(p))
	}
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), To: gethcommon.Address{19: 9}, Gas: 100000, GasTipCap: uint256.NewInt(1_000_000_000), GasFeeCap: uint256.NewInt(2_000_000_000), BlobFeeCap: uint256.NewInt(100), Value: new(uint256.Int), BlobHashes: []gethcommon.Hash{hash}}), gethtypes.NewCancunSigner(source.blockchain.Config().ChainID), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.WithBlobTxSidecar(side).MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var submitted types.Hash
	proposer.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(raw))
	deadline := time.Now().Add(12 * time.Second)
	for target.txPool.Get(submitted) == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	stored := target.txPool.Get(submitted)
	if stored == nil || !types.HasFullBlobSidecar(stored) || !bytes.Equal(stored.BlobSidecar().Blobs[0], blob) {
		t.Fatal("real eth/72 cells did not promote reconstructed signed transaction")
	}
	var mask types.CellMask
	mask[0] = 1
	mask[15] = 0x80
	maskText, _ := mask.MarshalText()
	var projection []*struct {
		Cells  []*string `json:"blob_cells"`
		Proofs []*string `json:"proofs"`
	}
	unknown := types.Hash{0: 1, 31: 99}
	receiver.call(t, "engine_getBlobsV4", &projection, []types.Hash{types.Hash(hash), unknown}, string(maskText))
	if len(projection) != 2 || len(projection[0].Cells) != 2 || projection[0].Cells[0] == nil || projection[0].Cells[1] == nil || *projection[0].Cells[1] != "0x"+hex.EncodeToString(all[127][:]) || projection[1] != nil {
		t.Fatal("Engine cells/order/DATA/null changed")
	}
	for i, index := range []uint64{0, 127} {
		if projection[0].Proofs[i] == nil {
			t.Fatal("cell proof missing")
		}
		cell, err := hex.DecodeString((*projection[0].Cells[i])[2:])
		if err != nil {
			t.Fatal(err)
		}
		proof, err := hex.DecodeString((*projection[0].Proofs[i])[2:])
		if err != nil {
			t.Fatal(err)
		}
		valid, err := backend.VerifyCellProof(commitment[:], cell, proof, index)
		if err != nil || !valid {
			t.Fatalf("Engine cell proof %d: %v", index, err)
		}
	}

	for _, method := range []string{"engine_getBlobsV2", "engine_getBlobsV3"} {
		var full []*gethengine.BlobAndProofV2
		receiver.call(t, method, &full, []types.Hash{types.Hash(hash)})
		if len(full) != 1 || full[0] == nil || !bytes.Equal(full[0].Blob, blob) || len(full[0].CellProofs) != 128 {
			t.Fatalf("%s SDK blob projection changed", method)
		}
		for i, proof := range full[0].CellProofs {
			if !bytes.Equal(proof, proofs[i][:]) {
				t.Fatalf("%s stored proof %d changed", method, i)
			}
		}
	}
	if err := receiver.invoke("engine_getBlobsV4", &projection, []types.Hash{types.Hash(hash)}, "0x01"); err == nil {
		t.Fatal("short cell bitmap accepted")
	}
	target.p2pServer.Stop()
	if err := target.Stop(); err != nil {
		t.Fatal(err)
	}
	reopened := newTestNode(t, targetCfg)
	reopened.ethHandler.PrepareBlobBuilding()
	t.Cleanup(reopened.p2pServer.Stop)
	if err := reopened.p2pServer.Start(); err != nil {
		t.Fatal(err)
	}
	connectSyncNodes(t, source, reopened)
	deadline = time.Now().Add(12 * time.Second)
	for reopened.txPool.Get(submitted) == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if tx := reopened.txPool.Get(submitted); tx == nil || !bytes.Equal(tx.BlobSidecar().Blobs[0], blob) {
		t.Fatal("fresh receiver did not reacquire authenticated data from peer pool")
	}
	reread := track4NodeRPC(t, reopened, token)
	reread.call(t, "engine_getBlobsV4", &projection, []types.Hash{types.Hash(hash), unknown}, string(maskText))
	if projection[0] == nil || projection[0].Cells[1] == nil || *projection[0].Cells[1] != "0x"+hex.EncodeToString(all[127][:]) || projection[1] != nil {
		t.Fatal("fresh receiver Engine projection changed")
	}
	t.Logf("real eth/72 signed=%s versioned=%s cells=0,127 reconstructed=%d bytes", submitted, types.Hash(hash), len(blob))
}
