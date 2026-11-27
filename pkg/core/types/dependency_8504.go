package types

import (
	"bytes"
	"errors"
	"fmt"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Dependency witnesses are out of band: they do not alter canonical transaction
// bytes, transaction hashes or signature hashes. Installation clones every byte.
type dependencyWitnesses8504 struct{ witnesses [][]byte }

func cloneWitnesses8504(w [][]byte) [][]byte {
	if w == nil {
		return nil
	}
	out := make([][]byte, len(w))
	for i := range w {
		out[i] = bytes.Clone(w[i])
	}
	return out
}
func (tx *Transaction) WithDependencyWitnesses8504(w [][]byte) *Transaction {
	copy := NewTransaction(tx.inner.copy())
	if s := tx.sidecar; s != nil {
		copy.sidecar = &BlobSidecarData{Version: s.Version, Blobs: cloneWitnesses8504(s.Blobs), Commitments: cloneWitnesses8504(s.Commitments), Proofs: cloneWitnesses8504(s.Proofs)}
	}
	copy.dependencyWitnesses = &dependencyWitnesses8504{cloneWitnesses8504(w)}
	return copy
}
func (tx *Transaction) DependencyWitnesses8504() [][]byte {
	if tx.dependencyWitnesses == nil {
		return nil
	}
	return cloneWitnesses8504(tx.dependencyWitnesses.witnesses)
}
func (tx *Transaction) HasDependencies8504() bool {
	f, ok := tx.Inner().(*FrameTx8141)
	return ok && f.HasProofFrames8504()
}

// RequiresDependencyVerification8504 detects both the active block schema
// (including an empty witness list) and bare proof declarations. Execution or
// proof backends that cannot enforce the separate witness validity predicate
// must reject these blocks; successful EVM execution is insufficient.
func (b *Block) RequiresDependencyVerification8504() bool {
	if (b.header != nil && b.header.DependencyWitnessesHash != nil) || b.body.DependencyWitnesses != nil {
		return true
	}
	for _, tx := range b.body.Transactions {
		if tx.HasDependencies8504() {
			return true
		}
	}
	return false
}

type dependencyContent8504 struct {
	Dependencies [][]byte
	Witnesses    [][]byte
}
type dependencyWrapper8504 struct {
	Transactions [][]byte
	Mode         uint64
	Content      dependencyContent8504
}

// The wrapper is [ [pooled_tx_bytes], 0, [resolved_dependencies, witnesses] ].
// It uses the unchanged EIP-8141 pooled form inside the single byte string.
func EncodeDependencyWrapper8504(tx *Transaction) ([]byte, error) {
	if !tx.HasDependencies8504() {
		return nil, errors.New("wrapper requires a dependency transaction")
	}
	pooled, err := encodePooledTransactionBase(tx)
	if err != nil {
		return nil, err
	}
	deps, err := tx.Inner().(*FrameTx8141).Dependencies8504()
	if err != nil {
		return nil, err
	}
	w := tx.DependencyWitnesses8504()
	if err := ValidateDependencyWitnessShape8504(deps, w, true); err != nil {
		return nil, err
	}
	encoded := make([][]byte, len(deps))
	for i, d := range deps {
		encoded[i] = d.Bytes()
	}
	return rlp.EncodeToBytes(dependencyWrapper8504{Transactions: [][]byte{pooled}, Content: dependencyContent8504{encoded, w}})
}
func DependencyWrapperHash8504(tx *Transaction) (Hash, error) {
	raw, err := EncodeDependencyWrapper8504(tx)
	if err != nil {
		return Hash{}, err
	}
	return Hash(gethcrypto.Keccak256Hash(raw)), nil
}

const MaxDependencyWrapperBytes8504 = 2 * 1024 * 1024 // Local public transport bound.

func DecodeDependencyWrapper8504(raw []byte) (*Transaction, error) {
	if len(raw) > MaxDependencyWrapperBytes8504 {
		return nil, errors.New("dependency wrapper exceeds local transport limit")
	}
	var wrapped dependencyWrapper8504
	if err := rlp.DecodeBytes(raw, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Mode != 0 || len(wrapped.Transactions) != 1 {
		return nil, errors.New("unsupported dependency wrapper mode or transaction count")
	}
	tx, err := decodePooledTransactionBase(wrapped.Transactions[0])
	if err != nil {
		return nil, err
	}
	if !tx.HasDependencies8504() {
		return nil, errors.New("wrapper transaction has no dependencies")
	}
	deps, err := tx.Inner().(*FrameTx8141).Dependencies8504()
	if err != nil {
		return nil, err
	}
	if len(deps) != len(wrapped.Content.Dependencies) {
		return nil, errors.New("wrapper dependency count mismatch")
	}
	for i, d := range deps {
		if !bytes.Equal(d.Bytes(), wrapped.Content.Dependencies[i]) {
			return nil, errors.New("wrapper dependencies do not match canonical transaction order")
		}
	}
	if err := ValidateDependencyWitnessShape8504(deps, wrapped.Content.Witnesses, true); err != nil {
		return nil, err
	}
	return tx.WithDependencyWitnesses8504(wrapped.Content.Witnesses), nil
}

func ValidateDependencyWitnessShape8504(deps []Dependency8504, w [][]byte, public bool) error {
	if len(deps) != len(w) || (public && (len(deps) == 0 || len(deps) > MaxPublicDependencies8504)) {
		return errors.New("dependency witness count or public cap mismatch")
	}
	for i, d := range deps {
		if d.Scheme != LeanSPHINCSScheme8504 || len(w[i]) != DependencyWitnessSize8504 {
			return fmt.Errorf("invalid dependency scheme or witness length at %d", i)
		}
	}
	return nil
}

func EncodeDependencySubmission8504(tx *Transaction) ([]byte, error) {
	if !tx.HasDependencies8504() {
		return nil, errors.New("submission requires a dependency transaction")
	}
	deps, err := tx.Inner().(*FrameTx8141).Dependencies8504()
	if err != nil {
		return nil, err
	}
	w := tx.DependencyWitnesses8504()
	if err := ValidateDependencyWitnessShape8504(deps, w, true); err != nil {
		return nil, err
	}
	plain, err := tx.EncodeRLP()
	if err != nil {
		return nil, err
	}
	fields := []any{rlp.RawValue(plain[1:]), w}
	if len(tx.BlobHashes()) != 0 {
		if err := ValidateFrameSidecar8141(tx); err != nil {
			return nil, err
		}
		s := tx.BlobSidecar()
		fields = []any{rlp.RawValue(plain[1:]), uint8(1), s.Blobs, s.Commitments, s.Proofs, w}
	}
	raw, err := rlp.EncodeToBytes(fields)
	return append([]byte{FrameTxType}, raw...), err
}

func decodeDependencySubmission8504(raw []byte, fields []rlp.RawValue) (*Transaction, error) {
	if len(raw) > MaxDependencyWrapperBytes8504 {
		return nil, errors.New("dependency submission exceeds local transport limit")
	}
	var witnesses [][]byte
	if err := rlp.DecodeBytes(fields[len(fields)-1], &witnesses); err != nil {
		return nil, err
	}
	var tx *Transaction
	var err error
	if len(fields) == 2 {
		tx, err = DecodeTxRLP(append([]byte{FrameTxType}, fields[0]...))
	} else {
		pooled, e := rlp.EncodeToBytes(fields[:5])
		if e != nil {
			return nil, e
		}
		tx, err = decodePooledTransactionBase(append([]byte{FrameTxType}, pooled...))
	}
	if err != nil {
		return nil, err
	}
	if !tx.HasDependencies8504() {
		return nil, errors.New("witness submission has no dependencies")
	}
	if (len(fields) == 6) != (len(tx.BlobHashes()) != 0) {
		return nil, errors.New("dependency submission blob composition mismatch")
	}
	deps, err := tx.Inner().(*FrameTx8141).Dependencies8504()
	if err != nil {
		return nil, err
	}
	if err := ValidateDependencyWitnessShape8504(deps, witnesses, true); err != nil {
		return nil, err
	}
	return tx.WithDependencyWitnesses8504(witnesses), nil
}

func DependencyWitnessesHash8504(w [][]byte) Hash {
	raw, _ := rlp.EncodeToBytes(w)
	return Hash(gethcrypto.Keccak256Hash(raw))
}
