package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPressureProofStreamBindsActualRequestAndIndependentHeader(t *testing.T) {
	input := capturedPayment(t)
	var proofIndex, headerIndex = -1, -1
	for i, record := range input.RPC {
		if record.Method == "eth_getProof" && proofIndex < 0 {
			proofIndex = i
		}
		if record.Method == "eth_getBlockByHash" && len(record.Params) > 0 && string(record.Params[0]) == `"`+input.Hashes[0].Hex()+`"` {
			headerIndex = i
		}
	}
	if proofIndex < 0 || headerIndex < 0 {
		t.Fatal("missing actual proof/header fixture")
	}
	proof := input.RPC[proofIndex]
	line := map[string]any{"kind": "proof", "expected_hash": input.Hashes[0], "header": input.RPC[headerIndex].Reply.Result,
		"request": map[string]any{"method": proof.Method, "params": proof.Params}, "response": proof.Reply}
	// Reply uses exported Go field names; JSON-RPC capture is lower-case.
	line["response"] = map[string]any{"result": proof.Reply.Result}
	path := filepath.Join(t.TempDir(), "proofs.jsonl")
	write := func() {
		t.Helper()
		data, err := json.Marshal(line)
		if err != nil {
			t.Fatal(err)
		}
		prefix := []byte("{\"kind\":\"header\",\"response\":{\"result\":{\"nonce\":\"0x0000000000000000\"}}}\n" +
			"{\"kind\":\"finite-call\",\"response\":{\"result\":\"0x2a\"}}\n" +
			"{\"kind\":\"ws-disconnect\",\"response\":\"raw transport reply\"}\n")
		if err := os.WriteFile(path, append(prefix, append(data, '\n')...), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if count, err := verifyPressureProofs(path); err != nil || count != 1 {
		t.Fatalf("actual proof stream = %d, %v", count, err)
	}
	line["expected_hash"] = input.Hashes[1]
	write()
	if _, err := verifyPressureProofs(path); err == nil {
		t.Fatal("accepted an independently valid proof bound to the wrong requested block")
	}
	line["kind"] = "unrelated"
	write()
	if _, err := verifyPressureProofs(path); err == nil {
		t.Fatal("vacuous proof stream passed")
	}
}

//go:embed testdata/actual-builder-payment.json.gz
var actualPayment []byte

func capturedPayment(t *testing.T) capture {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(actualPayment))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var input capture
	if err := json.NewDecoder(reader).Decode(&input); err != nil {
		t.Fatal(err)
	}
	return input
}

func TestActualBuilderWithdrawalHasThreeIndependentOriginalProofConsumers(t *testing.T) {
	results, err := verify(capturedPayment(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("verified endpoint count=%d", len(results))
	}
	for _, result := range results {
		if result.PaidWei != "101000000000000000" {
			t.Fatalf("actual committed payment changed: %+v", result)
		}
	}
}

func TestOriginalPaymentRejectsHeaderProofClaimAndWithdrawalMutations(t *testing.T) {
	for _, fault := range []string{"header", "balance", "account-proof", "withdrawal-amount", "withdrawal-duplicate", "missing-peer"} {
		t.Run(fault, func(t *testing.T) {
			input := capturedPayment(t)
			if fault == "missing-peer" {
				input.RPC = input.RPC[:len(input.RPC)-4]
			} else {
				method := "eth_getProof"
				if fault == "header" || fault == "withdrawal-amount" || fault == "withdrawal-duplicate" {
					method = "eth_getBlockByHash"
				}
				changed := false
				for i := range input.RPC {
					record := &input.RPC[i]
					if record.Method != method {
						continue
					}
					var result map[string]any
					if err := json.Unmarshal(record.Reply.Result, &result); err != nil {
						t.Fatal(err)
					}
					if method == "eth_getBlockByHash" {
						hash, ok := result["hash"].(string)
						if !ok || (hash != input.Hashes[0].Hex() && hash != input.Hashes[1].Hex()) {
							continue
						}
					}
					switch fault {
					case "header":
						result["stateRoot"] = "0x" + string(bytes.Repeat([]byte{'1'}, 64))
					case "balance":
						result["balance"] = "0x1"
					case "account-proof":
						nodes := result["accountProof"].([]any)
						node := nodes[0].(string)
						nodes[0] = node[:len(node)-2] + "ff"
					case "withdrawal-amount", "withdrawal-duplicate":
						withdrawals := result["withdrawals"].([]any)
						if len(withdrawals) == 0 {
							continue
						}
						if fault == "withdrawal-amount" {
							withdrawals[0].(map[string]any)["amount"] = "0x1"
						} else {
							result["withdrawals"] = append(withdrawals, withdrawals[0])
						}
					}
					raw, err := json.Marshal(result)
					if err != nil {
						t.Fatal(err)
					}
					record.Reply.Result = raw
					changed = true
					break
				}
				if !changed {
					t.Fatal("control did not alter an actual input")
				}
			}
			if _, err := verify(input); err == nil {
				t.Fatalf("accepted altered original %s", fault)
			}
		})
	}
}
