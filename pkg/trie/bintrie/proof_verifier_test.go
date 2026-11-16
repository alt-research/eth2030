package bintrie

import (
	"crypto/sha256"
	"testing"

	"github.com/eth2030/eth2030/core/types"
)

// helper: make a 32-byte key with a distinct stem prefix.
func makeKey(prefix byte, leafIdx byte) [HashSize]byte {
	var k [HashSize]byte
	k[0] = prefix
	k[StemSize] = leafIdx
	return k
}

// helper: make a 32-byte value.
func makeVal(b byte) [HashSize]byte {
	var v [HashSize]byte
	v[0] = b
	return v
}

func TestProofVerifierInclusionSingleEntry(t *testing.T) {
	trie := New()
	key := makeKey(0x00, 1)
	val := makeVal(0xAA)
	if err := trie.Put(key[:], val[:]); err != nil {
		t.Fatal(err)
	}

	proof, err := BuildInclusionProof(trie, key[:])
	if err != nil {
		t.Fatalf("BuildInclusionProof: %v", err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])
	if !pv.VerifyInclusion(proof) {
		t.Fatal("valid inclusion proof should verify")
	}
}

func TestProofVerifierInclusionMultiLevel(t *testing.T) {
	trie := New()
	// Insert keys with different first bits to force internal nodes.
	keys := [][HashSize]byte{
		makeKey(0x00, 1), // first bit = 0
		makeKey(0x80, 2), // first bit = 1
		makeKey(0x40, 3), // second bit differs
	}
	vals := [][HashSize]byte{
		makeVal(0x11),
		makeVal(0x22),
		makeVal(0x33),
	}
	for i, k := range keys {
		if err := trie.Put(k[:], vals[i][:]); err != nil {
			t.Fatal(err)
		}
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])

	for i, k := range keys {
		proof, err := BuildInclusionProof(trie, k[:])
		if err != nil {
			t.Fatalf("BuildInclusionProof(%d): %v", i, err)
		}
		if !pv.VerifyInclusion(proof) {
			t.Fatalf("inclusion proof %d should verify", i)
		}
	}
}

func TestProofVerifierExclusionMissingKey(t *testing.T) {
	trie := New()
	existing := makeKey(0x00, 1)
	val := makeVal(0xDD)
	if err := trie.Put(existing[:], val[:]); err != nil {
		t.Fatal(err)
	}

	missing := makeKey(0x80, 1) // different stem
	proof, err := BuildExclusionProof(trie, missing[:])
	if err != nil {
		t.Fatalf("BuildExclusionProof: %v", err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])
	if !pv.VerifyExclusion(proof) {
		t.Fatal("valid exclusion proof should verify")
	}
}

func TestProofVerifierTamperedSiblingHash(t *testing.T) {
	trie := New()
	k1 := makeKey(0x00, 1)
	k2 := makeKey(0x80, 2)
	v := makeVal(0xFF)
	if err := trie.Put(k1[:], v[:]); err != nil {
		t.Fatal(err)
	}
	if err := trie.Put(k2[:], v[:]); err != nil {
		t.Fatal(err)
	}

	proof, err := BuildInclusionProof(trie, k1[:])
	if err != nil {
		t.Fatal(err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])

	// Tamper with a sibling hash (flip one bit).
	if len(proof.Path.Siblings) > 0 {
		proof.Path.Siblings[0][0] ^= 0x01
		if pv.VerifyInclusion(proof) {
			t.Fatal("tampered proof should NOT verify")
		}
	}
}

func TestProofVerifierEmptyTrie(t *testing.T) {
	trie := New()
	key := makeKey(0x42, 5)

	// Inclusion proof should fail (key not found).
	_, err := BuildInclusionProof(trie, key[:])
	if err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}

	// Exclusion proof for empty trie.
	proof, err := BuildExclusionProof(trie, key[:])
	if err != nil {
		t.Fatalf("BuildExclusionProof: %v", err)
	}
	if !proof.EmptyAtPath {
		t.Fatal("exclusion proof for empty trie should have EmptyAtPath=true")
	}
}

func TestProofVerifierSingleElementTrie(t *testing.T) {
	trie := New()
	key := makeKey(0x10, 7)
	val := makeVal(0xBB)
	if err := trie.Put(key[:], val[:]); err != nil {
		t.Fatal(err)
	}

	proof, err := BuildInclusionProof(trie, key[:])
	if err != nil {
		t.Fatal(err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])
	if !pv.VerifyInclusion(proof) {
		t.Fatal("single element inclusion proof should verify")
	}
	if proof.Path.Len() != 0 {
		t.Fatal("single element trie should have 0 siblings in path")
	}
}

