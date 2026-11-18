// Command glamsterdam-compare records native versus independent Geth execution.
// It reports mismatches as data; it never labels a candidate corpus release-ready.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	bridge "github.com/eth2030/eth2030/geth"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

const blockGas = 150_000_000
const eipsRevision = "3db38a3725105cb67b8804ac3e147967113bc3ca"

var target = types.HexToAddress("0x1000000000000000000000000000000000000001")
var miner = types.HexToAddress("0x2000000000000000000000000000000000000002")
var child = types.HexToAddress("0x3000000000000000000000000000000000000003")
var delegate = types.HexToAddress("0x4000000000000000000000000000000000000004")

type scenario struct {
	Name                  string
	Gas                   uint64
	Value                 int64
	Creation, NewTarget   bool
	ZeroBaseFee           bool
	AuthorizationCase     string
	Code, Data, ChildCode []byte
	DelegateCode          []byte
	InitialSlot           uint64
	AccessList            gethtypes.AccessList
}

type accountView struct {
	Exists  bool              `json:"exists"`
	Balance string            `json:"balance"`
	Nonce   uint64            `json:"nonce"`
	Code    string            `json:"code"`
	Storage map[string]string `json:"storage"`
}
type logView struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}
type observation struct {
	Valid           bool                   `json:"valid"`
	ValidationError string                 `json:"validation_error,omitempty"`
	Status          uint64                 `json:"receipt_status"`
	GasUsed         uint64                 `json:"user_gas_used"`
	BlockGasUsed    uint64                 `json:"block_gas_used"`
	ExecutionGas    uint64                 `json:"block_execution_gas"`
	StateGas        uint64                 `json:"block_state_gas"`
	StateRoot       string                 `json:"post_state_root"`
	Accounts        map[string]accountView `json:"accounts"`
	Logs            []logView              `json:"logs"`
}
type input struct {
	Fork              string                 `json:"fork"`
	BlockNumber       uint64                 `json:"block_number"`
	Timestamp         uint64                 `json:"timestamp"`
	Slot              uint64                 `json:"slot"`
	BlockGasLimit     uint64                 `json:"block_gas_limit"`
	BaseFee           string                 `json:"base_fee"`
	Coinbase          string                 `json:"coinbase"`
	SignedTransaction string                 `json:"signed_transaction"`
	PreState          map[string]accountView `json:"pre_state"`
	NativeConfig      *config.ChainConfig    `json:"native_config"`
	ReferenceConfig   *params.ChainConfig    `json:"reference_config"`
	ReferenceCPSB     uint64                 `json:"reference_cost_per_state_byte"`
}

type comparison struct {
	Name        string      `json:"name"`
	Input       input       `json:"input"`
	Native      observation `json:"native"`
	Reference   observation `json:"reference"`
	Differences []string    `json:"differences"`
}
type report struct {
	Scope          string       `json:"scope"`
	Status         string       `json:"status"`
	NativeRevision string       `json:"native_revision"`
	OracleModule   string       `json:"oracle_module"`
	OracleVersion  string       `json:"oracle_version"`
	EIPsRevision   string       `json:"eips_revision"`
	NativeDirty    bool         `json:"native_dirty"`
	Cases          []comparison `json:"cases"`
	Uncovered      []string     `json:"uncovered"`
}

