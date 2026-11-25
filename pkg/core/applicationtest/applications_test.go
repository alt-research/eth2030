package applicationtest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	bridge "github.com/eth2030/eth2030/geth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	common "github.com/ethereum/go-ethereum/common"
	core "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

//go:embed testdata/*
var fixtures embed.FS

type artifact struct {
	ABI  abi.ABI
	Code []byte
}

func loadArtifact(t *testing.T, name string) artifact {
	t.Helper()
	raw, err := fixtures.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var a struct {
		ABI      json.RawMessage `json:"abi"`
		Bytecode string          `json:"bytecode"`
	}
	if err = json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(bytes.NewReader(a.ABI))
	if err != nil {
		t.Fatal(err)
	}
	code, err := hex.DecodeString(strings.TrimPrefix(a.Bytecode, "0x"))
	if err != nil || len(code) == 0 {
		t.Fatalf("artifact %s: %v", name, err)
	}
	return artifact{parsed, code}
}
func pack(t *testing.T, a artifact, method string, args ...any) []byte {
	t.Helper()
	data, err := a.ABI.Pack(method, args...)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPublishedApplicationArtifactIntegrity(t *testing.T) {
	raw, err := fixtures.ReadFile("testdata/sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var sources []struct {
		Artifact string `json:"artifact"`
		SHA      string `json:"artifact_sha256"`
	}
	if err = json.Unmarshal(raw, &sources); err != nil {
		t.Fatal(err)
	}
	if len(sources) != 6 {
		t.Fatal("incomplete application inventory")
	}
	for _, source := range sources {
		raw, err := fixtures.ReadFile("testdata/" + source.Artifact)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(raw)) != source.SHA {
			t.Fatalf("changed published artifact %s", source.Artifact)
		}
	}
}

type sequence struct {
	t         *testing.T
	amsterdam bool
	native    state.StateDB
	historyDB rawdb.Database
	reference *bridge.StateTestState
	cfg       *config.ChainConfig
	oracle    *params.ChainConfig
	key       *ecdsa.PrivateKey
	owner     common.Address
	miner     types.Address
	nonce     uint64
	step      uint64
	outerGas  uint64
}

