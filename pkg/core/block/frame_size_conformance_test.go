package block

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/types"
)

func TestFrameBlockRLPByteLimitBoundary(t *testing.T) {
	cfg := *config.TestConfig
	build := func(length int) *types.Block {
		tx := types.NewTransaction(&types.FrameTx8141{ChainID: big.NewInt(1), Frames: []types.Frame8141{{Mode: types.ModeVerify, Flags: 3, Data: make([]byte, length)}}})
		return types.NewBlock(&types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 30000000}, &types.Body{Transactions: []*types.Transaction{tx}})
	}
	sample := build(MaxRLPBlockSize)
	size, err := sample.RLPEncodedSize()
	if err != nil {
		t.Fatal(err)
	}
	overhead := int(size) - MaxRLPBlockSize
	for _, delta := range []int{-1, 0, 1} {
		block := build(MaxRLPBlockSize - overhead + delta)
		size, err := block.RLPEncodedSize()
		if err != nil || int(size) != MaxRLPBlockSize+delta {
			t.Fatalf("wrong encoded boundary: %d %v", size, err)
		}
		if err := validateRLPBlockSize(&cfg, block); (err == nil) != (delta <= 0) {
			t.Fatalf("delta=%d: %v", delta, err)
		}
	}
	// This test isolates byte accounting. The intentionally huge frame is also
	// independently invalid under 8141's transaction execution/floor gas cap.
}
