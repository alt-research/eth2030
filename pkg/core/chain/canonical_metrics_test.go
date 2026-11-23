package chain

import (
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/metrics"
	"testing"
)

func TestCanonicalHeightFollowsPublishedForkchoice(t *testing.T) {
	bc, st := testChain(t)
	metrics.ChainHeight.Set(0)
	blocks := makeChainBlocks(bc.Genesis(), 3, st.Copy())
	for _, block := range blocks {
		if err := bc.InsertSyncedBlock(block); err != nil {
			t.Fatal(err)
		}
	}
	if got := metrics.ChainHeight.Value(); got != 0 {
		t.Fatalf("staged head leaked into gauge: %d", got)
	}
	if err := bc.ApplyForkchoice(blocks[2].Hash(), bc.Genesis().Hash(), bc.Genesis().Hash()); err != nil {
		t.Fatal(err)
	}
	if got := metrics.ChainHeight.Value(); got != 3 {
		t.Fatalf("published head gauge = %d, want 3", got)
	}
	if err := bc.ApplyForkchoice(types.Hash{31: 99}, types.Hash{}, types.Hash{}); err == nil {
		t.Fatal("unknown head accepted")
	}
	if got := metrics.ChainHeight.Value(); got != 3 {
		t.Fatalf("failed forkchoice changed gauge: %d", got)
	}
	if err := bc.ApplyForkchoice(blocks[0].Hash(), types.Hash{}, bc.Genesis().Hash()); err != nil {
		t.Fatal(err)
	}
	if got := metrics.ChainHeight.Value(); got != 1 {
		t.Fatalf("rewound head gauge = %d, want 1", got)
	}
}

func TestCanonicalHeightRestoredFromIndexedCheckpoint(t *testing.T) {
	reopened, original, _ := indexedSnapRestart(t, "")
	metrics.ChainHeight.Set(0)
	if err := reopened.RestoreSnapPivot(); err != nil {
		t.Fatal(err)
	}
	if got, want := metrics.ChainHeight.Value(), int64(original.CurrentBlock().NumberU64()); got != want {
		t.Fatalf("restored head gauge = %d, want %d", got, want)
	}
}

func TestCanonicalHeightLegacyRewind(t *testing.T) {
	bc, st := testChain(t)
	blocks := makeChainBlocks(bc.Genesis(), 3, st.Copy())
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	if err := bc.SetHead(1); err != nil {
		t.Fatal(err)
	}
	if got := metrics.ChainHeight.Value(); got != 1 {
		t.Fatalf("legacy rewind gauge = %d", got)
	}
}
