package node

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

func TestHistoricalReceiptsAndTransactionsSurviveCacheEvictionAndNodeRestart(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	authorityKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
	if err != nil {
		t.Fatal(err)
	}
	blobKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 3))
	if err != nil {
		t.Fatal(err)
	}
	var blob kzg4844.Blob
	blob[31] = 1
	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := kzg4844.ComputeCellProofs(&blob)
	if err != nil {
		t.Fatal(err)
	}
	sidecar := &gethtypes.BlobTxSidecar{Version: gethtypes.BlobSidecarVersion1, Blobs: []kzg4844.Blob{blob}, Commitments: []kzg4844.Commitment{commitment}, Proofs: proofs[:]}
	for _, amsterdam := range []bool{false, true} {
		t.Run(map[bool]string{false: "Osaka", true: "Amsterdam"}[amsterdam], func(t *testing.T) {
			const token = "receipt-readback"
			raw, _, writer, _ := simulationV1Genesis(t, 0)
			var genesis map[string]any
			if err := json.Unmarshal(raw, &genesis); err != nil {
				t.Fatal(err)
			}
			if !amsterdam {
				delete(genesis["config"].(map[string]any), "amsterdamTime")
			}
			genesis["config"].(map[string]any)["blobSchedule"] = params.DefaultBlobSchedule
			from := gethcrypto.PubkeyToAddress(key.PublicKey)
			genesis["alloc"].(map[string]any)[from.Hex()] = map[string]string{"balance": "100000000000000000000000"}
			genesis["alloc"].(map[string]any)[gethcrypto.PubkeyToAddress(blobKey.PublicKey).Hex()] = map[string]string{"balance": "100000000000000000000000"}
			raw, err = json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}
			var sdkGenesis gethcore.Genesis
			if err := json.Unmarshal(raw, &sdkGenesis); err != nil {
				t.Fatal(err)
			}
			cfg := makeTestConfig(t)
			cfg.EngineAuthToken = token
			cfg.CacheBlockSize, cfg.CacheReceiptSize = 1, 1
			cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
			if err := os.WriteFile(cfg.GenesisPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			n := newTestNode(t, &cfg)
			rpc := track4NodeRPC(t, n, token)
			auth, err := gethtypes.SignSetCode(authorityKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: gethcommon.Address(writer)})
			if err != nil {
				t.Fatal(err)
			}
			destination := gethcommon.Address(writer)
			unsigned := []gethtypes.TxData{
				&gethtypes.LegacyTx{Nonce: 0, GasPrice: big.NewInt(2000000000), Gas: 600000, Value: new(big.Int), Data: hexutil.MustDecode("0x60015f5260205fa060005f5360015ff3")},
				&gethtypes.AccessListTx{ChainID: big.NewInt(1337), Nonce: 1, To: &destination, GasPrice: big.NewInt(2000000000), Gas: 600000, Value: new(big.Int), Data: []byte{1}},
				&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: 2, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(100000000), Gas: 600000, Value: new(big.Int), Data: hexutil.MustDecode("0x5f5ffd")},
				&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), Nonce: 0, To: destination, GasFeeCap: uint256.NewInt(2000000000), GasTipCap: uint256.NewInt(100000000), Gas: 600000, Value: new(uint256.Int), BlobFeeCap: uint256.NewInt(1000000000), BlobHashes: sidecar.BlobHashes()},
				&gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), Nonce: 3, To: destination, GasFeeCap: uint256.NewInt(2000000000), GasTipCap: uint256.NewInt(100000000), Gas: 600000, Value: new(uint256.Int), AuthList: []gethtypes.SetCodeAuthorization{auth}},
			}
			signed := make([]*gethtypes.Transaction, len(unsigned))
			for i, data := range unsigned {
				signingKey := key
				if _, blob := data.(*gethtypes.BlobTx); blob {
					signingKey = blobKey
				}
				tx, err := gethtypes.SignTx(gethtypes.NewTx(data), gethtypes.LatestSignerForChainID(big.NewInt(1337)), signingKey)
				if err != nil {
					t.Fatal(err)
				}
				signed[i] = tx
				if tx.Type() == gethtypes.BlobTxType {
					tx = tx.WithBlobTxSidecar(sidecar)
				}
				encoded, err := tx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var hash types.Hash
				rpc.call(t, "eth_sendRawTransaction", &hash, hexutil.Encode(encoded))
				if hash != types.Hash(tx.Hash()) {
					t.Fatal("signed transaction identity changed")
				}
			}
			build := func(parent types.Hash, timestamp, slot uint64) types.Hash {
				t.Helper()
				beacon := types.Hash{31: byte(slot)}
				attrs := map[string]any{"timestamp": hexutil.EncodeUint64(timestamp), "slotNumber": hexutil.EncodeUint64(slot), "targetGasLimit": "0x1c9c380", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{19: 0x55}, "parentBeaconBlockRoot": beacon}
				var fc payload.ForkchoiceUpdatedResult
				fcMethod, getMethod, newMethod := "engine_forkchoiceUpdatedV3", "engine_getPayloadV5", "engine_newPayloadV4"
				if amsterdam {
					fcMethod, getMethod, newMethod = "engine_forkchoiceUpdatedV4", "engine_getPayloadV6", "engine_newPayloadV5"
				}
				rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
				if fc.PayloadID == nil {
					t.Fatal("payload missing")
				}
				var built struct {
					ExecutionPayload  payload.ExecutionPayloadV5
					ExecutionRequests []string
				}
				rpc.call(t, getMethod, &built, fc.PayloadID)
				hashes := []types.Hash{}
				if slot == 1 {
					hashes = append(hashes, types.Hash(signed[3].BlobHashes()[0]))
					if len(built.ExecutionPayload.Transactions) != len(signed) {
						t.Fatalf("built %d of %d transactions", len(built.ExecutionPayload.Transactions), len(signed))
					}
				}
				var status payload.PayloadStatusV1
				if amsterdam {
					rpc.call(t, newMethod, &status, built.ExecutionPayload, hashes, beacon, built.ExecutionRequests)
				} else {
					rpc.call(t, newMethod, &status, built.ExecutionPayload.ExecutionPayloadV4.ExecutionPayloadV3, hashes, beacon, built.ExecutionRequests)
				}
				if status.Status != "VALID" {
					t.Fatal(status)
				}
				rpc.call(t, fcMethod, &fc, payload.ForkchoiceStateV1{HeadBlockHash: built.ExecutionPayload.BlockHash}, nil)
				if fc.PayloadStatus.Status != "VALID" {
					t.Fatal(fc)
				}
				if slot == 1 {
					byHash := make(map[gethcommon.Hash]*gethtypes.Transaction)
					for _, tx := range signed {
						byHash[tx.Hash()] = tx
					}
					for i, encoded := range built.ExecutionPayload.Transactions {
						var tx gethtypes.Transaction
						if err := tx.UnmarshalBinary(encoded); err != nil {
							t.Fatal(err)
						}
						signed[i] = byHash[tx.Hash()]
						if signed[i] == nil {
							t.Fatal("unexpected signed transaction in block")
						}
					}
				}
				return built.ExecutionPayload.BlockHash
			}
			first := build(n.blockchain.Genesis().Hash(), 12, 1)
			header := n.blockchain.GetBlock(first).Header()
			encodedHeader, err := header.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			var independentHeader gethtypes.Header
			if err := gethrlp.DecodeBytes(encodedHeader, &independentHeader); err != nil {
				t.Fatal(err)
			}
			if types.Hash(independentHeader.Hash()) != first {
				t.Fatal("independent header hash changed")
			}
			var reference []map[string]any
			read := func(label string) {
				t.Helper()
				var all []map[string]any
				rpc.call(t, "eth_getBlockReceipts", &all, first.Hex())
				if len(all) != len(signed) {
					t.Fatalf("%s receipt count %d", label, len(all))
				}
				var independentReceipts gethtypes.Receipts
				for i, tx := range signed {
					var receipt map[string]any
					rpc.call(t, "eth_getTransactionReceipt", &receipt, tx.Hash().Hex())
					if !reflect.DeepEqual(receipt, all[i]) {
						t.Fatalf("%s receipt lookup differs", label)
					}
					expectedFrom, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(big.NewInt(1337)), tx)
					if err != nil {
						t.Fatal(err)
					}
					if receipt["from"] != types.Address(expectedFrom).Hex() {
						t.Fatalf("%s sender missing: %v", label, receipt["from"])
					}
					encoded, err := json.Marshal(receipt)
					if err != nil {
						t.Fatal(err)
					}
					var actual gethtypes.Receipt
					if err := json.Unmarshal(encoded, &actual); err != nil {
						t.Fatal(err)
					}
					var expected gethtypes.Receipt
					binary, err := actual.MarshalBinary()
					if err != nil {
						t.Fatal(err)
					}
					if err := expected.UnmarshalBinary(binary); err != nil {
						t.Fatal(err)
					}
					price := eip4844.CalcBlobFee(sdkGenesis.Config, &independentHeader)
					expected.DeriveFields(gethtypes.MakeSigner(sdkGenesis.Config, header.Number, header.Time), gethtypes.DeriveReceiptContext{Tx: tx, BaseFee: header.BaseFee, BlobGasPrice: price, BlockHash: independentHeader.Hash(), BlockNumber: header.Number.Uint64(), BlockTime: header.Time, GasUsed: actual.GasUsed, TxIndex: uint(i)})
					if actual.EffectiveGasPrice == nil || actual.EffectiveGasPrice.Cmp(expected.EffectiveGasPrice) != 0 || actual.ContractAddress != expected.ContractAddress || actual.BlobGasUsed != expected.BlobGasUsed || (actual.BlobGasPrice == nil) != (expected.BlobGasPrice == nil) || actual.BlobGasPrice != nil && actual.BlobGasPrice.Cmp(expected.BlobGasPrice) != 0 {
						t.Fatalf("%s transaction %d derived receipt mismatch: price %v/%v, creation %s/%s, blob %d/%d", label, i, actual.EffectiveGasPrice, expected.EffectiveGasPrice, actual.ContractAddress, expected.ContractAddress, actual.BlobGasUsed, expected.BlobGasUsed)
					}
					if i == 2 && actual.Status != 0 || i != 2 && actual.Status != 1 {
						t.Fatalf("%s unexpected transaction %d status %d", label, i, actual.Status)
					}
					independentReceipts = append(independentReceipts, &actual)
					var transaction map[string]any
					rpc.call(t, "eth_getTransactionByHash", &transaction, tx.Hash().Hex())
					if transaction["from"] != types.Address(expectedFrom).Hex() || transaction["blockHash"] != first.Hex() {
						t.Fatalf("%s historical transaction context: %+v", label, transaction)
					}
				}
				if gethtypes.DeriveSha(independentReceipts, gethtrie.NewStackTrie(nil)) != independentHeader.ReceiptHash {
					t.Fatalf("%s full independent receipt trie differs", label)
				}
				if reference == nil {
					reference = all
				} else if !reflect.DeepEqual(reference, all) {
					t.Fatalf("%s changed complete HTTP receipt fields", label)
				}
			}
			read("hot")
			second := build(first, 24, 2)
			root := n.blockchain.CurrentBlock().Root()
			read("cache evicted")
			if err := n.Stop(); err != nil {
				t.Fatal(err)
			}
			n = newTestNode(t, &cfg)
			rpc = track4NodeRPC(t, n, token)
			if n.blockchain.CurrentBlock().Hash() != second || n.blockchain.CurrentBlock().Root() != root {
				t.Fatal("restart changed canonical commitment")
			}
			read("restarted")
		})
	}
}
