package bal

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
)

// addrKey is used as a map key for addresses.
type addrKey = types.Address

// slotKey uniquely identifies an (address, slot) pair.
type slotKey struct {
	Addr types.Address
	Slot types.Hash
}

// AccessTracker records state accesses during block execution and
// produces a BlockAccessList from the recorded data.
type AccessTracker struct {
	initialBalances map[types.Address]*big.Int
	initialNonces   map[types.Address]uint64
	reads           map[slotKey]types.Hash    // slot -> value read
	changes         map[slotKey][2]types.Hash // slot -> [old, new]
	balanceChanges  map[addrKey]*BalanceChange
	nonceChanges    map[addrKey]*NonceChange
	codeChanges     map[addrKey]*CodeChange
	touchedAddrs    map[addrKey]struct{}
}

// NewTracker creates a new AccessTracker with initialized maps.
func NewTracker() *AccessTracker {
	return &AccessTracker{
		initialBalances: make(map[types.Address]*big.Int),
		initialNonces:   make(map[types.Address]uint64),
		reads:           make(map[slotKey]types.Hash),
		changes:         make(map[slotKey][2]types.Hash),
		balanceChanges:  make(map[addrKey]*BalanceChange),
		nonceChanges:    make(map[addrKey]*NonceChange),
		codeChanges:     make(map[addrKey]*CodeChange),
		touchedAddrs:    make(map[addrKey]struct{}),
	}
}

// RecordStorageRead records a storage slot read.
func (t *AccessTracker) RecordStorageRead(addr types.Address, slot types.Hash, value types.Hash) {
	key := slotKey{Addr: addr, Slot: slot}
	t.reads[key] = value
	t.touchedAddrs[addr] = struct{}{}
}

// RecordStorageChange records a storage slot modification.
func (t *AccessTracker) RecordStorageChange(addr types.Address, slot, oldVal, newVal types.Hash) {
	key := slotKey{Addr: addr, Slot: slot}
	if previous, ok := t.changes[key]; ok {
		oldVal = previous[0]
	}
	t.changes[key] = [2]types.Hash{oldVal, newVal}
	t.touchedAddrs[addr] = struct{}{}
}

// RecordBalanceChange records a balance modification.
func (t *AccessTracker) RecordBalanceChange(addr types.Address, oldBal, newBal *big.Int) {
	if initial, ok := t.initialBalances[addr]; ok {
		oldBal = initial
	}
	if previous, ok := t.balanceChanges[addr]; ok {
		oldBal = previous.OldValue
	}
	t.balanceChanges[addr] = &BalanceChange{
		OldValue: new(big.Int).Set(oldBal),
		NewValue: new(big.Int).Set(newBal),
	}
	t.touchedAddrs[addr] = struct{}{}
}

// RecordNonceChange records a nonce modification.
func (t *AccessTracker) RecordNonceChange(addr types.Address, oldNonce, newNonce uint64) {
	if initial, ok := t.initialNonces[addr]; ok {
		oldNonce = initial
	}
	if previous, ok := t.nonceChanges[addr]; ok {
		oldNonce = previous.OldValue
	}
	t.nonceChanges[addr] = &NonceChange{OldValue: oldNonce, NewValue: newNonce}
	t.touchedAddrs[addr] = struct{}{}
}

// RecordAddressTouch records that an address was accessed (read or touched)
// without necessarily modifying any specific storage, balance, or code.
// This is used by opcodes like BALANCE, EXTCODEHASH, CALL target, etc.
func (t *AccessTracker) RecordAddressTouch(addr types.Address) {
	t.touchedAddrs[addr] = struct{}{}
}

// RecordCodeChange records a code modification.
func (t *AccessTracker) RecordCodeChange(addr types.Address, oldCode, newCode []byte) {
	if previous, ok := t.codeChanges[addr]; ok {
		oldCode = previous.OldCode
	}
	cc := &CodeChange{}
	if oldCode != nil {
		cc.OldCode = make([]byte, len(oldCode))
		copy(cc.OldCode, oldCode)
	}
	if newCode != nil {
		cc.NewCode = make([]byte, len(newCode))
		copy(cc.NewCode, newCode)
	}
	t.codeChanges[addr] = cc
	t.touchedAddrs[addr] = struct{}{}
}

