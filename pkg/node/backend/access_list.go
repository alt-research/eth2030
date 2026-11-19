package backend

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpcbackend "github.com/eth2030/eth2030/rpc/backend"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

// accessedState records accesses independently of state rollback. In particular,
// an inner call reverting must not erase the accesses discovered by simulation.
type accessedState struct {
	state.StateDB
	items map[types.Address]map[types.Hash]struct{}
}

func (s *accessedState) address(a types.Address) {
	if s.items[a] == nil {
		s.items[a] = make(map[types.Hash]struct{})
	}
}
func (s *accessedState) slot(a types.Address, k types.Hash) { s.address(a); s.items[a][k] = struct{}{} }
func (s *accessedState) GetBalance(a types.Address) *big.Int {
	s.address(a)
	return s.StateDB.GetBalance(a)
}
func (s *accessedState) GetNonce(a types.Address) uint64 { s.address(a); return s.StateDB.GetNonce(a) }
func (s *accessedState) GetCode(a types.Address) []byte  { s.address(a); return s.StateDB.GetCode(a) }
func (s *accessedState) GetCodeHash(a types.Address) types.Hash {
	s.address(a)
	return s.StateDB.GetCodeHash(a)
}
func (s *accessedState) GetCodeSize(a types.Address) int {
	s.address(a)
	return s.StateDB.GetCodeSize(a)
}
func (s *accessedState) Exist(a types.Address) bool { s.address(a); return s.StateDB.Exist(a) }
func (s *accessedState) Empty(a types.Address) bool { s.address(a); return s.StateDB.Empty(a) }
func (s *accessedState) GetState(a types.Address, k types.Hash) types.Hash {
	s.slot(a, k)
	return s.StateDB.GetState(a, k)
}
func (s *accessedState) GetCommittedState(a types.Address, k types.Hash) types.Hash {
	s.slot(a, k)
	return s.StateDB.GetCommittedState(a, k)
}
func (s *accessedState) SetState(a types.Address, k, v types.Hash) {
	s.slot(a, k)
	s.StateDB.SetState(a, k, v)
}
func (s *accessedState) AddAddressToAccessList(a types.Address) {
	s.address(a)
	s.StateDB.AddAddressToAccessList(a)
}
func (s *accessedState) AddSlotToAccessList(a types.Address, k types.Hash) {
	s.slot(a, k)
	s.StateDB.AddSlotToAccessList(a, k)
}

func (s *accessedState) list(excluded map[types.Address]bool) types.AccessList {
	out := make(types.AccessList, 0, len(s.items))
	for address, slots := range s.items {
		if excluded[address] && len(slots) == 0 {
			continue
		}
		keys := make([]types.Hash, 0, len(slots))
		for k := range slots {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i][:], keys[j][:]) < 0 })
		out = append(out, types.AccessTuple{Address: address, StorageKeys: keys})
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i].Address[:], out[j].Address[:]) < 0 })
	return out
}

// CreateAccessList simulates transaction execution with a converged access list.
// Every attempt has an independent state copy; the returned gas is measured with
// the returned list, even if omitting it would be cheaper under a newer schedule.
func (b *RPCBackend) CreateAccessList(args rpctypes.CallArgs, number rpctypes.BlockNumber) (*rpcbackend.AccessListResult, error) {
	header := b.HeaderByNumber(number)
	if header == nil {
		return nil, fmt.Errorf("block not found")
	}
	base, err := b.StateAtHeader(header)
	if err != nil {
		return nil, err
	}
	msg, err := parseSimulationMessage(args, header, base, b.node.Blockchain().Config())
	if err != nil {
		return nil, err
	}
	from, to, nonce, limit, list := msg.From, msg.To, msg.Nonce, msg.GasLimit, msg.AccessList
	cfg := b.node.Blockchain().Config()
	for attempt := 0; attempt < 64; attempt++ {
		tracked := &accessedState{StateDB: &simulationBalanceState{StateDB: base.Dup()}, items: make(map[types.Address]map[types.Hash]struct{})}
		excluded := map[types.Address]bool{from: true, header.Coinbase: true}
		if to != nil {
			excluded[*to] = true
		} else {
			excluded[vm.CreateAddressFromNonce(from, nonce)] = true
		}
		evm := vm.NewEVMWithState(vm.BlockContext{}, vm.TxContext{}, vm.Config{}, tracked)
		active, err := execution.ConfigureBlockEVM(cfg, header, evm)
		if err != nil {
			return nil, err
		}
		for a := range active {
			excluded[a] = true
		}
		candidate := *msg
		candidate.AccessList = list
		result, err := execution.SimulateMessage(cfg, b.node.Blockchain().GetHashFn(), tracked, header, &candidate, new(gaspool.GasPool).AddGas(limit), true, vm.Config{})
		if err != nil {
			return nil, err
		}
		next := tracked.list(excluded)
		if reflect.DeepEqual(list, next) {
			out := &rpcbackend.AccessListResult{AccessList: next, GasUsed: result.UsedGas}
			if result.Err != nil {
				out.ExecutionError = result.Err.Error()
			}
			return out, nil
		}
		list = next
	}
	return nil, fmt.Errorf("access-list discovery did not converge after 64 executions")
}
