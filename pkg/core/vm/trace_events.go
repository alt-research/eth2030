package vm

import (
	"errors"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"math/big"
)

type TraceTransaction struct {
	From      types.Address
	To        *types.Address
	Value     *big.Int
	Input     []byte
	Gas       uint64
	Amsterdam bool
	Reservoir uint64
}
type TraceGasUsage struct {
	UsedGas        uint64
	RegularGasUsed uint64
	StateGasUsed   uint64
	GasRefund      uint64
	Amsterdam      bool
}
type TransactionLogger interface {
	CaptureTransactionStart(TraceTransaction)
	CaptureTransactionEnd(TraceGasUsage, []byte, error, int)
}
type TraceFrameEntry struct {
	Depth     int
	Type      OpCode
	From      types.Address
	To        types.Address
	Value     *big.Int
	Input     []byte
	Gas       uint64
	Reservoir uint64
	Amsterdam bool
}
type FrameLogger interface {
	CaptureEnter(TraceFrameEntry)
	CaptureExit(int, []byte, uint64, error)
}
type OpcodeLoggerV2 interface {
	CaptureStateV2(uint64, OpCode, uint64, uint64, int64, uint64, *Stack, *Memory, int, error)
}
type LogLogger interface{ CaptureLog(*types.Log) }

func (evm *EVM) traceFrame(typ OpCode, from, to types.Address, input []byte, gas uint64, value *big.Int) func([]byte, uint64, error) {
	if !evm.Config.Debug {
		return nil
	}
	logger, ok := evm.Config.Tracer.(FrameLogger)
	if !ok {
		return nil
	}
	reservoir := evm.callReservoirForward
	if evm.depth == 0 && evm.initialStateGas != nil {
		reservoir = evm.initialStateGas.StateGasReservoir
	}
	if typ == SELFDESTRUCT {
		reservoir = 0
	}
	depth := evm.depth
	logger.CaptureEnter(TraceFrameEntry{Depth: depth, Type: typ, From: from, To: to, Value: value, Input: input, Gas: gas, Reservoir: reservoir, Amsterdam: evm.forkRules.IsAmsterdam})
	return func(output []byte, left uint64, err error) { logger.CaptureExit(depth, output, left, err) }
}
func (evm *EVM) traceLog(log *types.Log) {
	if evm.Config.Debug && log != nil {
		if logger, ok := evm.Config.Tracer.(LogLogger); ok {
			logger.CaptureLog(log)
		}
	}
}

// Match CREATE's actual precheck boundary; metadata reads make no state changes.
func (evm *EVM) pendingTraceCreation(kind OpCode, contract *Contract, stack *Stack, memory *Memory) uint64 {
	if evm.readOnly || evm.StateDB == nil || evm.depth > evm.Config.MaxCallDepth {
		return 0
	}
	value := stack.Back(0)
	if evm.StateDB.GetBalance(contract.Address).Cmp(value) < 0 {
		return 0
	}
	nonce := evm.StateDB.GetNonce(contract.Address)
	if nonce == ^uint64(0) {
		return 0
	}
	address := types.Address{}
	if kind == CREATE {
		address = createAddress(contract.Address, nonce)
	} else {
		offset, size := stack.Back(1).Uint64(), stack.Back(2).Uint64()
		code := make([]byte, size)
		if offset < uint64(memory.Len()) {
			copy(code, memory.Data()[offset:min(offset+size, uint64(memory.Len()))])
		}
		address = create2Address(contract.Address, stack.Back(3), crypto.Keccak256(code))
	}
	if evm.StateDB.Empty(address) {
		return AmsterdamNewAccountStateGas
	}
	return 0
}

func (evm *EVM) pendingTraceStorage(contract *Contract, stack *Stack) int64 {
	if evm.readOnly || evm.StateDB == nil {
		return 0
	}
	key, value := bigToHash(stack.Back(0)), bigToHash(stack.Back(1))
	original, current := evm.StateDB.GetCommittedState(contract.Address, key), evm.StateDB.GetState(contract.Address, key)
	if original != (types.Hash{}) {
		return 0
	}
	if current == (types.Hash{}) && value != (types.Hash{}) {
		amount := AmsterdamNewSlotStateGas
		if amount > contract.StateGasReservoir && amount-contract.StateGasReservoir > contract.Gas {
			return 0
		}
		return int64(amount)
	}
	if current != (types.Hash{}) && value == (types.Hash{}) {
		return -int64(AmsterdamNewSlotStateGas)
	}
	return 0
}

type OpcodeContextLogger interface {
	CaptureOpcodeContext(*EVM, *Contract, OpCode, *Stack)
	CaptureOpcodeResult(*EVM, *Contract, OpCode, error)
}

// Trace collection is request-owned. Capacity failure aborts the request rather
// than publishing a partial tree; the caller's explicit step limit is separate.
const traceCaptureBudget uint64 = 2 * MaxMemorySize

var ErrTraceCaptureLimit = errors.New("trace capture exceeds client memory limit")

type TraceFailureLogger interface{ TraceError() error }

func reserveTraceBytes(used *uint64, amount uint64) bool {
	if amount > traceCaptureBudget || *used > traceCaptureBudget-amount {
		return false
	}
	*used += amount
	return true
}
