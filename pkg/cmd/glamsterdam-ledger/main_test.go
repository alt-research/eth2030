package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"testing"
)

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
