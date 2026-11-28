package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/engine/payload"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
)

// A missing proof witness cannot make an already-invalid nonce, fee budget or
// dimensional reservation unknown. Both real Engine consumers are exercised.
func TestGroupEFramePreconditionsBeforeMissingWitness(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		t.Run(fmt.Sprintf("keyed_%t", keyed), func(t *testing.T) {
			const token = "heze-frame-preconditions"
			var genesis map[string]any
			if err := json.Unmarshal(builderRequestGenesisJSON(1337), &genesis); err != nil {
				t.Fatal(err)
			}
			fields := genesis["config"].(map[string]any)
			fields["terminalTotalDifficulty"], fields["amsterdamTime"], fields["bogotaTime"], fields["eip8504Time"] = 0, 0, 0, 0
			original, _ := proof8504.CanonicalTransaction(keyed)
			sender := original.FrameSender()
			genesis["gasLimit"] = "0xf4240"
			alloc := genesis["alloc"].(map[string]any)
			alloc[sender.Hex()] = map[string]string{"balance": "10000000000000000000", "code": "0x" + hex.EncodeToString(proof8504.CanonicalWalletCode()), "nonce": "0x0"}
			if keyed {
				fields["eip8250Time"] = 0
				alloc[types.NonceManager8250.Hex()] = map[string]string{"balance": "0x0", "code": "0x" + hex.EncodeToString(types.NonceManagerCode8250), "nonce": "0x1"}
			}
			wire, err := json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}
			cfg := makeTestConfig(t)
			cfg.EngineAuthToken = token
			cfg.GenesisPath = filepath.Join(cfg.DataDir, "genesis.json")
			if err := os.WriteFile(cfg.GenesisPath, wire, 0600); err != nil {
				t.Fatal(err)
			}
			n := newTestNode(t, &cfg)
			rpc := track4NodeRPC(t, n, token)
			beacon := types.Hash{31: 1}
			build := func(timestamp uint64, list []string) *payload.PayloadID {
				var fc payload.ForkchoiceUpdatedResult
				rpc.call(t, "engine_forkchoiceUpdatedV5", &fc, payload.ForkchoiceStateV1{HeadBlockHash: n.Blockchain().Genesis().Hash()}, map[string]any{"timestamp": fmt.Sprintf("0x%x", timestamp), "slotNumber": "0x1", "withdrawals": []any{}, "prevRandao": types.Hash{}, "suggestedFeeRecipient": types.Address{}, "parentBeaconBlockRoot": beacon, "targetGasLimit": "0xf4240", "inclusionListTransactions": list})
				if fc.PayloadID == nil {
					t.Fatal(fc)
				}
				return fc.PayloadID
			}
			var empty bogotaIntegrationPayload
			rpc.call(t, "engine_getPayloadV6", &empty, build(6, []string{}))
			if empty.ExecutionPayload.BaseFeePerGas == nil || empty.ExecutionPayload.BaseFeePerGas.Sign() <= 0 {
				t.Fatal("fixture requires positive base fee")
			}
			if empty.ExecutionPayload.GasLimit != 1_000_000 {
				t.Fatal("fixture gas limit", empty.ExecutionPayload.GasLimit)
			}
			rpc.call(t, "engine_newPayloadV6", new(bogotaIntegrationStatus), empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, []string{})
			cases := []struct {
				name   string
				change func(*types.FrameTx8141)
			}{
				{"nonce", func(f *types.FrameTx8141) { f.Nonce = 1 }},
				{"fee_budget", func(f *types.FrameTx8141) { f.Fees.MaxFee = new(big.Int) }},
				{"blob_fee_budget", func(f *types.FrameTx8141) {
					f.BlobVersionedHashes = []types.Hash{{0: 1, 31: 1}}
					var budget types.FrameGasBudget
					var e error
					if p := f.NonceProfile8250(); p != nil {
						budget, e = p.GasBudget()
					} else {
						budget, e = f.GasBudget()
					}
					if e != nil {
						t.Fatal(e)
					}
					f.Fees.MaxFee = new(big.Int).Mul(new(big.Int).SetUint64(budget.MaxGas), empty.ExecutionPayload.BaseFeePerGas)
				}},
				{"execution_reservation", func(f *types.FrameTx8141) {
					for i := range f.Frames {
						if f.Frames[i].Mode == types.ModeVerify {
							f.Frames[i].Limits.Execution = 1_000_000
							return
						}
					}
					t.Fatal("missing VERIFY")
				}},
				{"state_reservation", func(f *types.FrameTx8141) {
					for i := range f.Frames {
						if f.Frames[i].Mode == types.ModeVerify {
							f.Frames[i].Limits.State = 1_000_001
							return
						}
					}
					t.Fatal("missing VERIFY")
				}},
			}
			for i, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					tx, _ := proof8504.CanonicalTransaction(keyed)
					f := tx.Inner().(*types.FrameTx8141)
					tc.change(f)
					tx = types.NewTransaction(f)
					if !tx.HasDependencies8504() {
						t.Fatal("fixture lost dependency")
					}
					if err := f.ValidateWithExtensions(n.Blockchain().Config().FrameExtensions(6)); err != nil {
						t.Fatal("fixture must be structurally valid", err)
					}
					raw, err := tx.EncodeRLP()
					if err != nil {
						t.Fatal(err)
					}
					list := []string{"0x" + hex.EncodeToString(raw)}
					var status bogotaIntegrationStatus
					if err := rpc.invoke("engine_newPayloadV6", &status, empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, list); err != nil {
						t.Error("known-invalid precondition required omitted witness:", err)
					} else if status.Status != "VALID" || status.InclusionListSatisfied == nil || !*status.InclusionListSatisfied {
						t.Error("invalid omission verdict", status)
					}
					var built bogotaIntegrationPayload
					if err := rpc.invoke("engine_getPayloadV6", &built, build(uint64(7+i), list)); err != nil {
						t.Error("invalid private candidate required witness:", err)
					} else if len(built.ExecutionPayload.Transactions) != 0 || built.ExecutionPayload.GasUsed != 0 {
						t.Error("invalid candidate changed build")
					}
				})
			}
			// Passing preconditions does not establish proof validity or witness transport.
			raw, err := original.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			list := []string{"0x" + hex.EncodeToString(raw)}
			err = rpc.invoke("engine_newPayloadV6", new(bogotaIntegrationStatus), empty.ExecutionPayload, []types.Hash{}, beacon, empty.ExecutionRequests, list)
			if err == nil || !strings.Contains(err.Error(), "composition is unspecified") {
				t.Fatal("valid-precondition omission lost its explicit witness gate", err)
			}
			err = rpc.invoke("engine_getPayloadV6", new(bogotaIntegrationPayload), build(20, list))
			if err == nil || !strings.Contains(err.Error(), "composition is unspecified") {
				t.Fatal("private build invented witness availability", err)
			}
		})
	}
}
