package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

type validationRawTransactions [][]byte

func (r validationRawTransactions) Len() int                           { return len(r) }
func (r validationRawTransactions) EncodeIndex(i int, b *bytes.Buffer) { b.Write(r[i]) }

// Mutant headers commit to the exact raw transactions, including malformed
// envelopes. Rejection must name the intended guard, not a hash/root shortcut.
func TestSignedOrdinaryAdmissionAndBoundEngineRejections(t *testing.T) {
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
			const limit uint64 = 5000000000
			const token = "signed-envelope-validation"
			key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
			if err != nil {
				t.Fatal(err)
			}
			maxKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 3))
			if err != nil {
				t.Fatal(err)
			}
			wrappedKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 4))
			if err != nil {
				t.Fatal(err)
			}
			wrappedSender := gethcrypto.PubkeyToAddress(wrappedKey.PublicKey)
			sender := gethcrypto.PubkeyToAddress(key.PublicKey)
			maxSender := gethcrypto.PubkeyToAddress(maxKey.PublicKey)
			target := gethcommon.Address{19: 0x42}
			miner := types.Address{19: 0x43}
			var genesis map[string]any
			if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
				t.Fatal(err)
			}
			genesis["gasLimit"] = hexutil.EncodeUint64(limit)
			forks := genesis["config"].(map[string]any)
			forks["terminalTotalDifficulty"] = 0
			if amsterdam {
				forks["amsterdamTime"] = 0
				forks["bpo1Time"] = 0
				forks["bpo2Time"] = 0
			} else {
				delete(forks, "amsterdamTime")
			}
			alloc := genesis["alloc"].(map[string]any)
			for _, address := range []gethcommon.Address{sender, maxSender, wrappedSender, {}} {
				alloc[address.Hex()] = map[string]string{"balance": "100000000000000000000000"}
			}
			alloc[maxSender.Hex()] = map[string]string{"balance": "100000000000000000000000", "nonce": "0xffffffffffffffff"}
			alloc[target.Hex()] = map[string]string{"balance": "5", "nonce": "0x1", "code": "0x00"}
			alloc[miner.Hex()] = map[string]string{"balance": "1", "nonce": "0x1"}
			input, err := json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}
			cfg := func() *Config {
				c := makeTestConfig(t)
				c.EngineAuthToken = token
				c.MinerGasLimit = limit
				c.GenesisPath = filepath.Join(c.DataDir, "genesis.json")
				if err := os.WriteFile(c.GenesisPath, input, 0600); err != nil {
					t.Fatal(err)
				}
				return &c
			}
			producer, receiver := newTestNode(t, cfg()), newTestNode(t, cfg())
			writer, reader := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
			chain := big.NewInt(1337)
			tip, fee := big.NewInt(1000000000), big.NewInt(2000000000)
			sign := func(nonce, gas uint64, which int) *gethtypes.Transaction {
				t.Helper()
				k := key
				if which == 3 {
					k = maxKey
				}
				tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: chain, Nonce: nonce, Gas: gas, GasFeeCap: fee, GasTipCap: tip, To: &target, Value: new(big.Int)}), gethtypes.LatestSignerForChainID(chain), k)
				if err != nil {
					t.Fatal(err)
				}
				return tx
			}
			encode := func(tx *gethtypes.Transaction) []byte {
				t.Helper()
				raw, err := tx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				return raw
			}
			good := sign(0, 500000, 1)
			var submitted types.Hash
			writer.call(t, "eth_sendRawTransaction", &submitted, hexutil.Encode(encode(good)))
			if submitted != types.Hash(good.Hash()) {
				t.Fatal("positive signed hash changed")
			}
			fcu, get, newPayload := "engine_forkchoiceUpdatedV3", "engine_getPayloadV5", "engine_newPayloadV4"
			if amsterdam {
				fcu, get, newPayload = "engine_forkchoiceUpdatedV4", "engine_getPayloadV6", "engine_newPayloadV5"
			}
			beacon := types.Hash{31: 1}
			attrs := map[string]any{"timestamp": "0xc", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": miner, "parentBeaconBlockRoot": beacon}
			if amsterdam {
				attrs["slotNumber"] = "0x1"
				attrs["targetGasLimit"] = hexutil.EncodeUint64(limit)
			}
			parent := producer.blockchain.Genesis().Hash()
			var fc payload.ForkchoiceUpdatedResult
			writer.call(t, fcu, &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
			if fc.PayloadID == nil {
				t.Fatal("positive build did not start")
			}
			var built struct {
				ExecutionPayload  payload.ExecutionPayloadV5
				ExecutionRequests []string
			}
			writer.call(t, get, &built, fc.PayloadID)
			p := built.ExecutionPayload
			if len(p.Transactions) != 1 || p.GasLimit != limit {
				t.Fatalf("positive control count=%d gas limit=%d", len(p.Transactions), p.GasLimit)
			}
			send := func(t *testing.T, q payload.ExecutionPayloadV5, hashes []types.Hash) payload.PayloadStatusV1 {
				t.Helper()
				var status payload.PayloadStatusV1
				if hashes == nil {
					hashes = []types.Hash{}
				}
				if amsterdam {
					reader.call(t, newPayload, &status, q, hashes, beacon, built.ExecutionRequests)
				} else {
					reader.call(t, newPayload, &status, q.ExecutionPayloadV3, hashes, beacon, built.ExecutionRequests)
				}
				return status
			}
			if status := send(t, p, []types.Hash{}); status.Status != "VALID" {
				t.Fatalf("positive independent native receiver: %+v", status)
			}
			reader.call(t, fcu, &fc, payload.ForkchoiceStateV1{HeadBlockHash: p.BlockHash}, nil)
			if fc.PayloadStatus.Status != "VALID" {
				t.Fatal("positive head selection failed")
			}
			frozenRoot := receiver.blockchain.CurrentBlock().Root()
			headerBytes, err := receiver.blockchain.CurrentBlock().Header().EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			var reference gethtypes.Header
			if err := gethrlp.DecodeBytes(headerBytes, &reference); err != nil {
				t.Fatal(err)
			}
			if types.Hash(reference.Hash()) != p.BlockHash {
				t.Fatal("SDK disagrees with complete positive header commitment")
			}
			// Encode a correctly signed envelope whose sole malformed field is the
			// prohibited creation recipient. Native and SDK consensus parsers reject it.
			missingRecipient := func(kind byte) []byte {
				t.Helper()
				fields := []interface{}{chain, uint64(0), tip, fee, uint64(500000), []byte{}, new(big.Int), []byte{}, []gethtypes.AccessTuple{}}
				if kind == 3 {
					fields = append(fields, big.NewInt(1), []gethcommon.Hash{{0: 1, 31: 1}})
				} else {
					authKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
					if err != nil {
						t.Fatal(err)
					}
					auth, err := gethtypes.SignSetCode(authKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: target, Nonce: 0})
					if err != nil {
						t.Fatal(err)
					}
					fields = append(fields, []gethtypes.SetCodeAuthorization{auth})
				}
				unsigned, err := gethrlp.EncodeToBytes(fields)
				if err != nil {
					t.Fatal(err)
				}
				sig, err := gethcrypto.Sign(gethcrypto.Keccak256([]byte{kind}, unsigned), key)
				if err != nil {
					t.Fatal(err)
				}
				fields = append(fields, uint64(sig[64]), new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:64]))
				raw, err := gethrlp.EncodeToBytes(fields)
				if err != nil {
					t.Fatal(err)
				}
				return append([]byte{kind}, raw...)
			}
			v, r, s := good.RawSignatureValues()
			malleable := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: chain, Gas: good.Gas(), GasFeeCap: fee, GasTipCap: tip, To: &target, Value: new(big.Int), V: new(big.Int).Xor(v, big.NewInt(1)), R: r, S: new(big.Int).Sub(gethcrypto.S256().Params().N, s)})
			cap := uint64(1<<24) + 1
			if amsterdam {
				cap = uint64(1) << 32
			}
			emptyAuth, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), Gas: 500000, GasFeeCap: uint256.NewInt(2000000000), GasTipCap: uint256.NewInt(1000000000), To: target}), gethtypes.LatestSignerForChainID(chain), key)
			if err != nil {
				t.Fatal(err)
			}
			cases := []struct {
				name   string
				raw    []byte
				reason string
				hashes []types.Hash
			}{
				{"high_s", encode(malleable), "signature", nil},
				{"nonce_overflow", encode(sign(^uint64(0), 500000, 3)), "overflow", nil},
				{"empty_authorizations", encode(emptyAuth), "authorizations", nil},
				{"outer_gas_cap", encode(sign(0, cap, 1)), "maximum", nil},
				{"blob_creation_recipient", missingRecipient(3), "recipient", []types.Hash{{0: 1, 31: 1}}},
				{"set_code_creation_recipient", missingRecipient(4), "recipient", nil},
			}

			// A real pooled blob wrapper is valid RPC transport with valid KZG cell
			// proofs. The same raw bytes are forbidden in an execution block body.
			wrapped, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), Gas: 500000, GasFeeCap: uint256.NewInt(2000000000), GasTipCap: uint256.NewInt(1000000000), To: target, Value: new(uint256.Int), BlobFeeCap: uint256.NewInt(1), BlobHashes: sidecar.BlobHashes(), Sidecar: sidecar}), gethtypes.LatestSignerForChainID(chain), wrappedKey)
			if err != nil {
				t.Fatal(err)
			}
			wrappedRaw := encode(wrapped)
			var wrapperHash types.Hash
			reader.call(t, "eth_sendRawTransaction", &wrapperHash, hexutil.Encode(wrappedRaw))
			if wrapperHash != types.Hash(wrapped.Hash()) {
				t.Fatal("valid pooled wrapper identity changed")
			}
			wrappedPayload := p
			wrappedPayload.Transactions = [][]byte{wrappedRaw}
			wrappedPayload.BlobGasUsed = 131072
			wrappedHeader := reference
			wrappedHeader.TxHash = gethtypes.DeriveSha(validationRawTransactions(wrappedPayload.Transactions), gethtrie.NewStackTrie(nil))
			wrappedHeader.BlobGasUsed = new(uint64)
			*wrappedHeader.BlobGasUsed = 131072
			wrappedPayload.BlockHash = types.Hash(wrappedHeader.Hash())
			wrappedHashes := []types.Hash{types.Hash(wrapped.BlobHashes()[0])}
			wrapperStatus := send(t, wrappedPayload, wrappedHashes)
			if wrapperStatus.Status != "INVALID" || wrapperStatus.ValidationError == nil || !strings.Contains(*wrapperStatus.ValidationError, "wrapper") {
				t.Fatalf("valid pooled wrapper was not rejected at block ingress: %+v", wrapperStatus)
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					var hash types.Hash
					err := reader.invoke("eth_sendRawTransaction", &hash, hexutil.Encode(tc.raw))
					if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.reason) {
						t.Fatalf("admission did not reach %s guard: %v", tc.reason, err)
					}
					q := p
					q.Transactions = [][]byte{tc.raw}
					mutant := reference
					if len(tc.hashes) > 0 {
						q.BlobGasUsed = uint64(len(tc.hashes)) * 131072
						mutant.BlobGasUsed = new(uint64)
						*mutant.BlobGasUsed = q.BlobGasUsed
					}
					mutant.TxHash = gethtypes.DeriveSha(validationRawTransactions(q.Transactions), gethtrie.NewStackTrie(nil))
					q.BlockHash = types.Hash(mutant.Hash())
					status := send(t, q, tc.hashes)
					if status.Status != "INVALID" || status.ValidationError == nil || !strings.Contains(strings.ToLower(*status.ValidationError), tc.reason) {
						detail := ""
						if status.ValidationError != nil {
							detail = *status.ValidationError
						}
						t.Fatalf("bound receiver did not reach %s guard: status=%s detail=%s", tc.reason, status.Status, detail)
					}

					if tc.reason != "recipient" && (status.LatestValidHash == nil || *status.LatestValidHash != parent) {
						t.Fatalf("invalid branch latestValidHash must be its parent, not the current canonical head: %+v", status)
					}
					if receiver.blockchain.GetBlock(q.BlockHash) != nil {
						t.Fatal("rejected payload was stored as an imported block")
					}
					if receiver.blockchain.CurrentBlock().Hash() != p.BlockHash || receiver.blockchain.CurrentBlock().Root() != frozenRoot {
						t.Fatal("invalid payload or admission changed canonical head/state")
					}
				})
			}
		})
	}
}
