package node

import (
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	bridge "github.com/eth2030/eth2030/geth"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

func verifySimulationAgainstSDK(t *testing.T, rpc track4RPC, genesisJSON []byte, sender, writer types.Address) {
	for _, floor := range []bool{false, true} {
		name := "new-slot"
		if floor {
			name = "floor-refund-and-prestate-override"
		}
		t.Run(name, func(t *testing.T) { verifySimulationSDKCase(t, rpc, genesisJSON, sender, writer, floor) })
	}
}
func verifySimulationSDKCase(t *testing.T, rpc track4RPC, genesisJSON []byte, sender, writer types.Address, floor bool) {

	t.Helper()
	var genesis gethcore.Genesis
	if err := json.Unmarshal(genesisJSON, &genesis); err != nil {
		t.Fatalf("frozen genesis decode: %v", err)
	}
	pre := make(map[string]bridge.PreAccount)
	for address, account := range genesis.Alloc {
		storage := make(map[types.Hash]types.Hash)
		for slot, value := range account.Storage {
			storage[types.Hash(slot)] = types.Hash(value)
		}
		pre[address.Hex()] = bridge.PreAccount{Balance: account.Balance, Nonce: account.Nonce, Code: account.Code, Storage: storage}
	}
	if floor {
		account := pre[gethcommon.Address(writer).Hex()]
		if account.Storage == nil {
			account.Storage = make(map[types.Hash]types.Hash)
		}
		account.Storage[types.Hash{}] = types.Hash{31: 1}
		pre[gethcommon.Address(writer).Hex()] = account
	}
	reference, err := bridge.MakePreState(pre)
	if err != nil {
		t.Fatal(err)
	}
	defer reference.Close()
	var wire []json.RawMessage
	withdrawal := map[string]string{"index": "0x1", "validatorIndex": "0x2", "address": sender.Hex(), "amount": "0x9"}
	firstData := []byte{7}
	if floor {
		firstData = make([]byte, 128)
	}
	calls := []any{map[string]any{"from": sender.Hex(), "to": writer.Hex(), "input": hexutil.Encode(firstData), "gas": "0x7a120"}, map[string]any{"from": sender.Hex(), "to": writer.Hex(), "gas": "0x7a120"}}
	blockRequest := map[string]any{"blockOverrides": map[string]any{"withdrawals": []any{withdrawal}}, "calls": calls}
	if floor {
		blockRequest["stateOverrides"] = map[string]any{writer.Hex(): map[string]any{"stateDiff": map[string]string{(types.Hash{}).Hex(): types.Hash{31: 1}.Hex()}}}
	}
	rpc.call(t, "eth_simulateV1", &wire, map[string]any{"blockStateCalls": []any{blockRequest}}, "latest")
	if len(wire) != 1 {
		t.Fatal("missing complete block result")
	}
	var returned gethtypes.Header
	if err := json.Unmarshal(wire[0], &returned); err != nil {
		t.Fatalf("RPC header decode: %v", err)
	}
	var output struct {
		Hash         gethcommon.Hash
		Size         string
		Transactions []gethcommon.Hash
		Calls        []struct {
			GasUsed    string
			MaxUsedGas string
			ReturnData string
			Status     string
		}
	}
	if err := json.Unmarshal(wire[0], &output); err != nil {
		t.Fatal(err)
	}
	config := bridge.AmsterdamFixtureConfig()
	config.ChainID = big.NewInt(1337)
	parent := genesis.ToBlock().Header()
	random := gethcommon.Hash{}
	contextVM := gethvm.BlockContext{CanTransfer: gethcore.CanTransfer, Transfer: gethcore.Transfer, GetHash: func(number uint64) gethcommon.Hash {
		if number == 0 {
			return parent.Hash()
		}
		return gethcommon.Hash{}
	}, BlockNumber: big.NewInt(1), Time: 12, GasLimit: 30_000_000, BaseFee: new(big.Int), BlobBaseFee: new(big.Int), Difficulty: new(big.Int), Random: &random, CostPerStateByte: 1530}
	evm := gethvm.NewEVM(contextVM, reference.StateDB, config, gethvm.Config{NoBaseFee: true})
	defer evm.Release()
	beacon := gethcommon.Hash{}
	accesses := gethcore.PreExecution(context.Background(), &beacon, parent, config, evm, big.NewInt(1), 12)
	pool := gethcore.NewGasPool(30_000_000)
	var txs gethtypes.Transactions
	var receipts gethtypes.Receipts
	address := gethcommon.Address(writer)
	for i, data := range [][]byte{firstData, nil} {
		tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: uint64(i), Gas: 500000, GasFeeCap: new(big.Int), GasTipCap: new(big.Int), To: &address, Value: new(big.Int), Data: data})
		if len(output.Transactions) != 2 || output.Transactions[i] != tx.Hash() {
			t.Fatalf("unsigned transaction %d commitment changed", i)
		}
		reference.StateDB.SetTxContext(tx.Hash(), i, uint32(i+1))
		msg := (&gethcore.Message{From: gethcommon.Address(sender), To: &address, Nonce: uint64(i), GasLimit: 500000, Value: uint256.NewInt(0), GasPrice: uint256.NewInt(0), GasFeeCap: uint256.NewInt(0), GasTipCap: uint256.NewInt(0), Data: data}).SkipEOACheck().SkipNonceCheck().SkipGasLimitCapCheck().SkipExecutionGasCapCheck()
		result, err := gethcore.ApplyMessage(evm, msg, pool)
		if err != nil {
			t.Fatal(err)
		}
		accesses.Merge(reference.StateDB.Finalise(evm.GetRules()))
		receipt := gethcore.MakeReceipt(evm, result, reference.StateDB, big.NewInt(1), gethcommon.Hash{}, 12, tx, pool.CumulativeUsed(), nil)
		receipts = append(receipts, receipt)
		txs = append(txs, tx)
		gas, e1 := hexutil.DecodeUint64(output.Calls[i].GasUsed)
		peak, e2 := hexutil.DecodeUint64(output.Calls[i].MaxUsedGas)
		if e1 != nil || e2 != nil || gas != result.UsedGas || peak != result.MaxUsedGas || output.Calls[i].ReturnData != hexutil.Encode(result.Return()) || output.Calls[i].Status != "0x1" {
			t.Fatalf("actual call %d native=%+v SDK=%+v", i, output.Calls[i], result)
		}
	}
	withdrawals := gethtypes.Withdrawals{&gethtypes.Withdrawal{Index: 1, Validator: 2, Address: gethcommon.Address(sender), Amount: 9}}
	requests, post, err := gethcore.PostExecution(context.Background(), config, big.NewInt(1), 12, nil, withdrawals, evm, 3)
	if err != nil {
		t.Fatal(err)
	}
	accesses.Merge(post)
	balEncoding := accesses.ToEncodingObj()
	balHash := balEncoding.Hash()
	reqHash := gethtypes.CalcRequestsHash(requests)
	expected := &gethtypes.Header{ParentHash: parent.Hash(), UncleHash: gethtypes.EmptyUncleHash, Number: big.NewInt(1), Time: 12, Difficulty: new(big.Int), GasLimit: 30_000_000, GasUsed: pool.Used(), BaseFee: new(big.Int), Root: reference.StateDB.IntermediateRoot(evm.GetRules()), ParentBeaconRoot: &beacon, ExcessBlobGas: new(uint64), BlobGasUsed: new(uint64), RequestsHash: &reqHash, BlockAccessListHash: &balHash}
	if parent.SlotNumber != nil {
		slot := *parent.SlotNumber + 1
		expected.SlotNumber = &slot
	}
	block := gethtypes.NewBlock(expected, &gethtypes.Body{Transactions: txs, Withdrawals: withdrawals}, receipts, gethtrie.NewStackTrie(nil))
	expected = block.Header()
	if returned.Root != expected.Root || returned.ReceiptHash != expected.ReceiptHash || returned.TxHash != expected.TxHash || returned.WithdrawalsHash == nil || expected.WithdrawalsHash == nil || *returned.WithdrawalsHash != *expected.WithdrawalsHash || returned.BlockAccessListHash == nil || *returned.BlockAccessListHash != balHash || returned.RequestsHash == nil || *returned.RequestsHash != reqHash || returned.GasUsed != expected.GasUsed {
		t.Fatalf("native commitments state=%s receipt=%s tx=%s BAL=%v gas=%d; SDK state=%s receipt=%s tx=%s BAL=%s gas=%d", returned.Root, returned.ReceiptHash, returned.TxHash, returned.BlockAccessListHash, returned.GasUsed, expected.Root, expected.ReceiptHash, expected.TxHash, balHash, expected.GasUsed)
	}
	if output.Hash != expected.Hash() || returned.Hash() != expected.Hash() {
		for i := 0; i < reflect.TypeOf(returned).NumField(); i++ {
			native := reflect.ValueOf(returned).Field(i).Interface()
			reference := reflect.ValueOf(*expected).Field(i).Interface()
			if !reflect.DeepEqual(native, reference) {
				t.Logf("header field %s native=%v SDK=%v", reflect.TypeOf(returned).Field(i).Name, native, reference)
			}
		}
		t.Fatalf("complete header hash native=%s decoded=%s SDK=%s", output.Hash, returned.Hash(), expected.Hash())
	}
	encoded, err := gethrlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	size, err := hexutil.DecodeUint64(output.Size)
	if err != nil || size != uint64(len(encoded)) {
		t.Fatalf("complete RLP size native=%s SDK=%d", output.Size, len(encoded))
	}
	t.Logf("independent block hash=%s root=%s receipt=%s BAL=%s gas=%d", expected.Hash(), expected.Root, expected.ReceiptHash, balHash, expected.GasUsed)
}
