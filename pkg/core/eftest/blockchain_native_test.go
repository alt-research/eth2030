package eftest

import (
	"encoding/json"
	"testing"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
)

func TestNativeBlockchainOfficialRegressions(t *testing.T) {
	for _, file := range []string{"bal", "zero_withdrawal", "queue", "bal_floor", "bal_delegation", "bal_storage", "history"} {
		t.Run(file, func(t *testing.T) {
			tests, err := LoadBlockchainFixtures("testdata/blockchain/" + file + ".json")
			if err != nil {
				t.Fatal(err)
			}
			for name, fixture := range tests {
				result := fixture.RunNative()
				if result.Passed {
					continue
				}
				t.Errorf("%s: %s", name, result.Error)
				// On failure retain exact address-level evidence from the real processor.
				cfg, _ := fixture.nativeConfig()
				db, _ := fixtureState(fixture.Pre)
				encoded, _ := fixtureBytes(fixture.Blocks[0].RLP, -1)
				b, _ := types.DecodeBlockRLP(encoded)
				processor := execution.NewStateProcessor(cfg)
				processed, e := processor.ProcessWithBAL(b, db)
				if e != nil {
					t.Logf("processor: %v", e)
					continue
				}
				t.Logf("actual state %s expected %s", db.GetRoot(), b.Root())
				if db.GetRoot() != b.Root() {
					for key, want := range fixture.Post {
						address := types.HexToAddress(key)
						amount, _ := fixtureNumber(want.Balance, 256)
						nonce, _ := fixtureNumber(want.Nonce, 64)
						code, _ := fixtureBytes(want.Code, -1)
						if amount.Cmp(db.GetBalance(address)) != 0 || nonce.Uint64() != db.GetNonce(address) || string(code) != string(db.GetCode(address)) {
							t.Logf("state account %s balance=%s/%s nonce=%d/%s code=%x/%s", key, db.GetBalance(address), want.Balance, db.GetNonce(address), want.Nonce, db.GetCode(address), want.Code)
						}
						for slot, value := range want.Storage {
							k, _ := fixtureNumber(slot, 256)
							v, _ := fixtureNumber(value, 256)
							got := db.GetState(address, types.BytesToHash(k.Bytes()))
							if got != types.BytesToHash(v.Bytes()) {
								t.Logf("state storage %s/%s got=%s want=%s", key, slot, got, value)
							}
						}
					}
				}

				if processed.BlockAccessList != nil {
					actual, _ := processed.BlockAccessList.EncodeRLP()
					want, _ := fixtureBAL(fixture.Blocks[0].BAL)
					a, _ := bal.DecodeRLP(actual)
					w, _ := bal.DecodeRLP(want)
					expected := map[types.Address]bal.AccountChanges{}
					for _, account := range w {
						expected[account.Address] = account
					}
					for _, account := range a {
						v := expected[account.Address]
						x, _ := json.Marshal(account)
						y, _ := json.Marshal(v)
						if string(x) != string(y) {
							t.Logf("account %s actual %s expected %s", account.Address, x, y)
						}
						delete(expected, account.Address)
					}
					for address, account := range expected {
						x, _ := json.Marshal(account)
						t.Logf("missing account %s %s", address, x)
					}
				}
			}
		})
	}
}
