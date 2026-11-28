package main

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

func TestGroupEExternalFixtureIsCanonicalAndDomainBound(t *testing.T) {
	fixture, err := ordinary(7, 19)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(fixture["raw"].(string)[2:])
	if err != nil {
		t.Fatal(err)
	}
	var tx gethtypes.Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	sender, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(big.NewInt(fixtureChain)), &tx)
	if err != nil || sender.Hex() != fixture["sender"] || tx.Nonce() != 7 || tx.Value().Uint64() != 19 || tx.Gas() != 250000 || tx.Hash().Hex() != fixture["transaction"] {
		t.Fatal("fixture does not bind the selected isolated chain/nonce/value", err)
	}
	if _, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(big.NewInt(1)), &tx); err == nil {
		t.Fatal("fixture accepted outside its chain domain")
	}
	// Qualify the actual new-recipient state charge, rather than just signing
	// an Osaka-era 21,000-gas transfer that now fails at runtime.
	ordinaryTx, err := types.DecodeBlockTransaction(raw)
	if err != nil {
		t.Fatal(err)
	}
	db := state.NewMemoryStateDB()
	account := types.Address(sender)
	db.AddBalance(account, big.NewInt(1000000000000000000))
	db.SetNonce(account, 7)
	cfg := *config.TestConfigHogota
	cfg.ChainID = big.NewInt(fixtureChain)
	header := &types.Header{Number: big.NewInt(1), Time: 1, GasLimit: 30000000, BaseFee: big.NewInt(1)}
	receipt, _, err := execution.ApplyTransaction(&cfg, db, header, ordinaryTx, new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil || receipt.Status != types.ReceiptStatusSuccessful || receipt.StateGasUsed != 183600 || db.GetBalance(*ordinaryTx.To()).Cmp(big.NewInt(19)) != 0 {
		t.Fatal("ordinary external fixture does not execute under current EIP-8037", receipt, err)
	}
	pq, err := sample()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := hex.DecodeString(pq["raw"].(string)[2:])
	if err != nil || len(encoded) != pq["transaction_bytes"] || len(encoded) > 8192 {
		t.Fatal("invalid PQ transport fixture", err)
	}
	decoded, err := types.DecodeBlockTransaction(encoded)
	if err != nil || !decoded.HasDependencies8504() {
		t.Fatal("sample is not the canonical dependency envelope", err)
	}
	// The full-capacity control uses the second already-funded public account
	// and a distinct new recipient, without depending on the first sender's nonce.
	filler, err := ordinaryWithBudget(0, 1, 250000, 2)
	if err != nil {
		t.Fatal(err)
	}
	fillerRaw, _ := hex.DecodeString(filler["raw"].(string)[2:])
	fillerTx, err := types.DecodeBlockTransaction(fillerRaw)
	if err != nil {
		t.Fatal(err)
	}
	fillerSender := types.HexToAddress(filler["sender"].(string))
	db.AddBalance(fillerSender, big.NewInt(1000000000000000000))
	remaining := new(gaspool.GasPool).AddGas(header.GasLimit)
	filled, _, err := execution.ApplyTransaction(&cfg, db, header, fillerTx, remaining)
	if err != nil || filled.Status != types.ReceiptStatusSuccessful || filled.StateGasUsed != 183600 || fillerSender == account || *fillerTx.To() == *ordinaryTx.To() {
		t.Fatal("full-capacity filler is not independently executable", filled, err)
	}
	nearLimit, err := ordinaryWithBudget(db.GetNonce(account), 1, header.GasLimit-32768, 1)
	if err != nil {
		t.Fatal(err)
	}
	nearRaw, _ := hex.DecodeString(nearLimit["raw"].(string)[2:])
	nearTx, err := types.DecodeBlockTransaction(nearRaw)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := execution.ApplyTransaction(&cfg, db.Dup(), header, nearTx, remaining); !errors.Is(err, gaspool.ErrGasPoolExhausted) {
		t.Fatal("the filler did not make the declared reservation unavailable", err)
	}
	accepted, _, err := execution.ApplyTransaction(&cfg, db.Dup(), header, nearTx, new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil || accepted.Status != types.ReceiptStatusSuccessful {
		t.Fatal("capacity fixture must otherwise be a valid current-fork transaction", accepted, err)
	}
	for _, params := range [][2]uint64{{250000, 3}, {1 << 32, 1}} {
		if _, err := ordinaryWithBudget(0, 1, params[0], params[1]); err == nil {
			t.Fatal("fixture escaped its account or total-gas domain")
		}
	}
}
