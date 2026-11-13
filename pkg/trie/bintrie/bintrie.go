// Package bintrie implements the EIP-7864 binary trie for Ethereum state.
//
// The binary trie replaces MPT tries with a SHA-256-based binary
// tree structure that supports efficient stateless proofs. Each key is
// 32 bytes: the first 31 bytes form the "stem" that navigates through
// the internal nodes, and the final byte selects one of 256 leaves in
// a StemNode.
package bintrie

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

// Key encoding constants matching the go-ethereum reference.
const (
	BasicDataLeafKey        = 0
	CodeHashLeafKey         = 1
	BasicDataCodeSizeOffset = 5
	BasicDataNonceOffset    = 8
	BasicDataBalanceOffset  = 16
)

var (
	zeroHash                      = types.Hash{}
	headerStorageOffset           = new(big.Int).SetUint64(64)
	codeOffset                    = new(big.Int).SetUint64(128)
	codeStorageDelta              = new(big.Int).Sub(codeOffset, headerStorageOffset)
	nodeWidthLog2                 = 8
	mainStorageOffsetLshNodeWidth = new(big.Int).Lsh(big.NewInt(1), 248-uint(nodeWidthLog2))
)

// GetBinaryTreeKey computes the SHA-256 tree key for an address and value key.
// The key is: SHA256(zeroHash[:12] || addr || key[:31] || 0x00), with the
// last byte replaced by key[31].
func GetBinaryTreeKey(addr types.Address, key []byte) []byte {
	hasher := sha256.New()
	hasher.Write(zeroHash[:12])
	hasher.Write(addr[:])
	hasher.Write(key[:31])
	hasher.Write([]byte{0})
	k := hasher.Sum(nil)
	k[31] = key[31]
	return k
}

// GetBinaryTreeKeyBasicData returns the tree key for an account's basic data.
func GetBinaryTreeKeyBasicData(addr types.Address) []byte {
	var k [32]byte
	k[31] = BasicDataLeafKey
	return GetBinaryTreeKey(addr, k[:])
}

// GetBinaryTreeKeyCodeHash returns the tree key for an account's code hash.
func GetBinaryTreeKeyCodeHash(addr types.Address) []byte {
	var k [32]byte
	k[31] = CodeHashLeafKey
	return GetBinaryTreeKey(addr, k[:])
}

// GetBinaryTreeKeyStorageSlot returns the tree key for a storage slot.
func GetBinaryTreeKeyStorageSlot(address types.Address, key []byte) []byte {
	var k [32]byte

	// Header storage: key[:31] all zero and key[31] < 64
	if bytes.Equal(key[:31], zeroHash[:31]) && key[31] < 64 {
		k[31] = 64 + key[31]
		return GetBinaryTreeKey(address, k[:])
	}

	// Main storage offset
	k[0] = 1 // 1 << 248
	copy(k[1:], key[:31])
	k[31] = key[31]

	return GetBinaryTreeKey(address, k[:])
}

// GetBinaryTreeKeyCodeChunk returns the tree key for a code chunk.
func GetBinaryTreeKeyCodeChunk(address types.Address, chunknr uint64) []byte {
	chunkOffset := new(big.Int).Add(codeOffset, new(big.Int).SetUint64(chunknr))
	var buf [32]byte
	b := chunkOffset.Bytes()
	copy(buf[32-len(b):], b)
	return GetBinaryTreeKey(address, buf[:])
}

// StorageIndex computes the tree index and sub-index for a storage key.
func StorageIndex(storageKey []byte) (*big.Int, byte) {
	var key big.Int
	key.SetBytes(storageKey)
	if key.Cmp(codeStorageDelta) < 0 {
		key.Add(headerStorageOffset, &key)
		suffix := byte(key.Uint64() & 0xFF)
		return new(big.Int), suffix
	}
	suffix := storageKey[len(storageKey)-1]
	key.Rsh(&key, uint(nodeWidthLog2))
	key.Add(&key, mainStorageOffsetLshNodeWidth)
	return &key, suffix
}

// ChunkifyCode is defined in code_chunker.go and returns [][32]byte per EIP-7864.

// NewBinaryNode creates a new empty binary trie root node.
func NewBinaryNode() BinaryNode {
	return Empty{}
}

