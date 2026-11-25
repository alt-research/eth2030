package snap

import (
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

// Account bodies are raw RLP lists in both snap versions, unlike storage values
// and proof nodes, which are byte strings. A []byte reflection encoding adds an
// extra string layer and cannot interoperate with ordinary Ethereum peers.
type accountRangeWire struct {
	ID       uint64
	Accounts []struct {
		Hash types.Hash
		Body gethrlp.RawValue
	}
	Proof [][]byte
}

func validateSlimAccountBody(body []byte) error {
	var account gethtypes.SlimAccount
	if err := gethrlp.DecodeBytes(body, &account); err != nil {
		return fmt.Errorf("snap: invalid raw slim account: %w", err)
	}
	if (len(account.Root) != 0 && len(account.Root) != 32) || (len(account.CodeHash) != 0 && len(account.CodeHash) != 32) {
		return fmt.Errorf("snap: invalid slim account commitment length")
	}
	return nil
}

func (p AccountRangePacket) EncodeRLP() ([]byte, error) {
	wire := accountRangeWire{ID: p.ID, Proof: p.Proof}
	for _, account := range p.Accounts {
		if err := validateSlimAccountBody(account.Body); err != nil {
			return nil, err
		}
		wire.Accounts = append(wire.Accounts, struct {
			Hash types.Hash
			Body gethrlp.RawValue
		}{account.Hash, account.Body})
	}
	encoded, err := gethrlp.EncodeToBytes(wire)
	if err == nil && len(encoded) > HardResponseLimit {
		return nil, ErrRequestTooLarge
	}
	return encoded, err
}

func (p *AccountRangePacket) DecodeRLP(data []byte) error {
	if len(data) > HardResponseLimit {
		return ErrRequestTooLarge
	}
	var wire accountRangeWire
	if err := gethrlp.DecodeBytes(data, &wire); err != nil {
		return err
	}
	var accounts []AccountData
	for _, account := range wire.Accounts {
		if err := validateSlimAccountBody(account.Body); err != nil {
			return err
		}
		accounts = append(accounts, AccountData{Hash: account.Hash, Body: account.Body})
	}
	p.ID, p.Accounts, p.Proof = wire.ID, accounts, wire.Proof
	return nil
}
