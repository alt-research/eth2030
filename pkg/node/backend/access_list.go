package backend

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	rpcbackend "github.com/eth2030/eth2030/rpc/backend"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	invalid := func(field string, err error) error {
		return fmt.Errorf("%w: %s: %v", rpcbackend.ErrInvalidCallArgs, field, err)
	}
	address := func(value string) (types.Address, error) {
		raw, err := hexutil.Decode(value)
		if err != nil {
			return types.Address{}, err
		}
		if len(raw) != types.AddressLength {
			return types.Address{}, fmt.Errorf("address must contain 20 bytes")
		}
		return types.BytesToAddress(raw), nil
	}
	from := types.Address{}
	if args.From != nil {
		from, err = address(*args.From)
		if err != nil {
			return nil, invalid("from", err)
		}
	}
	var to *types.Address
	if args.To != nil {
		a, err := address(*args.To)
		if err != nil {
			return nil, invalid("to", err)
		}
		to = &a
	}
	limit := header.GasLimit
	if args.Gas != nil {
		limit, err = hexutil.DecodeUint64(*args.Gas)
		if err != nil {
			return nil, invalid("gas", err)
		}
	}
	price, value := new(big.Int), new(big.Int)
	if args.GasPrice != nil {
		price, err = hexutil.DecodeBig(*args.GasPrice)
		if err != nil {
			return nil, invalid("gasPrice", err)
		}
	}
	if args.Value != nil {
		value, err = hexutil.DecodeBig(*args.Value)
		if err != nil {
			return nil, invalid("value", err)
		}
	}
	nonce := base.GetNonce(from)
	if args.Nonce != nil {
		nonce, err = hexutil.DecodeUint64(*args.Nonce)
		if err != nil {
			return nil, invalid("nonce", err)
		}
	}
	input := args.Input
	if input == nil {
		input = args.Data
	}
	var data []byte
	if input != nil {
		data, err = hexutil.Decode(*input)
		if err != nil {
			return nil, invalid("input", err)
		}
	}
	list := make(types.AccessList, 0, len(args.AccessList))
	for _, item := range args.AccessList {
		a, err := address(item.Address)
		if err != nil {
			return nil, invalid("accessList address", err)
		}
		tuple := types.AccessTuple{Address: a, StorageKeys: make([]types.Hash, 0, len(item.StorageKeys))}
		for _, key := range item.StorageKeys {
			raw, err := hexutil.Decode(key)
			if err != nil {
				return nil, invalid("storage key", err)
			}
			if len(raw) != types.HashLength {
				return nil, invalid("storage key", fmt.Errorf("must contain 32 bytes"))
			}
			tuple.StorageKeys = append(tuple.StorageKeys, types.BytesToHash(raw))
		}
		list = append(list, tuple)
	}
	cfg := b.node.Blockchain().Config()
	for attempt := 0; attempt < 64; attempt++ {
		tracked := &accessedState{StateDB: base.Dup(), items: make(map[types.Address]map[types.Hash]struct{})}
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
		msg := &config.Message{From: from, To: to, Nonce: nonce, GasLimit: limit, GasPrice: price, Value: value, Data: data, AccessList: list}
		result, err := execution.ApplyMessage(cfg, b.node.Blockchain().GetHashFn(), tracked, header, msg, new(gaspool.GasPool).AddGas(limit))
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
