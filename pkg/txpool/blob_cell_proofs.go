package txpool

import "github.com/eth2030/eth2030/core/types"

// BlobAndCellProofs retains authenticated PeerDAS source data. It deliberately
// does not synthesize cell proofs from legacy blob-proof sidecars on retrieval.
type BlobAndCellProofs struct {
	Blob       []byte
	CellProofs [][]byte
}

func (pool *TxPool) GetBlobsWithCellProofs(hashes []types.Hash) []*BlobAndCellProofs {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	index := make(map[types.Hash]*BlobAndCellProofs)
	wanted := make(map[types.Hash]struct{}, len(hashes))
	for _, hash := range hashes {
		wanted[hash] = struct{}{}
	}
	for _, set := range []map[types.Address]*txSortedList{pool.pending, pool.queue} {
		for _, list := range set {
			for _, tx := range list.items {
				side := tx.BlobSidecar()
				if side == nil || (side.Version != 1 && !isCanonicalFrame8141(tx)) {
					continue
				}
				for i, hash := range tx.BlobHashes() {
					if _, requested := wanted[hash]; !requested {
						continue
					}
					if i >= len(side.Blobs) || len(side.Blobs[i]) != 131072 || (i+1)*128 > len(side.Proofs) {
						continue
					}
					if _, exists := index[hash]; !exists {
						index[hash] = &BlobAndCellProofs{Blob: side.Blobs[i], CellProofs: side.Proofs[i*128 : (i+1)*128]}
					}
				}
			}
		}
	}
	result := make([]*BlobAndCellProofs, len(hashes))
	for i, hash := range hashes {
		result[i] = index[hash]
	}
	return result
}
