package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func TestGroupECanonicalPQILUsesAuthenticatedPoolWitness(t *testing.T) {
	const token = "heze-pq-pool-witness"
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	fields := genesis["config"].(map[string]any)
	fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"], fields["eip8504Time"] = 0, 0, 0, 0
	tx, _ := proof8504.CanonicalTransaction(false)
	genesis["alloc"].(map[string]any)[tx.FrameSender().Hex()] = map[string]string{"balance": "10000000000000000000", "code": "0x" + hex.EncodeToString(proof8504.CanonicalWalletCode()), "nonce": "0x0"}
	wire, _ := json.Marshal(genesis)
	cfg := makeTestConfig(t)
	cfg.EngineAuthToken = token
	cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
	if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	n := newTestNode(t, &cfg)
	rpc := track4NodeRPC(t, n, token)
	pooled, err := types.EncodePooledTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := tx.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	beacon := types.Hash{31: 1}
	bad, err := types.DecodeBlockTransaction(canonical)
	if err != nil {
		t.Fatal(err)
	}
	badFrame := bad.Inner().(*types.FrameTx8141)
	badFrame.Signatures = []types.FrameSignature{{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}}
	bad = types.NewTransaction(badFrame)
	if !bad.HasDependencies8504() {
		t.Fatal("invalid-signature proof fixture lost dependency shape")
	}
	badRaw, err := bad.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	badList := []string{"0x" + hex.EncodeToString(badRaw)}
	var emptyFC payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV5", &emptyFC, payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": "0x6", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": badList})
	if emptyFC.PayloadID == nil {
		t.Fatal(emptyFC)
	}
	var empty bogotaIntegrationPayload
	rpc.call(t, "engine_getPayloadV6", &empty, emptyFC.PayloadID)
	var invalidStatus bogotaIntegrationStatus
	rpc.call(t, "engine_newPayloadV6", &invalidStatus, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, badList)
	if invalidStatus.Status != "VALID" || invalidStatus.InclusionListSatisfied == nil || !*invalidStatus.InclusionListSatisfied {
		t.Fatal("invalid proof signature treated as missing validity data", invalidStatus)
	}
	foreign, err := types.DecodeBlockTransaction(canonical)
	if err != nil {
		t.Fatal(err)
	}
	foreignFrame := foreign.Inner().(*types.FrameTx8141)
	foreignFrame.ChainID = big.NewInt(1338)
	foreign = types.NewTransaction(foreignFrame)
	keyed, _ := proof8504.CanonicalTransaction(true)
	for i, candidate := range []*types.Transaction{foreign, keyed} {
		t.Run([]string{"foreign_chain", "inactive_nonce_schema"}[i], func(t *testing.T) {
			raw, err := candidate.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			if !candidate.HasDependencies8504() {
				t.Fatal("context-invalid fixture lost dependencies")
			}
			invalidList := []string{"0x" + hex.EncodeToString(raw)}
			var status bogotaIntegrationStatus
			rpc.call(t, "engine_newPayloadV6", &status, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, invalidList)
			if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
				t.Fatal("known-invalid context required omitted witness", status)
			}
			var fc payload.ForkchoiceUpdatedResult
			rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": fmt.Sprintf("0x%x", 7+i), "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": invalidList})
			if fc.PayloadID == nil {
				t.Fatal(fc)
			}
			var built bogotaIntegrationPayload
			rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
			if len(built.ExecutionPayload.Transactions) != 0 {
				t.Fatal("context-invalid dependency tx built")
			}
		})
	}
	var submitted types.Hash
	rpc.call(t, "eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(pooled))
	list := []string{"0x" + hex.EncodeToString(canonical)}
	var fc payload.ForkchoiceUpdatedResult
	rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": list})
	if fc.PayloadID == nil {
		t.Fatal(fc)
	}
	var built bogotaIntegrationPayload
	rpc.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	if len(built.ExecutionPayload.Transactions) != 1 || !bytes.Equal(built.ExecutionPayload.Transactions[0], canonical) || len(built.ExecutionPayload.DependencyWitnesses) != 1 || !bytes.Equal(built.ExecutionPayload.DependencyWitnesses[0], tx.DependencyWitnesses8504()[0]) {
		t.Fatal("IL priority discarded authenticated pooled witness")
	}
	var status bogotaIntegrationStatus
	rpc.call(t, "engine_newPayloadV6", &status, built.ExecutionPayload, []types.Hash{}, beacon, built.ExecutionRequests, list)
	if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
		t.Fatal(status)
	}
}
