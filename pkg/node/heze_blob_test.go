package node

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethkzg "github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
)

func TestGroupELatest7805Type3IgnoredAndPoolBlobAvailability(t *testing.T) {
	const token = "heze-base-blob"
	key, _ := gethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	fields := genesis["config"].(map[string]any)
	fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"] = 0, 0, 0
	genesis["alloc"].(map[string]any)[gethcrypto.PubkeyToAddress(key.PublicKey).Hex()] = map[string]string{"balance": "1000000000000000000"}
	wire, _ := json.Marshal(genesis)
	cfg := makeTestConfig(t)
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	cfg.EngineAuthToken = token
	if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	blob := make([]byte, 131072)
	blob[31] = 9
	kzg := bls.DefaultKZGBackend()
	commitment, err := kzg.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	_, proofs, err := kzg.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), To: gethcommon.HexToAddress("0xbeef"), Gas: 100000, GasFeeCap: uint256.NewInt(2_000_000_000), GasTipCap: uint256.NewInt(1), BlobFeeCap: uint256.NewInt(1_000_000_000), Value: new(uint256.Int), BlobHashes: []gethcommon.Hash{gethcommon.Hash(hash)}}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	list := []string{"0x" + hex.EncodeToString(raw)}
	beacon := types.Hash{31: 1}
	head := payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}
	attrs := func(timestamp, slot string, list []string) map[string]any {
		return map[string]any{"timestamp": timestamp, "slotNumber": slot, "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": list}
	}
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, head, attrs("0xc", "0x1", []string{}))
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	var empty bogotaIntegrationPayload
	rpc.call(t, "engine_getPayloadV6", &empty, fc.PayloadID)
	var status bogotaIntegrationStatus
	rpc.call(t, "engine_newPayloadV6", &status, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal("merged EIP-7805 did not ignore type-3 omission", status)
	}
	bad, err := tx.WithSignature(gethtypes.LatestSignerForChainID(tx.ChainId()), make([]byte, 65))
	if err != nil {
		t.Fatal(err)
	}
	badRaw, _ := bad.MarshalBinary()
	rpc.call(t, "engine_newPayloadV6", &status, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, []string{"0x" + hex.EncodeToString(badRaw)})
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal("invalid blob signature not excused", status)
	}
	requested := attrs("0x18", "0x2", list)
	rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, head, requested)
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	var built struct {
		bogotaIntegrationPayload
		BlobsBundle struct{ Blobs, Commitments, Proofs []json.RawMessage }
	}
	rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	if len(built.ExecutionPayload.Transactions) != 0 || len(built.BlobsBundle.Blobs) != 0 {
		t.Fatal("ignored IL type 3 forced unavailable data into a payload")
	}
	sidecar := &gethtypes.BlobTxSidecar{Version: gethtypes.BlobSidecarVersion1, Blobs: []gethkzg.Blob{gethkzg.Blob(blob)}, Commitments: []gethkzg.Commitment{gethkzg.Commitment(commitment)}}
	for _, proof := range proofs {
		sidecar.Proofs = append(sidecar.Proofs, gethkzg.Proof(proof))
	}
	pooled, err := tx.WithBlobTxSidecar(sidecar).MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var submitted types.Hash
	rpc.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(pooled))
	var selected []string
	rpc.call(t, "engine_getInclusionListV1", &selected)
	if len(selected) != 0 {
		t.Fatal("IL generation selected a type-3 transaction", selected)
	}
	// A separately requested normal-pool build can still include an authenticated
	// blob; ignoring a type-3 IL does not disable normal blob transactions.
	rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, head, attrs("0x19", "0x2", list))
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	if len(built.ExecutionPayload.Transactions) != 1 || len(built.BlobsBundle.Blobs) != 1 || len(built.BlobsBundle.Commitments) != 1 || len(built.BlobsBundle.Proofs) != 128 {
		t.Fatal("authenticated blob wrapper lost at IL priority", len(built.ExecutionPayload.Transactions), len(built.BlobsBundle.Proofs))
	}
	rpc.call(t, "engine_newPayloadV6", &status, built.ExecutionPayload, []types.Hash{types.Hash(hash)}, beacon, built.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal(status)
	}
}
