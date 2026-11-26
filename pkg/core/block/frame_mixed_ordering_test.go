package block_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// EIP-8141's payment approval advances the account nonce unless nonempty
// EIP-8250 keys are selected. Blob classification must not reorder that domain.
func TestFrameCBlobAndOrdinarySenderOrdering(t *testing.T) {
	for _, profile := range []string{"base", "empty_keys", "keyed"} {
		for _, legacy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/legacy_builder_%t", profile, legacy), func(t *testing.T) {
				cfg := *config.TestConfigHogota
				cfg.ChainID = big.NewInt(1337)
				zero := uint64(0)
				if profile != "base" {
					cfg.EIP8250Time = &zero
				}
				key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
				sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
				target := types.Address{19: 0x90}
				alloc := newBuilderRequestTestState()
				alloc.AddBalance(sender, big.NewInt(100000000))
				alloc.SetNonce(target, 1)
				frame := &types.FrameTx8141{ChainID: cfg.ChainID, Sender: sender, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(1000000), MaxPriorityFeePerGas: big.NewInt(2)}}
				if profile != "base" {
					delta := &types.FrameTx8250Delta{ChainID: frame.ChainID, Sender: sender, Frames: frame.Frames, Fees: frame.Fees}
					if profile == "keyed" {
						alloc.SetNonce(sender, 1)
						alloc.SetNonce(types.NonceManager8250, 1)
						alloc.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
						delta.NonceKeys = []*big.Int{big.NewInt(42)}
						delta.NonceSeq = ^uint64(0)
						delta.Frames[0].Limits.State = 97920
					}
					frame = types.NewFrameTransaction8250(delta).Inner().(*types.FrameTx8141)
				}
				blob := make([]byte, types.FrameBytesPerBlob)
				kzg := bls.DefaultKZGBackend()
				commitment, err := kzg.BlobToCommitment(blob)
				if err != nil {
					t.Fatal(err)
				}
				_, proofs, err := kzg.ComputeCellsAndProofs(blob)
				if err != nil {
					t.Fatal(err)
				}
				hash := sha256.Sum256(commitment[:])
				hash[0] = 1
				frame.BlobVersionedHashes = []types.Hash{types.Hash(hash)}
				first := signFrameBlock(t, frame, 1)
				sidecar := &types.BlobSidecarData{Version: 1, Blobs: [][]byte{blob}, Commitments: [][]byte{bytes.Clone(commitment[:])}}
				for _, p := range proofs {
					sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(p[:]))
				}
				first.SetBlobSidecar(sidecar)
				receiver := gethcommon.Address(target)
				signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: cfg.ChainID, Nonce: 1, Gas: 21000, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(4), To: &receiver, Value: big.NewInt(1)}), gethtypes.LatestSignerForChainID(cfg.ChainID), key)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := signed.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				second, err := types.DecodeTxRLP(raw)
				if err != nil {
					t.Fatal(err)
				}
				pending := []*types.Transaction{second, first}
				builder := block.NewBlockBuilder(&cfg, nil, &mockTxPoolInteg{txs: pending})
				builder.SetState(alloc.Dup())
				parent := makeGenesisInteg(30000000, big.NewInt(1)).Header()
				var built *types.Block
				if legacy {
					built, _, err = builder.BuildBlockLegacy(parent, pending, 12, types.Address{}, nil)
				} else {
					slot := uint64(1)
					built, _, err = builder.BuildBlock(parent, &block.BuildBlockAttributes{Timestamp: 12, GasLimit: parent.GasLimit, SlotNumber: &slot})
				}
				if err != nil {
					t.Fatal(err)
				}
				txs := built.Transactions()
				if len(txs) != 2 {
					t.Fatalf("blob partition discarded an executable successor: included %d transactions", len(txs))
				}
				wantFirst := first
				if profile == "keyed" {
					wantFirst = second
				}
				if txs[0].Hash() != wantFirst.Hash() {
					t.Fatal("builder did not respect the selected replay domain")
				}
				replay := alloc.Dup()
				if _, err := execution.NewStateProcessor(&cfg).ProcessWithBAL(built, replay); err != nil {
					t.Fatal("built mixed payload does not replay", err)
				}
			})
		}
	}
}
