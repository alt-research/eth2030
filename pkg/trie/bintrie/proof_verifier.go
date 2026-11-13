// proof_verifier.go implements Merkle proof verification for binary trie
// inclusion and exclusion proofs, path validation, and root-hash consistency
// checks. This extends the existing proof.go with structured proof types
// and a standalone verifier that does not require access to the full trie.
package bintrie

import (
	"bytes"
	"crypto/sha256"
	"errors"

	"github.com/eth2030/eth2030/core/types"
	"lukechampine.com/blake3"
)

// Proof verifier errors.
var (
	ErrNilProof             = errors.New("proof_verifier: nil proof")
	ErrInvalidKeyLength     = errors.New("proof_verifier: key must be 32 bytes")
	ErrInvalidValueLength   = errors.New("proof_verifier: value must be 32 bytes")
	ErrKeyNotFound          = errors.New("proof_verifier: key not found in trie")
	ErrKeyExists            = errors.New("proof_verifier: key exists in trie (cannot build exclusion proof)")
	ErrEmptyRoot            = errors.New("proof_verifier: empty root hash")
	ErrUnsupportedExclusion = errors.New("proof_verifier: suffix exclusion is unsupported")
	ErrUnsupportedProofHash = errors.New("proof_verifier: unsupported hash function")
	ErrProofPathEmpty       = errors.New("proof_verifier: proof path has no siblings")
)

// ProofPath is an ordered sequence of sibling hashes along a path from the
// leaf to the root in the binary trie. Index 0 is the deepest sibling
// (closest to the leaf), and the last element is closest to the root.
type ProofPath struct {
	Siblings []types.Hash
}

// Len returns the number of sibling hashes in the path.
func (pp *ProofPath) Len() int {
	if pp == nil {
		return 0
	}
	return len(pp.Siblings)
}

// InclusionProof proves that a key-value pair exists in the trie.
type InclusionProof struct {
	// Key is the full 32-byte key.
	Key [HashSize]byte
	// Value is the 32-byte leaf value.
	Value [HashSize]byte
	// Path contains the sibling hashes from leaf to root.
	Path ProofPath
	// StemHash is the hash of the StemNode containing this key.
	StemHash types.Hash
	// ValueSiblings are the eight suffix-tree siblings, leaf first.
	ValueSiblings [8]types.Hash
}

// ExclusionProof proves that a key does NOT exist in the trie.
// It works by showing the divergence point: either the trie path leads
// to an empty node, or to a StemNode with a different stem.
type ExclusionProof struct {
	// Key is the full 32-byte key being proven absent.
	Key [HashSize]byte
	// Path contains the sibling hashes from divergence point to root.
	Path ProofPath
	// DivergentStem is the stem found at the path if a different stem
	// occupies that position. Nil if the path leads to an empty node.
	DivergentStem []byte
	// DivergentStemHash is the hash of the divergent stem node, if any.
	DivergentStemHash types.Hash
	// DivergentValuesRoot binds DivergentStem to its authenticated node hash.
	DivergentValuesRoot types.Hash
	// MissingSuffix proves a nil leaf within the matching stem.
	MissingSuffix bool
	ValueSiblings [8]types.Hash
	// EmptyAtPath is true if the path ends at an empty node.
	EmptyAtPath bool
}

// ProofVerifier verifies inclusion and exclusion proofs against a root hash.
type ProofVerifier struct {
	root     types.Hash
	hashFunc string
}

// NewProofVerifier creates a new ProofVerifier bound to the given root hash.
func NewProofVerifier(root []byte) *ProofVerifier {
	var h types.Hash
	if len(root) >= HashSize {
		copy(h[:], root[:HashSize])
	} else {
		copy(h[:], root)
	}
	return &ProofVerifier{root: h, hashFunc: HashFunctionSHA256}
}

// NewBlake3ProofVerifier verifies proofs against an EIP-7864 draft BLAKE3 root.
func NewBlake3ProofVerifier(root []byte) *ProofVerifier {
	pv := NewProofVerifier(root)
	pv.hashFunc = HashFunctionBlake3
	return pv
}

