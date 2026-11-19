package txpool

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
)

func validateOrdinaryBlobSidecar(tx *types.Transaction) error {
	// The current full-data pool accepts only authenticated wrappers. Sparse
	// relay must stage partial data separately before promoting a transaction.
	sidecar := tx.BlobSidecar()
	if sidecar == nil {
		return errors.New("blob sidecar is missing")
	}
	if !types.HasFullBlobSidecar(tx) {
		return errors.New("blob sidecar has invalid version, counts or lengths")
	}
	for i, commitment := range sidecar.Commitments {
		hash := sha256.Sum256(commitment)
		hash[0] = 1
		if types.Hash(hash) != tx.BlobHashes()[i] {
			return fmt.Errorf("blob commitment %d does not match transaction versioned hash", i)
		}
	}
	if sidecar.Version == 1 {
		return verifyCellProofSidecar(sidecar)
	}
	backend := bls.DefaultKZGBackend()
	for i, blob := range sidecar.Blobs {
		valid, err := backend.VerifyBlobProof(blob, sidecar.Commitments[i], sidecar.Proofs[i])
		if err != nil {
			return fmt.Errorf("blob proof %d: %w", i, err)
		}
		if !valid {
			return fmt.Errorf("blob proof %d is invalid", i)
		}
	}
	return nil
}

// Both callers check complete cell-proof shape before this shared verifier.
func verifyCellProofSidecar(sidecar *types.BlobSidecarData) error {
	backend := bls.DefaultKZGBackend()
	for i, blob := range sidecar.Blobs {
		proofs := sidecar.Proofs[i*bls.KZGCellsPerExtBlob : (i+1)*bls.KZGCellsPerExtBlob]
		if batch, ok := backend.(interface {
			VerifyBlobCellProofs([]byte, []byte, [][]byte) (bool, error)
		}); ok {
			valid, err := batch.VerifyBlobCellProofs(blob, sidecar.Commitments[i], proofs)
			if err != nil {
				return fmt.Errorf("blob cell-proof batch %d: %w", i, err)
			}
			if !valid {
				return errors.New("blob has an invalid KZG cell-proof batch")
			}
			continue
		}
		cells, err := backend.ComputeCells(blob)
		if err != nil {
			return err
		}
		if len(cells) != bls.KZGCellsPerExtBlob {
			return errors.New("invalid KZG extension cell count")
		}
		for j, cell := range cells {
			valid, err := backend.VerifyCellProof(sidecar.Commitments[i], cell[:], proofs[j], uint64(j))
			if err != nil {
				return err
			}
			if !valid {
				return errors.New("blob has an invalid KZG cell proof")
			}
		}
	}
	return nil
}

// ValidateSparseBlobMetadata applies ordinary economic and nonce checks without
// admitting a transaction or treating absent blob data as authenticated. The
// relay must separately bind commitments, recover the signed sender and verify
// cells; complete promotion still uses AddRemote and full-sidecar validation.
func (pool *TxPool) ValidateSparseBlobMetadata(tx *types.Transaction) error {
	if tx == nil || tx.Type() != types.BlobTxType {
		return errors.New("sparse metadata requires a blob transaction")
	}
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	if pool.chainID != nil && (tx.ChainId() == nil || tx.ChainId().Cmp(pool.chainID) != 0) {
		return errors.New("blob transaction chain ID mismatch")
	}
	if err := pool.validateTxData(tx, false); err != nil {
		return err
	}
	from := pool.senderOf(tx)
	nonce := pool.state.GetNonce(from)
	if tx.Nonce() < nonce {
		return ErrNonceTooLow
	}
	if tx.Nonce()-nonce > MaxNonceGap {
		return ErrNonceTooHigh
	}
	return nil
}

// BlobTransactions returns both executable and future-nonce full-data entries
// for blob relay/Engine lookup. The transaction objects are immutable after add.
func (pool *TxPool) BlobTransactions() []*types.Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	var out []*types.Transaction
	for _, set := range []map[types.Address]*txSortedList{pool.pending, pool.queue} {
		for _, list := range set {
			for _, tx := range list.items {
				if tx.Type() == types.BlobTxType {
					out = append(out, tx)
				}
			}
		}
	}
	return out
}
