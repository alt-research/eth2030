package eiptest_test

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/eiptest"
	"github.com/eth2030/eth2030/eiptest/contracts"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestLegacyFramePrototype_ApprovedFrameAndReplay(t *testing.T) {
	sender := types.Address{0x70}
	s := eiptest.New(t, eiptest.WithLegacyFrames(), eiptest.ForkAt(eiptest.ForkGlamsterdan, 0),
		eiptest.FundCode(sender, eiptest.Ether, contracts.Named{}.FrameApprove(), 0))
	pool := s.Pool()
	for seq := uint64(0); seq < 2; seq++ {
		tx := eiptest.UnsignedFrame(sender, s.Config().ChainID, new(big.Int).SetUint64(seq), 200_000,
			bigFeeCap, bigTipCap, []types.Frame{{Mode: types.ModeVerify, GasLimit: 100_000}})
		decoded := s.Encoding().RLPRoundTrip(t, tx)
		if decoded.FrameSender() != sender || len(decoded.Frames()) != 1 || decoded.Frames()[0].Mode != types.ModeVerify {
			t.Fatal("frame envelope lost its sender or VERIFY frame")
		}
		pool.ExpectAccepted(t, decoded)
		result := pool.CommitPending()
		eiptest.ExpectSuccess(t, result, 0)
		eiptest.ExpectNonce(t, s, sender, seq+1)
		pool.ExpectRejected(t, decoded, "nonce too low")
	}
}

// This frozen legacy Frame fixture verifies the sponsored
// flow end-to-end on the production block path: frame 0 (VERIFY, sender's
// contract) calls APPROVE(0) for execution approval; frame 1 (VERIFY, the
// sponsor's contract) calls APPROVE(1), which collects the total gas cost
// from the sponsor. The sender never pays; the refund routes to the payer.
func TestLegacyFramePrototype_SponsoredFrameTxBillerPath(t *testing.T) {
	sender := types.Address{0x71}
	sponsor := types.Address{0x72}
	s := eiptest.New(t, eiptest.WithLegacyFrames(), eiptest.ForkAt(eiptest.ForkGlamsterdan, 0),
		eiptest.FundCode(sender, eiptest.Wei(1), contracts.Named{}.FrameApproveExec(), 0),
		eiptest.FundCode(sponsor, eiptest.Ether, contracts.Named{}.FrameApprovePay(), 0),
	)

	sponsorBalanceBefore := s.State().GetBalance(sponsor)
	senderBalanceBefore := s.State().GetBalance(sender)

	tx := eiptest.UnsignedFrame(sender, s.Config().ChainID, new(big.Int).SetUint64(0), 300_000,
		bigFeeCap, bigTipCap, []types.Frame{
			{Mode: types.ModeVerify, GasLimit: 100_000, Target: &sender},
			{Mode: types.ModeVerify, GasLimit: 100_000, Target: &sponsor},
		})
	r := s.Commit(tx)
	eiptest.ExpectSuccess(t, r, 0)

	// The sender's nonce advanced and their balance is untouched (they never
	// pre-pay for frame transactions).
	eiptest.ExpectNonce(t, s, sender, 1)
	senderAfter := s.State().GetBalance(sender)
	if senderAfter.Cmp(senderBalanceBefore) != 0 {
		t.Fatalf("sender balance changed: %s -> %s", senderBalanceBefore, senderAfter)
	}

	// The sponsor paid the gas: their balance dropped by the final gas cost.
	sponsorAfter := s.State().GetBalance(sponsor)
	if sponsorAfter.Cmp(sponsorBalanceBefore) >= 0 {
		t.Fatalf("sponsor was not charged: %s -> %s", sponsorBalanceBefore, sponsorAfter)
	}
	charged := new(big.Int).Sub(sponsorBalanceBefore, sponsorAfter)
	recs0 := s.Chain().GetReceipts(r.Block.Hash())[0]
	expected := new(big.Int).Mul(recs0.EffectiveGasPrice, new(big.Int).SetUint64(recs0.GasUsed))
	if charged.Cmp(expected) != 0 {
		t.Fatalf("sponsor charged %s, want effectiveGasPrice*gasUsed = %s", charged, expected)
	}

	// The receipt records the sponsor as the payer per the EIP-8141 receipt
	// format [cumulativeGasUsed, payer, [frameReceipt, ...]], and carries
	// the per-frame outcomes.
	recs := s.Chain().GetReceipts(r.Block.Hash())
	if len(recs) != 1 {
		t.Fatalf("receipts = %d, want 1", len(recs))
	}
	if recs[0].Payer != sponsor {
		t.Fatalf("receipt payer = %s, want sponsor %s", recs[0].Payer.Hex(), sponsor.Hex())
	}
	if len(recs[0].FrameResults) != 2 {
		t.Fatalf("receipt frame results = %d, want 2", len(recs[0].FrameResults))
	}
}

// Exercise the current signed envelope through the actual pool, builder and
// state transition; historical prototype vectors above are separate evidence.
func TestEIP8141_CanonicalSignedFramePoolAndReplay(t *testing.T) {
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkHogota, 0))
	if s.Config().ExperimentalLegacyFrames {
		t.Fatal("canonical fixture enabled a legacy profile")
	}
	sender := s.Alice()
	pool := s.Pool()
	for nonce := uint64(0); nonce < 2; nonce++ {
		frame := &types.FrameTx8141{ChainID: s.Config().ChainID, Sender: sender.Address, Nonce: nonce,
			Fees:       types.FrameFeeBudget{MaxFee: big.NewInt(100_000_000_000_000_000), MaxPriorityFeePerGas: big.NewInt(1_000_000_000)},
			Frames:     []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}}},
			Signatures: []types.FrameSignature{{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}},
		}
		hash, err := frame.SigningHash()
		if err != nil {
			t.Fatal(err)
		}
		sig, err := gethcrypto.Sign(hash[:], sender.Key)
		if err != nil {
			t.Fatal(err)
		}
		frame.Signatures[0].Signature = append([]byte{sig[64]}, sig[:64]...)
		tx := s.Encoding().RLPRoundTrip(t, types.NewTransaction(frame))
		pool.ExpectAccepted(t, tx)
		result := pool.CommitPending()
		eiptest.ExpectSuccess(t, result, 0)
		eiptest.ExpectNonce(t, s, sender.Address, nonce+1)
		if err := pool.Submit(tx); err == nil {
			t.Fatal("canonical nonce replay accepted")
		}
	}
}