func newSequence(t *testing.T, amsterdam bool) *sequence {
	t.Helper()
	key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	owner := crypto.PubkeyToAddress(key.PublicKey)
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.ChainID = big.NewInt(1337)
	cfg.OsakaTime = &zero
	cfg.AmsterdamTime = nil
	cfg.BPO1Time = nil
	cfg.BPO2Time = nil
	cfg.GlamsterdanTime = nil
	cfg.HogotaTime = nil
	cfg.IPlusTime = nil
	oracle := bridge.AmsterdamFixtureConfig()
	oracle.ChainID = big.NewInt(1337)
	if amsterdam {
		cfg.AmsterdamTime = &zero
		cfg.BPO1Time = &zero
		cfg.BPO2Time = &zero
	} else {
		oracle.AmsterdamTime = nil
		oracle.BPO1Time = nil
		oracle.BPO2Time = nil
	}
	miner := types.Address{19: 0x77}
	fund := new(big.Int).Exp(big.NewInt(10), big.NewInt(25), nil)
	authKey, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 2))
	if err != nil {
		t.Fatal(err)
	}
	pre := map[string]bridge.PreAccount{owner.Hex(): {Balance: fund}, miner.Hex(): {Balance: big.NewInt(1), Nonce: 1}, crypto.PubkeyToAddress(authKey.PublicKey).Hex(): {Balance: big.NewInt(1000000000000000000)}}
	db := state.NewMemoryStateDB()
	for address, account := range pre {
		a := types.HexToAddress(address)
		db.CreateAccount(a)
		db.AddBalance(a, account.Balance)
		db.SetNonce(a, account.Nonce)
	}
	if _, err := db.Commit(); err != nil {
		t.Fatal(err)
	}
	ref, err := bridge.MakePreState(pre)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ref.Close)
	return &sequence{t: t, amsterdam: amsterdam, native: db, reference: ref, cfg: &cfg, oracle: oracle, key: key, owner: owner, miner: miner}
}
func (s *sequence) header() *types.Header {
	slot := s.step + 1
	return &types.Header{Number: new(big.Int).SetUint64(s.step + 1), Time: 12 * (s.step + 1), SlotNumber: &slot, GasLimit: 150000000, BaseFee: big.NewInt(1), Coinbase: s.miner, Difficulty: new(big.Int)}
}
func (s *sequence) context(h *types.Header) gethvm.BlockContext {
	random := common.Hash{}
	return gethvm.BlockContext{CanTransfer: core.CanTransfer, Transfer: core.Transfer, GetHash: func(uint64) common.Hash { return common.Hash{} }, Coinbase: common.Address(s.miner), GasLimit: h.GasLimit, BlockNumber: h.Number, Time: h.Time, Difficulty: new(big.Int), BaseFee: h.BaseFee, Random: &random, SlotNum: *h.SlotNumber, BlobBaseFee: big.NewInt(1), CostPerStateByte: 1530}
}
func (s *sequence) limit() uint64 {
	if s.outerGas != 0 {
		return s.outerGas
	}
	if s.amsterdam {
		return 100000000
	}
	return 8000000
}
func (s *sequence) send(name string, to *common.Address, data []byte, value *big.Int, status uint64) *types.Receipt {
	s.t.Helper()
	if value == nil {
		value = new(big.Int)
	}
	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: s.nonce, Gas: s.limit(), GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1), To: to, Value: value, Data: data})
	return s.apply(name, tx, status)
}
func (s *sequence) apply(name string, tx *gethtypes.Transaction, status uint64) *types.Receipt {
	s.t.Helper()
	t := s.t
	signed, err := gethtypes.SignTx(tx, gethtypes.LatestSignerForChainID(big.NewInt(1337)), s.key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	nativeTx, err := types.DecodeBlockTransaction(raw)
	if err != nil {
		t.Fatal(err)
	}
	h := s.header()
	pool := new(gaspool.GasPool).AddGas(h.GasLimit)
	before := new(big.Int).Set(s.native.GetBalance(types.Address(s.owner)))
	s.native.SetTxContext(nativeTx.Hash(), 0)
	receipt, _, err := execution.ApplyTransaction(s.cfg, s.native, h, nativeTx, pool)
	if err != nil {
		t.Fatalf("%s native transition: %v", name, err)
	}
	s.reference.StateDB.SetTxContext(signed.Hash(), 0, uint32(h.Number.Uint64()))
	evm := gethvm.NewEVM(s.context(h), s.reference.StateDB, s.oracle, gethvm.Config{})
	defer evm.Release()
	msg, err := core.TransactionToMessage(signed, gethtypes.MakeSigner(s.oracle, h.Number, h.Time), h.BaseFee)
	if err != nil {
		t.Fatal(err)
	}
	refPool := core.NewGasPool(h.GasLimit)
	result, err := core.ApplyMessage(evm, msg, refPool)
	if err != nil {
		t.Fatalf("%s independent transition: %v", name, err)
	}
	wantStatus := uint64(1)
	if result.Failed() {
		wantStatus = 0
	}
	if receipt.Status != wantStatus || receipt.Status != status || receipt.GasUsed != result.UsedGas {
		t.Fatalf("%s status/gas native=%d/%d independent=%d/%d want status=%d err=%v", name, receipt.Status, receipt.GasUsed, wantStatus, result.UsedGas, status, result.Err)
	}
	refLogs := s.reference.StateDB.GetLogs(signed.Hash(), h.Number.Uint64(), common.Hash{}, h.Time)
	if len(receipt.Logs) != len(refLogs) {
		t.Fatalf("%s log count %d/%d", name, len(receipt.Logs), len(refLogs))
	}
	for i, log := range receipt.Logs {
		want := refLogs[i]
		if common.Address(log.Address) != want.Address || !bytes.Equal(log.Data, want.Data) || len(log.Topics) != len(want.Topics) {
			t.Fatalf("%s log %d differs", name, i)
		}
		for j, topic := range log.Topics {
			if common.Hash(topic) != want.Topics[j] {
				t.Fatalf("%s topic differs", name)
			}
		}
	}
	rules := s.oracle.Rules(h.Number, true, h.Time)
	s.reference.StateDB.Finalise(rules)
	root, err := s.native.Commit()
	if err != nil {
		t.Fatal(err)
	}

	if s.historyDB != nil {
		batch := s.historyDB.NewBatch()
		if err := state.StoreTrieHistory(s.historyDB, batch, root, s.native); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		lazy, err := state.OpenTrieHistory(context.Background(), s.historyDB, root)
		if err != nil {
			t.Fatal(err)
		}
		s.native = lazy
	}
	wantRoot := s.reference.StateDB.IntermediateRoot(rules)
	if root != types.Hash(wantRoot) {
		t.Fatalf("%s complete post-state root native=%s independent=%s", name, root, wantRoot)
	}
	execGas, stateGas := h.GasLimit-pool.Gas(), h.GasLimit-pool.StateGas()
	if s.amsterdam && (execGas != refPool.CumulativeExecution() || stateGas != refPool.CumulativeState()) {
		t.Fatalf("%s block dimensions differ execution %d/%d state %d/%d", name, execGas, refPool.CumulativeExecution(), stateGas, refPool.CumulativeState())
	}
	// These transactions never transfer ETH back to the sender except wallet refund cases;
	// complete-root equality independently checks every sender/recipient/miner fee side effect.
	record := map[string]any{"name": name, "transaction": "0x" + hex.EncodeToString(raw), "block": h.Number, "timestamp": h.Time, "gas_limit": signed.Gas(), "status": receipt.Status, "gas_paid": receipt.GasUsed, "execution_gas": execGas, "state_gas": stateGas, "native_root": root, "independent_root": wantRoot, "sender_balance_before": before.String(), "sender_balance_after": s.native.GetBalance(types.Address(s.owner)).String(), "logs": receipt.Logs}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(encoded))
	s.nonce++
	s.step++
	return receipt
}
func (s *sequence) deploy(name string, a artifact, args ...any) common.Address {
	s.t.Helper()
	address := crypto.CreateAddress(s.owner, s.nonce)
	data := append(bytes.Clone(a.Code), pack(s.t, a, "", args...)...)
	s.send(name, nil, data, nil, 1)
	if len(s.native.GetCode(types.Address(address))) == 0 {
		s.t.Fatalf("%s deployed no code", name)
	}
	return address
}
func (s *sequence) query(a artifact, to common.Address, method string, args ...any) []any {
	s.t.Helper()
	t := s.t
	data := pack(t, a, method, args...)
	h := s.header()
	snapshot := s.native.Snapshot()
	refSnapshot := s.reference.StateDB.Snapshot()
	defer s.native.RevertToSnapshot(snapshot)
	defer s.reference.StateDB.RevertToSnapshot(refSnapshot)
	s.native.ClearAccessList()
	s.native.ClearTransientStorage()
	native := vm.NewEVMWithState(vm.BlockContext{BlockNumber: h.Number, Time: h.Time, GasLimit: h.GasLimit, BaseFee: h.BaseFee, Coinbase: h.Coinbase, BlobBaseFee: big.NewInt(1)}, vm.TxContext{Origin: types.Address(s.owner), GasPrice: big.NewInt(2)}, vm.Config{}, s.native)
	if _, err := execution.ConfigureBlockEVM(s.cfg, h, native); err != nil {
		t.Fatal(err)
	}
	got, _, err := native.StaticCall(types.Address(s.owner), types.Address(to), data, 3000000)
	if err != nil {
		t.Fatalf("%s native query: %v", method, err)
	}
	ref := gethvm.NewEVM(s.context(h), s.reference.StateDB, s.oracle, gethvm.Config{})
	defer ref.Release()
	want, _, err := ref.StaticCall(s.owner, to, data, gethvm.NewGasBudget(3000000, 0))
	if err != nil {
		t.Fatalf("%s independent query: %v", method, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s actual return bytes differ: %x/%x", method, got, want)
	}
	queryRecord, err := json.Marshal(map[string]any{"kind": "application_query", "method": method, "to": to, "input": "0x" + hex.EncodeToString(data), "native_output": "0x" + hex.EncodeToString(got), "independent_output": "0x" + hex.EncodeToString(want), "post_state_root": s.native.GetRoot()})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(queryRecord))
	decoded, err := a.ABI.Unpack(method, got)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}
