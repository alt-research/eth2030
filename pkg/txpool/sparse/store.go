// Package sparse stages authenticated EIP-8070 data outside executable pool
// contents. Incomplete entries never become block candidates.
package sparse

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const MaxEntries = 256
const MaxBytes = 64 << 20
const Lifetime = 20 * time.Minute

type entry struct {
	tx      *types.Transaction
	mask    types.CellMask
	cells   map[uint64][][]byte
	expires time.Time
	size    int
}
type Store struct {
	mu       sync.Mutex
	entries  map[types.Hash]*entry
	bytes    int
	validate func(*types.Transaction) error
}

func New(validate func(*types.Transaction) error) *Store {
	return &Store{entries: make(map[types.Hash]*entry), validate: validate}
}

func cloneTx(tx *types.Transaction) (*types.Transaction, error) {
	raw, err := tx.EncodeRLP()
	if err != nil {
		return nil, err
	}
	copyTx, err := types.DecodeTxRLP(raw)
	if err != nil {
		return nil, err
	}
	if s := tx.BlobSidecar(); s != nil {
		out := &types.BlobSidecarData{Version: s.Version}
		for _, v := range s.Commitments {
			out.Commitments = append(out.Commitments, bytes.Clone(v))
		}
		for _, v := range s.Proofs {
			out.Proofs = append(out.Proofs, bytes.Clone(v))
		}
		copyTx.SetBlobSidecar(out)
	}
	return copyTx, nil
}
func (s *Store) prune() {
	now := time.Now()
	for hash, e := range s.entries {
		if !now.Before(e.expires) {
			s.bytes -= e.size
			delete(s.entries, hash)
		}
	}
}

func (s *Store) Stage(tx *types.Transaction) error {
	if tx == nil || tx.Type() != types.BlobTxType {
		return errors.New("sparse transaction is not type 3")
	}
	tx, err := cloneTx(tx)
	if err != nil {
		return err
	}
	side := tx.BlobSidecar()
	count := len(tx.BlobHashes())
	if count == 0 || count > 6 || side == nil || side.Version != 1 || len(side.Commitments) != count || len(side.Proofs) != count*128 {
		return errors.New("invalid sparse blob metadata shape")
	}
	for i, c := range side.Commitments {
		hash := sha256.Sum256(c)
		hash[0] = 1
		if len(c) != 48 || types.Hash(hash) != tx.BlobHashes()[i] {
			return errors.New("sparse commitment hash mismatch")
		}
	}
	for _, p := range side.Proofs {
		if len(p) != 48 {
			return errors.New("invalid sparse cell proof length")
		}
	}
	raw, err := tx.EncodeRLP()
	if err != nil {
		return err
	}
	var signed gethtypes.Transaction
	if err = signed.UnmarshalBinary(raw); err != nil {
		return err
	}
	from, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(signed.ChainId()), &signed)
	if err != nil {
		return fmt.Errorf("sparse signed transaction: %w", err)
	}
	tx.SetSender(types.Address(from))
	// ValidateSignatureValues rejects high-S and malformed signatures even when
	// sender recovery would otherwise yield an address.
	v, r, t := signed.RawSignatureValues()
	if !v.IsUint64() || v.Uint64() > 1 || !gethcrypto.ValidateSignatureValues(byte(v.Uint64()), r, t, true) {
		return errors.New("invalid sparse transaction signature")
	}
	if s.validate != nil {
		if err = s.validate(tx); err != nil {
			return err
		}
	}
	size := len(raw) + count*48 + count*128*48
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	if _, ok := s.entries[tx.Hash()]; ok {
		return nil
	}
	if len(s.entries) >= MaxEntries || s.bytes+size > MaxBytes {
		return errors.New("sparse staging capacity reached")
	}
	s.entries[tx.Hash()] = &entry{tx: tx, cells: make(map[uint64][][]byte), expires: time.Now().Add(Lifetime), size: size}
	s.bytes += size
	return nil
}
func (s *Store) Get(hash types.Hash) *types.Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	e := s.entries[hash]
	if e == nil {
		return nil
	}
	tx, _ := cloneTx(e.tx)
	return tx
}
func (s *Store) Mask(hash types.Hash) types.CellMask {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	if e := s.entries[hash]; e != nil {
		return e.mask
	}
	return types.CellMask{}
}
func (s *Store) Hashes() []types.Hash {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	out := make([]types.Hash, 0, len(s.entries))
	for h := range s.entries {
		out = append(out, h)
	}
	return out
}