// Root returns the root hash this verifier checks against.
func (pv *ProofVerifier) Root() types.Hash {
	return pv.root
}

// VerifyInclusion walks the proof path from the stem node hash back to the
// root, recomputing each level using SHA-256(left || right). It returns true
// if the recomputed root matches the verifier's root.
func (pv *ProofVerifier) VerifyInclusion(proof *InclusionProof) bool {
	if proof == nil {
		return false
	}
	if pv.root == (types.Hash{}) || len(proof.Path.Siblings) > StemSize*8 {
		return false
	}

	// Recompute the stem node hash from key and value.
	stem := proof.Key[:StemSize]
	leafIdx := proof.Key[StemSize]
	stemHash := stemProofHash(stem, leafIdx, proof.Value[:], proof.ValueSiblings, pv.hashFunc)

	// If the proof has a stored stem hash, it must match our computation.
	if proof.StemHash != (types.Hash{}) && proof.StemHash != stemHash {
		return false
	}

	// Walk the siblings from bottom (leaf-side) to top (root-side).
	current := stemHash
	for i, sib := range proof.Path.Siblings {
		depth := len(proof.Path.Siblings) - 1 - i
		bit := getBit(stem, depth)
		current = proofPair(current, sib, bit, pv.hashFunc)
	}

	return current == pv.root
}

// VerifyExclusion verifies that a key does NOT exist in the trie.
// The proof demonstrates either:
//  1. The path leads to an empty node (EmptyAtPath=true), or
//  2. The path leads to a StemNode with a different stem (DivergentStem set).
func (pv *ProofVerifier) VerifyExclusion(proof *ExclusionProof) bool {
	if proof == nil {
		return false
	}
	if pv.root == (types.Hash{}) || len(proof.Path.Siblings) > StemSize*8 {
		return false
	}

	stem := proof.Key[:StemSize]

	var current types.Hash
	if proof.MissingSuffix {
		if proof.EmptyAtPath || len(proof.DivergentStem) != 0 {
			return false
		}
		current = emptyStemProofHash(stem, proof.Key[StemSize], proof.ValueSiblings, pv.hashFunc)
	} else if proof.EmptyAtPath {
		// Empty node hashes to zero.
		current = types.Hash{}
	} else if len(proof.DivergentStem) == StemSize {
		// A different stem exists at this position.
		if bytes.Equal(proof.DivergentStem, stem) {
			// The stem matches -- this is NOT an exclusion proof.
			return false
		}
		for depth := range len(proof.Path.Siblings) {
			if getBit(stem, depth) != getBit(proof.DivergentStem, depth) {
				return false
			}
		}
		current = hashStem(proof.DivergentStem, proof.DivergentValuesRoot, pv.hashFunc)
		if current != proof.DivergentStemHash {
			return false
		}
	} else {
		return false
	}

	// Walk siblings from bottom to top.
	for i, sib := range proof.Path.Siblings {
		depth := len(proof.Path.Siblings) - 1 - i
		bit := getBit(stem, depth)
		current = proofPair(current, sib, bit, pv.hashFunc)
	}

	return current == pv.root
}

// BuildInclusionProof constructs an inclusion proof for the given key
// from a live BinaryTrie. Returns an error if the key does not exist.
func BuildInclusionProof(trie *BinaryTrie, key []byte) (*InclusionProof, error) {
	if len(key) != HashSize {
		return nil, ErrInvalidKeyLength
	}

	if trie.hashFunc != HashFunctionSHA256 && trie.hashFunc != HashFunctionBlake3 {
		return nil, ErrUnsupportedProofHash
	}

	// Verify the key exists.
	val, err := trie.Get(key)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, ErrKeyNotFound
	}

	siblings, stemNode := collectVerifierSiblings(trie.root, key[:StemSize], 0, trie.hashFunc)

	proof := &InclusionProof{
		Path: ProofPath{Siblings: siblings},
	}
	copy(proof.Key[:], key)
	if len(val) >= HashSize {
		copy(proof.Value[:], val[:HashSize])
	} else {
		copy(proof.Value[HashSize-len(val):], val)
	}
	if stemNode != nil {
		proof.StemHash = proofNodeHash(stemNode, trie.hashFunc)
		proof.ValueSiblings, _ = stemValueProof(stemNode, key[StemSize], trie.hashFunc)
	}

	return proof, nil
}