var errInvalidRootType = errors.New("invalid root type")

// ErrDraftStateHashFunction indicates an operation requires a BLAKE3 draft trie.
var ErrDraftStateHashFunction = errors.New("bintrie: draft state operation requires BLAKE3")

// BinaryTrie is the primary trie structure implementing EIP-7864.
type BinaryTrie struct {
	root     BinaryNode
	hashFunc string // HashFunctionSHA256 or HashFunctionBlake3
}

// New creates a new empty binary trie using SHA-256 (default).
func New() *BinaryTrie {
	return &BinaryTrie{root: NewBinaryNode(), hashFunc: HashFunctionSHA256}
}

// NewWithHashFunc creates a new empty binary trie using the specified hash function.
// Pass HashFunctionBlake3 to activate EIP-7864 BLAKE3 hashing.
func NewWithHashFunc(hashFunc string) *BinaryTrie {
	if hashFunc == "" {
		hashFunc = HashFunctionSHA256
	}
	return &BinaryTrie{root: NewBinaryNode(), hashFunc: hashFunc}
}

// UpdateLeafMetadata writes epoch into the reserved metadata slot (subindex)
// of the stem node reached by stem. Used by state expiry to record last-access
// epochs in the trie (I+ roadmap, subindex 2).
func (t *BinaryTrie) UpdateLeafMetadata(stem []byte, subindex int, epoch uint64) error {
	if len(stem) < StemSize {
		return errors.New("bintrie: stem too short")
	}
	key := make([]byte, HashSize)
	copy(key[:StemSize], stem[:StemSize])
	key[StemSize] = byte(subindex)

	// Read current stem node if it exists.
	val, err := t.root.Get(key, nil)
	if err != nil || val == nil {
		// Stem not yet present; insert a fresh metadata value.
		// We still need to insert the metadata into the trie.
	}
	_ = val

	var v [HashSize]byte
	// Store epoch as big-endian uint64 in last 8 bytes.
	v[HashSize-8] = byte(epoch >> 56)
	v[HashSize-7] = byte(epoch >> 48)
	v[HashSize-6] = byte(epoch >> 40)
	v[HashSize-5] = byte(epoch >> 32)
	v[HashSize-4] = byte(epoch >> 24)
	v[HashSize-3] = byte(epoch >> 16)
	v[HashSize-2] = byte(epoch >> 8)
	v[HashSize-1] = byte(epoch)
	return t.Put(key, v[:])
}

// Get retrieves a value by its full 32-byte key.
func (t *BinaryTrie) Get(key []byte) ([]byte, error) {
	if len(key) != HashSize {
		return nil, ErrInvalidKeyLength
	}
	value, err := t.root.Get(key, nil)
	return slices.Clone(value), err
}

