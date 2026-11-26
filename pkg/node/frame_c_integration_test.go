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
	"reflect"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/engine/payload"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFrameCNativePeersPoolBuildSimulationAndRestart(t *testing.T) {
	for _, blob := range []bool{false, true} {
		t.Run(fmt.Sprintf("blob_%t", blob), func(t *testing.T) {
			const token = "frame-c-native"
			key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
			sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
			target := types.Address{18: 0x89}
			var genesis map[string]any
			if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
				t.Fatal(err)
			}
			genesis["gasLimit"] = "0x186a0"
			spec := genesis["config"].(map[string]any)
			spec["terminalTotalDifficulty"], spec["amsterdamTime"], spec["bogotaTime"] = 0, 0, 0
			alloc := genesis["alloc"].(map[string]any)
			alloc[sender.Hex()] = map[string]string{"balance": "1000000000000000000"}
			alloc[target.Hex()] = map[string]string{"balance": "0", "nonce": "1", "code": "0x60015f555f5fa060425f5360015ff3"}
			rawGenesis, _ := json.Marshal(genesis)
			config := func() *Config {
				cfg := makeTestConfig(t)
				cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
				cfg.EngineAuthToken = token
				if err := os.WriteFile(cfg.GenesisPath, rawGenesis, 0600); err != nil {
					t.Fatal(err)
				}
				return &cfg
			}
			source := newTestNode(t, config())
			receiverConfig := config()
			receiver := newTestNode(t, receiverConfig)
			if err := source.Start(); err != nil {
				t.Fatal(err)
			}
			if err := receiver.Start(); err != nil {
				t.Fatal(err)
			}
			connectSyncNodes(t, source, receiver)
			a, b := track4NodeRPC(t, source, token), track4NodeRPC(t, receiver, token)
			frame := &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100000000000000000), MaxPriorityFeePerGas: big.NewInt(1000000000)}, Frames: []types.Frame8141{
				{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}},
				{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 30000, State: 97920}},
			}, Signatures: []types.FrameSignature{{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}}}
			var sidecar *types.BlobSidecarData
			if blob {
				data := make([]byte, types.FrameBytesPerBlob)
				data[31] = 1
				backend := bls.DefaultKZGBackend()
				commitment, err := backend.BlobToCommitment(data)
				if err != nil {
					t.Fatal(err)
				}
				_, proofs, err := backend.ComputeCellsAndProofs(data)
				if err != nil {
					t.Fatal(err)
				}
				hash := sha256.Sum256(commitment[:])
				hash[0] = 1
				frame.BlobVersionedHashes = []types.Hash{types.Hash(hash)}
				sidecar = &types.BlobSidecarData{Version: 1, Blobs: [][]byte{data}, Commitments: [][]byte{bytes.Clone(commitment[:])}}
				for _, proof := range proofs {
					sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(proof[:]))
				}
			}
			signingHash, err := frame.SigningHash()
			if err != nil {
				t.Fatal(err)
			}
			signature, err := gethcrypto.Sign(signingHash[:], key)
			if err != nil {
				t.Fatal(err)
			}
			frame.Signatures[0].Signature = append([]byte{signature[64]}, signature[:64]...)
			tx := types.NewTransaction(frame)
			tx.SetBlobSidecar(sidecar)
			plain, err := tx.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			rawHex := "0x" + hex.EncodeToString(plain)
			options := map[string]any{"rawTransaction": rawHex, "slotNumber": "0x1", "blockOverrides": map[string]any{"time": "0xc"}, "trace": true}
			before := source.Blockchain().State().GetRoot()
			var simulated rpctypes.FrameSimulationResult
			a.call(t, "eth_simulateFrameV1", &simulated, options, "latest")
			if !simulated.Valid || simulated.Hash != tx.Hash().Hex() || simulated.SigningHash != signingHash.Hex() || simulated.Payer == nil || *simulated.Payer != sender.Hex() || len(simulated.Frames) != 2 || simulated.Frames[1].StateGasUsed != "0x17e80" || simulated.Frames[1].ReturnData != "0x42" {
				t.Fatalf("incomplete dimensional replay: %+v", simulated)
			}
			if source.Blockchain().State().GetRoot() != before || source.TxPool().Get(tx.Hash()) != nil {
				t.Fatal("simulation leaked canonical or pool effects")
			}
			if simulated.Frames[1].StructLogs == nil {
				t.Fatal("Frame trace omitted executed opcodes")
			}
			noTrace := map[string]any{"rawTransaction": rawHex, "slotNumber": "0x1", "blockOverrides": map[string]any{"time": "0xc"}}
			var untraced rpctypes.FrameSimulationResult
			a.call(t, "eth_simulateFrameV1", &untraced, noTrace, "latest")
			if simulated.GasUsed != untraced.GasUsed || simulated.Escrow != untraced.Escrow || simulated.Refund != untraced.Refund {
				t.Fatal("trace changed settlement")
			}
			for _, method := range []string{"eth_call", "eth_estimateGas", "eth_createAccessList", "debug_traceCall"} {
				var result any
				if err := a.invoke(method, &result, map[string]any{"type": "0x6", "to": target.Hex()}, "latest"); err == nil {
					t.Fatalf("%s silently accepted incomplete type-6 call", method)
				}
			}
			for _, invalid := range []map[string]any{{"rawTransaction": rawHex, "slotNumber": "0x01"}, {"rawTransaction": rawHex, "traceLimit": "0x0"}, {"rawTransaction": rawHex, "unsigned": true}} {
				var result any
				if err := a.invoke("eth_simulateFrameV1", &result, invalid); err == nil {
					t.Fatalf("invalid simulation accepted: %+v", invalid)
				}
			}
			network, err := types.EncodePooledTransaction(tx)
			if err != nil {
				t.Fatal(err)
			}
			if blob {
				// A bad last proof must not reach either public pool or any announcement.
				bad := bytes.Clone(sidecar.Proofs[127])
				sidecar.Proofs[127] = bytes.Clone(sidecar.Proofs[0])
				corrupt, err := types.EncodePooledTransaction(tx)
				if err != nil {
					t.Fatal(err)
				}
				var result any
				if err := a.invoke("eth_sendRawTransaction", &result, "0x"+hex.EncodeToString(corrupt)); err == nil {
					t.Fatal("invalid final cell proof admitted")
				}
				sidecar.Proofs[127] = bad
			}
			if !blob {
				for _, badDATA := range []string{rawHex[2:], "0x6", "0xzz"} {
					var result any
					if err := a.invoke("eth_sendRawTransaction", &result, badDATA); err == nil {
						t.Fatal("submission accepted malformed DATA")
					}
				}
			}
			var submitted types.Hash
			a.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(network))
			if submitted != tx.Hash() {
				t.Fatal("submission changed identity")
			}
			deadline := time.Now().Add(5 * time.Second)
			for receiver.TxPool().Get(tx.Hash()) == nil && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			propagated := receiver.TxPool().Get(tx.Hash())
			if propagated == nil {
				t.Fatal("connected peer did not fetch/admit Frame from public gossip")
			}
			peerRaw, err := types.EncodePooledTransaction(propagated)
			if err != nil || !bytes.Equal(peerRaw, network) {
				t.Fatal("P2P altered envelope or sidecar")
			}
			var pending map[string]any
			b.call(t, "eth_getTransactionByHash", &pending, tx.Hash().Hex())
			if pending["sender"] != sender.Hex() || pending["type"] != "0x6" {
				t.Fatal("pending query lost Frame fields")
			}
			// Build on the receiving node exclusively from its P2P-populated pool.
			beacon := types.Hash{31: 1}
			attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{19: 0xfe}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x186a0", "inclusionListTransactions": []string{}}
			var fc payload.ForkchoiceUpdatedResult
			b.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: receiver.Blockchain().Genesis().Hash()}, attrs)
			if fc.PayloadID == nil || fc.PayloadStatus.Status != "VALID" {
				t.Fatalf("ordinary pool build failed: %+v", fc)
			}
			var envelope bogotaIntegrationPayload
			b.call(t, "engine_getPayloadV6", &envelope, fc.PayloadID)
			if len(envelope.ExecutionPayload.Transactions) != 1 || !bytes.Equal(envelope.ExecutionPayload.Transactions[0], plain) {
				t.Fatal("builder did not select exact public-pool Frame")
			}
			for _, rpc := range []track4RPC{a, b} {
				var status bogotaIntegrationStatus
				rpc.call(t, "engine_newPayloadV6", &status, envelope.ExecutionPayload, append([]types.Hash{}, frame.BlobVersionedHashes...), beacon, envelope.ExecutionRequests, []string{})
				if status.Status != "VALID" {
					t.Fatalf("independent Frame import: %+v", status)
				}
				rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: envelope.ExecutionPayload.BlockHash}, nil)
			}
			if source.TxPool().Get(tx.Hash()) != nil || receiver.TxPool().Get(tx.Hash()) != nil {
				t.Fatal("canonical inclusion retained transaction/reservation")
			}
			var receipt, other map[string]any
			a.call(t, "eth_getTransactionReceipt", &receipt, tx.Hash().Hex())
			b.call(t, "eth_getTransactionReceipt", &other, tx.Hash().Hex())
			if !reflect.DeepEqual(receipt, other) || receipt["payer"] != sender.Hex() {
				t.Fatal("native peers differ on persisted Frame receipt")
			}
			if source.Blockchain().State().GetRoot() != receiver.Blockchain().State().GetRoot() {
				t.Fatal("native imports differ on state root")
			}
			if err := receiver.Stop(); err != nil {
				t.Fatal(err)
			}
			reopened := newTestNode(t, receiverConfig)
			read := track4NodeRPC(t, reopened, token)
			read.call(t, "eth_getTransactionReceipt", &other, tx.Hash().Hex())
			if !reflect.DeepEqual(receipt, other) {
				t.Fatal("restart changed Frame receipt")
			}
		})
	}
}
