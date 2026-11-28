package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/focil/heze"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

// Cross the actual mempool/RPC and CL serialization boundaries. Geth's signed
// MarshalBinary output is the independent oracle, not the selector's EncodeRLP.
func TestGroupELatest2718ExactEnvelopeBudget(t *testing.T) {
	const token = "heze-2718-byte-budget"
	key, err := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"legacy_exact", "typed_exact", "typed_one_over", "pooled_witness"} {
		t.Run(kind, func(t *testing.T) {
			var genesis map[string]any
			if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
				t.Fatal(err)
			}
			fields := genesis["config"].(map[string]any)
			fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"], fields["eip8504Time"] = 0, 0, 0, 0
			genesis["alloc"].(map[string]any)[gethcrypto.PubkeyToAddress(key.PublicKey).Hex()] = map[string]string{"balance": "10000000000000000000"}
			var canonical, pooled []byte
			if kind == "pooled_witness" {
				tx, _ := proof8504.CanonicalTransaction(false)
				genesis["alloc"].(map[string]any)[tx.FrameSender().Hex()] = map[string]string{"balance": "10000000000000000000", "code": "0x" + hex.EncodeToString(proof8504.CanonicalWalletCode())}
				canonical, err = tx.EncodeRLP()
				if err != nil {
					t.Fatal(err)
				}
				pooled, err = types.EncodePooledTransaction(tx)
				if err != nil || bytes.Equal(pooled, canonical) || len(pooled) <= len(canonical) {
					t.Fatal("fixture must carry an external witness", err)
				}
			} else {
				target := heze.MaxTransactionBytes
				if kind == "typed_one_over" {
					target++
				}
				to := gethcrypto.PubkeyToAddress(key.PublicKey)
				for size := target - 160; size < target; size++ {
					var inner gethtypes.TxData = &gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Gas: 1000000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1), To: &to, Data: make([]byte, size)}
					if kind == "legacy_exact" {
						inner = &gethtypes.LegacyTx{Gas: 1000000, GasPrice: big.NewInt(2000000000), To: &to, Data: make([]byte, size)}
					}
					tx, signErr := gethtypes.SignTx(gethtypes.NewTx(inner), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
					if signErr != nil {
						t.Fatal(signErr)
					}
					raw, encodeErr := tx.MarshalBinary()
					if encodeErr != nil {
						t.Fatal(encodeErr)
					}
					if len(raw) != target {
						continue
					}
					blockItem, encodeErr := gethrlp.EncodeToBytes(tx)
					if encodeErr != nil {
						t.Fatal(encodeErr)
					}
					if kind == "legacy_exact" && !bytes.Equal(blockItem, raw) || kind != "legacy_exact" && len(blockItem) <= len(raw) {
						t.Fatal("independent block-item oracle lost the EIP-2718 distinction")
					}
					canonical, pooled = raw, raw
					break
				}
				if len(canonical) != target {
					t.Fatal("could not produce exact signed EIP-2718 boundary", target)
				}
			}
			wire, err := json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}
			cfg := makeTestConfig(t)
			cfg.EngineAuthToken = token
			cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
			if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
				t.Fatal(err)
			}
			n := newTestNode(t, &cfg)
			rpc := track4NodeRPC(t, n, token)
			var submitted types.Hash
			rpc.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(pooled))
			var selected []string
			rpc.call(t, "engine_getInclusionListV1", &selected)
			if kind == "typed_one_over" {
				if len(selected) != 0 {
					t.Fatal("8193 canonical bytes admitted into one IL")
				}
				if n.TxPool().Count() != 1 {
					t.Fatal("IL limit was incorrectly applied to mempool admission")
				}
				return
			}
			if len(selected) != 1 || selected[0] != "0x"+hex.EncodeToString(canonical) {
				t.Fatalf("canonical EIP-2718 selection mismatch: got %d entries; want %d canonical bytes", len(selected), len(canonical))
			}
			message := heze.InclusionList{Transactions: [][]byte{canonical}}
			if err := heze.CheckTransactions(message.Transactions); err != nil {
				t.Fatal("CL byte validation disagreed with Engine selection", err)
			}
			ssz, err := message.MarshalSSZ()
			if err != nil || len(ssz) <= len(canonical) {
				t.Fatal("SSZ framing must remain outside the transaction-byte budget", err)
			}
			var decoded heze.InclusionList
			if err := decoded.UnmarshalSSZ(ssz); err != nil || len(decoded.Transactions) != 1 || !bytes.Equal(decoded.Transactions[0], canonical) {
				t.Fatal("CL framing changed the canonical transaction", err)
			}
		})
	}
}
