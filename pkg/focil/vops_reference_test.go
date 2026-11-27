package focil

import (
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func referenceFrame() *types.FrameTx8141 {
	return &types.FrameTx8141{ChainID: big.NewInt(1337), Sender: types.Address{19: 1}, Fees: types.FrameFeeBudget{MaxFee: big.NewInt(100)}, Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 200000}}}, Signatures: []types.FrameSignature{{Scheme: types.FrameSignatureArbitrary}}}
}
func TestGroupEVOPSFourShapesExpiryAndNoMempoolBudgetCoupling(t *testing.T) {
	for base := 0; base < 4; base++ {
		for expiry := 0; expiry < 2; expiry++ {
			tx := referenceFrame()
			f := tx.Frames[0]
			tx.Frames = nil
			if expiry != 0 {
				data := make([]byte, 8)
				binary.BigEndian.PutUint64(data, 100)
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Target: &types.FrameExpiryVerifier, Data: data, Limits: types.FrameGasLimits{Execution: 1000}})
			}
			if base&1 != 0 {
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeDefault, Limits: types.FrameGasLimits{Execution: 100, State: 100}})
			}
			if base&2 != 0 {
				f.Flags = 2
			}
			tx.Frames = append(tx.Frames, f)
			if base&2 != 0 {
				tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 1, Target: &types.Address{19: 2}, Limits: types.FrameGasLimits{Execution: 500}})
			}
			r, _, err := FillProfile2Reference([]*types.FrameTx8141{tx}, ReferenceBudget{1000000, 1000000}, types.FrameExtensions{})
			if err != nil || !r[0].Admitted {
				t.Fatal(base, expiry, r, err)
			}
			if _, err := tx.ValidationPrefix8141(0); err == nil {
				t.Fatal("research work incorrectly tied to 100000 public admission cap")
			}
			tx.Frames = append(tx.Frames, types.Frame8141{Mode: types.ModeVerify, Flags: 3})
			r, _, err = FillProfile2Reference([]*types.FrameTx8141{tx}, ReferenceBudget{1000000, 1000000}, types.FrameExtensions{})
			if err != nil || r[0].Admitted {
				t.Fatal("trailing VERIFY admitted", r, err)
			}
		}
	}
}
func TestGroupEVOPSOccurrenceBudgetSignatureDebitAndShapeFailure(t *testing.T) {
	tx := referenceFrame()
	tx.Frames[0].Limits.Execution = 1000
	r, left, err := FillProfile2Reference([]*types.FrameTx8141{tx, tx, tx}, ReferenceBudget{2200, 1100}, types.FrameExtensions{})
	if err != nil || !r[0].Admitted || !r[1].Admitted || r[2].Admitted || left != 0 {
		t.Fatal("dedup before occurrence debit", r, left, err)
	}
	tx.Signatures = []types.FrameSignature{{Scheme: types.FrameSignatureSecp256k1, Signature: make([]byte, 65)}}
	r, left, err = FillProfile2Reference([]*types.FrameTx8141{tx}, ReferenceBudget{4000, 4000}, types.FrameExtensions{})
	if err != nil || r[0].Admitted || left != 1200 {
		t.Fatal("bad signature must keep only signature debit", r, left, err)
	}
	tx.Signatures = []types.FrameSignature{{Scheme: types.FrameSignatureArbitrary}}
	tx.Frames[0].Flags = 2
	r, left, err = FillProfile2Reference([]*types.FrameTx8141{tx}, ReferenceBudget{4000, 4000}, types.FrameExtensions{})
	if err != nil || r[0].Admitted || left != 2900 {
		t.Fatal("failed candidate refunded declared work", r, left, err)
	}
	tx.Frames[0].Flags = 3
	r, left, err = FillProfile2Reference([]*types.FrameTx8141{tx}, ReferenceBudget{1099, 1099}, types.FrameExtensions{})
	if err != nil || r[0].Admitted || left != 1099 {
		t.Fatal("does-not-fit charged budget", r, left, err)
	}
}
func TestGroupEVOPSAccessCodeAndClaimedIndexActivationGate(t *testing.T) {
	s := ReferenceSurface{Sender: types.Address{19: 1}, Payer: types.Address{19: 2}, SlotCount: 4}
	for _, a := range []types.Address{s.Sender, s.Payer} {
		for i := byte(0); i < 4; i++ {
			if !s.AllowsStorage(a, types.Hash{31: i}) {
				t.Fatal("low slot rejected")
			}
		}
	}
	for _, slot := range []types.Hash{{31: 4}, types.Hash(gethcrypto.Keccak256Hash([]byte("erc7201"))), types.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")} {
		if s.AllowsStorage(s.Sender, slot) {
			t.Fatal("external/proxy slot allowed")
		}
	}
	if s.AllowsStorage(types.Address{19: 3}, types.Hash{}) {
		t.Fatal("third-party storage allowed")
	}
	code := []byte{0}
	hash := types.Hash(gethcrypto.Keccak256Hash(code))
	if AuthenticateReferenceCode(code, hash) != nil || AuthenticateReferenceCode(nil, hash) == nil {
		t.Fatal("missing authenticated code excused")
	}
	zero, out := uint64(0), uint64(11)
	if ClaimedIndex(&zero, false, 10) != 0 || ClaimedIndex(nil, false, 10) != 10 || ClaimedIndex(&out, false, 10) != 10 || ClaimedIndex(&zero, true, 10) != 10 {
		t.Fatal("index default changed")
	}
	if !errors.Is(RequireProfile2Enforcement(), ErrProfile2NotSpecified) || len(Profile2Prerequisites()) != 13 {
		t.Fatal("incomplete informational profile activated")
	}
	prereqs := Profile2Prerequisites()
	prereqs[0] = ""
	if Profile2Prerequisites()[0] == "" {
		t.Fatal("caller can erase activation gate")
	}
}
