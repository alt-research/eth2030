package node

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

func TestDependency8504ActualRPCSimulationEngineBodiesAndRestart(t *testing.T) {
	const token = "dependency-8504-native"
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	fields := genesis["config"].(map[string]any)
	fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"], fields["eip8504Time"] = 0, 0, 0, 0
	genesis["gasLimit"] = "0x1c9c380"
	tx, _ := proof8504.CanonicalTransaction(false)
	genesis["alloc"].(map[string]any)[tx.FrameSender().Hex()] = map[string]string{"balance": "10000000000000000000", "code": "0x" + hex.EncodeToString(proof8504.CanonicalWalletCode()), "nonce": "0x0"}
	rawGenesis, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	config := func() *Config {
		cfg := makeTestConfig(t)
		cfg.GenesisPath, cfg.EngineAuthToken = filepath.Join(cfg.DataDir, "genesis.json"), token
		if err := os.WriteFile(cfg.GenesisPath, rawGenesis, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	producer := newTestNode(t, config())
	receiverConfig := config()
	receiver := newTestNode(t, receiverConfig)
	source, destination := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	canonical, _ := tx.EncodeRLP()
	var submitted types.Hash
	if err := source.invoke("eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(canonical)); err == nil {
		t.Fatal("HTTP accepted missing witness")
	}
	w := tx.DependencyWitnesses8504()
	w[0][0] ^= 1
	bad, _ := types.EncodePooledTransaction(tx.WithDependencyWitnesses8504(w))
	if err := source.invoke("eth_sendRawTransaction", &submitted, "0x"+hex.EncodeToString(bad)); err == nil {
		t.Fatal("HTTP accepted invalid witness")
	}
	raw, err := types.EncodePooledTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	rawHex := "0x" + hex.EncodeToString(raw)
	var simulation map[string]any
	source.call(t, "eth_simulateFrameV1", &simulation, map[string]any{"rawTransaction": rawHex}, "latest")
	if simulation["valid"] != true {
		t.Fatal("native witness simulation", simulation)
	}
	source.call(t, "eth_sendRawTransaction", &submitted, rawHex)
	if submitted != tx.Hash() {
		t.Fatal("raw submission changed identity")
	}
	beacon := types.Hash{31: 1}
	var fc payload.ForkchoiceUpdatedResult
	source.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{19: 0xfe}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x1c9c380", "inclusionListTransactions": []string{}})
	if fc.PayloadID == nil {
		t.Fatal("native witness build", fc)
	}
	var envelope bogotaIntegrationPayload
	source.call(t, "engine_getPayloadV6", &envelope, fc.PayloadID)
	if len(envelope.ExecutionPayload.Transactions) != 1 || len(envelope.ExecutionPayload.DependencyWitnesses) != 1 {
		t.Fatal("getPayload dropped witness")
	}
	for _, kind := range []string{"missing", "bad", "extra"} {
		encoded, _ := json.Marshal(envelope)
		var candidate bogotaIntegrationPayload
		_ = json.Unmarshal(encoded, &candidate)
		switch kind {
		case "missing":
			candidate.ExecutionPayload.DependencyWitnesses = nil
		case "bad":
			candidate.ExecutionPayload.DependencyWitnesses[0][0] ^= 1
		case "extra":
			candidate.ExecutionPayload.DependencyWitnesses = append(candidate.ExecutionPayload.DependencyWitnesses, candidate.ExecutionPayload.DependencyWitnesses[0])
		}
		var status bogotaIntegrationStatus
		destination.call(t, "engine_newPayloadV6", &status, candidate.ExecutionPayload, []types.Hash{}, beacon, candidate.ExecutionRequests, []string{})
		if status.Status == "VALID" || status.Status == "ACCEPTED" {
			t.Fatal("invalid Engine witness accepted", kind, status)
		}
	}
	for _, rpc := range []track4RPC{source, destination} {
		var status bogotaIntegrationStatus
		rpc.call(t, "engine_newPayloadV6", &status, envelope.ExecutionPayload, []types.Hash{}, beacon, envelope.ExecutionRequests, []string{})
		if status.Status != "VALID" {
			t.Fatal("native witness import", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: envelope.ExecutionPayload.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" {
			t.Fatal("witness forkchoice", fc)
		}
	}
	readBodies := func(rpc track4RPC) {
		t.Helper()
		var bodies []map[string]any
		rpc.call(t, "engine_getPayloadBodiesByHashV2", &bodies, []types.Hash{envelope.ExecutionPayload.BlockHash})
		if len(bodies) != 1 {
			t.Fatal("body unavailable")
		}
		w, ok := bodies[0]["dependencyWitnesses"].([]any)
		if !ok || len(w) != 1 || w[0] != "0x"+hex.EncodeToString(tx.DependencyWitnesses8504()[0]) {
			t.Fatal("retrieval lost exact witness", bodies)
		}
	}
	readBodies(destination)
	var balHex string
	if err := json.Unmarshal(envelope.ExecutionPayload.BlockAccessList, &balHex); err != nil {
		t.Fatal(err)
	}
	balBytes, err := hex.DecodeString(balHex[2:])
	if err != nil {
		t.Fatal(err)
	}
	cl := types.NewConsensusPayload8504(receiver.Blockchain().GetBlock(envelope.ExecutionPayload.BlockHash), balBytes)
	encodedCL, err := cl.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	var decodedCL types.ConsensusPayload8504
	if err := decodedCL.UnmarshalSSZ(encodedCL); err != nil {
		t.Fatal(err)
	}
	clRoot, err := cl.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	decodedRoot, err := decodedCL.HashTreeRoot()
	if err != nil || clRoot != decodedRoot || decodedCL.BlockHash != envelope.ExecutionPayload.BlockHash || len(decodedCL.DependencyWitnesses) != 1 || !bytes.Equal(decodedCL.DependencyWitnesses[0], tx.DependencyWitnesses8504()[0]) || !bytes.Equal(decodedCL.BlockAccessList, balBytes) {
		t.Fatal("native Engine block to CL payload lost commitment/witness/BAL", err)
	}
	// Included proof transactions need no omitted-candidate trial. An omitted
	// proof cannot be authenticated from this canonical-bytes-only IL API.
	var satisfied bogotaIntegrationStatus
	destination.call(t, "engine_newPayloadV6", &satisfied, envelope.ExecutionPayload, []types.Hash{}, beacon, envelope.ExecutionRequests, []string{"0x" + hex.EncodeToString(canonical)})
	if satisfied.Status != "VALID" || satisfied.InclusionListSatisfied == nil || !*satisfied.InclusionListSatisfied {
		t.Fatal("included dependency IL", satisfied)
	}
	omitted := proof8504.Transaction(1)
	omittedRaw, _ := omitted.EncodeRLP()
	if err := destination.invoke("engine_newPayloadV6", &satisfied, envelope.ExecutionPayload, []types.Hash{}, beacon, envelope.ExecutionRequests, []string{"0x" + hex.EncodeToString(omittedRaw)}); err == nil {
		t.Fatal("omitted bare proof received a validity verdict without witness-bearing IL specification")
	}
	if err := receiver.Stop(); err != nil {
		t.Fatal(err)
	}
	receiver = newTestNode(t, receiverConfig)
	destination = track4NodeRPC(t, receiver, token)
	readBodies(destination)
	if receiver.Blockchain().CurrentBlock().Hash() != envelope.ExecutionPayload.BlockHash {
		t.Fatal("restart lost canonical witness block")
	}
}
