// Package proof8504 supplies an independent pinned EthLib verification vector
// to native consumer tests. The explicit message is deliberately shared between
// transactions, exercising block-union deduplication independently of execution.
package proof8504

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

//go:embed vector.json
var fixture []byte

func Dependency() (types.Dependency8504, []byte) {
	var v struct {
		PublicKey string `json:"public_key"`
		Message   string `json:"message"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(fixture, &v); err != nil {
		panic(err)
	}
	pk, err := hex.DecodeString(v.PublicKey[2:])
	if err != nil {
		panic(err)
	}
	msg, err := hex.DecodeString(v.Message[2:])
	if err != nil {
		panic(err)
	}
	sig, err := hex.DecodeString(v.Signature[2:])
	if err != nil {
		panic(err)
	}
	return types.Dependency8504{Scheme: types.LeanSPHINCSScheme8504, DataHash: types.BytesToHash(msg), VerificationKeyHash: types.BytesToHash(pk)}, sig
}

func Transaction(nonce uint64) *types.Transaction {
	d, w := Dependency()
	f := &types.FrameTx8141{ChainID: big.NewInt(1337), Nonce: nonce, Sender: types.Address{19: 0x90}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100_000_000), MaxPriorityFeePerGas: big.NewInt(2)}, Frames: []types.Frame8141{
		{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 5000}},
		{Mode: types.ModeProof8504, Data: d.Bytes(), Limits: types.FrameGasLimits{Execution: 3000}},
	}}
	return types.NewTransaction(f).WithDependencyWitnesses8504([][]byte{w})
}

var ApproveCode = []byte{0x60, 3, 0x5f, 0x5f, 0xaa}