// AddCells validates every supplied cell before committing any bytes. The group
// is blob-major, then ascending mask index, matching the devp2p wire schema.
func (s *Store) AddCells(hash types.Hash, mask types.CellMask, cells [][]byte) (*types.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	e := s.entries[hash]
	if e == nil {
		return nil, errors.New("cells lack signed metadata")
	}
	indices := mask.Indices()
	count := len(e.tx.BlobHashes())
	if len(indices) == 0 || len(cells) != count*len(indices) {
		return nil, errors.New("sparse cell count mismatch")
	}
	var comms, proofs [][]byte
	var ids []uint64
	for b := 0; b < count; b++ {
		for i, index := range indices {
			cell := cells[b*len(indices)+i]
			if len(cell) != 2048 {
				return nil, errors.New("invalid cell length")
			}
			comms = append(comms, e.tx.BlobSidecar().Commitments[b])
			proofs = append(proofs, e.tx.BlobSidecar().Proofs[b*128+int(index)])
			ids = append(ids, index)
		}
	}
	backend := bls.DefaultKZGBackend()
	if batch, ok := backend.(interface {
		VerifyCells([][]byte, []uint64, [][]byte, [][]byte) error
	}); ok {
		if err := batch.VerifyCells(comms, ids, cells, proofs); err != nil {
			return nil, fmt.Errorf("sparse cell proof: %w", err)
		}
	} else {
		for i := range cells {
			ok, err := backend.VerifyCellProof(comms[i], cells[i], proofs[i], ids[i])
			if err != nil || !ok {
				return nil, errors.New("invalid sparse cell proof")
			}
		}
	}
	delta := len(mask.Difference(e.mask).Indices()) * count * 2048
	if s.bytes+delta > MaxBytes {
		return nil, errors.New("sparse cell capacity reached")
	}
	for i, index := range indices {
		if e.mask.IsSet(index) {
			continue
		}
		group := make([][]byte, count)
		for b := 0; b < count; b++ {
			group[b] = bytes.Clone(cells[b*len(indices)+i])
		}
		e.cells[index] = group
	}
	e.mask = e.mask.Union(mask)
	e.size += delta
	s.bytes += delta
	if e.mask.OneCount() < 64 {
		return nil, nil
	}
	recovery, ok := backend.(interface {
		RecoverCells([]uint64, [][2048]byte) ([][2048]byte, error)
	})
	if !ok {
		return nil, errors.New("KZG backend cannot reconstruct cells")
	}
	available := e.mask.Indices()[:64]
	out, err := cloneTx(e.tx)
	if err != nil {
		return nil, err
	}
	side := out.BlobSidecar()
	for b := 0; b < count; b++ {
		part := make([][2048]byte, len(available))
		for i, index := range available {
			copy(part[i][:], e.cells[index][b])
		}
		all, err := recovery.RecoverCells(available, part)
		if err != nil {
			return nil, err
		}
		if len(all) != 128 {
			return nil, errors.New("invalid recovered extension")
		}
		blob := make([]byte, 131072)
		for i := 0; i < 64; i++ {
			copy(blob[i*2048:], all[i][:])
		}
		side.Blobs = append(side.Blobs, blob)
	}
	return out, nil
}
func (s *Store) Cells(hash types.Hash, mask types.CellMask) ([][]byte, types.CellMask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	e := s.entries[hash]
	if e == nil {
		return nil, types.CellMask{}
	}
	mask = mask.Intersection(e.mask)
	var out [][]byte
	for b := range e.tx.BlobHashes() {
		for _, index := range mask.Indices() {
			out = append(out, bytes.Clone(e.cells[index][b]))
		}
	}
	return out, mask
}

// BlobCells snapshots authenticated requested cells and their proofs under the
// staging lock. Readers retain the bytes even when promotion removes an entry.
func (s *Store) BlobCells(hashes []types.Hash, mask types.CellMask) []types.BlobCells {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	result := make([]types.BlobCells, len(hashes))
	indices := mask.Indices()
	for _, entry := range s.entries {
		side := entry.tx.BlobSidecar()
		for blob, hash := range entry.tx.BlobHashes() {
			for row, wanted := range hashes {
				if hash != wanted {
					continue
				}
				if result[row].Cells == nil {
					result[row].Cells = make([][]byte, len(indices))
					result[row].Proofs = make([][]byte, len(indices))
				}
				for column, index := range indices {
					if !entry.mask.IsSet(index) {
						continue
					}
					result[row].Cells[column] = bytes.Clone(entry.cells[index][blob])
					result[row].Proofs[column] = bytes.Clone(side.Proofs[blob*128+int(index)])
				}
			}
		}
	}
	return result
}

func (s *Store) Remove(hash types.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.entries[hash]; e != nil {
		s.bytes -= e.size
		delete(s.entries, hash)
	}
}

func (s *Store) Has(hash types.Hash) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	return s.entries[hash] != nil
}
