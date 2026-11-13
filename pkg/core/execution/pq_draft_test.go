package execution

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
)

func TestPQDraftActivationAndStateLookup(t *testing.T) {
	file, err := os.Open("../vm/testdata/eip8051_expanded.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var vectors []struct{ Name, Message, Signature, Key string }
	if err := json.NewDecoder(reader).Decode(&vectors); err != nil || len(vectors) != 2 {
		t.Fatalf("draft vectors: %v, got %d", err, len(vectors))
	}
	activation := uint64(1000)
	cfg := *config.TestConfig
	cfg.PragueTime = nil
	cfg.GlamsterdanTime = nil
	cfg.PQDraft = &config.PQDraftConfig{
		Time: &activation, SigRecover: types.Address{18: 0x70, 19: 1},
		MLDSA: types.Address{18: 0x70, 19: 2}, MLDSAETH: types.Address{18: 0x70, 19: 3},
	}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			decode := func(s string) []byte {
				out, err := hex.DecodeString(s)
				if err != nil {
					t.Fatal(err)
				}
				return out
			}
			message, signature, key := decode(vector.Message), decode(vector.Signature), decode(vector.Key)
			variant, direct := vm.EIP8051DraftNIST, cfg.PQDraft.MLDSA
			if vector.Name == "Keccak" {
				variant, direct = vm.EIP8051DraftETH, cfg.PQDraft.MLDSAETH
			}
			id := crypto.Keccak256(key)[12:]
			keyAccount := types.BytesToAddress(id)
			want := make([]byte, 32)
			copy(want[12:], crypto.Keccak256([]byte{variant}, key)[12:])
			envelope := bytes.Join([][]byte{{variant}, signature, id, message}, nil)
			payload := bytes.Join([][]byte{message, signature, key}, nil)

			run := func(time uint64, to types.Address, data, code []byte, warm bool) (*ExecutionResult, *bal.AccessTracker) {
				t.Helper()
				db := state.NewMemoryStateDB()
				from := setupSender(db)
				if code != nil {
					db.SetCode(keyAccount, code)
				}
				header := newTestHeader()
				header.Time = time
				tx := config.Message{
					From: from, To: &to, Value: new(big.Int), GasLimit: 700_000,
					GasPrice: big.NewInt(2), GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(2), Data: data,
				}
				if warm {
					tx.AccessList = types.AccessList{{Address: keyAccount}}
				}
				tracker := bal.NewTracker()
				result, err := ApplyMessage(&cfg, nil, db, header, &tx, new(gaspool.GasPool).AddGas(header.GasLimit), tracker)
				if err != nil || result.Failed() {
					t.Fatalf("execute at %d: %v, result %+v", time, err, result)
				}
				return result, tracker
			}

			code := append([]byte{0}, key...)
			before, _ := run(activation-1, cfg.PQDraft.SigRecover, envelope, code, false)
			if len(before.ReturnData) != 0 {
				t.Fatal("recovery ran before activation")
			}
			at, tracker := run(activation, cfg.PQDraft.SigRecover, envelope, code, false)
			if !bytes.Equal(at.ReturnData, want) {
				t.Fatalf("recovered %x, want %x", at.ReturnData, want)
			}
			var tracked bool
			for _, entry := range tracker.Build(0).Entries {
				tracked = tracked || entry.Address == keyAccount
			}
			if !tracked {
				t.Fatal("state-backed key read was absent from the BAL")
			}
			cold, warm := vm.ColdAccountAccessCost, vm.WarmStorageReadCost
			accessCost := accessListGas(types.AccessList{{Address: keyAccount}})
			intrinsic := intrinsicGas(envelope, false, cfg.IsMerge() && cfg.IsShanghai(activation), 0)
			const keyCopy = 1926 // 642 words at three gas per word.
			if at.UsedGas != intrinsic+6000+cold+keyCopy {
				t.Fatalf("cold key gas = %d, want %d", at.UsedGas, intrinsic+6000+cold+keyCopy)
			}
			warmed, _ := run(activation, cfg.PQDraft.SigRecover, envelope, code, true)
			if !bytes.Equal(warmed.ReturnData, want) || warmed.UsedGas != intrinsic+accessCost+6000+warm+keyCopy {
				t.Fatalf("warm key gas = %d, return %x", warmed.UsedGas, warmed.ReturnData)
			}
			corrupt := bytes.Clone(code)
			corrupt[len(corrupt)-1] ^= 1
			for _, rejectedCode := range [][]byte{nil, corrupt, key} {
				result, _ := run(activation, cfg.PQDraft.SigRecover, envelope, rejectedCode, false)
				if !bytes.Equal(result.ReturnData, make([]byte, 32)) {
					t.Fatal("accepted an absent, altered, or malformed key account")
				}
			}
			result, _ := run(activation, direct, payload, nil, false)
			if len(result.ReturnData) != 32 || result.ReturnData[31] != 1 {
				t.Fatalf("direct verification returned %x", result.ReturnData)
			}
		})
	}
}

func TestPQDraftAddressCollisionLeavesStateAndGasUntouched(t *testing.T) {
	activation := uint64(1000)
	for name, address := range map[string]types.Address{
		"factory": eips.FactoryAddress, "standard precompile": {19: 1}, "zero": {}, "duplicate": {18: 0x70, 19: 2},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := *config.TestConfig
			cfg.PQDraft = &config.PQDraftConfig{
				Time: &activation, SigRecover: address, MLDSA: types.Address{18: 0x70, 19: 2}, MLDSAETH: types.Address{18: 0x70, 19: 3},
			}
			db := state.NewMemoryStateDB()
			from := setupSender(db)
			root := db.GetRoot()
			header := newTestHeader()
			header.Time = activation
			pool := new(gaspool.GasPool).AddGas(header.GasLimit)
			message := &config.Message{From: from, To: &address, GasLimit: 50000, GasPrice: big.NewInt(2)}
			if _, err := ApplyMessage(&cfg, nil, db, header, message, pool); !errors.Is(err, vm.ErrPQDraftAddress) {
				t.Fatalf("colliding activation: %v", err)
			}
			if db.GetRoot() != root || pool.Gas() != header.GasLimit {
				t.Fatal("configuration error modified state or gas")
			}
		})
	}
}
