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

func TestGetBlobCellsPreservesAvailableData(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
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
	makeTx := func(nonce uint64) *types.Transaction {
		t.Helper()
		signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.BlobTx{
			ChainID: uint256.NewInt(1337), Nonce: nonce, Gas: 100000,
			GasFeeCap: uint256.NewInt(100), GasTipCap: uint256.NewInt(1),
			BlobFeeCap: uint256.NewInt(100), To: gethcommon.Address{19: 1},
			Value: new(uint256.Int), BlobHashes: []gethcommon.Hash{hash},
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
		return tx
	}
	for _, scenario := range []string{"full-and-metadata", "full-and-reconstructed-staging", "full-and-other-transaction", "sparse-union", "sparse-missing"} {
		t.Run(scenario, func(t *testing.T) {
			db := state.NewMemoryStateDB()
			db.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
			poolConfig := txpool.DefaultConfig()
			poolConfig.MinGasPrice = new(big.Int)
			pool := txpool.New(poolConfig, db)
			rules := *config.TestConfig
			rules.ChainID = big.NewInt(1337)
			pool.SetForkRules(&rules, 0)
			handler := NewHandler(newMockBlockchain(), pool, 1)
			full := makeTx(0)
			hasFull := strings.HasPrefix(scenario, "full-")
			if hasFull {
				if err := pool.AddRemote(full); err != nil {
					t.Fatal(err)
				}
			}
			staged := full
			if scenario == "full-and-other-transaction" {
				staged = makeTx(1)
			}
			if err := handler.relay.store.Stage(staged); err != nil {
				t.Fatal(err)
			}
			addCells := func(tx *types.Transaction, mask types.CellMask) *types.Transaction {
				t.Helper()
				var values [][]byte
				for _, index := range mask.Indices() {
					values = append(values, cells[index][:])
				}
				reconstructed, err := handler.relay.store.AddCells(tx.Hash(), mask, values)
				if err != nil {
					t.Fatal(err)
				}
				return reconstructed
			}
			if scenario == "full-and-reconstructed-staging" || scenario == "full-and-other-transaction" {
				var received types.CellMask
				for i := 0; i < 8; i++ {
					received[i] = 0xff
				}
				if addCells(staged, received) == nil {
					t.Fatal("fixture did not reach full reconstruction")
				}
				if handler.relay.store.Mask(staged.Hash()).IsSet(127) {
					t.Fatal("fixture unexpectedly fetched column 127")
				}
				// Hold the actual post-admission/pre-cleanup state deterministically.
			} else if !hasFull {
				if addCells(staged, types.CellMask{1}) != nil {
					t.Fatal("single cell unexpectedly reconstructed a blob")
				}
				if scenario == "sparse-union" {
					other := makeTx(1)
					if err := handler.relay.store.Stage(other); err != nil {
						t.Fatal(err)
					}
					var last types.CellMask
					last[15] = 0x80
					if addCells(other, last) != nil {
						t.Fatal("single cell unexpectedly reconstructed a blob")
					}
				}
				if pool.Get(staged.Hash()) != nil {
					t.Fatal("partial transaction entered executable pool")
				}
			}
			var requested types.CellMask
			requested[0], requested[15] = 1<<0|1<<5, 0x80
			unknown := types.Hash{0: 1, 31: 99}
			got := handler.GetBlobCells([]types.Hash{types.Hash(hash), unknown, types.Hash(hash)}, requested)
			if len(got) != 3 || got[1].Cells != nil || got[1].Proofs != nil {
				t.Fatal("unknown hash was not absent or request order changed")
			}
			for _, row := range []int{0, 2} {
				if len(got[row].Cells) != 3 || len(got[row].Proofs) != 3 {
					t.Fatalf("row %d: wrong projection dimensions", row)
				}
				for i, index := range []uint64{0, 5, 127} {
					available := hasFull || index == 0 || scenario == "sparse-union" && index == 127
					cell, proof := got[row].Cells[i], got[row].Proofs[i]
					if !available {
						if cell != nil || proof != nil {
							t.Fatalf("missing cell %d was fabricated", index)
						}
						continue
					}
					if !bytes.Equal(cell, cells[index][:]) || !bytes.Equal(proof, proofs[index][:]) {
						t.Fatalf("row %d: available cell/proof %d was overwritten by sparse metadata", row, index)
					}
					valid, err := backend.VerifyCellProof(commitment[:], cell, proof, index)
					if err != nil || !valid {
						t.Fatalf("cell %d KZG proof: %v", index, err)
					}
				}
			}
		})
	}
}
