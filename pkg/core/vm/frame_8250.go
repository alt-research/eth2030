package vm

import (
	"math"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// consumeFrameNonce is part of APPROVE's existing journaled transition. No
// protocol nonce read or write warms an account/slot or uses SSTORE pricing.
func (evm *EVM) consumeFrameNonce() error {
	f := evm.frame8141
	var keys []*big.Int
	if f.NonceProfile != nil {
		keys = f.NonceProfile.NonceKeys
	}
	if len(keys) == 0 {
		nonce := evm.StateDB.GetNonce(f.Tx.Sender)
		if nonce == math.MaxUint64 {
			return ErrFrameNonceOverflow
		}
		if evm.StateDB.Empty(f.Tx.Sender) {
			if err := f.Charge(frameNewAccountGas); err != nil {
				return err
			}
		}
		evm.StateDB.SetNonce(f.Tx.Sender, nonce+1)
		return nil
	}
	var firstUse uint64
	for _, key := range keys {
		if evm.StateDB.GetState(types.NonceManager8250, types.NonceSlot8250(f.Tx.Sender, key)).IsZero() {
			firstUse++
		}
	}
	if err := f.Charge(firstUse * types.KeyedNonceFirstUseStateGas8250); err != nil {
		return err
	}
	seq := f.NonceProfile.NonceSeq
	if seq != math.MaxUint64 {
		seq++
	}
	value := types.BytesToHash(new(big.Int).SetUint64(seq).Bytes())
	for _, key := range keys {
		slot := types.NonceSlot8250(f.Tx.Sender, key)
		before := evm.StateDB.GetState(types.NonceManager8250, slot)
		evm.StateDB.SetState(types.NonceManager8250, slot, value)
		if evm.balTracker != nil {
			evm.balTracker.RecordStorageChange(types.NonceManager8250, slot, before, value)
		}
	}
	return nil
}
