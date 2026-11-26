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
	runFrameBuildImportReorgAndRestart(t, false)
}
func TestFrame8250BuildImportReorgAndRestart(t *testing.T) {
	runFrameBuildImportReorgAndRestart(t, true)
}
func runFrameBuildImportReorgAndRestart(t *testing.T, keyed bool) {
	for _, blobs := range []bool{false, true} {
		t.Run(fmt.Sprintf("blob=%v", blobs), func(t *testing.T) {
			cfg := *config.TestConfig
			zero := uint64(0)
			cfg.ChainID, cfg.HogotaTime, cfg.GlamsterdanTime, cfg.AmsterdamTime = big.NewInt(1337), &zero, &zero, &zero
			if keyed {
				cfg.EIP8250Time = &zero
			}
			senderKey, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
			payerKey, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 2))
			sender := types.Address(gethcrypto.PubkeyToAddress(senderKey.PublicKey))
			payer := types.Address(gethcrypto.PubkeyToAddress(payerKey.PublicKey))
			target, coinbase := types.Address{19: 0x90}, types.Address{19: 0xff}
			alloc := newBuilderRequestTestState()
			if keyed {
				alloc.SetNonce(types.NonceManager8250, 1)
				alloc.SetCode(types.NonceManager8250, types.NonceManagerCode8250)
			}
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
			if keyed {
				frame.Frames[1].Limits.State = types.KeyedNonceFirstUseStateGas8250
				frame = types.NewFrameTransaction8250(&types.FrameTx8250Delta{ChainID: frame.ChainID, Sender: sender, NonceKeys: []*big.Int{big.NewInt(42)}, Frames: frame.Frames, Fees: frame.Fees}).Inner().(*types.FrameTx8141)
			}
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
			wantState := uint64(97920)
			if keyed {
				wantState *= 2
			}
			if built.GasUsed() != wantState || receipt.GasUsed <= built.GasUsed() || receipt.Payer != payer || len(receipt.Logs) != 1 {
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
			for name, corrupt := range map[string]func(*types.Header){
				"receipt_root":   func(h *types.Header) { h.ReceiptHash[0] ^= 1 },
				"state_root":     func(h *types.Header) { h.Root[0] ^= 1 },
				"bal_commitment": func(h *types.Header) { (*h.BlockAccessListHash)[0] ^= 1 },
				"block_gas":      func(h *types.Header) { h.GasUsed++ },
				"logs_bloom":     func(h *types.Header) { h.Bloom[0] ^= 1 },
			} {
				t.Run("reject_"+name, func(t *testing.T) {
					header := built.Header()
					corrupt(header)
					if err := bc.InsertBlock(types.NewBlock(header, built.Body())); err == nil {
						t.Fatal("native import accepted corrupted Frame block")
					}
					if bc.CurrentBlock().Hash() != genesis.Hash() || bc.State().GetRoot() != alloc.GetRoot() {
						t.Fatal("rejected Frame block changed canonical state")
					}
				})
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
				wantNonce := uint64(1)
				if keyed {
					wantNonce = 0
				}
				if keyed && bc.State().GetState(types.NonceManager8250, types.NonceSlot8250(sender, big.NewInt(42))) != (types.Hash{31: 1}) {
					t.Fatal("restart/reorg lost keyed nonce")
				}
				if bc.CurrentBlock().Hash() != built.Hash() || bc.State().GetNonce(sender) != wantNonce || bc.State().GetState(target, types.Hash{}) != (types.Hash{31: 1}) {
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
			if keyed && !bc.State().GetState(types.NonceManager8250, types.NonceSlot8250(sender, big.NewInt(42))).IsZero() {
				t.Fatal("reorg retained orphaned keyed consumption")
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

func TestRecentRoot8272CanonicalForkSwitch(t *testing.T) {
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.ChainID, cfg.HogotaTime, cfg.GlamsterdanTime, cfg.AmsterdamTime = big.NewInt(1337), &zero, &zero, &zero
	key, _ := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	alloc := newBuilderRequestTestState()
	alloc.AddBalance(sender, big.NewInt(100000000))
	alloc.SetNonce(types.RecentRootAddress, 1)
	alloc.SetCode(types.RecentRootAddress, types.RecentRootCode)
	h := makeGenesisInteg(30_000_000, big.NewInt(1)).Header()
	h.Root, h.SlotNumber = alloc.GetRoot(), &zero
	genesis := types.NewBlock(h, &types.Body{Withdrawals: []*types.Withdrawal{}})
	bc, err := chain.NewBlockchain(&cfg, genesis, alloc.Copy(), rawdb.NewMemoryDB())
	if err != nil {
		t.Fatal(err)
	}
	salt, root := types.Hash{31: 3}, types.Hash{31: 4}
	frame := &types.FrameTx8141{ChainID: cfg.ChainID, Sender: sender, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(10000000), MaxPriorityFeePerGas: big.NewInt(2)}, Frames: []types.Frame8141{
		{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}},
		{Mode: types.ModeSender, Target: &types.RecentRootAddress, Data: append(append([]byte{}, salt[:]...), root[:]...), Limits: types.FrameGasLimits{Execution: 30000, State: 97920}},
	}}
	slot := uint64(1)
	built, _, err := block.NewBlockBuilder(&cfg, bc, &mockTxPoolInteg{txs: []*types.Transaction{signFrameBlock(t, frame, 1)}}).BuildBlock(genesis.Header(), &block.BuildBlockAttributes{Timestamp: 12, SlotNumber: &slot, GasLimit: genesis.GasLimit()})
	if err != nil {
		t.Fatal(err)
	}
	if err = bc.InsertBlock(built); err != nil {
		t.Fatal(err)
	}
	ref := types.RecentRootReference{Source: types.RecentRootSource(sender, salt), Slot: 1, Root: root}
	if bc.State().GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() {
		t.Fatal("canonical root write missing")
	}
	slot = 20 // Empty consensus slots do not imply execution writes or timestamp-derived slots.
	frame.Nonce = 1
	frame.Frames = []types.Frame8141{{Mode: types.ModeVerify, Target: &types.RecentRootAddress, Data: ref.AppendTo(nil), Limits: types.FrameGasLimits{Execution: 12000}}, frame.Frames[0]}
	verified, receipts, err := block.NewBlockBuilder(&cfg, bc, &mockTxPoolInteg{txs: []*types.Transaction{signFrameBlock(t, frame, 1)}}).BuildBlock(built.Header(), &block.BuildBlockAttributes{Timestamp: 24, SlotNumber: &slot, GasLimit: genesis.GasLimit()})
	if err != nil || len(receipts) != 1 {
		t.Fatal("valid root after empty slots", err)
	}
	if err = bc.InsertBlock(verified); err != nil {
		t.Fatal(err)
	}
	side, _, err := block.NewBlockBuilder(&cfg, bc, nil).BuildBlock(genesis.Header(), &block.BuildBlockAttributes{Timestamp: 25, SlotNumber: &slot, GasLimit: genesis.GasLimit()})
	if err != nil {
		t.Fatal(err)
	}
	if err = bc.InsertBlock(side); err != nil {
		t.Fatal(err)
	}
	if err = bc.Reorg(side); err != nil {
		t.Fatal(err)
	}
	if !bc.State().GetState(types.RecentRootAddress, ref.StorageKey()).IsZero() {
		t.Fatal("fork switch retained orphaned root")
	}
	frame.Nonce = 0
	slot++
	orphan, _, err := block.NewBlockBuilder(&cfg, bc, &mockTxPoolInteg{txs: []*types.Transaction{signFrameBlock(t, frame, 1)}}).BuildBlock(side.Header(), &block.BuildBlockAttributes{Timestamp: 36, SlotNumber: &slot, GasLimit: genesis.GasLimit()})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphan.Transactions()) != 0 {
		t.Fatal("builder included orphaned-root verifier")
	}
	if err = bc.Reorg(verified); err != nil {
		t.Fatal(err)
	}
	if bc.State().GetState(types.RecentRootAddress, ref.StorageKey()) != ref.EntryHash() {
		t.Fatal("fork restoration lost root state")
	}
}
