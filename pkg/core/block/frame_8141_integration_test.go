package block_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func signFrameBlock(t *testing.T, frame *types.FrameTx8141, keys ...int) *types.Transaction {
	t.Helper()
	frame.Signatures = make([]types.FrameSignature, len(keys))
	for i, key := range keys {
		private, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", key))
		if err != nil {
			t.Fatal(err)
		}
		signer := types.Address(gethcrypto.PubkeyToAddress(private.PublicKey))
		frame.Signatures[i] = types.FrameSignature{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}
		if signer != frame.Sender {
			frame.Signatures[i].Signer = &signer
		}
	}
	hash, err := frame.SigningHash()
	if err != nil {
		t.Fatal(err)
	}
	for i, key := range keys {
		private, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", key))
		sig, err := gethcrypto.Sign(hash[:], private)
		if err != nil {
			t.Fatal(err)
		}
		frame.Signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
	}
	encoded, err := types.EncodeFrameTx8141(frame)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := types.DecodeTxRLP(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

// Exercise the production builder, block validator, processor, canonical switch,
// disk receipt decoder and startup replay using the exact signed envelope.
func TestCanonicalFrame8141BuildImportReorgAndRestart(t *testing.T) {
	for _, blobs := range []bool{false, true} {
		t.Run(fmt.Sprintf("blob=%v", blobs), func(t *testing.T) {
			cfg := *config.TestConfig
			zero := uint64(0)
			cfg.ChainID, cfg.HogotaTime, cfg.GlamsterdanTime, cfg.AmsterdamTime = big.NewInt(1337), &zero, &zero, &zero
			senderKey, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
			payerKey, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
			sender := types.Address(gethcrypto.PubkeyToAddress(senderKey.PublicKey))
			payer := types.Address(gethcrypto.PubkeyToAddress(payerKey.PublicKey))
			target, coinbase := types.Address{19: 0x90}, types.Address{19: 0xff}
			alloc := state.NewMemoryStateDB()
			alloc.AddBalance(sender, big.NewInt(100_000_000))
			alloc.AddBalance(payer, big.NewInt(100_000_000))
			alloc.SetCode(target, []byte{0x60, 1, 0x5f, 0x55, 0x5f, 0x5f, 0xa0, 0})
			h := makeGenesisInteg(30_000_000, big.NewInt(1)).Header()
			h.Root, h.SlotNumber = alloc.GetRoot(), &zero
			genesis := types.NewBlock(h, &types.Body{Withdrawals: []*types.Withdrawal{}})
			path := t.TempDir()
			db, err := rawdb.NewFileDB(path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			bc, err := chain.NewBlockchain(&cfg, genesis, alloc.Copy(), db)
			if err != nil {
				t.Fatal(err)
			}
			frame := &types.FrameTx8141{ChainID: cfg.ChainID, Sender: sender,
				Fees: types.FrameFeeBudget{MaxFee: big.NewInt(10_000_000), MaxPriorityFeePerGas: big.NewInt(2)},
				Frames: []types.Frame8141{
					{Mode: types.ModeVerify, Flags: 2, Limits: types.FrameGasLimits{Execution: 1000}},
					{Mode: types.ModeVerify, Flags: 1, Target: &payer, Limits: types.FrameGasLimits{Execution: 4000}},
					{Mode: types.ModeSender, Target: &target, Limits: types.FrameGasLimits{Execution: 20000, State: 97920}},
				}}
			var sidecar *types.BlobSidecarData
			if blobs {
				sidecar = &types.BlobSidecarData{}
				kzg := bls.DefaultKZGBackend()
				for i := 0; i < 2; i++ {
					blob := make([]byte, types.FrameBytesPerBlob)
					blob[31] = byte(i + 1)
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
					frame.BlobVersionedHashes = append(frame.BlobVersionedHashes, types.Hash(hash))
					sidecar.Blobs = append(sidecar.Blobs, blob)
					sidecar.Commitments = append(sidecar.Commitments, commitment[:])
					for _, proof := range proofs {
						sidecar.Proofs = append(sidecar.Proofs, bytes.Clone(proof[:]))
					}
				}
			}
			tx := signFrameBlock(t, frame, 1, 2)
			tx.SetBlobSidecar(sidecar)
			builder := block.NewBlockBuilder(&cfg, bc, &mockTxPoolInteg{txs: []*types.Transaction{tx}})
			slot := uint64(1)
			attrs := &block.BuildBlockAttributes{Timestamp: 12, FeeRecipient: coinbase, GasLimit: genesis.GasLimit(), SlotNumber: &slot}
			built, receipts, err := builder.BuildBlock(genesis.Header(), attrs)
			if err != nil {
				t.Fatal(err)
			}
			if len(built.Transactions()) != 1 || len(receipts) != 1 {
				t.Fatalf("builder omitted Frame: %d txs", len(built.Transactions()))
			}
			receipt := receipts[0]
			if built.GasUsed() != 97920 || receipt.GasUsed <= built.GasUsed() || receipt.Payer != payer || len(receipt.Logs) != 1 {
				t.Fatalf("split gas, payer or logs mismatch: header=%d receipt=%+v", built.GasUsed(), receipt)
			}
			if blobs && (built.Header().BlobGasUsed == nil || *built.Header().BlobGasUsed != 2*131072) {
				t.Fatal("Frame blob gas was omitted")
			}
			// Re-execute against a second state before accepting the built payload.
			replay := alloc.Copy()
			result, err := execution.NewStateProcessor(&cfg).ProcessWithBAL(built, replay)
			if err != nil {
				t.Fatal(err)
			}
			if replay.GetRoot() != built.Root() || result.BlockAccessList.Hash() != *built.Header().BlockAccessListHash {
				t.Fatal("builder/replay state or BAL differs")
			}
			if err := bc.InsertBlock(built); err != nil {
				t.Fatalf("canonical import: %v", err)
			}
			expectedReceipt, err := receipt.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				if bc.CurrentBlock().Hash() != built.Hash() || bc.State().GetNonce(sender) != 1 || bc.State().GetState(target, types.Hash{}) != (types.Hash{31: 1}) {
					t.Fatal("canonical state was not recovered")
				}
				stored := bc.GetReceipts(built.Hash())
				if len(stored) != 1 {
					t.Fatal("stored receipt absent")
				}
				encoded, err := stored[0].EncodeRLP()
				if err != nil || !bytes.Equal(encoded, expectedReceipt) {
					t.Fatalf("receipt changed: %v", err)
				}
				expectedCharge := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), new(big.Int).Add(built.Header().BaseFee, big.NewInt(2)))
				if blobs {
					expectedCharge.Add(expectedCharge, big.NewInt(2*131072))
				}
				charge := new(big.Int).Sub(alloc.GetBalance(payer), bc.State().GetBalance(payer))
				if charge.Cmp(expectedCharge) != 0 || bc.State().GetBalance(sender).Cmp(alloc.GetBalance(sender)) != 0 {
					t.Fatalf("sponsorship settlement differs: %s / %s", charge, expectedCharge)
				}
			}
			check()
			side, _, err := block.NewBlockBuilder(&cfg, bc, nil).BuildBlock(genesis.Header(), &block.BuildBlockAttributes{Timestamp: 13, GasLimit: genesis.GasLimit(), SlotNumber: &slot})
			if err != nil {
				t.Fatal(err)
			}
			if err := bc.InsertBlock(side); err != nil {
				t.Fatal(err)
			}
			if err := bc.Reorg(side); err != nil {
				t.Fatal(err)
			}
			if bc.State().GetNonce(sender) != 0 || bc.State().GetBalance(payer).Cmp(alloc.GetBalance(payer)) != 0 {
				t.Fatal("reorg retained Frame settlement")
			}
			if err := bc.Reorg(built); err != nil {
				t.Fatal(err)
			}
			check()
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			db, err = rawdb.NewFileDB(path)
			if err != nil {
				t.Fatal(err)
			}
			bc, err = chain.NewBlockchain(&cfg, genesis, alloc.Copy(), db)
			if err != nil {
				t.Fatal(err)
			}
			if err := bc.RestoreSnapPivot(); err != nil {
				t.Fatal(err)
			}
			check()
		})
	}
}
