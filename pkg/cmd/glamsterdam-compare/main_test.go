package main

import (
	"encoding/json"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"strings"
	"testing"
)

func TestSignedOsakaControl(t *testing.T) {
	result, err := runScenario(scenarios()[0], false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Native.Valid || !result.Reference.Valid || result.Native.Status != 1 || result.Reference.Status != 1 || result.Native.GasUsed != 21000 || len(result.Differences) != 0 {
		t.Fatalf("independent signed-transaction control disagrees: %+v", result)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded comparison
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Input.SignedTransaction != result.Input.SignedTransaction || len(decoded.Input.PreState) != len(result.Input.PreState) {
		t.Fatal("serialized corpus lost its input")
	}
}

func TestReferenceNewAccountRuntimeOOGIsIncluded(t *testing.T) {
	var selected scenario
	for _, sc := range scenarios() {
		if sc.Name == "new_account_runtime_oog" {
			selected = sc
		}
	}
	result, err := runScenario(selected, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Reference.Valid || result.Reference.Status != 0 || result.Reference.Accounts[target.Hex()].Exists {
		t.Fatalf("new-account pre-execution OOG did not reach its intended included failure: %+v", result.Reference)
	}
}

func TestReferenceFundedReservoirChargesActualState(t *testing.T) {
	var ordinary, funded scenario
	for _, sc := range scenarios() {
		switch sc.Name {
		case "allocate_slot":
			ordinary = sc
		case "funded_reservoir_allocate":
			funded = sc
		}
	}
	left, err := runScenario(ordinary, true)
	if err != nil {
		t.Fatal(err)
	}
	right, err := runScenario(funded, true)
	if err != nil {
		t.Fatal(err)
	}
	if left.Reference.StateGas != 64*1530 || right.Reference.StateGas != 64*1530 || left.Reference.GasUsed != right.Reference.GasUsed || left.Reference.StateRoot != right.Reference.StateRoot {
		t.Fatalf("funded reservoir changed the price or state of identical work: bare=%+v funded=%+v", left.Reference, right.Reference)
	}
}

// This exercises native signed transaction execution and both gas-pool
// dimensions, not the helper formulas. Full block/pool/API gates are separate.
func TestNativeAmsterdamSignedTransitionsMatchPinnedReference(t *testing.T) {
	for _, scenario := range scenarios() {
		t.Run(scenario.Name, func(t *testing.T) {
			result, err := runScenario(scenario, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Differences) != 0 {
				t.Fatalf("signed native/reference fields differ: %v", result.Differences)
			}
		})
	}
}

func TestNativeAmsterdamAuthorizationObligations(t *testing.T) {
	expected := map[string]struct {
		status, paid, execution, state, nonce uint64
		delegated                             bool
	}{
		"existing":              {1, 70006, 34816, 35190, 1, true},
		"new":                   {1, 253606, 34816, 218790, 1, true},
		"preexecution_oog":      {0, 50000, 50000, 0, 0, false},
		"late_preexecution_oog": {0, 100000, 100000, 0, 0, false},
		"revert":                {0, 70010, 34820, 35190, 1, true},
		"bad_nonce":             {1, 22921, 22921, 0, 0, false},
		"max_nonce":             {1, 25821, 25821, 0, 0, false},
		"clear_reset":           {1, 85638, 50448, 35190, 3, true},
		"existing_indicator":    {1, 42632, 42632, 0, 2, true},
		"sender":                {1, 58006, 22816, 35190, 2, true},
	}
	for _, sc := range scenarios() {
		want, ok := expected[sc.AuthorizationCase]
		if !ok {
			continue
		}
		t.Run(sc.Name, func(t *testing.T) {
			result, err := runScenario(sc, true)
			if err != nil {
				t.Fatal(err)
			}
			got := result.Native
			if !got.Valid || got.Status != want.status || got.GasUsed != want.paid || got.ExecutionGas != want.execution || got.StateGas != want.state {
				t.Fatalf("authority gas/failure boundary: %+v want=%+v", got, want)
			}
			keySuffix := "2"
			if sc.AuthorizationCase == "sender" {
				keySuffix = "1"
			}
			key, err := gethcrypto.HexToECDSA(strings.Repeat("0", 63) + keySuffix)
			if err != nil {
				t.Fatal(err)
			}
			authority := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey)).Hex()
			account := got.Accounts[authority]
			if account.Nonce != want.nonce {
				t.Fatalf("authority nonce=%d want=%d", account.Nonce, want.nonce)
			}
			if want.delegated {
				if account.Code != "0xef0100"+strings.TrimPrefix(delegate.Hex(), "0x") {
					t.Fatalf("authority delegation lost: %+v", account)
				}
			} else if account.Code != "0x" {
				t.Fatalf("failed/skipped authorization installed code: %+v", account)
			}
		})
	}
}
