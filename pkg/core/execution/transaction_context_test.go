package execution

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	bridge "github.com/eth2030/eth2030/geth"
	common "github.com/ethereum/go-ethereum/common"
	core "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	crypto "github.com/ethereum/go-ethereum/crypto"
)

func TestConsecutiveSignedTransactionsIsolateRefundAndTransientState(t *testing.T) {
	for _, amsterdam := range []bool{false, true} {
		fork := "Osaka"
		if amsterdam {
			fork = "Amsterdam"
		}
		t.Run(fork, func(t *testing.T) {
			key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", 1))
			if err != nil {
				t.Fatal(err)
			}
			sender := crypto.PubkeyToAddress(key.PublicKey)
			target := common.Address{19: 0x42}
			miner := common.Address{19: 0x43}
			// First call makes and clears a slot (refund), then TSTOREs. Second call
			// persists TLOAD's result: transaction isolation requires that result be zero.
			code := []byte{0x36, 0x60, 0x14, 0x57, 0x60, 1, 0x60, 0, 0x55, 0x60, 0, 0x60, 0, 0x55, 0x60, 42, 0x60, 7, 0x5d, 0, 0x5b, 0x60, 7, 0x5c, 0x60, 1, 0x55, 0}
			pre := map[string]bridge.PreAccount{sender.Hex(): {Balance: big.NewInt(1000000000000)}, target.Hex(): {Balance: big.NewInt(1), Nonce: 1, Code: code}, miner.Hex(): {Balance: big.NewInt(1), Nonce: 1}}
			db := state.NewMemoryStateDB()
			for addr, p := range pre {
				a := types.HexToAddress(addr)
				db.CreateAccount(a)
				db.AddBalance(a, p.Balance)
				db.SetNonce(a, p.Nonce)
				db.SetCode(a, p.Code)
			}
			if _, err := db.Commit(); err != nil {
				t.Fatal(err)
			}
			ref, err := bridge.MakePreState(pre)
			if err != nil {
				t.Fatal(err)
			}
			defer ref.Close()
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
			slot := uint64(1)
			h := &types.Header{Number: big.NewInt(1), Time: 12, SlotNumber: &slot, GasLimit: 30000000, BaseFee: big.NewInt(1), Coinbase: types.Address(miner), Difficulty: new(big.Int)}
			pool := new(gaspool.GasPool).AddGas(h.GasLimit)
			refPool := core.NewGasPool(h.GasLimit)
			for nonce := uint64(0); nonce < 3; nonce++ {
				var data []byte
				if nonce > 0 {
					data = []byte{1}
				}
				tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: nonce, Gas: 500000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1), To: &target, Value: new(big.Int), Data: data}), gethtypes.LatestSignerForChainID(big.NewInt(1337)), key)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := tx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				native, err := types.DecodeBlockTransaction(raw)
				if err != nil {
					t.Fatal(err)
				}
				db.SetTxContext(native.Hash(), int(nonce))
				receipt, _, err := ApplyTransaction(&cfg, db, h, native, pool)
				if err != nil {
					t.Fatal(err)
				}
				random := common.Hash{}
				ctx := gethvm.BlockContext{CanTransfer: core.CanTransfer, Transfer: core.Transfer, Coinbase: miner, GasLimit: h.GasLimit, BlockNumber: h.Number, Time: h.Time, Difficulty: new(big.Int), BaseFee: h.BaseFee, Random: &random, SlotNum: slot, BlobBaseFee: big.NewInt(1), CostPerStateByte: 1530}
				evm := gethvm.NewEVM(ctx, ref.StateDB, oracle, gethvm.Config{})
				msg, err := core.TransactionToMessage(tx, gethtypes.MakeSigner(oracle, h.Number, h.Time), h.BaseFee)
				if err != nil {
					t.Fatal(err)
				}
				ref.StateDB.SetTxContext(tx.Hash(), int(nonce), 1)
				result, err := core.ApplyMessage(evm, msg, refPool)
				evm.Release()
				if err != nil || result.Failed() {
					t.Fatalf("independent tx %d: %v/%v", nonce, err, result.Err)
				}
				rules := oracle.Rules(h.Number, true, h.Time)
				ref.StateDB.Finalise(rules)
				if receipt.Status != 1 || receipt.GasUsed != result.UsedGas {
					t.Fatalf("tx %d gas native=%d independent=%d status=%d", nonce, receipt.GasUsed, result.UsedGas, receipt.Status)
				}
				if nonce > 0 && db.GetState(types.Address(target), types.Hash{31: 1}) != (types.Hash{}) {
					t.Fatal("transient storage crossed the transaction boundary into persistent storage")
				}
				if got, want := db.GetRoot(), types.Hash(ref.StateDB.IntermediateRoot(rules)); got != want {
					t.Fatalf("tx %d full root %s/%s", nonce, got, want)
				}
				t.Logf("%s tx %d gas=%d root=%s refund=%d", fork, nonce, receipt.GasUsed, db.GetRoot(), db.GetRefund())
				// No Commit: all three signed transactions share the live state/gas pool.
			}
		})
	}
}
