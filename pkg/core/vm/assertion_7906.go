package vm

import (
	"errors"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

const (
	TXTRACE       OpCode = 0xb6
	EVENTDATACOPY OpCode = 0xb7
)

var ErrAssertionTrace7906 = errors.New("invalid EIP-7906 trace selector, index or context")

type AssertionChange7906 struct {
	Address       types.Address
	Before, After *big.Int
	Key           types.Hash
}

// AssertionTrace7906 is built lazily from the transaction journal, not from an
// EVM state read. Entries are already sorted; the VM only indexes these tables.
type AssertionTrace7906 struct {
	Balances, Slots, Codes, Nonces []AssertionChange7906
	Events                         []*types.Log
}

func (evm *EVM) assertionTrace7906() (*AssertionTrace7906, error) {
	f := evm.frame8141
	if !evm.forkRules.IsEIP7906 || f == nil || f.Tx.Frames[f.Index].Mode != types.ModeAssert7906 || evm.Config.FrameAssertionTrace == nil {
		return nil, ErrAssertionTrace7906
	}
	return evm.Config.FrameAssertionTrace(), nil
}

func opTxTrace7906(_ *uint64, evm *EVM, _ *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	trace, err := evm.assertionTrace7906()
	if err != nil {
		return nil, err
	}
	param, index := stack.Pop(), stack.Pop()
	if !param.IsUint64() || param.Uint64() > 0x1a {
		return nil, ErrAssertionTrace7906
	}
	p := param.Uint64()
	value := new(big.Int)
	if p <= 3 || p == 0x11 || p >= 0x19 {
		if index.Sign() != 0 {
			return nil, ErrAssertionTrace7906
		}
		switch p {
		case 0:
			value.SetUint64(uint64(len(trace.Balances)))
		case 1:
			value.SetUint64(uint64(len(trace.Slots)))
		case 2:
			value.SetUint64(uint64(len(trace.Codes)))
		case 3:
			value.SetUint64(uint64(len(trace.Nonces)))
		case 0x11:
			value.SetUint64(uint64(len(trace.Events)))
		case 0x19:
			if evm.frame8141.Payment != nil {
				value.Set(evm.frame8141.Payment.Escrow())
			}
		case 0x1a:
			if evm.frame8141.Payer != nil {
				value.SetBytes(evm.frame8141.Payer[:])
			}
		}
	} else {
		if !index.IsUint64() {
			return nil, ErrAssertionTrace7906
		}
		i := index.Uint64()
		if p >= 0x12 {
			if i >= uint64(len(trace.Events)) {
				return nil, ErrAssertionTrace7906
			}
			event := trace.Events[i]
			switch p {
			case 0x12:
				value.SetBytes(event.Address[:])
			case 0x13:
				value.SetUint64(uint64(len(event.Topics)))
			case 0x18:
				value.SetUint64(uint64(len(event.Data)))
			default:
				topic := p - 0x14
				if topic >= uint64(len(event.Topics)) {
					return nil, ErrAssertionTrace7906
				}
				value.SetBytes(event.Topics[topic][:])
			}
		} else {
			var table []AssertionChange7906
			var field uint64
			switch {
			case p <= 6:
				table, field = trace.Balances, p-4
			case p <= 0xa:
				table, field = trace.Slots, p-7
			case p <= 0xd:
				table, field = trace.Codes, p-0xb
			default:
				table, field = trace.Nonces, p-0xe
			}
			if i >= uint64(len(table)) {
				return nil, ErrAssertionTrace7906
			}
			entry := table[i]
			switch field {
			case 0:
				value.SetBytes(entry.Address[:])
			case 1:
				value.Set(entry.Before)
			case 2:
				value.Set(entry.After)
			case 3:
				value.SetBytes(entry.Key[:])
			}
		}
	}
	stack.Push(value)
	return nil, nil
}

func eventData7906(evm *EVM, stack *Stack) ([]byte, error) {
	trace, err := evm.assertionTrace7906()
	if err != nil {
		return nil, err
	}
	i, offset, length := stack.Back(0), stack.Back(2), stack.Back(3)
	if !i.IsUint64() || i.Uint64() >= uint64(len(trace.Events)) {
		return nil, ErrAssertionTrace7906
	}
	data := trace.Events[i.Uint64()].Data
	// Big integer addition deliberately does not wrap at 256 bits, including
	// zero-length copies with a nonzero or unrepresentable source offset.
	end := new(big.Int).Add(offset, length)
	if !end.IsUint64() || end.Uint64() > uint64(len(data)) {
		return nil, ErrAssertionTrace7906
	}
	return data, nil
}

func memoryEventData7906(stack *Stack) (uint64, bool) {
	return checkedMemoryRange(stack.Back(1), stack.Back(3))
}

func gasEventData7906(evm *EVM, c *Contract, stack *Stack, mem *Memory, size uint64) (uint64, error) {
	if _, err := eventData7906(evm, stack); err != nil {
		return 0, err
	}
	cost, err := gasMemExpansion(evm, c, stack, mem, size)
	return safeAdd(cost, safeMul(3, toWordSize(stack.Back(3).Uint64()))), err
}

func opEventDataCopy7906(_ *uint64, evm *EVM, _ *Contract, mem *Memory, stack *Stack) ([]byte, error) {
	data, err := eventData7906(evm, stack)
	if err != nil {
		return nil, err
	}
	stack.Pop()
	destination, offset, length := stack.Pop(), stack.Pop(), stack.Pop()
	if length.Sign() != 0 {
		mem.Set(destination.Uint64(), length.Uint64(), data[offset.Uint64():offset.Uint64()+length.Uint64()])
	}
	return nil, nil
}
