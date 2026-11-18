package sparse

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func signedFixture(t *testing.T) (*types.Transaction, [][]byte, [][][2048]byte) {
	t.Helper()
	backend := bls.DefaultKZGBackend()
	side := &types.BlobSidecarData{Version: 1}
	var hashes []gethcommon.Hash
	var blobs [][]byte
	var cells [][][2048]byte
	for b := 0; b < 2; b++ {
		blob := make([]byte, 131072)
		blob[31] = byte(b + 1)
		blob[200*32+31] = byte(b + 3)
		commitment, err := backend.BlobToCommitment(blob)
		if err != nil {
			t.Fatal(err)
		}
		all, proofs, err := backend.ComputeCellsAndProofs(blob)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(commitment[:])
		hash[0] = 1
		hashes = append(hashes, hash)
		side.Commitments = append(side.Commitments, bytes.Clone(commitment[:]))
		for _, p := range proofs {
			side.Proofs = append(side.Proofs, bytes.Clone(p[:]))
		}
		blobs = append(blobs, blob)
		cells = append(cells, all)
	}
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{ChainID: uint256.NewInt(1337), GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(100), Gas: 100000, To: gethcommon.Address{19: 1}, Value: uint256.NewInt(0), BlobFeeCap: uint256.NewInt(100), BlobHashes: hashes}), gethtypes.NewCancunSigner(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	native, err := types.DecodeTxRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	native.SetBlobSidecar(side)
	return native, blobs, cells
}
func TestSparseSignedCellsAtomicValidationAndReconstruction(t *testing.T) {
	tx, blobs, cells := signedFixture(t)
	validated := 0
	s := New(func(actual *types.Transaction) error {
		validated++
		if actual.Sender() == nil {
			t.Fatal("unsigned metadata reached economic validator")
		}
		return nil
	})
	if err := s.Stage(tx); err != nil {
		t.Fatal(err)
	}
	if validated != 1 {
		t.Fatal("metadata bypassed validator")
	}
	var first types.CellMask
	first[0] = 1
	good := [][]byte{cells[0][0][:], cells[1][0][:]}
	bad := [][]byte{bytes.Clone(good[0]), bytes.Clone(good[1])}
	bad[1][31] ^= 1
	if _, err := s.AddCells(tx.Hash(), first, bad); err == nil {
		t.Fatal("bad second blob accepted")
	}
	if s.Mask(tx.Hash()).OneCount() != 0 {
		t.Fatal("invalid batch partially published cells")
	}
	full, err := s.AddCells(tx.Hash(), first, good)
	if err != nil || full != nil {
		t.Fatalf("insufficient reconstruction: %v %v", full, err)
	}
	var mask types.CellMask
	for i := 0; i < 64; i++ {
		mask[(2*i)/8] |= 1 << uint((2*i)%8)
	}
	var group [][]byte
	for b := range cells {
		for _, index := range mask.Indices() {
			group = append(group, cells[b][index][:])
		}
	}
	full, err = s.AddCells(tx.Hash(), mask, group)
	if err != nil {
		t.Fatal(err)
	}
	if full == nil || full.Hash() != tx.Hash() || !types.HasFullBlobSidecar(full) {
		t.Fatal("complete reconstruction absent")
	}
	for b, blob := range blobs {
		if !bytes.Equal(full.BlobSidecar().Blobs[b], blob) {
			t.Fatalf("reconstructed blob %d differs", b)
		}
	}
	served, available := s.Cells(tx.Hash(), mask)
	if available != mask || len(served) != 128 || !bytes.Equal(served[65], cells[1][2][:]) {
		t.Fatal("blob-major ascending cell order changed")
	}
	served[0][31] ^= 1
	again, _ := s.Cells(tx.Hash(), first)
	if !bytes.Equal(again[0], good[0]) {
		t.Fatal("caller mutated staged data")
	}
}
func TestSparseRejectsMetadataAndExpiredData(t *testing.T) {
	tx, _, _ := signedFixture(t)
	for _, kind := range []string{"commitment", "proof-width", "version", "signature"} {
		t.Run(kind, func(t *testing.T) {
			copyTx, err := cloneTx(tx)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "commitment":
				copyTx.BlobSidecar().Commitments[0][0] ^= 1
			case "proof-width":
				copyTx.BlobSidecar().Proofs[255] = nil
			case "version":
				copyTx.BlobSidecar().Version = 0
			case "signature":
				inner := *copyTx.Inner().(*types.BlobTx)
				inner.R = new(big.Int)
				copyTx = types.NewTransaction(&inner)
				copyTx.SetBlobSidecar(tx.BlobSidecar())
			}
			if err := New(nil).Stage(copyTx); err == nil {
				t.Fatal("invalid metadata staged")
			}
		})
	}
}

func TestSparseExpiryDropsCellsAndAccounting(t *testing.T) {
	tx, _, _ := signedFixture(t)
	s := New(nil)
	if err := s.Stage(tx); err != nil {
		t.Fatal(err)
	}
	s.entries[tx.Hash()].expires = time.Now().Add(-time.Second)
	if s.Get(tx.Hash()) != nil || len(s.Hashes()) != 0 || s.bytes != 0 {
		t.Fatal("expired sparse metadata retained")
	}
}
