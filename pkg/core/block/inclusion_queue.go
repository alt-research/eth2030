package block

import (
	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

type buildCandidate struct {
	tx        *types.Transaction
	inclusion bool
	queued    bool
	included  bool
}

// inclusionQueue wakes omitted Profile 1 candidates only when the sender's
// actual balance/nonce/code changes, and only at its current nonce. Reverse
// nonce chains do not repeatedly execute all future transactions. Full-state
// experimental Frame candidates have dynamic dependencies and are retried after
// a successful state transition; they do not represent AA-VOPS enforcement.
type inclusionQueue struct {
	ready    []*buildCandidate
	position int
	waiting  map[types.Address]map[uint64][]*buildCandidate
	dynamic  []*buildCandidate
	all      []*buildCandidate
}

func newInclusionQueue(raw [][]byte, pending []*types.Transaction, pool TxPoolReader) *inclusionQueue {
	q := &inclusionQueue{waiting: make(map[types.Address]map[uint64][]*buildCandidate)}
	seen := make(map[types.Hash]bool)
	add := func(tx *types.Transaction, il bool) {
		if tx == nil || seen[tx.Hash()] {
			return
		}
		seen[tx.Hash()] = true
		c := &buildCandidate{tx: tx, inclusion: il}
		q.all = append(q.all, c)
		q.enqueue(c)
	}
	for _, data := range raw {
		tx, err := types.DecodeBlockTransaction(data)
		if err != nil || tx.Type() == types.BlobTxType {
			continue
		}
		if tx.HasDependencies8504() || len(tx.BlobHashes()) > 0 {
			if source, ok := pool.(interface {
				Get(types.Hash) *types.Transaction
			}); ok {
				if wrapped := source.Get(tx.Hash()); wrapped != nil {
					tx = wrapped
				}
			}
		}
		add(tx, true)
	}
	for _, tx := range pending {
		add(tx, false)
	}
	return q
}
func (q *inclusionQueue) enqueue(c *buildCandidate) {
	if c.included || c.queued {
		return
	}
	c.queued = true
	q.ready = append(q.ready, c)
}
func (q *inclusionQueue) next() (*buildCandidate, bool) {
	for q.position < len(q.ready) {
		c := q.ready[q.position]
		q.ready[q.position] = nil
		q.position++
		c.queued = false
		if !c.included {
			return c, true
		}
	}
	return nil, false
}
func (q *inclusionQueue) block(c *buildCandidate) {
	if !c.inclusion {
		return
	}
	if _, frame := c.tx.Inner().(*types.FrameTx8141); frame {
		q.dynamic = append(q.dynamic, c)
		return
	}
	if sender := c.tx.Sender(); sender != nil {
		if q.waiting[*sender] == nil {
			q.waiting[*sender] = make(map[uint64][]*buildCandidate)
		}
		q.waiting[*sender][c.tx.Nonce()] = append(q.waiting[*sender][c.tx.Nonce()], c)
	}
}
func (q *inclusionQueue) wake(db state.StateDB, changes *bal.BlockAccessList) {
	if changes == nil {
		for _, c := range q.all {
			if c.inclusion {
				q.enqueue(c)
			}
		}
		q.waiting = make(map[types.Address]map[uint64][]*buildCandidate)
	} else {
		for _, e := range changes.Entries {
			if e.BalanceChange == nil && e.NonceChange == nil && e.CodeChange == nil {
				continue
			}
			byNonce := q.waiting[e.Address]
			nonce := db.GetNonce(e.Address)
			for _, c := range byNonce[nonce] {
				q.enqueue(c)
			}
			delete(byNonce, nonce)
		}
	}
	for _, c := range q.dynamic {
		q.enqueue(c)
	}
	q.dynamic = nil
}
