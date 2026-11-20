package node

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	bridge "github.com/eth2030/eth2030/geth"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// Exercise the signed transaction through the public pool and Engine HTTP
// consumers. The independent SDK uses its own unchanged authorization executor.
func TestOsakaInvalidAuthorizationWarmsThroughEngine(t *testing.T) {
	const token = "osaka-authorization-regression"
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	authorityKey, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
	if err != nil {
		t.Fatal(err)
	}
	sender := gethcrypto.PubkeyToAddress(key.PublicKey)
	authority := gethcrypto.PubkeyToAddress(authorityKey.PublicKey)
	target := gethcommon.Address{19: 0x42}
	miner := types.Address{19: 0x43}
	code := append([]byte{0x73}, authority.Bytes()...)
	code = append(code, 0x31, 0x50, 0x00) // PUSH20 authority; BALANCE; POP; STOP.
	var genesis map[string]any
	if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
		t.Fatal(err)
	}
	delete(genesis["config"].(map[string]any), "amsterdamTime")
	genesis["config"].(map[string]any)["terminalTotalDifficulty"] = 0
	alloc := genesis["alloc"].(map[string]any)
	for address, runtime := range map[gethcommon.Address][]byte{params.WithdrawalQueueAddress: params.WithdrawalQueueCode, params.ConsolidationQueueAddress: params.ConsolidationQueueCode, params.BeaconRootsAddress: params.BeaconRootsCode, params.HistoryStorageAddress: params.HistoryStorageCode} {
		alloc[address.Hex()] = map[string]string{"balance": "0", "nonce": "0x1", "code": hexutil.Encode(runtime)}
	}
	alloc[sender.Hex()] = map[string]string{"balance": "100000000000000000000000"}
	alloc[authority.Hex()] = map[string]string{"balance": "5"}
	alloc[target.Hex()] = map[string]string{"balance": "5", "nonce": "0x1", "code": hexutil.Encode(code)}
	alloc[miner.Hex()] = map[string]string{"balance": "1", "nonce": "0x1"}
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
	producer, receiver := newTestNode(t, makeCfg()), newTestNode(t, makeCfg())
	writer, reader := track4NodeRPC(t, producer, token), track4NodeRPC(t, receiver, token)
	auth, err := gethtypes.SignSetCode(authorityKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: target, Nonce: 1})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), Gas: 500000, GasFeeCap: uint256.NewInt(2000000000), GasTipCap: uint256.NewInt(1000000000), To: target, Value: new(uint256.Int), AuthList: []gethtypes.SetCodeAuthorization{auth}}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var hash types.Hash
	writer.call(t, "eth_sendRawTransaction", &hash, "0x"+hex.EncodeToString(raw))
	if hash != types.Hash(tx.Hash()) {
		t.Fatal("signed hash changed")
	}
	beacon := types.Hash{31: 1}
	var fc payload.ForkchoiceUpdatedResult
	attrs := map[string]any{"timestamp": "0xc", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": miner, "parentBeaconBlockRoot": beacon}
	writer.call(t, "engine_forkchoiceUpdatedV3", &fc, payload.ForkchoiceStateV1{HeadBlockHash: producer.blockchain.Genesis().Hash()}, attrs)
	if fc.PayloadID == nil {
		t.Fatal("no build ID")
	}
	var built struct {
		ExecutionPayload  *payload.ExecutionPayloadV3
		ExecutionRequests []string
	}
	writer.call(t, "engine_getPayloadV5", &built, fc.PayloadID)
	p := built.ExecutionPayload
	if p == nil || len(p.Transactions) != 1 {
		t.Fatal("signed authorization excluded from block")
	}
	for _, rpc := range []track4RPC{writer, reader} {
		var status payload.PayloadStatusV1
		rpc.call(t, "engine_newPayloadV4", &status, p, []types.Hash{}, beacon, built.ExecutionRequests)
		if status.Status != "VALID" {
			t.Fatalf("import: %+v", status)
		}
		rpc.call(t, "engine_forkchoiceUpdatedV3", &fc, payload.ForkchoiceStateV1{HeadBlockHash: p.BlockHash}, nil)
		if fc.PayloadStatus.Status != "VALID" {
			t.Fatalf("select: %+v", fc)
		}
	}
	var receipt struct {
		GasUsed string
		Status  string
	}
	reader.call(t, "eth_getTransactionReceipt", &receipt, hash)
	if p.GasUsed != 46105 || receipt.GasUsed != hexutil.EncodeUint64(46105) || receipt.Status != "0x1" {
		t.Fatalf("warm authority gas: block=%d receipt=%+v", p.GasUsed, receipt)
	}
	if receiver.blockchain.State().GetNonce(types.Address(authority)) != 0 || len(receiver.blockchain.State().GetCode(types.Address(authority))) != 0 {
		t.Fatal("skipped authorization changed nonce or code")
	}

	var sdkGenesis gethcore.Genesis
	if err := json.Unmarshal(input, &sdkGenesis); err != nil {
		t.Fatal(err)
	}
	pre := make(map[string]bridge.PreAccount)
	for address, account := range sdkGenesis.Alloc {
		storage := make(map[types.Hash]types.Hash)
		for slot, value := range account.Storage {
			storage[types.Hash(slot)] = types.Hash(value)
		}
		pre[address.Hex()] = bridge.PreAccount{Balance: account.Balance, Nonce: account.Nonce, Code: account.Code, Storage: storage}
	}
	reference, err := bridge.MakePreState(pre)
	if err != nil {
		t.Fatal(err)
	}
	defer reference.Close()
	config := bridge.ToGethChainConfig(producer.blockchain.Config())
	parent := sdkGenesis.ToBlock().Header()
	random := gethcommon.Hash(p.PrevRandao)
	evm := gethvm.NewEVM(gethvm.BlockContext{CanTransfer: gethcore.CanTransfer, Transfer: gethcore.Transfer, GetHash: func(n uint64) gethcommon.Hash {
		if n == 0 {
			return parent.Hash()
		}
		return gethcommon.Hash{}
	}, Coinbase: gethcommon.Address(miner), BlockNumber: new(big.Int).SetUint64(p.BlockNumber), Time: p.Timestamp, GasLimit: p.GasLimit, BaseFee: p.BaseFeePerGas, BlobBaseFee: big.NewInt(1), Difficulty: new(big.Int), Random: &random}, reference.StateDB, config, gethvm.Config{})
	defer evm.Release()
	sdkBeacon := gethcommon.Hash(beacon)
	gethcore.PreExecution(context.Background(), &sdkBeacon, parent, config, evm, big.NewInt(1), p.Timestamp)
	msg, err := gethcore.TransactionToMessage(tx, gethtypes.MakeSigner(config, big.NewInt(1), p.Timestamp), p.BaseFeePerGas)
	if err != nil {
		t.Fatal(err)
	}
	reference.StateDB.SetTxContext(tx.Hash(), 0, 1)
	result, err := gethcore.ApplyMessage(evm, msg, gethcore.NewGasPool(p.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	reference.StateDB.Finalise(evm.GetRules())
	if _, _, err := gethcore.PostExecution(context.Background(), config, big.NewInt(1), p.Timestamp, nil, gethtypes.Withdrawals{}, evm, 2); err != nil {
		t.Fatal(err)
	}
	root := reference.StateDB.IntermediateRoot(evm.GetRules())
	if result.UsedGas != p.GasUsed || types.Hash(root) != p.StateRoot {
		t.Fatalf("independent Osaka result gas=%d root=%s; native gas=%d root=%s", result.UsedGas, root, p.GasUsed, p.StateRoot)
	}
	t.Logf("independent Osaka gas=%d root=%s", result.UsedGas, root)
}
