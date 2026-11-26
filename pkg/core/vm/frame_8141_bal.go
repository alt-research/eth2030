package vm

import (
	"bytes"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// frameBALJournal retains the access set across reverts, but emits changes only
// from the transaction's initial values to its final journaled state.
// Skipped frames never access their targets and therefore add no BAL entries.
type frameBALJournal struct {
	output   BALTracker
	db       StateDB
	slots    map[frameStorageKey]struct{}
	balances map[types.Address]*big.Int
	nonces   map[types.Address]uint64
	codes    map[types.Address][]byte
}

func newFrameBALJournal(output BALTracker, db StateDB) *frameBALJournal {
	return &frameBALJournal{output: output, db: db, slots: make(map[frameStorageKey]struct{}), balances: make(map[types.Address]*big.Int), nonces: make(map[types.Address]uint64), codes: make(map[types.Address][]byte)}
}
func (b *frameBALJournal) RecordAddressTouch(address types.Address) {
	b.output.RecordAddressTouch(address)
}
func (b *frameBALJournal) RecordStorageRead(address types.Address, slot, value types.Hash) {
	b.slots[frameStorageKey{address, slot}] = struct{}{}
	b.RecordAddressTouch(address)
}
func (b *frameBALJournal) RecordStorageChange(address types.Address, slot, old, new types.Hash) {
	b.RecordStorageRead(address, slot, old)
}
func (b *frameBALJournal) RecordBalanceChange(address types.Address, old, new *big.Int) {
	b.RecordAddressTouch(address)
}
func (b *frameBALJournal) beforeBalance(address types.Address) {
	if _, known := b.balances[address]; !known {
		b.balances[address] = new(big.Int).Set(b.db.GetBalance(address))
	}
	b.RecordAddressTouch(address)
}
func (b *frameBALJournal) beforeNonce(address types.Address) {
	if _, known := b.nonces[address]; !known {
		b.nonces[address] = b.db.GetNonce(address)
	}
	b.RecordAddressTouch(address)
}
func (b *frameBALJournal) beforeCode(address types.Address) {
	if _, known := b.codes[address]; !known {
		b.codes[address] = bytes.Clone(b.db.GetCode(address))
	}
	b.RecordAddressTouch(address)
}
func (evm *EVM) FlushFrameBAL8141() {
	b := evm.frame8141.bal
	if b == nil {
		return
	}
	for key := range b.slots {
		original, final := b.db.GetCommittedState(key.address, key.slot), b.db.GetState(key.address, key.slot)
		if original == final {
			b.output.RecordStorageRead(key.address, key.slot, original)
		} else {
			b.output.RecordStorageChange(key.address, key.slot, original, final)
		}
	}
	for address, original := range b.balances {
		final := b.db.GetBalance(address)
		if original.Cmp(final) != 0 {
			b.output.RecordBalanceChange(address, original, final)
		}
	}
	if output, ok := b.output.(interface {
		RecordNonceChange(types.Address, uint64, uint64)
	}); ok {
		for address, original := range b.nonces {
			if final := b.db.GetNonce(address); original != final {
				output.RecordNonceChange(address, original, final)
			}
		}
	}
	if output, ok := b.output.(interface {
		RecordCodeChange(types.Address, []byte, []byte)
	}); ok {
		for address, original := range b.codes {
			if final := b.db.GetCode(address); !bytes.Equal(original, final) {
				output.RecordCodeChange(address, original, final)
			}
		}
	}
}

func (s *frameJournalState) AddBalance(address types.Address, amount *big.Int) {
	if amount == nil || amount.Sign() == 0 {
		return
	}
	if s.frame.bal != nil {
		s.frame.bal.beforeBalance(address)
	}
	s.StateDB.AddBalance(address, amount)
}
func (s *frameJournalState) SubBalance(address types.Address, amount *big.Int) {
	if amount == nil || amount.Sign() == 0 {
		return
	}
	if s.frame.bal != nil {
		s.frame.bal.beforeBalance(address)
	}
	s.StateDB.SubBalance(address, amount)
}
func (s *frameJournalState) CreateAccount(address types.Address) {
	// Account reconstruction clears leaf fields before prefunding is restored.
	// Capture the transaction's original values before that reset, not on the
	// first subsequent AddBalance/SetNonce/SetCode (EIP-7928 net changes).
	if s.frame.bal != nil {
		s.frame.bal.beforeBalance(address)
		s.frame.bal.beforeNonce(address)
		s.frame.bal.beforeCode(address)
	}
	s.StateDB.CreateAccount(address)
}
func (s *frameJournalState) SetNonce(address types.Address, nonce uint64) {
	if s.frame.bal != nil {
		s.frame.bal.beforeNonce(address)
	}
	s.StateDB.SetNonce(address, nonce)
}
func (s *frameJournalState) SetCode(address types.Address, code []byte) {
	if s.frame.bal != nil {
		s.frame.bal.beforeCode(address)
	}
	s.StateDB.SetCode(address, code)
}
func (s *frameJournalState) GetCode(address types.Address) []byte {
	if s.frame.bal != nil {
		s.frame.bal.RecordAddressTouch(address)
	}
	return s.StateDB.GetCode(address)
}
func (s *frameJournalState) GetCodeHash(address types.Address) types.Hash {
	if s.frame.bal != nil {
		s.frame.bal.RecordAddressTouch(address)
	}
	return s.StateDB.GetCodeHash(address)
}
func (s *frameJournalState) GetBalance(address types.Address) *big.Int {
	if s.frame.bal != nil {
		s.frame.bal.RecordAddressTouch(address)
	}
	return s.StateDB.GetBalance(address)
}
func (s *frameJournalState) Empty(address types.Address) bool {
	if s.frame.bal != nil {
		s.frame.bal.RecordAddressTouch(address)
	}
	return s.StateDB.Empty(address)
}