// Build produces a BlockAccessList from the recorded accesses, tagging
// all entries with the given txIndex.
func (t *AccessTracker) Build(txIndex uint64) *BlockAccessList {
	bal := NewBlockAccessList()

	// Sort addresses for deterministic output.
	addrs := make([]types.Address, 0, len(t.touchedAddrs))
	for addr := range t.touchedAddrs {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return addrLess(addrs[i], addrs[j])
	})

	for _, addr := range addrs {
		entry := AccessEntry{
			Address:     addr,
			AccessIndex: txIndex,
		}

		// Collect storage reads for this address.
		for key, val := range t.reads {
			if key.Addr == addr {
				entry.StorageReads = append(entry.StorageReads, StorageAccess{
					Slot:  key.Slot,
					Value: val,
				})
			}
		}
		sort.Slice(entry.StorageReads, func(i, j int) bool {
			return hashLess(entry.StorageReads[i].Slot, entry.StorageReads[j].Slot)
		})

		// Collect storage changes for this address.
		for key, vals := range t.changes {
			if key.Addr == addr {
				entry.StorageChanges = append(entry.StorageChanges, StorageChange{
					Slot:     key.Slot,
					OldValue: vals[0],
					NewValue: vals[1],
				})
			}
		}
		sort.Slice(entry.StorageChanges, func(i, j int) bool {
			return hashLess(entry.StorageChanges[i].Slot, entry.StorageChanges[j].Slot)
		})

		if bc, ok := t.balanceChanges[addr]; ok {
			entry.BalanceChange = bc
		}
		if nc, ok := t.nonceChanges[addr]; ok {
			entry.NonceChange = nc
		}
		if cc, ok := t.codeChanges[addr]; ok {
			entry.CodeChange = cc
		}

		bal.AddEntry(entry)
	}

	return bal
}

// Reset clears all recorded accesses so the tracker can be reused.
func (t *AccessTracker) Reset() {
	t.initialBalances = make(map[types.Address]*big.Int)
	t.initialNonces = make(map[types.Address]uint64)
	t.reads = make(map[slotKey]types.Hash)
	t.changes = make(map[slotKey][2]types.Hash)
	t.balanceChanges = make(map[addrKey]*BalanceChange)
	t.nonceChanges = make(map[addrKey]*NonceChange)
	t.codeChanges = make(map[addrKey]*CodeChange)
	t.touchedAddrs = make(map[addrKey]struct{})
}

// addrLess compares two addresses lexicographically.
func addrLess(a, b types.Address) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// hashLess compares two hashes lexicographically.
func hashLess(a, b types.Hash) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// SeedAccount retains a transaction-boundary baseline without recording an
// access. Upfront fees and the transaction nonce can precede EVM BAL hooks.
func (t *AccessTracker) SeedAccount(address types.Address, balance *big.Int, nonce uint64) {
	if _, ok := t.initialBalances[address]; !ok {
		t.initialBalances[address] = new(big.Int).Set(balance)
	}
	if _, ok := t.initialNonces[address]; !ok {
		t.initialNonces[address] = nonce
	}
}

// Reconcile discards rolled-back writes and transient changes using finalized
// transaction state. Accesses remain recorded, including no-op and reverted
// SSTOREs, which become storage reads under EIP-7928.
func (t *AccessTracker) Reconcile(db interface {
	GetState(types.Address, types.Hash) types.Hash
	GetBalance(types.Address) *big.Int
	GetNonce(types.Address) uint64
	GetCode(types.Address) []byte
}) {
	for key, change := range t.changes {
		change[1] = db.GetState(key.Addr, key.Slot)
		if change[0] == change[1] {
			delete(t.changes, key)
			t.reads[key] = change[0]
		} else {
			t.changes[key] = change
			delete(t.reads, key)
		}
	}
	for address, change := range t.balanceChanges {
		change.NewValue = new(big.Int).Set(db.GetBalance(address))
		if change.OldValue.Cmp(change.NewValue) == 0 {
			delete(t.balanceChanges, address)
		}
	}
	for address, change := range t.nonceChanges {
		change.NewValue = db.GetNonce(address)
		if change.OldValue == change.NewValue {
			delete(t.nonceChanges, address)
		}
	}
	for address, change := range t.codeChanges {
		change.NewCode = bytes.Clone(db.GetCode(address))
		if change.NewCodeHash != nil && crypto.Keccak256Hash(change.NewCode) != *change.NewCodeHash {
			change.NewCodeHash = nil
		}
		if bytes.Equal(change.OldCode, change.NewCode) {
			delete(t.codeChanges, address)
		}
	}
}

// RecordCodeAdoption preserves the original pre-value and emits only the hash.
func (t *AccessTracker) RecordCodeAdoption(addr types.Address, old []byte, hash types.Hash) {
	if previous, ok := t.codeChanges[addr]; ok {
		old = previous.OldCode
	}
	t.codeChanges[addr] = &CodeChange{OldCode: bytes.Clone(old), NewCodeHash: &hash}
	t.touchedAddrs[addr] = struct{}{}
}

// ReconcileCodeOrigins8298 uses the same snapshot journal as EVM state. A
// reverted child adoption must not change the origin of an earlier installation.
func (t *AccessTracker) ReconcileCodeOrigins8298(origins map[types.Address]types.Hash) {
	for address, change := range t.codeChanges {
		if hash, adopted := origins[address]; adopted {
			change.NewCodeHash = &hash
		} else {
			change.NewCodeHash = nil
		}
	}
}
