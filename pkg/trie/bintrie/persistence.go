package bintrie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
)

// NodeStore is the subset of a key-value database used for immutable trie nodes.
type NodeStore interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte) error
}

var ErrInvalidStoredNode = errors.New("bintrie: invalid or corrupt stored node")

func storedNodeKey(hashFunc string, hash types.Hash) []byte {
	key := append([]byte("eth2030:binary-trie:v1:"), []byte(hashFunc)...)
	key = append(key, ':')
	return append(key, hash[:]...)
}

// Persist writes immutable nodes by hash and returns the root only when all
// writes succeed. Publication of the returned root is the caller's responsibility.
// This local storage format is not an EIP-7864 wire format.
func (t *BinaryTrie) Persist(store NodeStore) (types.Hash, error) {
	if store == nil || (t.hashFunc != HashFunctionBlake3 && t.hashFunc != HashFunctionSHA256) {
		return types.Hash{}, ErrInvalidStoredNode
	}
	var write func(BinaryNode, int) (types.Hash, error)
	write = func(node BinaryNode, depth int) (types.Hash, error) {
		if node == nil {
			return types.Hash{}, nil
		}
		switch n := node.(type) {
		case Empty:
			return types.Hash{}, nil
		case *InternalNode:
			if depth >= StemSize*8 {
				return types.Hash{}, ErrInvalidStoredNode
			}
			left, err := write(n.left, depth+1)
			if err != nil {
				return types.Hash{}, err
			}
			right, err := write(n.right, depth+1)
			if err != nil {
				return types.Hash{}, err
			}
			blob := make([]byte, 65)
			blob[0] = nodeTypeInternal
			copy(blob[1:33], left[:])
			copy(blob[33:], right[:])
			h := proofPair(left, right, 0, t.hashFunc)
			if h == (types.Hash{}) {
				return types.Hash{}, ErrInvalidStoredNode
			}
			if err := store.Put(storedNodeKey(t.hashFunc, h), blob); err != nil {
				return types.Hash{}, err
			}
			return h, nil
		case *StemNode:
			if len(n.Stem) != StemSize || len(n.Values) != StemNodeWidth {
				return types.Hash{}, ErrInvalidStoredNode
			}
			for _, value := range n.Values {
				if value != nil && len(value) != HashSize {
					return types.Hash{}, ErrInvalidStoredNode
				}
			}
			h := proofNodeHash(n, t.hashFunc)
			if h == (types.Hash{}) {
				return types.Hash{}, ErrInvalidStoredNode
			}
			if err := store.Put(storedNodeKey(t.hashFunc, h), SerializeNode(n)); err != nil {
				return types.Hash{}, err
			}
			return h, nil
		default:
			return types.Hash{}, ErrInvalidStoredNode
		}
	}
	root, err := write(t.root, 0)
	if err != nil {
		return types.Hash{}, err
	}
	if root != t.Hash() {
		return types.Hash{}, ErrInvalidStoredNode
	}
	return root, nil
}

// OpenPersisted eagerly reloads and authenticates every node against root.
// It rejects malformed nodes, wrong paths, missing data and corrupt hashes.
func OpenPersisted(store NodeStore, hashFunc string, root types.Hash) (*BinaryTrie, error) {
	if store == nil || (hashFunc != HashFunctionBlake3 && hashFunc != HashFunctionSHA256) {
		return nil, ErrInvalidStoredNode
	}
	var read func(types.Hash, []byte) (BinaryNode, error)
	read = func(hash types.Hash, path []byte) (BinaryNode, error) {
		if hash == (types.Hash{}) {
			return Empty{}, nil
		}
		if len(path) > StemSize*8 {
			return nil, ErrInvalidStoredNode
		}
		blob, err := store.Get(storedNodeKey(hashFunc, hash))
		if err != nil {
			return nil, fmt.Errorf("bintrie: load node %x: %w", hash, err)
		}
		node, err := DeserializeNode(blob, len(path))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidStoredNode, err)
		}
		var actual types.Hash
		switch n := node.(type) {
		case *StemNode:
			if !bytes.Equal(SerializeNode(n), blob) {
				return nil, ErrInvalidStoredNode
			}
			actual = proofNodeHash(n, hashFunc)
			for depth, bit := range path {
				if getBit(n.Stem, depth) != bit {
					return nil, ErrInvalidStoredNode
				}
			}
		case *InternalNode:
			if len(path) >= StemSize*8 {
				return nil, ErrInvalidStoredNode
			}
			left := types.Hash(n.left.(HashedNode))
			right := types.Hash(n.right.(HashedNode))
			actual = proofPair(left, right, 0, hashFunc)
			leftPath := append(bytes.Clone(path), 0)
			rightPath := append(bytes.Clone(path), 1)
			n.left, err = read(left, leftPath)
			if err != nil {
				return nil, err
			}
			n.right, err = read(right, rightPath)
			if err != nil {
				return nil, err
			}
		default:
			return nil, ErrInvalidStoredNode
		}
		if actual != hash {
			return nil, ErrInvalidStoredNode
		}
		return node, nil
	}
	node, err := read(root, nil)
	if err != nil {
		return nil, err
	}
	return &BinaryTrie{root: node, hashFunc: hashFunc}, nil
}
