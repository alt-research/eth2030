package vm

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/proofs"
)

// This is a deliberately non-cryptographic test double that only checks
// block-binding dispatch. It does not prove EVM execution.
type testBlockFrameProver struct {
	proofs.StubValidationFrameProver
	reject bool
}

func frameTestCommitment(block *types.Block) [32]byte {
	h := sha256.New()
	header := block.Hash()
	h.Write(header[:])
	var size [8]byte
	for _, tx := range block.Transactions() {
		raw, err := tx.EncodeRLP()
		if err != nil {
			panic(err)
		}
		binary.BigEndian.PutUint64(size[:], uint64(len(raw)))
		h.Write(size[:])
		h.Write(raw)
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}
func (*testBlockFrameProver) ProveBlockValidationFrames(block *types.Block) (*proofs.STARKProofData, error) {
	return &proofs.STARKProofData{TraceCommitment: frameTestCommitment(block)}, nil
}
func (p *testBlockFrameProver) VerifyBlockValidationFrames(block *types.Block, proof *proofs.STARKProofData) bool {
	return !p.reject && proof != nil && proof.TraceCommitment == frameTestCommitment(block)
}

func TestFrameProofRequiresBlockBoundProvider(t *testing.T) {
	a := makeTestBlock([]*types.Transaction{makeFrameTx([]byte{0xaa})})
	b := makeTestBlock([]*types.Transaction{makeFrameTx([]byte{0xbb})})
	stub := &proofs.StubValidationFrameProver{}
	fakeProof, err := stub.ProveAllValidationFrames([][]byte{{0xaa}})
	if err != nil {
		t.Fatal(err)
	}
	if VerifyBlockFrameProof(a, fakeProof, stub) || VerifyBlockFrameProof(b, fakeProof, stub) {
		t.Fatal("accepted an unbound legacy proof")
	}
	if _, proof, err := ReplaceValidationFrames(a, proofs.NewSTARKValidationFrameProver()); proof != nil || err == nil {
		t.Fatal("accepted placeholder STARK prover")
	}
	bound := &testBlockFrameProver{}
	original, proof, err := ReplaceValidationFrames(a, bound)
	if err != nil || original != a || proof == nil || !VerifyBlockFrameProof(a, proof, bound) {
		t.Fatalf("block-bound dispatch: %v", err)
	}
	if VerifyBlockFrameProof(b, proof, bound) {
		t.Fatal("accepted proof from a different block")
	}
	proof.TraceCommitment[0] ^= 1
	if VerifyBlockFrameProof(a, proof, bound) {
		t.Fatal("accepted tampered proof")
	}
	bound.reject = true
	if _, proof, err := ReplaceValidationFrames(a, bound); proof != nil || err == nil {
		t.Fatal("returned proof rejected by provider")
	}
}

func TestFrameProofOnlyAppliesToVerifyFrames(t *testing.T) {
	noFrames := makeTestBlock([]*types.Transaction{makeLegacyTx()})
	if !VerifyBlockFrameProof(noFrames, nil, nil) {
		t.Fatal("no-frame block should not require proof")
	}
	tx := makeFrameTx([]byte{1})
	tx.Inner().(*types.FrameTx).Frames[0].Mode = types.ModeDefault
	defaultOnly := makeTestBlock([]*types.Transaction{tx})
	if _, proof, err := ReplaceValidationFrames(defaultOnly, nil); err != nil || proof != nil {
		t.Fatalf("default frame incorrectly required proof: %v", err)
	}
	if !VerifyBlockFrameProof(defaultOnly, nil, nil) {
		t.Fatal("default frame incorrectly required proof")
	}
	verify := makeTestBlock([]*types.Transaction{makeFrameTx([]byte{1})})
	if VerifyBlockFrameProof(verify, nil, nil) {
		t.Fatal("verify frame accepted without proof")
	}
	if VerifyBlockFrameProof(nil, nil, nil) {
		t.Fatal("nil block accepted")
	}
	invalid := makeTestBlock([]*types.Transaction{nil})
	if VerifyBlockFrameProof(invalid, nil, nil) {
		t.Fatal("nil transaction accepted")
	}
	if _, _, err := ReplaceValidationFrames(invalid, nil); err == nil {
		t.Fatal("nil transaction accepted for proving")
	}
}
