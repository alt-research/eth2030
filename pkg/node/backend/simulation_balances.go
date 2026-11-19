package backend

import (
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/holiman/uint256"
	"math/big"
)

// RPC overrides can construct balances near uint256's boundary, unlike a real
// chain with bounded ETH supply. Apply the EVM's uint256 arithmetic on isolated
// simulation state instead of letting the native big.Int adapter overflow it.
type simulationBalanceState struct{ state.StateDB }

func (s *simulationBalanceState) AddBalance(address types.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		s.StateDB.AddBalance(address, amount)
		return
	}
	before := s.StateDB.GetBalance(address)
	current, _ := uint256.FromBig(before)
	delta, _ := uint256.FromBig(amount)
	after := new(uint256.Int).Add(current, delta).ToBig()
	s.assign(address, before, after)
}
func (s *simulationBalanceState) SubBalance(address types.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		s.StateDB.SubBalance(address, amount)
		return
	}
	before := s.StateDB.GetBalance(address)
	current, _ := uint256.FromBig(before)
	delta, _ := uint256.FromBig(amount)
	after := new(uint256.Int).Sub(current, delta).ToBig()
	s.assign(address, before, after)
}
func (s *simulationBalanceState) assign(address types.Address, before, after *big.Int) {
	change := new(big.Int).Sub(after, before)
	if change.Sign() > 0 {
		s.StateDB.AddBalance(address, change)
	} else if change.Sign() < 0 {
		s.StateDB.SubBalance(address, change.Neg(change))
	}
}
func (s *simulationBalanceState) FinalizePreState() {
	if target, ok := s.StateDB.(interface{ FinalizePreState() }); ok {
		target.FinalizePreState()
	}
}
func (s *simulationBalanceState) FinalizeTransaction() {
	if target, ok := s.StateDB.(interface{ FinalizeTransaction() }); ok {
		target.FinalizeTransaction()
	}
}
