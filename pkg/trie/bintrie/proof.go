package bintrie

import (
	"bytes"
	"slices"

	"github.com/eth2030/eth2030/core/types"
)

// Proof contains a Merkle inclusion proof for a key in the legacy SHA-256 trie.
type Proof struct {
	Key           []byte
	Value         []byte
	Siblings      []types.Hash // root to leaf
	Stem          []byte
	LeafIndex     byte
	ValueSiblings [8]types.Hash // leaf to stem
}

// Prove constructs a proof for key. A missing key returns its key and a nil value.
func (t *BinaryTrie) Prove(key []byte) (*Proof, error) {
	if t.hashFunc != HashFunctionSHA256 {
		return nil, ErrUnsupportedProofHash
	}
	if len(key) != HashSize {
		return nil, ErrInvalidKeyLength
	}
	proof := &Proof{Key: slices.Clone(key), Stem: slices.Clone(key[:StemSize]), LeafIndex: key[StemSize]}
	value, err := t.Get(key)
	if err != nil || value == nil {
		return proof, err
	}
	siblings, node := collectVerifierSiblings(t.root, key[:StemSize], 0, t.hashFunc)
	if node == nil {
		return nil, ErrKeyNotFound
	}
	slices.Reverse(siblings)
	proof.Value, proof.Siblings = value, siblings
	proof.ValueSiblings, _ = stemValueProof(node, proof.LeafIndex, t.hashFunc)
	return proof, nil
}

// VerifyProof verifies a Merkle inclusion proof against the known SHA-256 root.
func VerifyProof(root types.Hash, proof *Proof) bool {
	if proof == nil || len(proof.Key) != HashSize || len(proof.Value) != HashSize ||
		len(proof.Siblings) > StemSize*8 || !bytes.Equal(proof.Key[:StemSize], proof.Stem) ||
		proof.Key[StemSize] != proof.LeafIndex || root == (types.Hash{}) {
		return false
	}

	current := stemProofHash(proof.Stem, proof.LeafIndex, proof.Value, proof.ValueSiblings, HashFunctionSHA256)
	for i := len(proof.Siblings) - 1; i >= 0; i-- {
		current = hashPair(current, proof.Siblings[i], getBit(proof.Stem, i))
	}
	return current == root
}

func computeStemHash(stem []byte, leafIndex byte, value []byte) types.Hash {
	return stemProofHash(stem, leafIndex, value, [8]types.Hash{}, HashFunctionSHA256)
}
