package main

import (
	"encoding/json"
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