func scenarios() []scenario {
	slots := []byte{0x60, 1, 0x60, 0, 0x55}
	clear := []byte{0x60, 0, 0x60, 0, 0x55}
	callChild := []byte{0x60, 0, 0x60, 0, 0x60, 0, 0x60, 0, 0x73}
	callChild = append(callChild, child[:]...)
	callChild = append(callChild, 0x5a, 0xf4, 0x50, 0)
	nested := append(bytes.Clone(slots), callChild...)
	callSequence := func(op byte, value bool, maximum bool) []byte {
		count := 4
		if value {
			count = 5
		}
		code := bytes.Repeat([]byte{0x5f}, count)
		code = append(code, 0x73)
		code = append(code, child[:]...)
		if maximum {
			code = append(code, 0x7f)
			word := make([]byte, 32)
			word[0] = 1
			code = append(code, word...)
		} else {
			code = append(code, 0x5a)
		}
		return append(code, op, 0x50, 0)
	}

	return []scenario{
		{Name: "existing_zero_value", Gas: 100000},
		{Name: "authorization_existing", Gas: 500000, AuthorizationCase: "existing"},
		{Name: "authorization_new", Gas: 500000, AuthorizationCase: "new"},
		{Name: "authorization_preexecution_oog", Gas: 50000, AuthorizationCase: "preexecution_oog"},
		{Name: "authorization_frame_revert", Gas: 500000, AuthorizationCase: "revert"},
		{Name: "authorization_bad_nonce_warms", Gas: 500000, AuthorizationCase: "bad_nonce"},
		{Name: "authorization_max_nonce_not_warm", Gas: 500000, AuthorizationCase: "max_nonce"},
		{Name: "authorization_clear_reset_once", Gas: 500000, AuthorizationCase: "clear_reset"},
		{Name: "authorization_existing_indicator_reset", Gas: 500000, AuthorizationCase: "existing_indicator"},
		{Name: "authorization_sender", Gas: 500000, AuthorizationCase: "sender"},

		{Name: "delegated_call", Gas: 500000, Code: callSequence(0xf1, true, false), ChildCode: types.AddressToDelegation(delegate), DelegateCode: append(bytes.Clone(slots), 0)},
		{Name: "delegated_callcode", Gas: 500000, Code: callSequence(0xf2, true, false), ChildCode: types.AddressToDelegation(delegate), DelegateCode: append(bytes.Clone(slots), 0)},
		{Name: "delegated_delegatecall", Gas: 500000, Code: callSequence(0xf4, false, false), ChildCode: types.AddressToDelegation(delegate), DelegateCode: append(bytes.Clone(slots), 0)},
		{Name: "delegated_staticcall", Gas: 100000, Code: callSequence(0xfa, false, false), ChildCode: types.AddressToDelegation(delegate), DelegateCode: []byte{0x5f, 0x54, 0x50, 0}},
		{Name: "wide_gas_delegatecall", Gas: 500000, Code: callSequence(0xf4, false, true), ChildCode: append(bytes.Clone(slots), 0)},

		{Name: "zero_base_fee_dynamic", Gas: 100000, ZeroBaseFee: true},
		{Name: "zero_base_fee_gasprice", Gas: 500000, ZeroBaseFee: true, Code: []byte{0x3a, 0x5f, 0x55, 0}},
		{Name: "existing_value", Gas: 100000, Value: 7},
		{Name: "new_account_value", Gas: 400000, Value: 7, NewTarget: true},
		{Name: "new_account_runtime_oog", Gas: 21000, Value: 7, NewTarget: true},
		{Name: "calldata_floor", Gas: 500000, Data: bytes.Repeat([]byte{1}, 128)},
		{Name: "access_list", Gas: 500000, AccessList: gethtypes.AccessList{{Address: gethcommon.Address(target), StorageKeys: []gethcommon.Hash{{}}}}},
		{Name: "allocate_slot", Gas: 500000, Code: append(bytes.Clone(slots), 0)},
		{Name: "clear_original_slot", Gas: 500000, Code: append(bytes.Clone(clear), 0), InitialSlot: 1},
		{Name: "allocate_then_clear", Gas: 500000, Code: append(append(bytes.Clone(slots), clear...), 0)},
		{Name: "allocate_then_revert", Gas: 500000, Code: append(bytes.Clone(slots), 0x60, 0, 0x60, 0, 0xfd)},
		{Name: "allocate_then_exception", Gas: 500000, Code: append(bytes.Clone(slots), 0xfe)},
		{Name: "nested_refill_success", Gas: 500000, Code: nested, ChildCode: append(bytes.Clone(clear), 0)},
		{Name: "nested_refill_revert", Gas: 500000, Code: nested, ChildCode: append(bytes.Clone(clear), 0x60, 0, 0x60, 0, 0xfd)},
		{Name: "create_empty", Gas: 500000, Creation: true, Data: []byte{0}},
		{Name: "create_maximum_code_7954", Gas: 110_000_000, Creation: true, Data: []byte{0x62, 0x01, 0x00, 0x00, 0x5f, 0xf3}},
		{Name: "create_revert", Gas: 500000, Creation: true, Data: []byte{0x60, 0, 0x60, 0, 0xfd}},
		{Name: "create_selfdestruct", Gas: 500000, Creation: true, Value: 7, Data: append(bytes.Clone(slots), 0x30, 0xff)},
		{Name: "above_execution_cap", Gas: 16777217},
		{Name: "above_total_cap", Gas: 4294967296},
		{Name: "funded_reservoir_allocate", Gas: 16777216 + 200000, Code: append(bytes.Clone(slots), 0)},
		{Name: "partial_reservoir_allocate_clear", Gas: 16777216 + 50000, Code: append(append(bytes.Clone(slots), clear...), 0)},
		{Name: "funded_reservoir_nested_refill", Gas: 16777216 + 200000, Code: nested, ChildCode: append(bytes.Clone(clear), 0)},
		{Name: "funded_reservoir_exception", Gas: 16777216 + 200000, Code: append(bytes.Clone(slots), 0xfe)},
	}
}