func TestProofVerifierDeepPath(t *testing.T) {
	trie := New()
	// Two keys that share most bits to force deep paths.
	var k1, k2 [HashSize]byte
	k1[0] = 0x00
	k2[0] = 0x01 // differs at bit 6
	k1[StemSize] = 1
	k2[StemSize] = 2
	v := makeVal(0xCC)
	if err := trie.Put(k1[:], v[:]); err != nil {
		t.Fatal(err)
	}
	if err := trie.Put(k2[:], v[:]); err != nil {
		t.Fatal(err)
	}

	proof, err := BuildInclusionProof(trie, k1[:])
	if err != nil {
		t.Fatal(err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])
	if !pv.VerifyInclusion(proof) {
		t.Fatal("deep path inclusion proof should verify")
	}
	if proof.Path.Len() == 0 {
		t.Fatal("deep path should have siblings")
	}
}

func TestProofVerifierMultipleProofsSameRoot(t *testing.T) {
	trie := New()
	keys := make([][HashSize]byte, 5)
	for i := range keys {
		keys[i] = makeKey(byte(i*0x30), byte(i))
		v := makeVal(byte(i + 1))
		if err := trie.Put(keys[i][:], v[:]); err != nil {
			t.Fatal(err)
		}
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])

	for i, k := range keys {
		proof, err := BuildInclusionProof(trie, k[:])
		if err != nil {
			t.Fatalf("key %d: %v", i, err)
		}
		if !pv.VerifyInclusion(proof) {
			t.Fatalf("key %d: valid inclusion proof should verify", i)
		}
	}
}

func TestProofVerifierBuildAndVerifyRoundTrip(t *testing.T) {
	trie := New()
	key := makeKey(0x55, 0)
	val := makeVal(0xEE)
	if err := trie.Put(key[:], val[:]); err != nil {
		t.Fatal(err)
	}

	// Build, then verify.
	proof, err := BuildInclusionProof(trie, key[:])
	if err != nil {
		t.Fatal(err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])
	if !pv.VerifyInclusion(proof) {
		t.Fatal("round-trip proof should verify")
	}

	// Verify the proof value matches.
	if proof.Value != val {
		t.Fatal("proof value should match inserted value")
	}
}

func TestProofVerifierInvalidRootDetection(t *testing.T) {
	trie := New()
	key := makeKey(0xAB, 3)
	val := makeVal(0xCD)
	if err := trie.Put(key[:], val[:]); err != nil {
		t.Fatal(err)
	}

	proof, err := BuildInclusionProof(trie, key[:])
	if err != nil {
		t.Fatal(err)
	}

	// Use a wrong root.
	wrongRoot := sha256.Sum256([]byte("wrong root"))
	pv := NewProofVerifier(wrongRoot[:])
	if pv.VerifyInclusion(proof) {
		t.Fatal("proof with wrong root should NOT verify")
	}
}

func TestProofVerifierNilInputs(t *testing.T) {
	root := sha256.Sum256([]byte("some root"))
	pv := NewProofVerifier(root[:])

	if pv.VerifyInclusion(nil) {
		t.Fatal("nil inclusion proof should not verify")
	}
	if pv.VerifyExclusion(nil) {
		t.Fatal("nil exclusion proof should not verify")
	}
}

func TestProofVerifierEmptyRootHash(t *testing.T) {
	pv := NewProofVerifier([]byte{})
	proof := &InclusionProof{}
	if pv.VerifyInclusion(proof) {
		t.Fatal("empty root should not verify any proof")
	}

	exProof := &ExclusionProof{EmptyAtPath: true}
	if !pv.VerifyExclusion(exProof) {
		t.Fatal("empty tree should prove any key absent")
	}
	exProof.Path.Siblings = []types.Hash{{1}}
	if pv.VerifyExclusion(exProof) {
		t.Fatal("empty root accepted nonempty proof path")
	}
}

