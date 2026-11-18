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
