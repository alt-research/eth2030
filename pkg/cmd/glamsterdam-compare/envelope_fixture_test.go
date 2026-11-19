package main

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/eth2030/eth2030/core/eftest"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestAmsterdamFiveSignedEnvelopeFixtureConsumersMatchIndependentSDK(t *testing.T) {
	seen := make(map[uint8]bool)
	negatives := 0
	for _, sc := range scenarios() {
		if sc.ChainID != 1 {
			continue
		}
		c, err := runScenario(sc, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Differences) != 0 {
			t.Fatalf("envelope %s independent differences: %v", sc.Name, c.Differences)
		}
		if sc.BlobCase != "" {
			negatives++
			if c.Native.Valid || c.Reference.Valid {
				t.Fatalf("invalid blob envelope accepted: %s", sc.Name)
			}
			continue
		}
		if !c.Reference.Valid {
			t.Fatalf("positive envelope %s did not execute: %+v", sc.Name, c.Reference)
		}
		raw, err := hex.DecodeString(c.Input.SignedTransaction[2:])
		if err != nil {
			t.Fatal(err)
		}
		var tx gethtypes.Transaction
		if err := tx.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		if seen[tx.Type()] {
			t.Fatalf("duplicate rather than distinct envelope %d", tx.Type())
		}
		seen[tx.Type()] = true
		sender, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(big.NewInt(1)), &tx)
		if err != nil {
			t.Fatal(err)
		}
		pre := make(map[string]any)
		for address, account := range c.Input.PreState {
			balance, ok := new(big.Int).SetString(account.Balance, 10)
			if !ok {
				t.Fatal("bad independent balance")
			}
			pre[address] = map[string]any{"balance": hexutil.EncodeBig(balance), "nonce": hexutil.EncodeUint64(account.Nonce), "code": account.Code, "storage": account.Storage}
		}
		var sdkLogs []*gethtypes.Log
		for _, log := range c.Reference.Logs {
			l := &gethtypes.Log{Address: gethcommon.HexToAddress(log.Address), Data: gethcommon.FromHex(log.Data)}
			for _, topic := range log.Topics {
				l.Topics = append(l.Topics, gethcommon.HexToHash(topic))
			}
			sdkLogs = append(sdkLogs, l)
		}
		encoded, err := gethrlp.EncodeToBytes(sdkLogs)
		if err != nil {
			t.Fatal(err)
		}
		post := map[string]any{"hash": c.Reference.StateRoot, "logs": gethcrypto.Keccak256Hash(encoded).Hex(), "txbytes": c.Input.SignedTransaction, "indexes": map[string]int{"data": 0, "gas": 0, "value": 0}}
		baseFee, ok := new(big.Int).SetString(c.Input.BaseFee, 10)
		if !ok {
			t.Fatal("bad base fee")
		}
		env := map[string]any{"currentCoinbase": c.Input.Coinbase, "currentGasLimit": hexutil.EncodeUint64(c.Input.BlockGasLimit), "currentNumber": hexutil.EncodeUint64(c.Input.BlockNumber), "currentTimestamp": hexutil.EncodeUint64(c.Input.Timestamp), "currentSlotNumber": hexutil.EncodeUint64(c.Input.Slot), "currentBaseFee": hexutil.EncodeBig(baseFee), "currentRandom": (gethcommon.Hash{}).Hex(), "currentDifficulty": "0x0"}
		if c.Input.ExcessBlobGas != nil {
			env["currentExcessBlobGas"] = hexutil.EncodeUint64(*c.Input.ExcessBlobGas)
		}
		fixture := map[string]any{sc.Name: map[string]any{"env": env, "pre": pre, "transaction": map[string]any{"data": []string{hexutil.Encode(tx.Data())}, "gasLimit": []string{hexutil.EncodeUint64(tx.Gas())}, "value": []string{hexutil.EncodeBig(tx.Value())}, "sender": sender.Hex()}, "post": map[string]any{"Amsterdam": []any{post}}}}
		data, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "independent-amsterdam.json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		native, err := eftest.LoadStateTests(path)
		if err != nil {
			t.Fatal(err)
		}
		r := native[sc.Name].Run(eftest.StateSubtest{Fork: "Amsterdam"})
		if !r.Passed || r.Error != nil {
			t.Fatalf("type %d SDK-derived fixture failed actual native consumer: %+v", tx.Type(), r)
		}
	}
	if len(seen) != 5 || negatives != 3 {
		t.Fatalf("incomplete matrix: types=%v invalid=%d", seen, negatives)
	}
}
