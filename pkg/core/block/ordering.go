package block

import (
	"container/heap"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"sort"
)

type senderCandidates struct {
	order    int
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
	return h[i].order < h[j].order
}
func (h candidateHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *candidateHeap) Push(v any)   { *h = append(*h, v.(*senderCandidates)) }
func (h *candidateHeap) Pop() any     { last := (*h)[len(*h)-1]; *h = (*h)[:len(*h)-1]; return last }

// orderTransactionsAt compares only executable sender heads. Sorting all
// envelopes by price/hash independently can put nonce N+1 ahead of N.
func orderTransactionsAt(input []*types.Transaction, baseFee, blobFee *big.Int) []*types.Transaction {
	groups := make(map[string][]*types.Transaction)
	order := make(map[string]int)
	for _, tx := range input {
		key := "hash:" + tx.Hash().Hex()
		if from := tx.Sender(); from != nil {
			key = "sender:" + from.Hex()
		} else if frame, ok := tx.Inner().(*types.FrameTx8141); ok {
			key = "sender:" + frame.Sender.Hex()
		} else if tx.Type() <= types.SetCodeTxType {
			chainID := uint64(0)
			if chain := tx.ChainId(); chain != nil && chain.IsUint64() {
				chainID = chain.Uint64()
			}
			if from, err := types.MakeSigner(chainID, tx.Type()).Sender(tx); err == nil {
				key = "sender:" + from.Hex()
			}
		}
		// Keyed replay domains are independent of the account-nonce sequence.
		// Public policy admits one per sender; private input can contain many.
		if frame, ok := tx.Inner().(*types.FrameTx8141); ok && frame.HasKeyedNonces() {
			key = "keyed:" + tx.Hash().Hex()
		}
		if _, exists := groups[key]; !exists {
			order[key] = len(order)
		}
		groups[key] = append(groups[key], tx)
	}
	queue := new(candidateHeap)
	for key, txs := range groups {
		sort.SliceStable(txs, func(i, j int) bool { return txs[i].Nonce() < txs[j].Nonce() })
		heap.Push(queue, &senderCandidates{order: order[key], txs: txs, price: effectiveGasPriceAt(txs[0], baseFee, blobFee)})
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
