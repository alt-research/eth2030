package bintrie

import (
	"bytes"
	"slices"
	"sync"

	"github.com/eth2030/eth2030/core/types"
)

// The BLAKE3 tree shares immutable subtrees across copies. Writes replace only
// nodes along the changed path. The legacy SHA-256 tree uses mutable nodes.
func putDraftNode(node BinaryNode, key, value []byte, depth int) BinaryNode {
	switch n := node.(type) {
	case *InternalNode:
		next := *n
		if getBit(key, depth) == 0 {
			next.left = putDraftNode(n.left, key, value, depth+1)
		} else {
			next.right = putDraftNode(n.right, key, value, depth+1)
		}
		if next.left == n.left && next.right == n.right {
			return n
		}
		next.draft = new(draftNodeHash)
		return &next
	case *StemNode:
		if bytes.Equal(n.Stem, key[:StemSize]) {
			if bytes.Equal(n.Values[key[StemSize]], value) {
				return n
			}
			next := *n
			next.draft = new(draftNodeHash)
			next.Values = slices.Clone(n.Values)
			next.Values[key[StemSize]] = value
			return &next
		}

		existing := *n
		existing.depth = depth + 1
		next := &InternalNode{depth: depth, left: Empty{}, right: Empty{}, draft: new(draftNodeHash)}
		if getBit(n.Stem, depth) == 0 {
			next.left = &existing
		} else {
			next.right = &existing
		}
		if getBit(key, depth) == 0 {
			next.left = putDraftNode(next.left, key, value, depth+1)
		} else {
			next.right = putDraftNode(next.right, key, value, depth+1)
		}
		return next
	default:
		values := make([][]byte, StemNodeWidth)
		values[key[StemSize]] = value
		return &StemNode{Stem: slices.Clone(key[:StemSize]), Values: values, depth: depth, draft: new(draftNodeHash)}
	}
}

func deleteDraftNode(node BinaryNode, key []byte, depth int) BinaryNode {
	switch n := node.(type) {
	case *StemNode:
		if !bytes.Equal(n.Stem, key[:StemSize]) || n.Values[key[StemSize]] == nil {
			return n
		}
		next := *n
		next.draft = new(draftNodeHash)
		next.Values = slices.Clone(n.Values)
		next.Values[key[StemSize]] = nil
		for _, value := range next.Values {
			if value != nil {
				return &next
			}
		}
		return Empty{}
	case *InternalNode:
		next := *n
		next.draft = new(draftNodeHash)
		if getBit(key, depth) == 0 {
			next.left = deleteDraftNode(n.left, key, depth+1)
		} else {
			next.right = deleteDraftNode(n.right, key, depth+1)
		}
		leftEmpty, rightEmpty := IsEmptyNode(next.left), IsEmptyNode(next.right)
		if leftEmpty && rightEmpty {
			return Empty{}
		}
		if leftEmpty {
			if stem, ok := next.right.(*StemNode); ok {
				merged := *stem
				merged.depth = depth
				return &merged
			}
		}
		if rightEmpty {
			if stem, ok := next.left.(*StemNode); ok {
				merged := *stem
				merged.depth = depth
				return &merged
			}
		}
		return &next
	default:
		return node
	}
}

type draftNodeHash struct {
	once sync.Once
	hash types.Hash
}

func (m *draftNodeHash) get(compute func() types.Hash) types.Hash {
	if m == nil {
		return compute()
	}
	m.once.Do(func() { m.hash = compute() })
	return m.hash
}

func draftHash(node BinaryNode) types.Hash {
	switch n := node.(type) {
	case *InternalNode:
		return n.draft.get(func() types.Hash {
			left, right := draftHash(n.left), draftHash(n.right)
			return hashBLAKE3(left[:], right[:])
		})
	case *StemNode:
		return n.draft.get(n.HashBlake3)
	default:
		return types.Hash{}
	}
}
