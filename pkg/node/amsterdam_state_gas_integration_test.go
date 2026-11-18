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

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestAmsterdamStateGasThroughPoolHTTPBlockImportAndRestart(t *testing.T) {
	const token = "amsterdam-state-gas-http"
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	miner := types.Address{18: 0x88, 19: 1}
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	genesis["gasLimit"] = "0x8f0d180"
	forks := genesis["config"].(map[string]any)
	forks["osakaTime"], forks["amsterdamTime"], forks["terminalTotalDifficulty"] = 0, 0, 0
	initial := new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil)
	genesis["alloc"].(map[string]any)[sender.Hex()] = map[string]string{"balance": initial.String()}
	genesis["alloc"].(map[string]any)[miner.Hex()] = map[string]string{"balance": "1", "nonce": "0x1"}
	input, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	makeCfg := func() *Config {
		cfg := makeTestConfig(t)
		cfg.EngineAuthToken = token
		cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
		if err := os.WriteFile(cfg.GenesisPath, input, 0600); err != nil {
			t.Fatal(err)
		}
		return &cfg
	}
	producerCfg, receiverCfg := makeCfg(), makeCfg()
	producer, receiver := newTestNode(t, producerCfg), newTestNode(t, receiverCfg)
	writer, reader := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	feeCap, tip := big.NewInt(2_000_000_000), big.NewInt(1_000_000_000)
	submit := func(nonce, limit uint64, to *gethcommon.Address, data []byte) types.Hash {
		t.Helper()
		signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: nonce, Gas: limit, GasFeeCap: feeCap, GasTipCap: tip, To: to, Value: new(big.Int), Data: data}), gethtypes.NewCancunSigner(big.NewInt(1337)), key)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := signed.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var hash types.Hash
		writer.call(t, "eth_sendRawTransaction", &hash, "0x"+hex.EncodeToString(raw))
		if hash != types.Hash(signed.Hash()) {
			t.Fatal("signed transaction hash changed")
		}
		return hash
	}
	createHash := submit(0, 110_000_000, nil, []byte{0x62, 0x01, 0x00, 0x00, 0x5f, 0xf3})
	self := gethcommon.Address(sender)
	selfHash := submit(1, 100000, &self, nil)
	beacon := types.Hash{31: 1}
	attrs := map[string]any{"timestamp": "0xc", "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": miner, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0x8f0d180"}
	parent := producer.blockchain.Genesis().Hash()
	var fc payload.ForkchoiceUpdatedResult
	writer.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: parent}, attrs)
	if fc.PayloadID == nil || fc.PayloadStatus.Status != "VALID" {
		t.Fatalf("build request: %+v", fc)
	}
	var built struct {
		ExecutionPayload  payload.ExecutionPayloadV5
		ExecutionRequests []string
	}
	writer.call(t, "engine_getPayloadV6", &built, fc.PayloadID)
	p := built.ExecutionPayload
	if len(p.Transactions) != 2 {
		t.Fatalf("pool/builder excluded valid high-total state transaction: %d", len(p.Transactions))
	}
	wantState := uint64((120 + 65536) * 1530)
	if p.GasUsed != wantState {
		t.Fatalf("header gas=%d want bottleneck state=%d", p.GasUsed, wantState)
	}
	for _, rpc := range []track4RPC{writer, reader} {
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV5", &status, p, []types.Hash{}, beacon, built.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatalf("independent receiver import: %+v", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV4", &fc, payload.ForkchoiceStateV1{HeadBlockHash: p.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" {
			t.Fatalf("select state-gas block: %+v", fc)
		}
	}
	type receipt struct {
		GasUsed           string
		CumulativeGasUsed string
		EffectiveGasPrice string
		Status            string
	}
	var first, last receipt
	reader.call(t, "eth_getTransactionReceipt", &first, createHash)
	reader.call(t, "eth_getTransactionReceipt", &last, selfHash)
	gas0, err := hexutil.DecodeUint64(first.GasUsed)
	if err != nil {
		t.Fatal(err)
	}
	gas1, err := hexutil.DecodeUint64(last.GasUsed)
	if err != nil {
		t.Fatal(err)
	}
	cumulative, err := hexutil.DecodeUint64(last.CumulativeGasUsed)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "0x1" || last.Status != "0x1" || gas1 != 12000 || cumulative != gas0+gas1 || cumulative <= p.GasUsed {
		t.Fatalf("receipt payment was confused with block dimensions: first=%+v last=%+v header=%d", first, last, p.GasUsed)
	}
	header := receiver.blockchain.CurrentBlock().Header()
	encoded, err := header.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	var independent gethtypes.Header
	if err := gethrlp.DecodeBytes(encoded, &independent); err != nil {
		t.Fatal(err)
	}
	if types.Hash(independent.Hash()) != p.BlockHash {
		t.Fatal("SDK disagrees on complete header commitment")
	}
	// Rehash a header with the wrong scalar receipt sum. The hash is internally
	// consistent, so rejection must come from actual block gas accounting.
	altered := p
	altered.GasUsed = cumulative
	independent.GasUsed = cumulative
	altered.BlockHash = types.Hash(independent.Hash())
	var invalid payload.PayloadStatusV1
	reader.call(t, "engine_newPayloadV5", &invalid, altered, []types.Hash{}, beacon, built.ExecutionRequests)
	if invalid.Status != "INVALID" || invalid.ValidationError == nil || !strings.Contains(strings.ToLower(*invalid.ValidationError), "gas") {
		t.Fatalf("scalar receipt sum header not rejected by gas validator: %+v", invalid)
	}
	created := types.Address(gethcrypto.CreateAddress(gethcommon.Address(sender), 0))
	assertState := func(node *Node) {
		t.Helper()
		if node.blockchain.CurrentBlock().Hash() != p.BlockHash || node.blockchain.CurrentBlock().Root() != p.StateRoot || len(node.blockchain.State().GetCode(created)) != 65536 {
			t.Fatal("state-heavy deployment/head/root changed")
		}
		price := new(big.Int).Add(header.BaseFee, tip)
		if price.Cmp(feeCap) > 0 {
			price.Set(feeCap)
		}
		wantBalance := new(big.Int).Sub(initial, new(big.Int).Mul(new(big.Int).SetUint64(cumulative), price))
		if node.blockchain.State().GetBalance(sender).Cmp(wantBalance) != 0 || node.blockchain.State().GetNonce(sender) != 2 {
			t.Fatal("sender payment/nonce mismatch")
		}
	}
	assertState(receiver)
	if err := receiver.Stop(); err != nil {
		t.Fatal(err)
	}
	reopened := newTestNode(t, receiverCfg)
	assertState(reopened)
	reread := track4NodeRPC(t, reopened, token)
	var after receipt
	reread.call(t, "eth_getTransactionReceipt", &after, selfHash)
	if after.CumulativeGasUsed != last.CumulativeGasUsed || after.GasUsed != last.GasUsed {
		t.Fatal("receipt payment changed after restart")
	}
	t.Logf("signed creation=%s self=%s codeBytes=65536 state=%d header=%d userCumulative=%d", createHash, selfHash, wantState, p.GasUsed, cumulative)
}
