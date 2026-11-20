package eth

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/txpool"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type promotionSnapshotPool struct {
	*txpool.TxPool
	afterSnapshot func()
}

func (p *promotionSnapshotPool) BlobTransactions() []*types.Transaction {
	snapshot := p.TxPool.BlobTransactions()
	if callback := p.afterSnapshot; callback != nil {
		p.afterSnapshot = nil
		callback()
	}
	return snapshot
}

// The wrapper schedules real authenticated promotion/removal between the two
// production snapshots. No client code, proof checks or expected bytes change.
func TestBlobPromotionKeepsCellsAcrossPoolSnapshot(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	db := state.NewMemoryStateDB()
	db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
	poolConfig := txpool.DefaultConfig()
	poolConfig.MinGasPrice = new(big.Int)
	nativePool := txpool.New(poolConfig, db)
	rules := *config.TestConfig
	rules.ChainID = big.NewInt(1337)
	nativePool.SetForkRules(&rules, 0)
	pool := &promotionSnapshotPool{TxPool: nativePool}
	handler := NewHandler(newMockBlockchain(), pool, 1)
	backend := bls.DefaultKZGBackend()
	blob := make([]byte, 131072)
	blob[31] = 9
	commitment, err := backend.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	cells, proofs, err := backend.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{
		ChainID: uint256.NewInt(1337), Gas: 100000, GasFeeCap: uint256.NewInt(100),
		GasTipCap: uint256.NewInt(1), BlobFeeCap: uint256.NewInt(100),
		To: gethcommon.Address{19: 1}, Value: new(uint256.Int), BlobHashes: []gethcommon.Hash{hash},
	}), gethtypes.NewCancunSigner(big.NewInt(1337)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := types.DecodeTxRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	side := &types.BlobSidecarData{Version: 1, Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
	for _, proof := range proofs {
		side.Proofs = append(side.Proofs, bytes.Clone(proof[:]))
	}
	tx.SetBlobSidecar(side)
	if err := handler.relay.store.Stage(tx); err != nil {
		t.Fatal(err)
	}
	var received types.CellMask
	var partial [][]byte
	for i := 0; i < 8; i++ {
		received[i] = 0xff
	}
	for i := 0; i < 64; i++ {
		partial = append(partial, cells[i][:])
	}
	full, err := handler.relay.store.AddCells(tx.Hash(), received, partial)
	if err != nil || full == nil {
		t.Fatalf("reconstruction fixture: %v", err)
	}
	if nativePool.Get(tx.Hash()) != nil || !handler.relay.store.Mask(tx.Hash()).IsSet(0) {
		t.Fatal("fixture not initially sparse with cell 0 available")
	}
	pool.afterSnapshot = func() {
		if err := nativePool.AddRemote(full); err != nil {
			t.Fatal(err)
		}
		handler.relay.store.Remove(tx.Hash())
	}
	got := handler.GetBlobCells([]types.Hash{types.Hash(hash)}, types.CellMask{1})
	if !types.HasFullBlobSidecar(nativePool.Get(tx.Hash())) {
		t.Fatal("promotion control did not complete")
	}
	control := handler.GetBlobCells([]types.Hash{types.Hash(hash)}, types.CellMask{1})
	if len(control[0].Cells) != 1 || !bytes.Equal(control[0].Cells[0], cells[0][:]) {
		t.Fatal("post-promotion control did not return cell 0")
	}
	valid, err := backend.VerifyCellProof(commitment[:], control[0].Cells[0], control[0].Proofs[0], 0)
	if err != nil || !valid {
		t.Fatalf("control KZG proof: %v", err)
	}
	t.Logf("handoff_cells=%d control_cells=%d full_pool_present=%t", len(got[0].Cells), len(control[0].Cells), nativePool.Get(tx.Hash()) != nil)
	if len(got[0].Cells) != 1 || !bytes.Equal(got[0].Cells[0], cells[0][:]) {
		t.Fatal("Engine lost cell 0 across promotion, although it was available in staging before and full pool after the two snapshots")
	}
}
