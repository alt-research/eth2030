package execution

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	gethtests "github.com/ethereum/go-ethereum/tests"
)

func TestOfficialOsakaDelegationReentryMatchesIndependentState(t *testing.T) {
	data, err := os.ReadFile("testdata/eels-osaka-delegation-reentry.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 {
		t.Fatal("expected one complete frozen official case")
	}
	number := func(s string) *big.Int {
		t.Helper()
		n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
		if !ok {
			t.Fatalf("invalid frozen quantity %q", s)
		}
		return n
	}
	decode := func(s string) []byte {
		t.Helper()
		b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for name, encoded := range raw {
		t.Run(name, func(t *testing.T) {
			var fixture struct {
				Env struct {
					Coinbase string `json:"currentCoinbase"`
					GasLimit string `json:"currentGasLimit"`
					Number   string `json:"currentNumber"`
					Time     string `json:"currentTimestamp"`
					BaseFee  string `json:"currentBaseFee"`
					Random   string `json:"currentRandom"`
				}
				Pre map[string]struct {
					Balance, Nonce, Code string
					Storage              map[string]string
				}
				Post map[string][]struct {
					Hash    string
					TxBytes string `json:"txbytes"`
				}
			}
			if err := json.Unmarshal(encoded, &fixture); err != nil {
				t.Fatal(err)
			}
			var oracle gethtests.StateTest
			if err := json.Unmarshal(encoded, &oracle); err != nil {
				t.Fatal(err)
			}
			reference, root, used, err := oracle.RunNoVerify(gethtests.StateSubtest{Fork: "Osaka"}, gethvm.Config{}, false, "hash")
			defer reference.Close()
			if err != nil {
				t.Fatal(err)
			}
			post := fixture.Post["Osaka"][0]
			if types.Hash(root) != types.HexToHash(post.Hash) {
				t.Fatal("unchanged SDK disagrees with official expected root")
			}
			db := state.NewMemoryStateDB()
			for a, account := range fixture.Pre {
				address := types.HexToAddress(a)
				db.CreateAccount(address)
				db.AddBalance(address, number(account.Balance))
				db.SetNonce(address, number(account.Nonce).Uint64())
				db.SetCode(address, decode(account.Code))
				for k, v := range account.Storage {
					db.SetState(address, types.BytesToHash(number(k).Bytes()), types.BytesToHash(number(v).Bytes()))
				}
			}
			if _, err := db.Commit(); err != nil {
				t.Fatal(err)
			}
			cfg := *config.TestConfig
			cfg.ChainID = big.NewInt(1)
			zero := uint64(0)
			cfg.OsakaTime = &zero
			cfg.BPO1Time = nil
			cfg.BPO2Time = nil
			cfg.AmsterdamTime = nil
			cfg.GlamsterdanTime = nil
			cfg.HogotaTime = nil
			cfg.IPlusTime = nil
			header := &types.Header{Coinbase: types.HexToAddress(fixture.Env.Coinbase), GasLimit: number(fixture.Env.GasLimit).Uint64(), Number: number(fixture.Env.Number), Time: number(fixture.Env.Time).Uint64(), BaseFee: number(fixture.Env.BaseFee), MixDigest: types.HexToHash(fixture.Env.Random), Difficulty: new(big.Int)}
			tx, err := types.DecodeTxRLP(decode(post.TxBytes))
			if err != nil {
				t.Fatal(err)
			}
			db.SetTxContext(tx.Hash(), 0)
			receipt, _, err := ApplyTransaction(&cfg, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit))
			if err != nil {
				t.Fatal(err)
			}
			got, err := db.Commit()
			if err != nil {
				t.Fatal(err)
			}
			if got != types.Hash(root) || receipt.GasUsed != used {
				t.Fatalf("native root/gas %s/%d; independent official root/gas %s/%d", got, receipt.GasUsed, root, used)
			}
		})
	}
}
