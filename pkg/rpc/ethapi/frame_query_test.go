package ethapi

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/rpc/internal/testutil"
)

func TestFrameCPendingAndMinedQueryLocations(t *testing.T) {
	for _, mined := range []bool{false, true} {
		for _, route := range []string{"native", "pool", "direct"} {
			t.Run(route+map[bool]string{false: "/pending", true: "/mined"}[mined], func(t *testing.T) {
				backend := testutil.NewMockBackend()
				tx := types.NewTransaction(&types.FrameTx8141{ChainID: big.NewInt(1337), Sender: types.Address{19: 42}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(2)}})
				info := &testutil.MockTxInfo{Tx: tx}
				if mined {
					info.BlockNum, info.Index = 42, 3
				}
				backend.Transactions[tx.Hash()] = info
				hash, _ := json.Marshal(tx.Hash().Hex())
				request := &Request{JSONRPC: "2.0", Method: "eth_getTransactionByHash", Params: []json.RawMessage{hash}, ID: json.RawMessage(`1`)}
				var result any
				switch route {
				case "direct":
					var err error
					result, err = NewEthDirectAPI(backend).GetTransactionByHash(tx.Hash().Hex())
					if err != nil {
						t.Fatal(err)
					}
				default:
					var response *Response
					if route == "native" {
						response = NewEthAPI(backend, nil).HandleRequest(request)
					} else {
						response = NewTxPoolAPI(backend).GetTransactionByHash(request)
					}
					if response.Error != nil {
						t.Fatal(response.Error)
					}
					result = response.Result
				}
				encoded, err := json.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]any
				if err := json.Unmarshal(encoded, &fields); err != nil {
					t.Fatal(err)
				}
				if fields["sender"] != tx.FrameSender().Hex() || fields["fees"] == nil || fields["frames"] == nil {
					t.Fatal("query adapter lost authenticated Frame fields")
				}
				want := map[string]any{"blockHash": nil, "blockNumber": nil, "transactionIndex": nil}
				if mined {
					want = map[string]any{"blockHash": backend.Headers[42].Hash().Hex(), "blockNumber": "0x2a", "transactionIndex": "0x3"}
				}
				for name, expected := range want {
					if value, present := fields[name]; !present || value != expected {
						t.Fatalf("EIP-1474 %s=%v present=%t, want %v", name, value, present, expected)
					}
				}
			})
		}
	}
}

func TestFrame8250TxPoolMapDoesNotAliasLegacyNonce(t *testing.T) {
	sender := types.Address{19: 42}
	keyed := types.NewFrameTransaction8250(&types.FrameTx8250Delta{Sender: sender, NonceKeys: []*big.Int{big.NewInt(42)}})
	ordinary := types.NewTransaction(&types.LegacyTx{Nonce: 0, GasPrice: big.NewInt(1)})
	txs := map[types.Address][]*types.Transaction{sender: {ordinary, keyed}}
	content := FormatTxPoolMap(txs)[sender.Hex()]
	inspect := formatTxPoolInspect(txs)[sender.Hex()]
	if len(content) != 2 || len(inspect) != 2 || content["0x0"] == nil || content["keyed:"+keyed.Hash().Hex()] == nil {
		t.Fatal("txpool nonce map overwrote an independent keyed domain")
	}
}
