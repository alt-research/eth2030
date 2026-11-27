package bal

import (
	"fmt"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func (a AccountChanges) AdoptedCodeHash(index uint64) (types.Hash, bool) {
	hash, ok := a.adoptedCodeHashes[index]
	return hash, ok
}

// EncodeAccountChanges retains the historical two-element form for ordinary
// installation and emits the EIP-8298 hash-only third element for adoption.
func EncodeAccountChanges(accounts []AccountChanges) ([]byte, error) {
	wire := make([]any, 0, len(accounts))
	for _, a := range accounts {
		codes := make([]any, 0, len(a.CodeChanges))
		for _, c := range a.CodeChanges {
			if hash, adopted := a.AdoptedCodeHash(c.Index); adopted {
				if len(c.Value) != 0 || hash == (types.Hash{}) || hash == types.EmptyCodeHash {
					return nil, fmt.Errorf("invalid adopted code change")
				}
				codes = append(codes, []any{c.Index, []byte{}, hash})
			} else {
				codes = append(codes, []any{c.Index, c.Value})
			}
		}
		wire = append(wire, []any{a.Address, a.StorageChanges, a.StorageReads, a.BalanceChanges, a.NonceChanges, codes})
	}
	return rlp.EncodeToBytes(wire)
}

func decodeAccountChanges(encoded []byte, accounts *[]AccountChanges) error {
	var wire []struct {
		Address        types.Address
		StorageChanges []SlotChanges
		StorageReads   []*big.Int
		BalanceChanges []IndexedChange[*big.Int]
		NonceChanges   []IndexedChange[uint64]
		CodeChanges    []gethrlp.RawValue
	}
	if err := gethrlp.DecodeBytes(encoded, &wire); err != nil {
		return err
	}
	*accounts = make([]AccountChanges, len(wire))
	for i, w := range wire {
		a := &(*accounts)[i]
		a.Address, a.StorageChanges, a.StorageReads, a.BalanceChanges, a.NonceChanges = w.Address, w.StorageChanges, w.StorageReads, w.BalanceChanges, w.NonceChanges
		a.CodeChanges = make([]IndexedChange[[]byte], 0, len(w.CodeChanges))
		for _, code := range w.CodeChanges {
			var fields []gethrlp.RawValue
			if err := gethrlp.DecodeBytes(code, &fields); err != nil {
				return err
			}
			if len(fields) != 2 && len(fields) != 3 {
				return fmt.Errorf("BAL code change arity")
			}
			var c IndexedChange[[]byte]
			if err := gethrlp.DecodeBytes(fields[0], &c.Index); err != nil {
				return err
			}
			if err := gethrlp.DecodeBytes(fields[1], &c.Value); err != nil {
				return err
			}
			if len(fields) == 3 {
				var hash types.Hash
				if err := gethrlp.DecodeBytes(fields[2], &hash); err != nil {
					return err
				}
				if len(c.Value) != 0 || hash == (types.Hash{}) || hash == types.EmptyCodeHash {
					return fmt.Errorf("BAL code adoption must contain only a nonempty hash")
				}
				if a.adoptedCodeHashes == nil {
					a.adoptedCodeHashes = make(map[uint64]types.Hash)
				}
				a.adoptedCodeHashes[c.Index] = hash
			}
			a.CodeChanges = append(a.CodeChanges, c)
		}
	}
	return nil
}

// ResolveCodeChanges enforces strictly-earlier availability. preBlock must
// consult the authenticated pre-block code store, not a historical cache or
// the post-block state. No same-index creation can satisfy an adoption.
func ResolveCodeChanges(accounts []AccountChanges, active bool, preBlock func(types.Hash) ([]byte, bool)) (map[types.Hash][]byte, error) {
	type installed struct {
		index uint64
		code  []byte
	}
	installedCodes := make(map[types.Hash]installed)
	for _, a := range accounts {
		for _, c := range a.CodeChanges {
			if _, adopted := a.AdoptedCodeHash(c.Index); adopted {
				continue
			}
			hash := crypto.Keccak256Hash(c.Value)
			if old, found := installedCodes[hash]; !found || c.Index < old.index {
				installedCodes[hash] = installed{c.Index, c.Value}
			}
		}
	}
	resolved := make(map[types.Hash][]byte)
	for _, a := range accounts {
		for _, c := range a.CodeChanges {
			hash, adopted := a.AdoptedCodeHash(c.Index)
			if !adopted {
				continue
			}
			if !active {
				return nil, fmt.Errorf("BAL code adoption before EIP-8298")
			}
			code, ok := preBlock(hash)
			if !ok {
				if earlier, found := installedCodes[hash]; found && earlier.index < c.Index {
					code, ok = earlier.code, true
				}
			}
			if !ok || len(code) == 0 || code[0] == 0xef || crypto.Keccak256Hash(code) != hash {
				return nil, fmt.Errorf("unavailable BAL adopted code %s at %d", hash, c.Index)
			}
			resolved[hash] = append([]byte(nil), code...)
		}
	}
	return resolved, nil
}