func TestProofVerifierInvalidKeyLength(t *testing.T) {
	trie := New()
	_, err := BuildInclusionProof(trie, []byte{1, 2, 3})
	if err != ErrInvalidKeyLength {
		t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
	}
	_, err = BuildExclusionProof(trie, []byte{})
	if err != ErrInvalidKeyLength {
		t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestProofVerifierExclusionExistingKeyFails(t *testing.T) {
	trie := New()
	key := makeKey(0x77, 1)
	val := makeVal(0xDD)
	if err := trie.Put(key[:], val[:]); err != nil {
		t.Fatal(err)
	}

	_, err := BuildExclusionProof(trie, key[:])
	if err != ErrKeyExists {
		t.Fatalf("expected ErrKeyExists, got %v", err)
	}
}

func TestProofVerifierExclusionWithDivergentStem(t *testing.T) {
	trie := New()
	existing := makeKey(0x00, 5)
	val := makeVal(0xAA)
	if err := trie.Put(existing[:], val[:]); err != nil {
		t.Fatal(err)
	}

	// Key with same first bit but different second bit -- goes into
	// the same internal node branch but with a different stem.
	missing := makeKey(0x40, 5)
	proof, err := BuildExclusionProof(trie, missing[:])
	if err != nil {
		t.Fatalf("BuildExclusionProof: %v", err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])

	if proof.EmptyAtPath {
		// Depending on tree depth it may be empty or divergent.
		if !pv.VerifyExclusion(proof) {
			t.Fatal("exclusion proof (empty) should verify")
		}
	} else if len(proof.DivergentStem) > 0 {
		if !pv.VerifyExclusion(proof) {
			t.Fatal("exclusion proof (divergent) should verify")
		}
	}
}

func TestProofVerifierRootAccessor(t *testing.T) {
	root := types.HexToHash("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	pv := NewProofVerifier(root[:])
	if pv.Root() != root {
		t.Fatal("Root() should return the bound root hash")
	}
}

func TestProofPathLen(t *testing.T) {
	var pp *ProofPath
	if pp.Len() != 0 {
		t.Fatal("nil ProofPath should have length 0")
	}

	pp2 := &ProofPath{Siblings: make([]types.Hash, 5)}
	if pp2.Len() != 5 {
		t.Fatalf("expected length 5, got %d", pp2.Len())
	}
}

func TestProofVerifierExclusionFakeProofReject(t *testing.T) {
	// Craft a fake exclusion proof with matching stem (should be rejected).
	trie := New()
	key := makeKey(0x10, 0)
	val := makeVal(0xBB)
	if err := trie.Put(key[:], val[:]); err != nil {
		t.Fatal(err)
	}

	root := trie.Hash()
	pv := NewProofVerifier(root[:])

	// Create a bogus exclusion proof claiming the key's own stem diverges.
	fakeProof := &ExclusionProof{
		DivergentStem:     key[:StemSize],
		DivergentStemHash: types.Hash{},
		EmptyAtPath:       false,
	}
	copy(fakeProof.Key[:], key[:])
	if pv.VerifyExclusion(fakeProof) {
		t.Fatal("exclusion proof with matching stem should be rejected")
	}
}

func TestProofVerifierMultiValueAndForgedExclusion(t *testing.T) {
	trie := New()
	first := makeKey(0x10, 1)
	second := makeKey(0x10, 200)
	firstValue, secondValue := makeVal(1), makeVal(2)
	if err := trie.Put(first[:], firstValue[:]); err != nil {
		t.Fatal(err)
	}
	if err := trie.Put(second[:], secondValue[:]); err != nil {
		t.Fatal(err)
	}
	root := trie.Hash()
	verifier := NewProofVerifier(root[:])
	proof, err := BuildInclusionProof(trie, first[:])
	if err != nil || !verifier.VerifyInclusion(proof) {
		t.Fatalf("multi-value proof failed: %v", err)
	}
	proof.ValueSiblings[0][0] ^= 1
	if verifier.VerifyInclusion(proof) {
		t.Fatal("accepted modified suffix sibling")
	}
	missing := makeKey(0x80, 1)
	exclusion, err := BuildExclusionProof(trie, missing[:])
	if err != nil || !verifier.VerifyExclusion(exclusion) {
		t.Fatalf("exclusion failed: %v", err)
	}
	// The previous verifier accepted the existing node hash with an invented stem.
	exclusion.Key = first
	exclusion.DivergentStem[0] ^= 0x20
	if verifier.VerifyExclusion(exclusion) {
		t.Fatal("accepted forged divergent stem")
	}
	sameStem := makeKey(0x10, 3)
	suffixProof, err := BuildExclusionProof(trie, sameStem[:])
	if err != nil || !verifier.VerifyExclusion(suffixProof) {
		t.Fatalf("missing suffix proof: %v", err)
	}
	suffixProof.Key = first
	if verifier.VerifyExclusion(suffixProof) {
		t.Fatal("accepted missing suffix proof for present key")
	}
}

func TestBlake3Proofs(t *testing.T) {
	trie := NewWithHashFunc(HashFunctionBlake3)
	a, b, c := makeKey(0x10, 1), makeKey(0x10, 9), makeKey(0x90, 2)
	v := makeVal(42)
	for _, key := range [][32]byte{a, b, c} {
		if err := trie.Put(key[:], v[:]); err != nil {
			t.Fatal(err)
		}
	}
	root := trie.Hash()
	verifier := NewBlake3ProofVerifier(root[:])
	inclusion, err := BuildInclusionProof(trie, a[:])
	if err != nil || !verifier.VerifyInclusion(inclusion) {
		t.Fatalf("inclusion: %v", err)
	}
	if NewProofVerifier(root[:]).VerifyInclusion(inclusion) {
		t.Fatal("SHA-256 verifier accepted BLAKE3 proof")
	}
	missing := makeKey(0x10, 3)
	suffix, err := BuildExclusionProof(trie, missing[:])
	if err != nil || !verifier.VerifyExclusion(suffix) {
		t.Fatalf("suffix exclusion: %v", err)
	}
	suffix.ValueSiblings[0][0] ^= 1
	if verifier.VerifyExclusion(suffix) {
		t.Fatal("accepted tampered suffix")
	}
	other := makeKey(0x20, 1)
	divergent, err := BuildExclusionProof(trie, other[:])
	if err != nil || !verifier.VerifyExclusion(divergent) {
		t.Fatalf("stem exclusion: %v", err)
	}
	divergent.DivergentStem[0] ^= 1
	if verifier.VerifyExclusion(divergent) {
		t.Fatal("accepted modified divergent stem")
	}
}
