package txpool

import (
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
)

// An adoption invalidates already pending ECDSA origins too. All ordinary
// lists share the same lookup removal path, so replacement/queue/reorg metadata
// cannot retain transactions whose key has been retired by regular code.
func (pool *TxPool) evictRetiredOrigins8298() {
	if !pool.frameConfig.IsEIP8298(pool.frameTime) {
		return
	}
	db, ok := pool.state.(interface{ GetCode(types.Address) []byte })
	if !ok {
		return
	}
	retired := make(map[types.Address]bool)
	for _, tx := range pool.lookup.all {
		if tx.Type() > types.SetCodeTxType {
			continue
		}
		addr := pool.senderOf(tx)
		code := db.GetCode(addr)
		if state.ReadError(pool.state) != nil {
			return
		}
		if len(code) != 0 {
			if _, delegated := types.ParseDelegation(code); !delegated {
				retired[addr] = true
			}
		}
	}
	for addr := range retired {
		for _, lists := range []map[types.Address]*txSortedList{pool.pending, pool.queue} {
			if list := lists[addr]; list != nil {
				for _, tx := range append([]*types.Transaction(nil), list.items...) {
					if tx.Type() <= types.SetCodeTxType {
						list.RemoveTx(tx)
						pool.lookup.Remove(tx.Hash())
					}
				}
				if list.Len() == 0 {
					delete(lists, addr)
				}
			}
		}
		pool.reconcileScalarPending(addr)
	}
}