// Put inserts or updates a key-value pair.
func (t *BinaryTrie) Put(key, value []byte) error {
	if len(key) != HashSize {
		return ErrInvalidKeyLength
	}
	if len(value) != HashSize {
		return ErrInvalidValueLength
	}
	root, err := t.root.Insert(key, slices.Clone(value), nil, 0)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

// Delete removes a draft BLAKE3 leaf; legacy SHA-256 trees store zero.
func (t *BinaryTrie) Delete(key []byte) error {
	if len(key) != HashSize {
		return ErrInvalidKeyLength
	}
	if t.hashFunc == HashFunctionBlake3 {
		root, err := deleteDraftNode(t.root, key, 0)
		if err != nil {
			return err
		}
		t.root = root
		return nil
	}
	var zeroVal [HashSize]byte
	root, err := t.root.Insert(key, zeroVal[:], nil, 0)
	if err != nil {
		return err
	}
	t.root = root
	return nil
}

// IsDraftBlake3 reports whether the tree uses the pinned EIP-7864 encoding.
func (t *BinaryTrie) IsDraftBlake3() bool { return t.hashFunc == HashFunctionBlake3 }

// Hash returns the root hash of the trie.
func (t *BinaryTrie) Hash() types.Hash {
	if t.hashFunc == HashFunctionBlake3 {
		return NewBinaryHasherBlake3(0).Hash(t.root)
	}
	return t.root.Hash()
}

// GetAccount returns the account information for the given address.
func (t *BinaryTrie) GetAccount(addr types.Address) (*types.Account, error) {
	var (
		values [][]byte
		err    error
		key    = t.accountKey(addr)
	)
	switch r := t.root.(type) {
	case *InternalNode:
		values, err = r.GetValuesAtStem(key[:StemSize], nil)
	case *StemNode:
		values, err = r.GetValuesAtStem(key[:StemSize], nil)
	case Empty:
		return nil, nil
	default:
		return nil, errInvalidRootType
	}
	if err != nil {
		return nil, fmt.Errorf("GetAccount (%x) error: %v", addr, err)
	}

	emptyAccount := true
	for i := 0; values != nil && i <= CodeHashLeafKey && emptyAccount; i++ {
		emptyAccount = emptyAccount && values[i] == nil
	}
	if emptyAccount {
		return nil, nil
	}

	// Check for deleted accounts (nonce=0, basic data zero, code hash zero)
	if bytes.Equal(values[BasicDataLeafKey], zero[:]) && len(values) > 10 && len(values[10]) > 0 && bytes.Equal(values[CodeHashLeafKey], zero[:]) {
		return nil, nil
	}

	if len(values[BasicDataLeafKey]) != HashSize {
		return nil, ErrInvalidValueLength
	}
	acc := &types.Account{
		Balance: new(big.Int),
	}
	acc.Nonce = binary.BigEndian.Uint64(values[BasicDataLeafKey][BasicDataNonceOffset:])
	var balance [16]byte
	copy(balance[:], values[BasicDataLeafKey][BasicDataBalanceOffset:])
	acc.Balance.SetBytes(balance[:])
	acc.CodeHash = slices.Clone(values[CodeHashLeafKey])

	return acc, nil
}

// UpdateAccount updates the account information for the given address.
func (t *BinaryTrie) UpdateAccount(addr types.Address, acc *types.Account, codeLen int) error {
	if t.hashFunc == HashFunctionBlake3 {
		if acc == nil || acc.Balance == nil || acc.Balance.Sign() < 0 || acc.Balance.BitLen() > 128 || codeLen < 0 || codeLen > 0xffffff || len(acc.CodeHash) != HashSize {
			return ErrInvalidValueLength
		}
	}
	var (
		err       error
		basicData [HashSize]byte
		values    = make([][]byte, StemNodeWidth)
		stem      = t.accountKey(addr)
	)
	binary.BigEndian.PutUint32(basicData[BasicDataCodeSizeOffset-1:], uint32(codeLen))
	binary.BigEndian.PutUint64(basicData[BasicDataNonceOffset:], acc.Nonce)

	balanceBytes := acc.Balance.Bytes()
	if len(balanceBytes) > 16 {
		balanceBytes = balanceBytes[16:]
	}
	copy(basicData[HashSize-len(balanceBytes):], balanceBytes[:])
	values[BasicDataLeafKey] = basicData[:]
	values[CodeHashLeafKey] = acc.CodeHash[:]

	if t.hashFunc == HashFunctionBlake3 {
		if len(values[CodeHashLeafKey]) != HashSize {
			return ErrInvalidValueLength
		}
		if err := t.Put(stem, values[BasicDataLeafKey]); err != nil {
			return err
		}
		codeKey := GetTreeKeyForCodeHash(addr)
		return t.Put(codeKey[:], values[CodeHashLeafKey])
	}
	t.root, err = t.root.InsertValuesAtStem(stem, values, nil, 0)
	return err
}

// GetStorage returns the value for a storage slot.
func (t *BinaryTrie) GetStorage(addr types.Address, key []byte) ([]byte, error) {
	k, err := t.storageKey(addr, key)
	if err != nil {
		return nil, err
	}
	return t.Get(k)
}

// UpdateStorage sets a storage slot value.
func (t *BinaryTrie) UpdateStorage(address types.Address, key, value []byte) error {
	k, err := t.storageKey(address, key)
	if err != nil {
		return err
	}
	if t.hashFunc == HashFunctionBlake3 && len(value) > HashSize {
		return ErrInvalidValueLength
	}
	var v [HashSize]byte
	if len(value) >= HashSize {
		copy(v[:], value[:HashSize])
	} else {
		copy(v[HashSize-len(value):], value[:])
	}
	if t.hashFunc == HashFunctionBlake3 && v == ([HashSize]byte{}) {
		return t.Delete(k)
	}
	root, err := t.root.Insert(k, v[:], nil, 0)
	if err != nil {
		return fmt.Errorf("UpdateStorage (%x) error: %w", address, err)
	}
	t.root = root
	return nil
}

// DeleteStorage removes a draft storage leaf; legacy SHA-256 trees store zero.
func (t *BinaryTrie) DeleteStorage(addr types.Address, key []byte) error {
	k, err := t.storageKey(addr, key)
	if err != nil {
		return err
	}
	if t.hashFunc == HashFunctionBlake3 {
		return t.Delete(k)
	}
	var zeroVal [HashSize]byte
	root, err := t.root.Insert(k, zeroVal[:], nil, 0)
	if err != nil {
		return fmt.Errorf("DeleteStorage (%x) error: %w", addr, err)
	}
	t.root = root
	return nil
}

// UpdateContractCode writes code chunks without updating the account header or removing old chunks.
// Use ReplaceContractCode for an existing BLAKE3 account.
func (t *BinaryTrie) UpdateContractCode(addr types.Address, code []byte) error {
	chunks := ChunkifyCode(code)
	if t.hashFunc == HashFunctionBlake3 {
		for i, chunk := range chunks {
			key := GetTreeKeyForCodeChunk(addr, uint64(i))
			if err := t.Put(key[:], chunk[:]); err != nil {
				return err
			}
		}
		return nil
	}
	var (
		values [][]byte
		key    []byte
		err    error
	)
	for chunknr, chunk := range chunks {
		chunkNr := uint64(chunknr)
		groupOffset := (chunkNr + 128) % StemNodeWidth
		if groupOffset == 0 || chunkNr == 0 {
			values = make([][]byte, StemNodeWidth)
			var offset [HashSize]byte
			binary.LittleEndian.PutUint64(offset[24:], chunkNr+128)
			key = GetBinaryTreeKey(addr, offset[:])
		}
		c := chunk // copy to avoid loop variable aliasing
		values[groupOffset] = c[:]

		if groupOffset == StemNodeWidth-1 || chunknr == len(chunks)-1 {
			err = t.UpdateStem(key[:StemSize], values)
			if err != nil {
				return fmt.Errorf("UpdateContractCode (addr=%x) error: %w", addr[:], err)
			}
		}
	}
	return nil
}

// UpdateStem updates the values for a given stem key.
func (t *BinaryTrie) UpdateStem(key []byte, values [][]byte) error {
	var err error
	t.root, err = t.root.InsertValuesAtStem(key, values, nil, 0)
	return err
}

// Copy creates a deep copy of the trie.
func (t *BinaryTrie) Copy() *BinaryTrie {
	return &BinaryTrie{root: t.root.Copy(), hashFunc: t.hashFunc}
}

// Root returns the root node (for testing and iteration).
func (t *BinaryTrie) Root() BinaryNode {
	return t.root
}

func (t *BinaryTrie) accountKey(addr types.Address) []byte {
	if t.hashFunc == HashFunctionBlake3 {
		key := GetTreeKeyForBasicData(addr)
		return key[:]
	}
	return GetBinaryTreeKey(addr, zero[:])
}
func (t *BinaryTrie) storageKey(addr types.Address, slot []byte) ([]byte, error) {
	if len(slot) != HashSize {
		return nil, ErrInvalidKeyLength
	}
	if t.hashFunc == HashFunctionBlake3 {
		key := GetTreeKeyForStorageSlot(addr, new(big.Int).SetBytes(slot))
		return key[:], nil
	}
	return GetBinaryTreeKeyStorageSlot(addr, slot), nil
}

// deleteDraftNode removes an EIP-7864 leaf and maintains minimal stem paths.
func deleteDraftNode(node BinaryNode, key []byte, depth int) (BinaryNode, error) {
	switch n := node.(type) {
	case nil, Empty:
		return Empty{}, nil
	case *StemNode:
		if !bytes.Equal(n.Stem, key[:StemSize]) || n.Values[key[StemSize]] == nil {
			return n, nil
		}
		n.Values[key[StemSize]] = nil
		for _, value := range n.Values {
			if value != nil {
				return n, nil
			}
		}
		return Empty{}, nil
	case *InternalNode:
		if depth >= StemSize*8 {
			return nil, ErrInvalidStoredNode
		}
		var err error
		if getBit(key, depth) == 0 {
			n.left, err = deleteDraftNode(n.left, key, depth+1)
		} else {
			n.right, err = deleteDraftNode(n.right, key, depth+1)
		}
		if err != nil {
			return nil, err
		}
		leftEmpty := n.left == nil || IsEmptyNode(n.left)
		rightEmpty := n.right == nil || IsEmptyNode(n.right)
		if leftEmpty && rightEmpty {
			return Empty{}, nil
		}
		if leftEmpty {
			if stem, ok := n.right.(*StemNode); ok {
				stem.depth = depth
				return stem, nil
			}
		}
		if rightEmpty {
			if stem, ok := n.left.(*StemNode); ok {
				stem.depth = depth
				return stem, nil
			}
		}
		return n, nil
	default:
		return nil, ErrInvalidStoredNode
	}
}

// ReplaceContractCode atomically updates code, code hash and code size for an
// existing BLAKE3 account and removes obsolete chunks. It does not publish a root.
func (t *BinaryTrie) ReplaceContractCode(addr types.Address, code []byte) error {
	if t.hashFunc != HashFunctionBlake3 {
		return ErrDraftStateHashFunction
	}
	if len(code) > 0xffffff {
		return ErrInvalidValueLength
	}
	account, err := t.GetAccount(addr)
	if err != nil {
		return err
	}
	if account == nil {
		return ErrKeyNotFound
	}
	basic, err := t.Get(t.accountKey(addr))
	if err != nil {
		return err
	}
	if len(basic) != HashSize {
		return ErrInvalidValueLength
	}
	oldSize := int(basic[5])<<16 | int(basic[6])<<8 | int(basic[7])
	oldChunks, newChunks := (oldSize+StemSize-1)/StemSize, (len(code)+StemSize-1)/StemSize
	next := t.Copy()
	if err := next.UpdateContractCode(addr, code); err != nil {
		return err
	}
	for i := newChunks; i < oldChunks; i++ {
		key := GetTreeKeyForCodeChunk(addr, uint64(i))
		if err := next.Delete(key[:]); err != nil {
			return err
		}
	}
	account.CodeHash = crypto.Keccak256(code)
	if err := next.UpdateAccount(addr, account, len(code)); err != nil {
		return err
	}
	t.root = next.root
	return nil
}

// GetContractCode reconstructs and authenticates an account's code chunks.
func (t *BinaryTrie) GetContractCode(addr types.Address) ([]byte, error) {
	if t.hashFunc != HashFunctionBlake3 {
		return nil, ErrDraftStateHashFunction
	}
	account, err := t.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrKeyNotFound
	}
	basic, err := t.Get(t.accountKey(addr))
	if err != nil {
		return nil, err
	}
	if len(basic) != HashSize || len(account.CodeHash) != HashSize {
		return nil, ErrInvalidValueLength
	}
	size := int(basic[5])<<16 | int(basic[6])<<8 | int(basic[7])
	result := make([]byte, 0, (size+StemSize-1)/StemSize*StemSize)
	for i := 0; i < (size+StemSize-1)/StemSize; i++ {
		key := GetTreeKeyForCodeChunk(addr, uint64(i))
		chunk, err := t.Get(key[:])
		if err != nil {
			return nil, err
		}
		if len(chunk) != HashSize {
			return nil, ErrKeyNotFound
		}
		result = append(result, chunk[1:]...)
	}
	result = result[:size]
	if !bytes.Equal(crypto.Keccak256(result), account.CodeHash) {
		return nil, ErrInvalidStoredNode
	}
	chunks := ChunkifyCode(result)
	for i, expected := range chunks {
		key := GetTreeKeyForCodeChunk(addr, uint64(i))
		chunk, err := t.Get(key[:])
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(chunk, expected[:]) {
			return nil, ErrInvalidStoredNode
		}
	}
	return result, nil
}
