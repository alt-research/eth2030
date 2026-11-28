package chain

import (
	"bytes"
	"testing"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
)

func TestBogotaPayloadStorageRetainsFinalizedAndPendingViews(t *testing.T) {
	dir := t.TempDir()
	open := func() *Blockchain {
		t.Helper()
		db, err := rawdb.NewFileDB(dir)
		if err != nil {
			t.Fatal(err)
		}
		return &Blockchain{db: db}
	}
	bc := open()
	older, omitted, abandoned, checkpoint, pending := types.Hash{31: 1}, types.Hash{31: 4}, types.Hash{31: 5}, types.Hash{31: 2}, types.Hash{31: 3}
	for hash, view := range map[types.Hash]*BogotaPayload{
		older:      {Number: 10, Satisfied: true, SatisfactionVersion: BogotaSatisfactionVersion, InclusionList: [][]byte{{9}}},
		omitted:    {Number: 10, Satisfied: false, InclusionList: [][]byte{{8}}},
		abandoned:  {Number: 10, Block: []byte{7}, InclusionList: [][]byte{{6}}},
		checkpoint: {Number: 11, Satisfied: false},
		pending:    {Number: 12, Block: []byte{1, 2}, InclusionList: [][]byte{{3}}},
	} {
		if err := bc.WriteBogotaPayload(hash, view); err != nil {
			t.Fatal(err)
		}
	}
	if err := bc.PruneBogotaPayloads(11); err != nil {
		t.Fatal(err)
	}
	// Repeated finalization is idempotent after moving completed records out
	// of the live-view namespace.
	if err := bc.PruneBogotaPayloads(11); err != nil {
		t.Fatal(err)
	}
	if err := bc.db.Close(); err != nil {
		t.Fatal(err)
	}
	bc = open()
	t.Cleanup(func() { _ = bc.db.Close() })
	for hash, satisfied := range map[types.Hash]bool{older: true, omitted: false} {
		if view, err := bc.ReadBogotaPayload(hash); err != nil || view == nil || view.Satisfied != satisfied || view.HasInclusionListView() || (hash == older && view.SatisfactionVersion != BogotaSatisfactionVersion) || len(view.Block) != 0 || len(view.InclusionList) != 0 {
			t.Fatalf("compacted verdict %+v, %v (want %v)", view, err, satisfied)
		}
	}
	if view, err := bc.ReadBogotaPayload(abandoned); err != nil || view != nil {
		t.Fatalf("abandoned accepted view %+v, %v", view, err)
	}
	if view, err := bc.ReadBogotaPayload(checkpoint); err != nil || view == nil || view.Satisfied || len(view.Block) != 0 {
		t.Fatalf("checkpoint verdict %+v, %v", view, err)
	}
	view, err := bc.ReadBogotaPayload(pending)
	if err != nil || view == nil || !bytes.Equal(view.Block, []byte{1, 2}) || len(view.InclusionList) != 1 ||
		!bytes.Equal(view.InclusionList[0], []byte{3}) {
		t.Fatalf("accepted payload view %+v, %v", view, err)
	}
	// A new explicit CL view must override a compact historical verdict, both
	// immediately and after that view is compacted by a later cleanup pass.
	if err := bc.WriteBogotaPayload(older, &BogotaPayload{Number: 10, Satisfied: false, SatisfactionVersion: BogotaSatisfactionVersion, InclusionList: [][]byte{{4}}}); err != nil {
		t.Fatal(err)
	}
	if view, err := bc.ReadBogotaPayload(older); err != nil || view == nil || view.Satisfied || !view.HasInclusionListView() || len(view.InclusionList) != 1 {
		t.Fatalf("replacement view lost behind compact verdict %+v, %v", view, err)
	}
	if err := bc.PruneBogotaPayloads(11); err != nil {
		t.Fatal(err)
	}
	if view, err := bc.ReadBogotaPayload(older); err != nil || view == nil || view.Satisfied || view.HasInclusionListView() || view.SatisfactionVersion != BogotaSatisfactionVersion || len(view.InclusionList) != 0 {
		t.Fatalf("replacement compact verdict %+v, %v", view, err)
	}

}
