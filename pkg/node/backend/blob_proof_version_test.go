package backend

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"testing"
	"time"

	coreconfig "github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/txpool"
)

func TestPeerDASBlobQueriesDoNotUpgradeLegacyProofs(t *testing.T) {
	kzg, err := bls.NewGoEthKZGBackend()
	if err != nil {
		t.Fatal(err)
	}
	blob := make([]byte, 131072)
	blob[31] = 7
	commitment, err := kzg.BlobToCommitment(blob)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := kzg.ComputeBlobProof(blob, commitment)
	if err != nil {
		t.Fatal(err)
	}
	_, cells, err := kzg.ComputeCellsAndProofs(blob)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(commitment[:])
	hash[0] = 1
	for _, version := range []byte{0, 1} {
		node := newTestBackendNode(t)
		sender := types.Address{19: 42}
		db := state.NewMemoryStateDB()
		db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil))
		cfg := txpool.DefaultConfig()
		cfg.MinGasPrice = new(big.Int)
		node.txPool = txpool.New(cfg, db)
		rules := *coreconfig.TestConfig
		rules.OsakaTime = nil
		rules.AmsterdamTime = nil
		node.txPool.SetForkRules(&rules, 0)
		tx := types.NewTransaction(&types.BlobTx{ChainID: rules.ChainID, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(1), Gas: 100000, BlobFeeCap: big.NewInt(100), To: types.Address{19: 9}, Value: new(big.Int), BlobHashes: []types.Hash{types.Hash(hash)}})
		tx.SetSender(sender)
		side := &types.BlobSidecarData{Version: version, Blobs: [][]byte{blob}, Commitments: [][]byte{commitment[:]}}
		if version == 0 {
			side.Proofs = [][]byte{legacy[:]}
		} else {
			for _, proof := range cells {
				side.Proofs = append(side.Proofs, append([]byte(nil), proof[:]...))
			}
		}
		tx.SetBlobSidecar(side)
		if err := node.txPool.AddLocal(tx); err != nil {
			t.Fatalf("version %d setup: %v", version, err)
		}
		// Existing entries can straddle the Osaka activation. Retrieval must not
		// silently turn a legacy blob proof into a PeerDAS sidecar.
		node.txPool.SetForkRules(coreconfig.TestConfig, 0)
		b := NewEngineBackend(node)
		t.Cleanup(b.Close)
		got := b.GetBlobsV3ByVersionedHashes([]types.Hash{types.Hash(hash)})
		if len(got) != 1 {
			t.Fatalf("version %d: lost query position", version)
		}
		if version == 0 && got[0] != nil {
			t.Fatal("legacy sidecar was implicitly upgraded and served as PeerDAS")
		}
		if version == 1 && (got[0] == nil || len(got[0].Proofs) != 128) {
			t.Fatal("cell-proof sidecar unavailable")
		}
		if version == 1 {
			hashes := make([]types.Hash, 128)
			for i := range hashes {
				hashes[i] = types.Hash(hash)
			}
			start := time.Now()
			repeated := b.GetBlobsV3ByVersionedHashes(hashes)
			t.Logf("128 stored-proof responses: %s", time.Since(start))
			if len(repeated) != 128 || repeated[127] == nil || len(repeated[127].Proofs) != 128 {
				t.Fatal("128 known blobs not supported")
			}
			repeated[0].Blob[31] ^= 1
			repeated[0].Proofs[0][0] ^= 1
			reread := b.GetBlobsV3ByVersionedHashes([]types.Hash{types.Hash(hash)})
			if !bytes.Equal(repeated[127].Blob, blob) || !bytes.Equal(reread[0].Blob, blob) || !bytes.Equal(reread[0].Proofs[0], cells[0][:]) {
				t.Fatal("response mutation changed another position or authenticated source")
			}
		}

	}
}