func nativeConfig(amsterdam bool) *config.ChainConfig {
	cfg := *config.TestConfig
	cfg.ChainID = big.NewInt(1337)
	zero := uint64(0)
	cfg.OsakaTime = &zero
	cfg.AmsterdamTime = nil
	cfg.GlamsterdanTime = nil
	cfg.HogotaTime = nil
	cfg.IPlusTime = nil
	if amsterdam {
		cfg.AmsterdamTime = &zero
	}
	return &cfg
}
func referenceConfig(amsterdam bool) *params.ChainConfig {
	cfg := bridge.AmsterdamFixtureConfig()
	cfg.ChainID = big.NewInt(1337)
	if !amsterdam {
		cfg.AmsterdamTime = nil
	}
	return cfg
}
func runScenario(sc scenario, amsterdam bool) (comparison, error) {
	key, err := gethcrypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	if err != nil {
		return comparison{}, err
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	pre := map[string]bridge.PreAccount{
		sender.Hex(): {Balance: big.NewInt(1_000_000_000_000)},
		miner.Hex():  {Balance: big.NewInt(5), Nonce: 1},
	}
	if !sc.NewTarget && !sc.Creation {
		pre[target.Hex()] = bridge.PreAccount{Balance: big.NewInt(5), Nonce: 1, Code: bytes.Clone(sc.Code), Storage: map[types.Hash]types.Hash{{}: types.BytesToHash(new(big.Int).SetUint64(sc.InitialSlot).Bytes())}}
	}
	if len(sc.ChildCode) > 0 {
		pre[child.Hex()] = bridge.PreAccount{Balance: big.NewInt(1), Nonce: 1, Code: bytes.Clone(sc.ChildCode)}
	}
	if len(sc.DelegateCode) > 0 {
		pre[delegate.Hex()] = bridge.PreAccount{Balance: big.NewInt(1), Nonce: 1, Code: bytes.Clone(sc.DelegateCode)}
	}
	var authority types.Address
	var authorizations []gethtypes.SetCodeAuthorization
	if sc.AuthorizationCase != "" {
		authorityKey, err := gethcrypto.HexToECDSA(strings.Repeat("0", 63) + "2")
		if err != nil {
			return comparison{}, err
		}
		authority = types.Address(gethcrypto.PubkeyToAddress(authorityKey.PublicKey))
		if sc.AuthorizationCase == "sender" {
			authorityKey = key
			authority = sender
		}
		if sc.AuthorizationCase != "new" && authority != sender {
			pre[authority.Hex()] = bridge.PreAccount{Balance: big.NewInt(5)}
		}
		pre[delegate.Hex()] = bridge.PreAccount{Balance: big.NewInt(1), Nonce: 1, Code: []byte{0}}
		if sc.AuthorizationCase == "revert" {
			account := pre[delegate.Hex()]
			account.Code = []byte{0x5f, 0x5f, 0xfd}
			pre[delegate.Hex()] = account
		}
		if sc.AuthorizationCase == "existing_indicator" {
			account := pre[authority.Hex()]
			account.Code = types.AddressToDelegation(delegate)
			pre[authority.Hex()] = account
		}
		tuples := []struct {
			nonce  uint64
			target types.Address
		}{{0, delegate}}
		switch sc.AuthorizationCase {
		case "sender":
			tuples[0].nonce = 1
		case "bad_nonce":
			tuples[0].nonce = 1
		case "max_nonce":
			tuples[0].nonce = ^uint64(0)
		case "clear_reset":
			tuples = append(tuples, struct {
				nonce  uint64
				target types.Address
			}{1, types.Address{}}, struct {
				nonce  uint64
				target types.Address
			}{2, delegate})
		case "existing_indicator":
			tuples = []struct {
				nonce  uint64
				target types.Address
			}{{0, types.Address{}}, {1, delegate}}
		}
		for _, tuple := range tuples {
			auth, err := gethtypes.SignSetCode(authorityKey, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: gethcommon.Address(tuple.target), Nonce: tuple.nonce})
			if err != nil {
				return comparison{}, err
			}
			authorizations = append(authorizations, auth)
		}
		if sc.AuthorizationCase == "bad_nonce" || sc.AuthorizationCase == "max_nonce" {
			account := pre[target.Hex()]
			account.Code = append([]byte{0x73}, authority[:]...)
			account.Code = append(account.Code, 0x31, 0x50, 0)
			pre[target.Hex()] = account
		}
	}
	var to *gethcommon.Address
	if !sc.Creation {
		addr := gethcommon.Address(target)
		to = &addr
	}
	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), To: to, Gas: sc.Gas, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1), Value: big.NewInt(sc.Value), Data: sc.Data, AccessList: sc.AccessList})
	if sc.AuthorizationCase != "" {
		recipient := gethcommon.Address(authority)
		if sc.AuthorizationCase == "bad_nonce" || sc.AuthorizationCase == "max_nonce" {
			recipient = gethcommon.Address(target)
		}
		tx = gethtypes.NewTx(&gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), To: recipient, Gas: sc.Gas, GasFeeCap: uint256.NewInt(2), GasTipCap: uint256.NewInt(1), Value: uint256.NewInt(uint64(sc.Value)), Data: sc.Data, AccessList: sc.AccessList, AuthList: authorizations})
	}

	tx, err = gethtypes.SignTx(tx, gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
	if err != nil {
		return comparison{}, err
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		return comparison{}, err
	}
	slot := uint64(7)
	header := &types.Header{Number: big.NewInt(1), Time: 12, SlotNumber: &slot, GasLimit: blockGas, BaseFee: big.NewInt(1), Coinbase: miner, Difficulty: new(big.Int)}
	if sc.ZeroBaseFee {
		header.BaseFee.SetUint64(0)
	}
	ndb := state.NewMemoryStateDB()
	for addrHex, account := range pre {
		addr := types.HexToAddress(addrHex)
		ndb.CreateAccount(addr)
		ndb.AddBalance(addr, account.Balance)
		ndb.SetNonce(addr, account.Nonce)
		ndb.SetCode(addr, account.Code)
		for key, value := range account.Storage {
			ndb.SetState(addr, key, value)
		}
	}
	if _, err = ndb.Commit(); err != nil {
		return comparison{}, err
	}
	ntx, err := types.DecodeTxRLP(raw)
	if err != nil {
		return comparison{}, err
	}
	// Decode the same signed consensus bytes in both clients. No cached sender,
	// custom opcode table, custom Geth precompile injection or native fee hooks.
	npool := new(gaspool.GasPool).AddGas(blockGas)
	ndb.SetTxContext(ntx.Hash(), 0)
	nr, _, nerr := execution.ApplyTransaction(nativeConfig(amsterdam), ndb, header, ntx, npool)
	native := observation{Valid: nerr == nil, Logs: []logView{}}
	if nerr != nil {
		native.ValidationError = nerr.Error()
	} else {
		native.Status = nr.Status
		native.GasUsed = nr.GasUsed
		native.ExecutionGas = blockGas - npool.Gas()
		native.StateGas = blockGas - npool.StateGas()
		native.BlockGasUsed = max(native.ExecutionGas, native.StateGas)
		for _, log := range nr.Logs {
			view := logView{Address: log.Address.Hex(), Topics: []string{}, Data: hexData(log.Data)}
			for _, topic := range log.Topics {
				view.Topics = append(view.Topics, topic.Hex())
			}
			native.Logs = append(native.Logs, view)
		}
	}
	addresses := []types.Address{sender, miner, target, child, types.Address(gethcrypto.CreateAddress(gethcommon.Address(sender), 0))}
	if len(sc.DelegateCode) > 0 {
		addresses = append(addresses, delegate)
	}
	if sc.AuthorizationCase != "" {
		addresses = append(addresses, authority, delegate)
	}

	native.Accounts = readAccounts(addresses, ndb.Exist, ndb.GetBalance, ndb.GetNonce, ndb.GetCode, ndb.GetState)
	root, err := ndb.Commit()
	if err != nil {
		return comparison{}, err
	}
	native.StateRoot = root.Hex()
	gst, err := bridge.MakePreState(pre)
	if err != nil {
		return comparison{}, err
	}
	defer gst.Close()
	rcfg := referenceConfig(amsterdam)
	random := gethcommon.Hash{}
	context := gethvm.BlockContext{CanTransfer: gethcore.CanTransfer, Transfer: gethcore.Transfer, GetHash: func(uint64) gethcommon.Hash { return gethcommon.Hash{} }, Coinbase: gethcommon.Address(miner), GasLimit: blockGas, BlockNumber: big.NewInt(1), Time: 12, Difficulty: new(big.Int), BaseFee: new(big.Int).Set(header.BaseFee), Random: &random, SlotNum: slot, CostPerStateByte: params.CostPerStateByte}
	evm := gethvm.NewEVM(context, gst.StateDB, rcfg, gethvm.Config{})
	msg, err := gethcore.TransactionToMessage(tx, gethtypes.MakeSigner(rcfg, header.Number, header.Time), header.BaseFee)
	if err != nil {
		return comparison{}, err
	}
	gst.StateDB.SetTxContext(tx.Hash(), 0, 1)
	gpool := gethcore.NewGasPool(blockGas)
	snapshot := gst.StateDB.Snapshot()
	gr, gerr := gethcore.ApplyMessage(evm, msg, gpool)
	reference := observation{Valid: gerr == nil, Logs: []logView{}}
	if gerr != nil {
		reference.ValidationError = gerr.Error()
		gst.StateDB.RevertToSnapshot(snapshot)
	} else {
		reference.Status = 1
		if gr.Failed() {
			reference.Status = 0
		}
		reference.GasUsed = gr.UsedGas
		reference.BlockGasUsed = gpool.Used()
		reference.ExecutionGas = gpool.CumulativeExecution()
		reference.StateGas = gpool.CumulativeState()
		if !amsterdam {
			reference.ExecutionGas = reference.BlockGasUsed
		}
		for _, log := range gst.StateDB.GetLogs(tx.Hash(), 1, gethcommon.Hash{}, 12) {
			view := logView{Address: types.Address(log.Address).Hex(), Topics: []string{}, Data: hexData(log.Data)}
			for _, topic := range log.Topics {
				view.Topics = append(view.Topics, types.Hash(topic).Hex())
			}
			reference.Logs = append(reference.Logs, view)
		}
	}
	rules := rcfg.Rules(header.Number, true, header.Time)
	gst.StateDB.Finalise(rules)
	reference.Accounts = readAccounts(addresses, func(a types.Address) bool { return gst.StateDB.Exist(gethcommon.Address(a)) }, func(a types.Address) *big.Int { return gst.StateDB.GetBalance(gethcommon.Address(a)).ToBig() }, func(a types.Address) uint64 { return gst.StateDB.GetNonce(gethcommon.Address(a)) }, func(a types.Address) []byte { return gst.StateDB.GetCode(gethcommon.Address(a)) }, func(a types.Address, k types.Hash) types.Hash {
		return types.Hash(gst.StateDB.GetState(gethcommon.Address(a), gethcommon.Hash(k)))
	})
	reference.StateRoot = gst.StateDB.IntermediateRoot(rules).Hex()
	fork := "osaka_control"
	if amsterdam {
		fork = "amsterdam_merged_text_candidate"
	}
	out := comparison{Name: sc.Name, Input: input{Fork: fork, BlockNumber: 1, Timestamp: 12, Slot: 7, BlockGasLimit: blockGas, BaseFee: header.BaseFee.String(), Coinbase: miner.Hex(), SignedTransaction: hexData(raw), PreState: freezePreState(pre), NativeConfig: nativeConfig(amsterdam), ReferenceConfig: rcfg, ReferenceCPSB: context.CostPerStateByte}, Native: native, Reference: reference, Differences: []string{}}
	for _, field := range []struct {
		name              string
		native, reference any
	}{
		{"validity", native.Valid, reference.Valid}, {"receipt_status", native.Status, reference.Status}, {"user_gas_used", native.GasUsed, reference.GasUsed}, {"block_gas_used", native.BlockGasUsed, reference.BlockGasUsed}, {"block_execution_gas", native.ExecutionGas, reference.ExecutionGas}, {"block_state_gas", native.StateGas, reference.StateGas}, {"post_state_root", native.StateRoot, reference.StateRoot}, {"accounts", native.Accounts, reference.Accounts}, {"logs", native.Logs, reference.Logs},
	} {
		if !reflect.DeepEqual(field.native, field.reference) {
			out.Differences = append(out.Differences, field.name)
		}
	}
	return out, nil
}
func hexData(data []byte) string { return "0x" + hex.EncodeToString(data) }
func readAccounts(addresses []types.Address, exists func(types.Address) bool, balance func(types.Address) *big.Int, nonce func(types.Address) uint64, code func(types.Address) []byte, storage func(types.Address, types.Hash) types.Hash) map[string]accountView {
	out := make(map[string]accountView)
	for _, address := range addresses {
		view := accountView{Exists: exists(address), Balance: balance(address).String(), Nonce: nonce(address), Code: hexData(code(address)), Storage: make(map[string]string)}
		for _, slot := range []types.Hash{{}, {31: 1}} {
			view.Storage[slot.Hex()] = storage(address, slot).Hex()
		}
		out[address.Hex()] = view
	}
	return out
}
func makeReport() (report, error) {
	out := report{Status: "development corpus; not release-qualified", Scope: "signed transaction transitions; pool and outer block validation are not invoked", EIPsRevision: eipsRevision, OracleModule: "github.com/ethereum/go-ethereum", OracleVersion: "unknown", Uncovered: []string{"pool admission and outer block validation", "full EEST corpus", "7702 authorizations and application invariants", "multi-transaction block settlement and next base fee", "method-specific RPC semantics", "cross-client process/devnet recovery", "unmerged alternative gas models"}}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == out.OracleModule {
				out.OracleVersion = dep.Version
				if dep.Replace != nil {
					return out, fmt.Errorf("oracle module replaced: %s", dep.Replace.Path)
				}
			}
		}
	}
	if out.OracleVersion != "v1.17.10" {
		return out, fmt.Errorf("oracle version %s requires explicit requalification", out.OracleVersion)
	}
	if raw, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		out.NativeRevision = strings.TrimSpace(string(raw))
	}
	if raw, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		out.NativeDirty = len(raw) > 0
	}
	for _, amsterdam := range []bool{false, true} {
		for _, sc := range scenarios() {
			result, err := runScenario(sc, amsterdam)
			if err != nil {
				return out, fmt.Errorf("%s Amsterdam=%t: %w", sc.Name, amsterdam, err)
			}
			out.Cases = append(out.Cases, result)
		}
	}
	sort.SliceStable(out.Cases, func(i, j int) bool {
		return out.Cases[i].Input.Fork+out.Cases[i].Name < out.Cases[j].Input.Fork+out.Cases[j].Name
	})
	return out, nil
}
func main() {
	result, err := makeReport()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func freezePreState(pre map[string]bridge.PreAccount) map[string]accountView {
	out := make(map[string]accountView, len(pre))
	for address, account := range pre {
		value := accountView{Exists: true, Balance: account.Balance.String(), Nonce: account.Nonce, Code: hexData(account.Code), Storage: make(map[string]string)}
		for slot, stored := range account.Storage {
			value.Storage[slot.Hex()] = stored.Hex()
		}
		out[address] = value
	}
	return out
}
