package block

import (
	"container/heap"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"sort"
)

type senderCandidates struct {
	txs      []*types.Transaction
	position int
	price    *big.Int
}
type candidateHeap []*senderCandidates

func (h candidateHeap) Len() int { return len(h) }
func (h candidateHeap) Less(i, j int) bool {
	if order := h[i].price.Cmp(h[j].price); order != 0 {
		return order > 0
	}
	return h[i].txs[h[i].position].Hash().Hex() < h[j].txs[h[j].position].Hash().Hex()
}
func (h candidateHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *candidateHeap) Push(v any)   { *h = append(*h, v.(*senderCandidates)) }
func (h *candidateHeap) Pop() any     { last := (*h)[len(*h)-1]; *h = (*h)[:len(*h)-1]; return last }

// orderTransactionsAt compares only executable sender heads. Sorting all
// envelopes by price/hash independently can put nonce N+1 ahead of N.
func orderTransactionsAt(input []*types.Transaction, baseFee, blobFee *big.Int) []*types.Transaction {
	groups := make(map[string][]*types.Transaction)
	for _, tx := range input {
		key := "hash:" + tx.Hash().Hex()
		if from := tx.Sender(); from != nil {
			key = "sender:" + from.Hex()
		} else if frame, ok := tx.Inner().(*types.FrameTx8141); ok {
			key = "sender:" + frame.Sender.Hex()
		}
		groups[key] = append(groups[key], tx)
	}
	queue := new(candidateHeap)
	for _, txs := range groups {
		sort.SliceStable(txs, func(i, j int) bool { return txs[i].Nonce() < txs[j].Nonce() })
		heap.Push(queue, &senderCandidates{txs: txs, price: effectiveGasPriceAt(txs[0], baseFee, blobFee)})
	}
	result := make([]*types.Transaction, 0, len(input))
	for queue.Len() > 0 {
		group := heap.Pop(queue).(*senderCandidates)
		result = append(result, group.txs[group.position])
		group.position++
		if group.position < len(group.txs) {
			group.price = effectiveGasPriceAt(group.txs[group.position], baseFee, blobFee)
			heap.Push(queue, group)
		}
	}
	return result
}