// BuildExclusionProof constructs an exclusion proof for the given key
// from a live BinaryTrie. Returns an error if the key exists.
func BuildExclusionProof(trie *BinaryTrie, key []byte) (*ExclusionProof, error) {
	if len(key) != HashSize {
		return nil, ErrInvalidKeyLength
	}

	if trie.hashFunc != HashFunctionSHA256 && trie.hashFunc != HashFunctionBlake3 {
		return nil, ErrUnsupportedProofHash
	}

	// Verify the key does NOT exist.
	val, err := trie.Get(key)
	if err != nil {
		return nil, err
	}
	if val != nil {
		return nil, ErrKeyExists
	}

	siblings, divergentStem := collectExclusionSiblings(trie.root, key[:StemSize], 0, trie.hashFunc)
	// A matching stem with an absent suffix needs a suffix-tree exclusion proof.
	_, matchingStem := collectVerifierSiblings(trie.root, key[:StemSize], 0, trie.hashFunc)

	proof := &ExclusionProof{
		Path: ProofPath{Siblings: siblings},
	}
	copy(proof.Key[:], key)

	if matchingStem != nil {
		proof.MissingSuffix = true
		proof.ValueSiblings, _ = stemValueProof(matchingStem, key[StemSize], trie.hashFunc)
	} else if divergentStem != nil {
		proof.DivergentStem = make([]byte, len(divergentStem.Stem))
		copy(proof.DivergentStem, divergentStem.Stem)
		proof.DivergentStemHash = proofNodeHash(divergentStem, trie.hashFunc)
		_, proof.DivergentValuesRoot = stemValueProof(divergentStem, 0, trie.hashFunc)
		proof.EmptyAtPath = false
	} else {
		proof.EmptyAtPath = true
	}

	return proof, nil
}

// collectVerifierSiblings walks the trie collecting sibling hashes for an
// inclusion proof. Returns siblings in leaf-to-root order and the stem node.
func collectVerifierSiblings(node BinaryNode, stem []byte, depth int, hashFunc string) ([]types.Hash, *StemNode) {
	switch n := node.(type) {
	case *InternalNode:
		bit := getBit(stem, depth)
		var sibling, child BinaryNode
		if bit == 0 {
			child = n.left
			sibling = n.right
		} else {
			child = n.right
			sibling = n.left
		}

		var sibHash types.Hash
		if sibling != nil {
			sibHash = proofNodeHash(sibling, hashFunc)
		}

		deeper, stemNode := collectVerifierSiblings(child, stem, depth+1, hashFunc)
		// Append sibling at the end (leaf-to-root order: deeper first).
		result := make([]types.Hash, len(deeper)+1)
		copy(result, deeper)
		result[len(deeper)] = sibHash
		return result, stemNode

	case *StemNode:
		if bytes.Equal(n.Stem, stem) {
			return nil, n
		}
		return nil, nil

	case Empty:
		return nil, nil

	default:
		return nil, nil
	}
}

// collectExclusionSiblings walks the trie collecting sibling hashes for an
// exclusion proof. Returns siblings in leaf-to-root order and any divergent stem.
func collectExclusionSiblings(node BinaryNode, stem []byte, depth int, hashFunc string) ([]types.Hash, *StemNode) {
	switch n := node.(type) {
	case *InternalNode:
		bit := getBit(stem, depth)
		var sibling, child BinaryNode
		if bit == 0 {
			child = n.left
			sibling = n.right
		} else {
			child = n.right
			sibling = n.left
		}

		var sibHash types.Hash
		if sibling != nil {
			sibHash = proofNodeHash(sibling, hashFunc)
		}

		deeper, divergent := collectExclusionSiblings(child, stem, depth+1, hashFunc)
		result := make([]types.Hash, len(deeper)+1)
		copy(result, deeper)
		result[len(deeper)] = sibHash
		return result, divergent

	case *StemNode:
		if !bytes.Equal(n.Stem, stem) {
			return nil, n
		}
		// Stem matches; this is not an exclusion (leaf might be nil though).
		return nil, nil

	case Empty:
		return nil, nil

	default:
		return nil, nil
	}
}