func (s *sequence) balance(a artifact, token, owner common.Address) *big.Int {
	s.t.Helper()
	return s.query(a, token, "balanceOf", owner)[0].(*big.Int)
}

func TestPublishedWalletAndDEXSignedSequences(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		name := "Osaka"
		if amsterdam {
			name = "Amsterdam"
		}
		t.Run(name, func(t *testing.T) { runApplications(t, amsterdam) })
	}
}
func TestPublishedWalletAndDEXLazyHistorySequences(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		name := "Osaka"
		if amsterdam {
			name = "Amsterdam"
		}
		t.Run(name, func(t *testing.T) { runApplications(t, amsterdam, true) })
	}
}
func runApplications(t *testing.T, amsterdam bool, lazyHistory ...bool) {
	s := newSequence(t, amsterdam)
	if len(lazyHistory) > 0 && lazyHistory[0] {
		s.historyDB = rawdb.NewMemoryDB()
		root := s.native.GetRoot()
		batch := s.historyDB.NewBatch()
		if err := state.StoreTrieHistory(s.historyDB, batch, root, s.native); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		lazy, err := state.OpenTrieHistory(context.Background(), s.historyDB, root)
		if err != nil {
			t.Fatal(err)
		}
		s.native = lazy
	}
	token := loadArtifact(t, "uniswap-v2-erc20")
	factoryCode := loadArtifact(t, "uniswap-v2-factory")
	pairCode := loadArtifact(t, "uniswap-v2-pair")
	supply := new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)
	a := s.deploy("deploy token A", token, supply)
	b := s.deploy("deploy token B", token, supply)
	factory := s.deploy("deploy canonical factory", factoryCode, s.owner)
	s.send("factory CREATE2 pair", &factory, pack(t, factoryCode, "createPair", a, b), nil, 1)
	pair := s.query(factoryCode, factory, "getPair", a, b)[0].(common.Address)
	first, second := a, b
	if bytes.Compare(a[:], b[:]) > 0 {
		first, second = b, a
	}
	salt := crypto.Keccak256Hash(first[:], second[:])
	expectedPair := crypto.CreateAddress2(factory, [32]byte(salt), crypto.Keccak256(pairCode.Code))
	if pair != expectedPair {
		t.Fatalf("factory address %s differs from published CREATE2 commitment %s", pair, expectedPair)
	}
	reserve := new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil)
	s.send("seed pair A", &a, pack(t, token, "transfer", pair, reserve), nil, 1)
	s.send("seed pair B", &b, pack(t, token, "transfer", pair, reserve), nil, 1)
	s.send("mint actual liquidity", &pair, pack(t, pairCode, "mint", s.owner), nil, 1)
	reserves := s.query(pairCode, pair, "getReserves")
	if reserves[0].(*big.Int).Cmp(reserve) != 0 || reserves[1].(*big.Int).Cmp(reserve) != 0 {
		t.Fatal("mint did not establish reserves")
	}
	input := big.NewInt(1000000000000000000)
	s.send("swap input", &a, pack(t, token, "transfer", pair, input), nil, 1)
	numerator := new(big.Int).Mul(input, big.NewInt(997))
	output := new(big.Int).Div(new(big.Int).Mul(numerator, reserve), new(big.Int).Add(new(big.Int).Mul(reserve, big.NewInt(1000)), numerator))
	amount0, amount1 := new(big.Int), new(big.Int)
	if b == first {
		amount0 = output
	} else {
		amount1 = output
	}
	prior := s.balance(token, b, s.owner)
	s.send("actual pair swap", &pair, pack(t, pairCode, "swap", amount0, amount1, s.owner, []byte{}), nil, 1)
	if new(big.Int).Sub(s.balance(token, b, s.owner), prior).Cmp(output) != 0 {
		t.Fatal("swap output not paid")
	}
	lp := s.balance(pairCode, pair, s.owner)
	half := new(big.Int).Div(lp, big.NewInt(2))
	s.send("return LP shares", &pair, pack(t, pairCode, "transfer", pair, half), nil, 1)
	s.send("burn actual liquidity", &pair, pack(t, pairCode, "burn", s.owner), nil, 1)
	s.send("duplicate pair reverts", &factory, pack(t, factoryCode, "createPair", a, b), nil, 0)
	s.send("insufficient token transfer reverts", &a, pack(t, token, "transfer", s.owner, new(big.Int).Mul(supply, big.NewInt(2))), nil, 0)
	runWallet(t, s, token, a, b)
}

