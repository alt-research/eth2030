package frameproof

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/internal/testutil/proof8504"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestDependency8504IndependentSubmissionWrapperAndBlock(t *testing.T) {
	tx := proof8504.Transaction(0)
	ext := types.FrameExtensions{Proof8504: true}
	if err := VerifyTransaction(tx, ext, true); err != nil {
		t.Fatal(err)
	}
	canonical, _ := tx.EncodeRLP()
	before := tx.Hash()
	raw, err := types.EncodePooledTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	var submission []rlp.RawValue
	if err := rlp.DecodeBytes(raw[1:], &submission); err != nil || len(submission) != 2 || !bytes.Equal(submission[0], canonical[1:]) {
		t.Fatal("submission nested payload", err)
	}
	decoded, err := types.DecodePooledTransaction(raw)
	if err != nil || decoded.Hash() != before {
		t.Fatal("submission changed canonical hash", err)
	}
	wrapped, err := types.EncodeDependencyWrapper8504(decoded)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Transactions [][]byte
		Mode         uint64
		Content      struct{ Deps, Witnesses [][]byte }
	}
	if err := rlp.DecodeBytes(wrapped, &wire); err != nil || len(wire.Transactions) != 1 || !bytes.Equal(wire.Transactions[0], canonical) || wire.Mode != 0 || len(wire.Content.Deps) != 1 {
		t.Fatal("wrapper shape", err)
	}
	decoded, err = types.DecodeDependencyWrapper8504(wrapped)
	if err != nil || decoded.Hash() != before || VerifyTransaction(decoded, ext, true) != nil {
		t.Fatal("wrapper roundtrip", err)
	}
	txs := []*types.Transaction{decoded, proof8504.Transaction(1)}
	w, err := Collect(txs)
	if err != nil || len(w) != 1 {
		t.Fatal("block union shared witness", err)
	}
	hash := types.DependencyWitnessesHash8504(w)
	slot := uint64(1)
	b := types.NewBlock(&types.Header{Number: big.NewInt(1), GasLimit: 30_000_000, SlotNumber: &slot, DependencyWitnessesHash: &hash}, &types.Body{Transactions: txs, Withdrawals: []*types.Withdrawal{}, DependencyWitnesses: w})
	encoded, err := b.EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if size, err := b.RLPEncodedSize(); err != nil || size != len(encoded) {
		t.Fatal("witness omitted from block size", size, len(encoded), err)
	}
	b, err = types.DecodeBlockRLP(encoded)
	if err != nil || VerifyBlock(b, true, ext) != nil {
		t.Fatal("block roundtrip", err)
	}
	if len(b.Transactions()[0].DependencyWitnesses8504()) != 0 {
		t.Fatal("canonical block contains pooled attachments")
	}
	if err := VerifyBlock(b, false, ext); err == nil {
		t.Fatal("preactivation fields accepted")
	}
	// Mutation of returned attachments cannot invalidate a verified transaction.
	returned := tx.DependencyWitnesses8504()
	returned[0][0] ^= 1
	if err := VerifyTransaction(tx, ext, true); err != nil {
		t.Fatal("witness aliases caller")
	}
}

func TestDependency8504MissingTamperedAndWrapperControls(t *testing.T) {
	tx := proof8504.Transaction(0)
	ext := types.FrameExtensions{Proof8504: true}
	canonical, _ := tx.EncodeRLP()
	bare, _ := types.DecodeTxRLP(canonical)
	if err := VerifyTransaction(bare, ext, true); err == nil {
		t.Fatal("bare dependency admitted")
	}
	if err := VerifyTransaction(tx, types.FrameExtensions{}, true); err == nil {
		t.Fatal("proof admitted before activation")
	}
	for _, mutate := range []func(*types.FrameTx8141, [][]byte) [][]byte{
		func(_ *types.FrameTx8141, w [][]byte) [][]byte { return nil },
		func(_ *types.FrameTx8141, w [][]byte) [][]byte { return append(w, w[0]) },
		func(_ *types.FrameTx8141, w [][]byte) [][]byte { w[0] = w[0][:len(w[0])-1]; return w },
		func(_ *types.FrameTx8141, w [][]byte) [][]byte { w[0][500] ^= 1; return w },
		func(f *types.FrameTx8141, w [][]byte) [][]byte { f.Frames[1].Data[64] ^= 1; return w },
		func(f *types.FrameTx8141, w [][]byte) [][]byte { f.Frames[1].Data[32] ^= 1; return w },
		func(f *types.FrameTx8141, w [][]byte) [][]byte { f.Frames[1].Data[31] = 0x11; return w },
		func(f *types.FrameTx8141, w [][]byte) [][]byte { clear(f.Frames[1].Data[32:64]); return w },
	} {
		candidate := proof8504.Transaction(0)
		w := mutate(candidate.Inner().(*types.FrameTx8141), candidate.DependencyWitnesses8504())
		candidate = candidate.WithDependencyWitnesses8504(w)
		if err := VerifyTransaction(candidate, ext, true); err == nil {
			t.Fatal("invalid witness/dependency accepted")
		}
	}
	raw, _ := types.EncodeDependencyWrapper8504(tx)
	var fields []rlp.RawValue
	_ = rlp.DecodeBytes(raw, &fields)
	for _, bad := range [][]rlp.RawValue{fields[:2], append(append([]rlp.RawValue{}, fields...), rlp.RawValue{0x80}), {fields[0], {1}, fields[2]}, {rlp.RawValue{0xc0}, fields[1], fields[2]}} {
		encoded, _ := rlp.EncodeToBytes(bad)
		if _, err := types.DecodeDependencyWrapper8504(encoded); err == nil {
			t.Fatal("malformed wrapper admitted")
		}
	}
	d, w := proof8504.Dependency()
	c := NewCache(1)
	if err := c.Verify([]types.Dependency8504{d}, [][]byte{w}); err != nil {
		t.Fatal(err)
	}
	if len(c.entries) != 1 {
		t.Fatal("successful verification not cached")
	}
	bad := bytes.Clone(w)
	bad[0] ^= 1
	if err := c.Verify([]types.Dependency8504{d}, [][]byte{bad}); err == nil || len(c.entries) != 1 {
		t.Fatal("cache bypass or failed insertion")
	}
}

func TestDependency8504EmptyBlockCommitment(t *testing.T) {
	hash := types.DependencyWitnessesHash8504([][]byte{})
	if hash != types.HexToHash("1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347") {
		t.Fatal("empty witness commitment", hash)
	}
	h := &types.Header{GasLimit: 1, DependencyWitnessesHash: &hash}
	for _, w := range [][][]byte{nil, {}, {{1}}} {
		b := types.NewBlock(h, &types.Body{DependencyWitnesses: w})
		err := VerifyBlock(b, true, types.FrameExtensions{Proof8504: true})
		if (err == nil) != (w != nil && len(w) == 0) {
			t.Fatal("active-empty presence/count", w, err)
		}
	}
}
