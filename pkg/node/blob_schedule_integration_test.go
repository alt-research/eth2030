package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	geth4844 "github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestSignedBlobSchedulesThroughActualHTTPAndRestart(t *testing.T) {
	const token = "blob-schedule-test"
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	contract := types.Address{18: 0xbb, 19: 2}
	prague := &params.BlobConfig{Target: 3, Max: 7, UpdateFraction: 500_000}
	bpo1 := &params.BlobConfig{Target: 4, Max: 10, UpdateFraction: 100_000}
	bpo2 := &params.BlobConfig{Target: 14, Max: 21, UpdateFraction: 200_000}
	var genesis map[string]any
	if err := json.Unmarshal(minimalGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["bpo1Time"], forks["bpo2Time"] = 0, 24, 36
	forks["blobSchedule"] = map[string]any{"cancun": params.DefaultCancunBlobConfig, "prague": prague, "bpo1": bpo1, "bpo2": bpo2, "osaka": prague}
	alloc := genesis["alloc"].(map[string]any)
	alloc[sender.Hex()] = map[string]string{"balance": "1000000000000000000000"}
	alloc[contract.Hex()] = map[string]string{"balance": "0", "nonce": "0x1", "code": "0x4a5f5500"}
	input, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	makeConfig := func() *Config {
		cfg := makeTestConfig(t)
		cfg.EngineAuthToken = token
		cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
		if err := os.WriteFile(cfg.GenesisPath, input, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	producerCfg, receiverCfg := makeConfig(), makeConfig()
	producer, receiver := newTestNode(t, producerCfg), newTestNode(t, receiverCfg)
	proposer, verifier := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	reference := *params.AllDevChainProtocolChanges
	zero, at24, at36 := uint64(0), uint64(24), uint64(36)
	reference.PragueTime, reference.OsakaTime, reference.BPO1Time, reference.BPO2Time = &zero, &zero, &at24, &at36
	reference.AmsterdamTime, reference.BogotaTime = nil, nil
	reference.BlobScheduleConfig = &params.BlobScheduleConfig{Cancun: params.DefaultCancunBlobConfig, Prague: prague, BPO1: bpo1, BPO2: bpo2}
	var blob kzg4844.Blob
	blob[31] = 1
	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		t.Fatal(err)
	}
	cellProofs, err := kzg4844.ComputeCellProofs(&blob)
	if err != nil {
		t.Fatal(err)
	}
	var nonce uint64
	beacon := types.Hash{1}
	for _, step := range []struct {
		time   uint64
		counts []int
	}{{12, []int{6, 1}}, {24, []int{6, 4}}, {36, []int{6, 6, 6, 3}}} {
		parent := producer.Blockchain().CurrentBlock()
		parentRLP, err := parent.Header().EncodeRLP()
		if err != nil {
			t.Fatal(err)
		}
		var refParent gethtypes.Header
		if err := gethrlp.DecodeBytes(parentRLP, &refParent); err != nil {
			t.Fatal(err)
		}
		before := new(big.Int).Set(producer.Blockchain().State().GetBalance(sender))
		hashes := []types.Hash{}
		transactionHashes := []types.Hash{}
		for i, count := range step.counts {
			sidecar := &gethtypes.BlobTxSidecar{Version: gethtypes.BlobSidecarVersion1, Blobs: make([]kzg4844.Blob, count), Commitments: make([]kzg4844.Commitment, count), Proofs: make([]kzg4844.Proof, count*kzg4844.CellProofsPerBlob)}
			for j := 0; j < count; j++ {
				sidecar.Blobs[j], sidecar.Commitments[j] = blob, commitment
				copy(sidecar.Proofs[j*kzg4844.CellProofsPerBlob:], cellProofs)
			}
			body := &gethtypes.BlobTx{ChainID: uint256.NewInt(1337), Nonce: nonce, GasTipCap: uint256.NewInt(uint64(5-i) * 1_000_000_000), GasFeeCap: uint256.NewInt(6_000_000_000), Gas: 100_000, To: gethcommon.Address(contract), Value: new(uint256.Int), BlobFeeCap: uint256.NewInt(1_000_000_000), BlobHashes: sidecar.BlobHashes(), Sidecar: sidecar}
			tx, err := gethtypes.SignNewTx(key, gethtypes.LatestSignerForChainID(big.NewInt(1337)), body)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := tx.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var admitted types.Hash
			proposer.call(t, "eth_sendRawTransaction", &admitted, "0x"+hex.EncodeToString(encoded))
			if admitted != types.Hash(tx.Hash()) {
				t.Fatal("pool changed signed transaction identity")
			}
			transactionHashes = append(transactionHashes, admitted)
			for _, hash := range tx.BlobHashes() {
				hashes = append(hashes, types.Hash(hash))
			}
			nonce++
		}
		var fc payload.ForkchoiceUpdatedResult
		attrs := payload.PayloadAttributesV3{PayloadAttributesV2: payload.PayloadAttributesV2{PayloadAttributesV1: payload.PayloadAttributesV1{Timestamp: step.time}, Withdrawals: []*payload.Withdrawal{}}, ParentBeaconBlockRoot: beacon}
		proposer.call(t, "engine_forkchoiceUpdatedV3", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent.Hash()}, attrs)
		if fc.PayloadID == nil {
			t.Fatalf("no build at %d", step.time)
		}
		var built struct {
			ExecutionPayload  *payload.ExecutionPayloadV3
			ExecutionRequests []string
		}
		proposer.call(t, "engine_getPayloadV5", &built, fc.PayloadID)
		block := built.ExecutionPayload
		wantCount := 0
		for _, count := range step.counts {
			wantCount += count
		}
		if block == nil || len(block.Transactions) != len(step.counts) || block.BlobGasUsed != uint64(wantCount)*gas.GasPerBlob {
			t.Fatalf("time %d lost configured block capacity: %+v", step.time, block)
		}
		wantExcess := geth4844.CalcExcessBlobGas(&reference, &refParent, step.time)
		if block.ExcessBlobGas != wantExcess {
			t.Fatalf("time %d excess %d want independent %d", step.time, block.ExcessBlobGas, wantExcess)
		}
		price := geth4844.CalcBlobFee(&reference, &gethtypes.Header{Time: step.time, ExcessBlobGas: &wantExcess})
		for _, rpc := range []track4RPC{proposer, verifier} {
			var status payload.PayloadStatusV1
			rpc.call(t, "engine_newPayloadV4", &status, block, hashes, beacon, built.ExecutionRequests)
			if status.Status != "VALID" {
				t.Fatalf("independent import at %d: %+v", step.time, status)
			}
			rpc.call(t, "engine_forkchoiceUpdatedV3", &fc, payload.ForkchoiceStateV1{HeadBlockHash: block.BlockHash}, nil)
		}
		spent := new(big.Int)
		for _, hash := range transactionHashes {
			var wireReceipt struct {
				BlobGasPrice string
				BlobGasUsed  string
			}
			proposer.call(t, "eth_getTransactionReceipt", &wireReceipt, hash)
			if wireReceipt.BlobGasPrice != fmt.Sprintf("0x%x", price) {
				t.Fatalf("HTTP receipt price %s want %s", wireReceipt.BlobGasPrice, price)
			}
			receipt := producer.Blockchain().GetReceipts(block.BlockHash)
			found := false
			for _, r := range receipt {
				if r.TxHash == hash {
					found = true
					if r.BlobGasPrice == nil || r.BlobGasPrice.Cmp(price) != 0 {
						t.Fatal("receipt has a different blob price")
					}
					spent.Add(spent, new(big.Int).Mul(new(big.Int).SetUint64(r.GasUsed), r.EffectiveGasPrice))
					spent.Add(spent, new(big.Int).Mul(new(big.Int).SetUint64(r.BlobGasUsed), price))
				}
			}
			if !found {
				t.Fatal("signed receipt unavailable")
			}
		}
		after := producer.Blockchain().State().GetBalance(sender)
		t.Logf("time=%d blobs=%d excess=%d blobFee=%s block=%s root=%s senderSpent=%s", step.time, wantCount, wantExcess, price, block.BlockHash, block.StateRoot, spent)
		if new(big.Int).Sub(before, after).Cmp(spent) != 0 {
			t.Fatal("actual sender settlement differs from execution plus blob fees")
		}
		for _, n := range []*Node{producer, receiver} {
			if n.Blockchain().State().GetState(contract, types.Hash{}) != types.BytesToHash(price.Bytes()) {
				t.Fatal("actual BLOBBASEFEE/state consumer differs")
			}
		}
		var history struct {
			BaseFeePerBlobGas []string
			BlobGasUsedRatio  []float64
		}
		proposer.call(t, "eth_feeHistory", &history, "0x1", "latest", []float64{})
		refBlockBytes, _ := producer.Blockchain().CurrentBlock().Header().EncodeRLP()
		var refBlock gethtypes.Header
		if err := gethrlp.DecodeBytes(refBlockBytes, &refBlock); err != nil {
			t.Fatal(err)
		}
		nextExcess := geth4844.CalcExcessBlobGas(&reference, &refBlock, step.time)
		nextPrice := geth4844.CalcBlobFee(&reference, &gethtypes.Header{Time: step.time, ExcessBlobGas: &nextExcess})
		if len(history.BaseFeePerBlobGas) != 2 || history.BaseFeePerBlobGas[0] != fmt.Sprintf("0x%x", price) || history.BaseFeePerBlobGas[1] != fmt.Sprintf("0x%x", nextPrice) || len(history.BlobGasUsedRatio) != 1 || history.BlobGasUsedRatio[0] != 1 {
			t.Fatalf("fork-aware feeHistory: %+v", history)
		}
		// Change only the announced blob gas and recompute its hash independently.
		bad := *block
		bad.BlobGasUsed += gas.GasPerBlob
		refBlock.BlobGasUsed = &bad.BlobGasUsed
		bad.BlockHash = types.Hash(refBlock.Hash())
		var rejected payload.PayloadStatusV1
		verifier.call(t, "engine_newPayloadV4", &rejected, &bad, hashes, beacon, built.ExecutionRequests)
		if rejected.Status != "INVALID" || rejected.ValidationError == nil || !strings.Contains(*rejected.ValidationError, "exceeds maximum") || receiver.Blockchain().CurrentBlock().Hash() != block.BlockHash {
			t.Fatalf("over-capacity header did not fail intended guard: %+v", rejected)
		}
	}
	head, root := receiver.Blockchain().CurrentBlock().Hash(), receiver.Blockchain().CurrentBlock().Root()
	if err := receiver.Stop(); err != nil {
		t.Fatal(err)
	}
	restarted := newTestNode(t, receiverCfg)
	if restarted.Blockchain().CurrentBlock().Hash() != head || restarted.Blockchain().CurrentBlock().Root() != root || restarted.Blockchain().State().GetBalance(sender).Cmp(producer.Blockchain().State().GetBalance(sender)) != 0 {
		t.Fatal("configured signed blob history did not survive fresh-node replay")
	}
	authority, err := restarted.db.Get(persistedChainConfigKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Stop(); err != nil {
		t.Fatal(err)
	}
	// Keep the legacy no-op alias consistent so it cannot mask the actual
	// historical parameter guard under test.
	changed := *prague
	changed.UpdateFraction++
	schedule := forks["blobSchedule"].(map[string]any)
	schedule["prague"], schedule["osaka"] = &changed, &changed
	altered, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiverCfg.GenesisPath, altered, 0600); err != nil {
		t.Fatal(err)
	}
	badNode, err := New(receiverCfg)
	if badNode != nil {
		_ = badNode.Stop()
	}
	if err == nil || !strings.Contains(err.Error(), "incompatible historical blobSchedule prague") {
		t.Fatalf("historical pricing changed silently: %v", err)
	}
	if err := os.WriteFile(receiverCfg.GenesisPath, input, 0600); err != nil {
		t.Fatal(err)
	}
	restored := newTestNode(t, receiverCfg)
	afterAuthority, err := restored.db.Get(persistedChainConfigKey)
	if err != nil || string(afterAuthority) != string(authority) || restored.Blockchain().CurrentBlock().Hash() != head {
		t.Fatal("rejected pricing change overwrote authority/history")
	}
}