func runWallet(t *testing.T, s *sequence, token artifact, a, b common.Address) {
	safe := loadArtifact(t, "safe")
	proxy := loadArtifact(t, "safe-proxy")
	multisend := loadArtifact(t, "safe-multisend")
	singleton := s.deploy("deploy canonical Safe", safe)
	wallet := s.deploy("deploy actual Safe proxy", proxy, singleton)
	multi := s.deploy("deploy actual MultiSend", multisend)
	zero := common.Address{}
	setup := pack(t, safe, "setup", []common.Address{s.owner}, big.NewInt(1), zero, []byte{}, zero, zero, new(big.Int), zero)
	s.send("setup one-owner wallet", &wallet, setup, nil, 1)
	if !reflect.DeepEqual(s.query(safe, wallet, "getOwners")[0], []common.Address{s.owner}) || s.query(safe, wallet, "getThreshold")[0].(*big.Int).Cmp(big.NewInt(1)) != 0 {
		t.Fatal("Safe setup ownership differs")
	}
	amount := big.NewInt(1000000000000000000)
	for _, target := range []common.Address{a, b} {
		s.send("fund wallet tokens", &target, pack(t, token, "transfer", wallet, new(big.Int).Mul(amount, big.NewInt(10))), nil, 1)
	}
	s.send("fund wallet fee refunds", &wallet, nil, amount, 1)
	signature := make([]byte, 65)
	copy(signature[12:32], s.owner[:])
	signature[64] = 1
	execute := func(name string, wallet, target common.Address, data []byte, operation uint8, forward uint64, wantSuccess bool) *types.Receipt {
		t.Helper()
		calldata := pack(t, safe, "execTransaction", target, new(big.Int), data, operation, new(big.Int).SetUint64(forward), new(big.Int), big.NewInt(1), zero, zero, signature)
		walletBefore := new(big.Int).Set(s.native.GetBalance(types.Address(wallet)))
		senderBefore := new(big.Int).Set(s.native.GetBalance(types.Address(s.owner)))
		receipt := s.send(name, &wallet, calldata, nil, 1)
		event := "ExecutionSuccess"
		if !wantSuccess {
			event = "ExecutionFailure"
		}
		var payment *big.Int
		found := false
		for _, log := range receipt.Logs {
			if common.Address(log.Address) == wallet && len(log.Topics) > 0 && common.Hash(log.Topics[0]) == safe.ABI.Events[event].ID {
				values, err := safe.ABI.Events[event].Inputs.NonIndexed().Unpack(log.Data)
				if err != nil {
					t.Fatal(err)
				}
				if len(values) != 1 {
					t.Fatalf("%s malformed published refund event", name)
				}
				payment = values[0].(*big.Int)
				found = true
			}
		}
		if !found {
			t.Fatalf("%s did not emit %s", name, event)
		}
		walletPaid := new(big.Int).Sub(walletBefore, s.native.GetBalance(types.Address(wallet)))
		if walletPaid.Cmp(payment) != 0 {
			t.Fatalf("%s actual wallet refund %s differs from event %s", name, walletPaid, payment)
		}
		paidFee := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), big.NewInt(2))
		senderDelta := new(big.Int).Sub(s.native.GetBalance(types.Address(s.owner)), senderBefore)
		if senderDelta.Cmp(new(big.Int).Sub(payment, paidFee)) != 0 {
			t.Fatalf("%s sender/refund/fee conservation failed", name)
		}
		return receipt
	}
	recipient := common.Address{19: 0x51}
	execute("wallet bounded token transfer", wallet, a, pack(t, token, "transfer", recipient, amount), 0, 1000000, true)
	if s.balance(token, a, recipient).Cmp(amount) != 0 {
		t.Fatal("wallet transfer not paid")
	}
	fresh := common.Address{19: 0x52}
	// Measure the actual published child, including repriced cold accesses.
	// A 30k forwarded execution budget succeeds even though Amsterdam's state
	// charge alone exceeds that budget; 15k fails on both selected forks.
	bounded := execute("wallet 30k-forwarded-gas fresh storage", wallet, a, pack(t, token, "transfer", fresh, amount), 0, 30000, true)
	if s.amsterdam && bounded.StateGasUsed <= 30000 {
		t.Fatal("application did not exercise state gas beyond the forwarded execution budget")
	}
	if s.balance(token, a, fresh).Cmp(amount) != 0 {
		t.Fatal("bounded child storage outcome differs")
	}
	starved := common.Address{19: 0x55}
	execute("wallet 15k-forwarded-gas failure handler", wallet, a, pack(t, token, "transfer", starved, amount), 0, 15000, false)
	if s.balance(token, a, starved).Sign() != 0 {
		t.Fatal("starved child leaked token side effects")
	}
	subcall := func(target common.Address, data []byte) []byte {
		out := append([]byte{0}, target[:]...)
		out = append(out, make([]byte, 32)...)
		out = append(out, common.LeftPadBytes(new(big.Int).SetUint64(uint64(len(data))).Bytes(), 32)...)
		return append(out, data...)
	}
	failedReceiver := common.Address{19: 0x53}
	batch := append(subcall(a, pack(t, token, "transfer", failedReceiver, amount)), subcall(b, pack(t, token, "transfer", failedReceiver, new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)))...)
	execute("wallet delegate batch failure rolls back first transfer", wallet, multi, pack(t, multisend, "multiSend", batch), 1, 1000000, false)
	if s.balance(token, a, failedReceiver).Sign() != 0 || s.balance(token, b, failedReceiver).Sign() != 0 {
		t.Fatal("failed nested MultiSend leaked token side effects")
	}
	batch = append(subcall(a, pack(t, token, "transfer", failedReceiver, amount)), subcall(b, pack(t, token, "transfer", failedReceiver, amount))...)
	execute("wallet delegate batch succeeds", wallet, multi, pack(t, multisend, "multiSend", batch), 1, 1000000, true)
	if s.balance(token, a, failedReceiver).Cmp(amount) != 0 || s.balance(token, b, failedReceiver).Cmp(amount) != 0 {
		t.Fatal("successful nested batch did not pay both tokens")
	}
	// Two actual token subcalls touch the same originally-zero allowance.
	// Below TX_MAX_GAS_LIMIT there is no initial reservoir: allocation and
	// cross-frame clearing must refill the exact gas pools in LIFO order.
	s.outerGas = 1000000
	spender := common.Address{19: 0x58}
	approvals := append(subcall(a, pack(t, token, "approve", spender, big.NewInt(1))), subcall(a, pack(t, token, "approve", spender, new(big.Int)))...)
	execute("wallet no-reservoir sibling allowance refill", wallet, multi, pack(t, multisend, "multiSend", approvals), 1, 500000, true)
	if s.query(token, a, "allowance", wallet, spender)[0].(*big.Int).Sign() != 0 {
		t.Fatal("sibling refill retained a temporary allowance")
	}
	approvals = append(subcall(a, pack(t, token, "approve", spender, big.NewInt(1))), subcall(b, pack(t, token, "transfer", failedReceiver, new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)))...)
	execute("wallet no-reservoir nested rollback", wallet, multi, pack(t, multisend, "multiSend", approvals), 1, 500000, false)
	if s.query(token, a, "allowance", wallet, spender)[0].(*big.Int).Sign() != 0 {
		t.Fatal("nested revert retained a temporary allowance")
	}
	s.outerGas = 0
	authKey, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 2))
	if err != nil {
		t.Fatal(err)
	}
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	auth, err := gethtypes.SignSetCode(authKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: singleton, Nonce: 0})
	if err != nil {
		t.Fatal(err)
	}
	tx := gethtypes.NewTx(&gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), Nonce: s.nonce, Gas: s.limit(), GasFeeCap: uint256.NewInt(2), GasTipCap: uint256.NewInt(1), To: authority, Value: new(uint256.Int), Data: setup, AuthList: []gethtypes.SetCodeAuthorization{auth}})
	s.apply("authorize actual Safe runtime and initialize delegated EOA", tx, 1)
	s.send("fund delegated EOA tokens", &a, pack(t, token, "transfer", authority, amount), nil, 1)
	receiver := common.Address{19: 0x54}
	execute("delegated EOA wallet executes token transfer", authority, a, pack(t, token, "transfer", receiver, amount), 0, 1000000, true)
	if s.balance(token, a, receiver).Cmp(amount) != 0 {
		t.Fatal("delegated wallet did not pay recipient")
	}
	auth, err = gethtypes.SignSetCode(authKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: zero, Nonce: 1})
	if err != nil {
		t.Fatal(err)
	}
	tx = gethtypes.NewTx(&gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), Nonce: s.nonce, Gas: s.limit(), GasFeeCap: uint256.NewInt(2), GasTipCap: uint256.NewInt(1), To: s.owner, Value: new(uint256.Int), AuthList: []gethtypes.SetCodeAuthorization{auth}})
	s.apply("revoke delegated wallet authority", tx, 1)
	if len(s.native.GetCode(types.Address(authority))) != 0 || s.native.GetNonce(types.Address(authority)) != 2 {
		t.Fatal("delegation revocation did not preserve nonce lifecycle")
	}
}
