package state

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	"github.com/eth2030/eth2030/trie"
	"github.com/eth2030/eth2030/trie/bintrie"
)

var ErrBinaryMigrationSource = errors.New("binary migration: source state does not match committed root")

// MigrateCommittedToBinary migrates the committed TrieStateDB flat tables to
// an EIP-7864 draft BLAKE3 tree. The source must remain quiescent for the full
// operation. expectedRoot is the trusted MPT root; account and storage roots
// and code hashes are independently checked before any binary nodes are saved.
// It does not publish or activate the returned root. The caller owns durability,
// root publication and exclusion of concurrent writers.
func MigrateCommittedToBinary(source rawdb.Database, expectedRoot types.Hash, dest bintrie.NodeStore) (types.Hash, error) {
	if source == nil || dest == nil {
		return types.Hash{}, ErrBinaryMigrationSource
	}
	mpt := trie.New()
	binary := bintrie.NewWithHashFunc(bintrie.HashFunctionBlake3)
	accounts := source.NewIterator(dbPrefixAccount)
	defer accounts.Release()
	for accounts.Next() {
		key, body := accounts.Key(), accounts.Value()
		if len(key) != len(dbPrefixAccount)+len(types.Address{}) {
			return types.Hash{}, ErrBinaryMigrationSource
		}
		var address types.Address
		copy(address[:], key[len(dbPrefixAccount):])
		var account rlpAccount
		if err := rlp.DecodeBytes(body, &account); err != nil || account.Balance == nil || len(account.Root) != 32 || len(account.CodeHash) != 32 {
			return types.Hash{}, fmt.Errorf("%w: invalid account %x", ErrBinaryMigrationSource, address)
		}
		if err := mpt.Put(crypto.Keccak256(address[:]), body); err != nil {
			return types.Hash{}, err
		}
		var code []byte
		if !bytes.Equal(account.CodeHash, types.EmptyCodeHash[:]) {
			var err error
			code, err = source.Get(codeDBKey(account.CodeHash))
			if err != nil || !bytes.Equal(crypto.Keccak256(code), account.CodeHash) {
				return types.Hash{}, fmt.Errorf("%w: missing or invalid code for %x", ErrBinaryMigrationSource, address)
			}
		}
		stateAccount := &types.Account{Nonce: account.Nonce, Balance: account.Balance, CodeHash: account.CodeHash}
		if err := binary.UpdateAccount(address, stateAccount, len(code)); err != nil {
			return types.Hash{}, fmt.Errorf("binary migration: account %x: %w", address, err)
		}
		if err := binary.UpdateContractCode(address, code); err != nil {
			return types.Hash{}, err
		}
		slots := source.NewIterator(storageDBPrefix(address))
		obj := newStateObject()
		for slots.Next() {
			sk, value := slots.Key(), slots.Value()
			if len(sk) != len(dbPrefixStorage)+len(address)+32 || len(value) != 32 {
				slots.Release()
				return types.Hash{}, fmt.Errorf("%w: invalid storage for %x", ErrBinaryMigrationSource, address)
			}
			var slot, val types.Hash
			copy(slot[:], sk[len(dbPrefixStorage)+len(address):])
			copy(val[:], value)
			if val == (types.Hash{}) {
				continue
			}
			obj.committedStorage[slot] = val
			if err := binary.UpdateStorage(address, slot[:], val[:]); err != nil {
				slots.Release()
				return types.Hash{}, err
			}
		}
		slots.Release()
		actualStorageRoot := computeStorageRoot(obj)
		if !bytes.Equal(actualStorageRoot[:], account.Root) {
			return types.Hash{}, fmt.Errorf("%w: storage root for %x", ErrBinaryMigrationSource, address)
		}
	}
	if mpt.Hash() != expectedRoot {
		return types.Hash{}, ErrBinaryMigrationSource
	}
	return binary.Persist(dest)
}
