package backend

import (
	"context"
	"fmt"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func frameSimulationFixture(t *testing.T) (*RPCBackend, *state.MemoryStateDB, *types.Header, rpctypes.FrameSimulationOptions) {
	t.Helper()
	db := state.NewMemoryStateDB()
	key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	db.AddBalance(sender, big.NewInt(1000000000))
	zero := uint64(0)
	cfg := *config.TestConfig
	cfg.ChainID = big.NewInt(1337)
	cfg.HogotaTime = &zero
	header := &types.Header{Number: big.NewInt(1), Time: 100, SlotNumber: &zero, GasLimit: 30000000, BaseFee: big.NewInt(1), Root: db.GetRoot()}
	bc, err := chain.NewBlockchain(&cfg, types.NewBlock(header, &types.Body{}), db, rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	backend := NewRPCBackend(&mockNodeDeps{blockchain: bc, config: &Config{}}, nil).(*RPCBackend)
	frame := &types.FrameTx8141{ChainID: cfg.ChainID, Sender: sender, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(2)}, Signatures: []types.FrameSignature{{Scheme: 1, Signature: make([]byte, 65)}}}
	hash, _ := frame.SigningHash()
	sig, _ := gethcrypto.Sign(hash[:], key)
	frame.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
	raw, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	return backend, db, header, rpctypes.FrameSimulationOptions{RawTransaction: hexutil.Encode(raw), Trace: true}
}
func TestFrameCSimulationAuthenticationOverridesAndCancellation(t *testing.T) {
	backend, db, header, options := frameSimulationFixture(t)
	root := db.GetRoot()
	result, err := backend.SimulateFrame(options, header)
	if err != nil || !result.Valid {
		t.Fatalf("replay=%+v err=%v", result, err)
	}
	// Hypothetical nonce mismatch is execution failure, never ordinary CALL success.
	sender := *result.Payer
	nonce := "0x1"
	options.StateOverrides = rpctypes.StateOverride{sender: {Nonce: &nonce}}
	result, err = backend.SimulateFrame(options, header)
	if err != nil || result.Valid || result.Error == "" {
		t.Fatalf("invalid nonce replay=%+v err=%v", result, err)
	}
	if db.GetRoot() != root {
		t.Fatal("override changed canonical state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scoped := backend.WithContext(ctx).(*RPCBackend)
	if _, err := scoped.SimulateFrame(options, header); err == nil {
		t.Fatal("cancelled simulation executed")
	}
	raw, _ := hexutil.Decode(options.RawTransaction)
	raw[len(raw)-1] ^= 1
	options.RawTransaction = hexutil.Encode(raw)
	if result, err := backend.SimulateFrame(options, header); err == nil {
		t.Fatalf("mutated signed envelope accepted: %+v", result)
	}
}