// getBit returns the bit at position depth in the stem byte slice.
// depth 0 is the MSB of stem[0].
func getBit(stem []byte, depth int) byte {
	if depth/8 >= len(stem) {
		return 0
	}
	return (stem[depth/8] >> (7 - (depth % 8))) & 1
}

// hashPair computes SHA-256(left || right) based on the direction bit.
// If bit=0, current is left and sibling is right.
// If bit=1, current is right and sibling is left.
func hashPair(current, sibling types.Hash, bit byte) types.Hash {
	h := sha256.New()
	if bit == 0 {
		h.Write(current[:])
		h.Write(sibling[:])
	} else {
		h.Write(sibling[:])
		h.Write(current[:])
	}
	return types.BytesToHash(h.Sum(nil))
}

// stemValueProof computes a suffix opening and the complete value subtree root.
func stemValueProof(node *StemNode, index byte, hashFunc string) ([8]types.Hash, types.Hash) {
	var nodes [StemNodeWidth]types.Hash
	for i, value := range node.Values {
		if value != nil {
			nodes[i] = proofLeaf(value, hashFunc)
		}
	}
	var siblings [8]types.Hash
	pos, width := int(index), StemNodeWidth
	for level := range 8 {
		siblings[level] = nodes[pos^1]
		for i := 0; i < width; i += 2 {
			nodes[i/2] = hashValuePair(nodes[i], nodes[i+1], hashFunc)
		}
		pos /= 2
		width /= 2
	}
	return siblings, nodes[0]
}
func hashValuePair(left, right types.Hash, hashFunc string) types.Hash {
	if left == (types.Hash{}) && right == (types.Hash{}) {
		return types.Hash{}
	}
	return proofPair(left, right, 0, hashFunc)
}
func hashStem(stem []byte, root types.Hash, hashFunc string) types.Hash {
	var input [64]byte
	copy(input[:31], stem)
	copy(input[32:], root[:])
	if hashFunc == HashFunctionBlake3 {
		return hashBLAKE3(input[:32], input[32:])
	}
	return sha256.Sum256(input[:])
}
func stemProofHash(stem []byte, index byte, value []byte, siblings [8]types.Hash, hashFunc string) types.Hash {
	current := proofLeaf(value, hashFunc)
	for level, sibling := range siblings {
		if (index>>level)&1 == 0 {
			current = hashValuePair(current, sibling, hashFunc)
		} else {
			current = hashValuePair(sibling, current, hashFunc)
		}
	}
	return hashStem(stem, current, hashFunc)
}

func proofLeaf(value []byte, hashFunc string) types.Hash {
	if hashFunc == HashFunctionBlake3 {
		return blake3.Sum256(value)
	}
	return sha256.Sum256(value)
}
func proofPair(current, sibling types.Hash, bit byte, hashFunc string) types.Hash {
	if hashFunc == HashFunctionBlake3 {
		if bit == 0 {
			return HashPairBlake3(current, sibling)
		}
		return HashPairBlake3(sibling, current)
	}
	return hashPair(current, sibling, bit)
}
func proofNodeHash(node BinaryNode, hashFunc string) types.Hash {
	if hashFunc == HashFunctionBlake3 {
		return NewBinaryHasherBlake3(0).Hash(node)
	}
	return node.Hash()
}

func emptyStemProofHash(stem []byte, index byte, siblings [8]types.Hash, hashFunc string) types.Hash {
	var current types.Hash
	for level, sibling := range siblings {
		if (index>>level)&1 == 0 {
			current = hashValuePair(current, sibling, hashFunc)
		} else {
			current = hashValuePair(sibling, current, hashFunc)
		}
	}
	return hashStem(stem, current, hashFunc)
}
