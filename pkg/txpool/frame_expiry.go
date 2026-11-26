package txpool

import (
	"container/heap"
	"encoding/binary"
	"github.com/eth2030/eth2030/core/types"
	"math"
)

type frameExpiryEntry struct {
	hash  types.Hash
	when  uint64
	index int
}
type frameExpiryQueue struct {
	items  []*frameExpiryEntry
	byHash map[types.Hash]*frameExpiryEntry
}

func (q *frameExpiryQueue) Len() int { return len(q.items) }
func (q *frameExpiryQueue) Less(i, j int) bool {
	a, b := q.items[i], q.items[j]
	if a.when != b.when {
		return a.when < b.when
	}
	return a.hash.Hex() < b.hash.Hex()
}
func (q *frameExpiryQueue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
	q.items[i].index = i
	q.items[j].index = j
}
func (q *frameExpiryQueue) Push(value any) {
	item := value.(*frameExpiryEntry)
	item.index = len(q.items)
	q.items = append(q.items, item)
	q.byHash[item.hash] = item
}
func (q *frameExpiryQueue) Pop() any {
	last := len(q.items) - 1
	item := q.items[last]
	q.items[last] = nil
	q.items = q.items[:last]
	delete(q.byHash, item.hash)
	item.index = -1
	return item
}
func (q *frameExpiryQueue) remove(hash types.Hash) {
	if item := q.byHash[hash]; item != nil {
		heap.Remove(q, item.index)
	}
}
func (q *frameExpiryQueue) set(hash types.Hash, when uint64) {
	if q.byHash == nil {
		q.byHash = make(map[types.Hash]*frameExpiryEntry)
	}
	if item := q.byHash[hash]; item != nil {
		if item.when != when {
			item.when = when
			heap.Fix(q, item.index)
		}
		return
	}
	heap.Push(q, &frameExpiryEntry{hash: hash, when: when})
}
func (l *txLookup) indexFrameExpiry(hash types.Hash) {
	tx := l.all[hash]
	if tx == nil {
		return
	}
	frame, ok := tx.Inner().(*types.FrameTx8141)
	if !ok {
		return
	}
	hasTime, hasSlot := false, false
	var deadline, expirySlot uint64
	logical := 0
	for _, f := range frame.Frames {
		if f.Mode == types.ModeProof8504 {
			continue
		}
		i := logical
		logical++
		if i == 0 && f.Mode == types.ModeVerify && f.Flags == 0 && f.Target != nil && *f.Target == types.FrameExpiryVerifier && len(f.Data) == 8 {
			hasTime = true
			deadline = binary.BigEndian.Uint64(f.Data)
		}
		if f.IsRecentRootVerifier() {
			refs, _ := types.DecodeRecentRootReferences(f.Data)
			for _, ref := range refs {
				// An overflowed expiration lies outside the representable slot domain.
				if ref.Slot > math.MaxUint64-types.RecentRootLength {
					continue
				}
				expiry := ref.Slot + types.RecentRootLength
				if !hasSlot || expiry < expirySlot {
					hasSlot = true
					expirySlot = expiry
				}
			}
		}
	}
	if hasTime {
		l.frameDeadlines.set(hash, deadline)
	} else {
		l.frameDeadlines.remove(hash)
	}
	if hasSlot {
		l.frameRootExpiries.set(hash, expirySlot)
	} else {
		l.frameRootExpiries.remove(hash)
	}
}
func (pool *TxPool) expireFrames8141() {
	for pool.lookup.frameDeadlines.Len() > 0 {
		item := pool.lookup.frameDeadlines.items[0]
		if item.when >= pool.frameTime {
			break
		}
		pool.removeFrameLocked(item.hash)
	}
	if pool.frameSlot != nil && *pool.frameSlot < math.MaxUint64 {
		next := *pool.frameSlot + 1
		for pool.lookup.frameRootExpiries.Len() > 0 {
			item := pool.lookup.frameRootExpiries.items[0]
			if item.when > next {
				break
			}
			pool.removeFrameLocked(item.hash)
		}
	}
}
