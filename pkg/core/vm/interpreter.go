package vm

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	"github.com/eth2030/eth2030/rlp"
	"github.com/eth2030/eth2030/zkvm"
)

var (
	ErrOutOfGas                = errors.New("out of gas")
	ErrStackOverflow           = errors.New("stack overflow")
	ErrStackUnderflow          = errors.New("stack underflow")
	ErrInvalidJump             = errors.New("invalid jump destination")
	ErrWriteProtection         = errors.New("write protection")
	ErrExecutionReverted       = errors.New("execution reverted")
	ErrMaxCallDepthExceeded    = errors.New("max call depth exceeded")
	ErrInvalidOpCode           = errors.New("invalid opcode")
	ErrReturnDataOutOfBounds   = errors.New("return data out of bounds")
	ErrMaxInitCodeSizeExceeded = errors.New("max initcode size exceeded")
)

// GetHashFunc returns the hash of the block with the given number.
type GetHashFunc func(uint64) types.Hash

// BlockContext provides the EVM with block-level information.
type BlockContext struct {
	GetHash     GetHashFunc
	BlockNumber *big.Int
	ChainID     *big.Int
	Time        uint64
	Coinbase    types.Address
	GasLimit    uint64
	BaseFee     *big.Int
	PrevRandao  types.Hash
	BlobBaseFee *big.Int
	SlotNumber  uint64 // EIP-7843: beacon chain slot number
}

// TxContext provides the EVM with transaction-level information.
type TxContext struct {
	Origin     types.Address
	GasPrice   *big.Int
	BlobHashes []types.Hash
}

// StateDB provides the EVM with access to Ethereum world state.
// This interface is defined in the vm package to avoid circular imports
// with core/state. Any implementation of core/state.StateDB satisfies it.
type StateDB interface {
	// Account operations
	CreateAccount(addr types.Address)
	GetBalance(addr types.Address) *big.Int
	AddBalance(addr types.Address, amount *big.Int)
	SubBalance(addr types.Address, amount *big.Int)
	GetNonce(addr types.Address) uint64
	SetNonce(addr types.Address, nonce uint64)
	GetCode(addr types.Address) []byte
	SetCode(addr types.Address, code []byte)
	GetCodeHash(addr types.Address) types.Hash

	GetCodeSize(addr types.Address) int

	// Storage
	GetState(addr types.Address, key types.Hash) types.Hash
	SetState(addr types.Address, key types.Hash, value types.Hash)
	GetCommittedState(addr types.Address, key types.Hash) types.Hash

	// Transient storage (EIP-1153)
	GetTransientState(addr types.Address, key types.Hash) types.Hash
	SetTransientState(addr types.Address, key types.Hash, value types.Hash)
	ClearTransientStorage()

	// Self-destruct
	SelfDestruct(addr types.Address)
	HasSelfDestructed(addr types.Address) bool

	// Account existence
	Exist(addr types.Address) bool
	Empty(addr types.Address) bool

	// Snapshot and revert
	Snapshot() int
	RevertToSnapshot(id int)

	// Logs
	AddLog(log *types.Log)

	// Refund counter (EIP-3529)
	AddRefund(gas uint64)
	SubRefund(gas uint64)
	GetRefund() uint64

	// Access list (EIP-2929 warm/cold tracking)
	AddAddressToAccessList(addr types.Address)
	AddSlotToAccessList(addr types.Address, slot types.Hash)
	AddressInAccessList(addr types.Address) bool
	SlotInAccessList(addr types.Address, slot types.Hash) (addressOk bool, slotOk bool)
}

// Config holds EVM configuration options.
type Config struct {
	OnFrameLegacyNonceRead       func(types.Address) // optional isolated-profile dependency instrumentation
	FrameObserver                FrameObserver       // optional bounded per-frame simulation observer
	ExperimentalFramePaymasterV1 bool                // local research template; not upstream canonical code
	Context                      context.Context     // optional cancellation for isolated RPC execution
	Debug                        bool
	Tracer                       EVMLogger
	MaxCallDepth                 int
}

// EVM is the Ethereum Virtual Machine execution environment.
type EVM struct {
	createdContracts     map[types.Address]bool
	createdOrder         []types.Address
	Context              BlockContext
	TxContext            TxContext
	Config               Config
	StateDB              StateDB
	depth                int
	readOnly             bool
	jumpTable            JumpTable
	precompiles          map[types.Address]PrecompiledContract
	returnData           []byte // return data from the last CALL/CREATE
	callGasTemp          uint64 // temporary storage for CALL gas (set by dynamic gas, read by opCall)
	callReservoirForward uint64 // reservoir forwarded to child call (set by opCall, read by evm.Call)
	callReservoirReturn  uint64 // reservoir returned from child call (set by evm.Call, read by opCall)
	initialStateGas      *Contract
	returnedStateGas     *Contract
	activeContract       *Contract
	dimGasUsage          *TxDimGasUsage     // per-dimension gas tracking (GAP-2.1, nil if not active)
	witnessGas           *WitnessGasTracker // EIP-4762: witness gas tracking (nil if not Verkle)
	balTracker           BALTracker         // EIP-7928: BAL state access tracking (nil if not active)
	txIndex              uint64             // EIP-7928: current transaction index for BAL
	forkRules            ForkRules          // active fork rules for this block
	frame8141            *FrameStateGas
	frameValidationErr   error
	FrameCtx             *FrameContext       // EIP-8141: frame transaction approval context (nil if not frame tx)
	guestRegistry        *zkvm.GuestRegistry // EL-3: RISC-V guest registry for RVCREATE (nil if not I+)
}

// NewEVM creates a new EVM instance.
func NewEVM(blockCtx BlockContext, txCtx TxContext, config Config) *EVM {
	if config.MaxCallDepth == 0 {
		config.MaxCallDepth = 1024
	}
	return &EVM{
		Context:   blockCtx,
		TxContext: txCtx,
		Config:    config,
		jumpTable: NewCancunJumpTable(),
	}
}

// NewEVMWithState creates a new EVM instance with state access.
func NewEVMWithState(blockCtx BlockContext, txCtx TxContext, config Config, stateDB StateDB) *EVM {
	evm := NewEVM(blockCtx, txCtx, config)
	evm.StateDB = stateDB
	return evm
}

// SetJumpTable replaces the EVM's jump table. Use SelectJumpTable to pick
// the correct table for a given fork.
func (evm *EVM) SetJumpTable(jt JumpTable) {
	evm.jumpTable = jt
}

// SetPrecompiles replaces the EVM's precompile map.
func (evm *EVM) SetPrecompiles(p map[types.Address]PrecompiledContract) {
	evm.precompiles = p
}

// SetGuestRegistry sets the RISC-V guest registry used by RVCREATE (EL-3).
// Pass nil to disable RISC-V contract routing.
func (evm *EVM) SetGuestRegistry(r *zkvm.GuestRegistry) {
	evm.guestRegistry = r
}

// SetForkRules sets the active fork rules for this EVM instance.
func (evm *EVM) SetForkRules(rules ForkRules) {
	evm.forkRules = rules
}

// GetForkRules returns the active fork rules.
func (evm *EVM) GetForkRules() ForkRules {
	return evm.forkRules
}

// SetDimGasUsage attaches per-dimension gas usage tracking to this EVM.
// When non-nil, gasSstoreGlamst routes SSTORE state-creation premiums to
// DimStorage and records all SSTORE ops in the tracker (GAP-2.1).
func (evm *EVM) SetDimGasUsage(u *TxDimGasUsage) {
	evm.dimGasUsage = u
}

// DimGasUsage returns the per-dimension gas tracker, or nil if not set.
func (evm *EVM) DimGasUsage() *TxDimGasUsage {
	return evm.dimGasUsage
}

// SetInitialReservoir seeds the state-creation gas reservoir forwarded to
// the first contract call. Call this after NewEVM before evm.Call/Create
// so the top-level contract receives the reservoir (GAP-1.2/1.3).
func (evm *EVM) SetInitialReservoir(reservoir uint64) {
	evm.callReservoirForward = reservoir
}

// SetWitnessGasTracker enables EIP-4762 witness gas tracking. When set, the
// Verkle jump table charges gas based on witness size for state accesses.
func (evm *EVM) SetWitnessGasTracker(t *WitnessGasTracker) {
	evm.witnessGas = t
}

// GetWitnessGasTracker returns the current witness gas tracker (may be nil).
func (evm *EVM) GetWitnessGasTracker() *WitnessGasTracker {
	return evm.witnessGas
}

// SetBALTracker enables EIP-7928 Block Access List tracking. When set,
// EVM opcodes that access state will record accesses to the tracker.
func (evm *EVM) SetBALTracker(t BALTracker, txIdx uint64) {
	evm.balTracker = t
	evm.txIndex = txIdx
}

// GetBALTracker returns the current BAL tracker (may be nil).
func (evm *EVM) GetBALTracker() BALTracker {
	return evm.balTracker
}

// precompile returns the precompiled contract at addr, falling back to the
// default Cancun precompile set if no custom map has been set.
func (evm *EVM) precompile(addr types.Address) (PrecompiledContract, bool) {
	m := evm.precompiles
	if m == nil {
		m = PrecompiledContractsCancun
	}
	p, ok := m[addr]
	return p, ok
}

// runPrecompile executes a precompiled contract and returns the output,
// remaining gas, and any error.
func runPrecompile(p PrecompiledContract, input []byte, gas uint64) ([]byte, uint64, error) {
	gasCost := p.RequiredGas(input)
	if gas < gasCost {
		return nil, 0, ErrOutOfGas
	}
	output, err := p.Run(input)
	if err != nil {
		return nil, 0, err
	}
	return output, gas - gasCost, err
}

// ForkRules mirrors the chain configuration fork flags needed to select
// the correct jump table. The caller (processor) converts ChainConfig.Rules
// into this struct to avoid a circular import.
type ForkRules struct {
	IsVerkle         bool // EIP-4762: statelessness gas cost changes
	IsGlamsterdan    bool
	IsLegacyFrames   bool
	IsAmsterdam      bool
	IsPrague         bool
	IsOsaka          bool
	IsCancun         bool
	IsShanghai       bool
	IsMerge          bool
	IsLondon         bool
	IsBerlin         bool
	IsIstanbul       bool
	IsConstantinople bool
	IsByzantium      bool
	IsHomestead      bool
	IsEIP158         bool // EIP-158: empty account cleanup
	IsEIP7708        bool // EIP-7708: ETH transfers emit a log
	IsEIP7954        bool // EIP-7954: increased max contract code size
	IsIPlus          bool // I+ fork: NTT/NII precompiles (EIP-7885)
	LegacyNativeBLS  bool
}

// SelectPrecompiles returns the correct precompile map for the given fork rules.
func SelectPrecompiles(rules ForkRules) map[types.Address]PrecompiledContract {
	selected := selectPrecompileBase(rules)
	if !rules.LegacyNativeBLS && !rules.IsOsaka && !rules.IsAmsterdam && !rules.IsGlamsterdan && !rules.IsIPlus {
		selected = maps.Clone(selected)
		delete(selected, types.Address{18: 1}) // EIP-7951 activates at Osaka, not Cancun/Prague.
	}
	if rules.IsOsaka {
		selected = maps.Clone(selected)
		selected[types.Address{19: 5}] = &osakaModExp{}
	}
	return selected
}

func selectPrecompileBase(rules ForkRules) map[types.Address]PrecompiledContract {
	if rules.LegacyNativeBLS {
		switch {
		case rules.IsIPlus:
			return PrecompiledContractsIPlus
		case rules.IsGlamsterdan:
			return PrecompiledContractsGlamsterdan
		default:
			return PrecompiledContractsCancun
		}
	}
	switch {
	case rules.IsIPlus:
		return precompiledContractsIPlusStandardBLS
	case rules.IsGlamsterdan:
		return precompiledContractsGlamsterdanStandardBLS
	case rules.IsPrague:
		return PrecompiledContractsPrague
	default:
		return precompiledContractsCancunStandardBLS
	}
}

// SelectJumpTable returns the correct jump table for the given fork rules.
func SelectJumpTable(rules ForkRules) JumpTable {
	table := selectBaseJumpTable(rules)
	if rules.IsLegacyFrames {
		table = enableLegacyFrameOpcodes(table)
	}
	return table
}

func selectBaseJumpTable(rules ForkRules) JumpTable {
	switch {
	case rules.IsIPlus:
		return NewIPlusJumpTable()
	case rules.IsVerkle:
		return NewVerkleJumpTable()
	case rules.IsGlamsterdan:
		table := NewGlamsterdanJumpTable()
		if rules.IsAmsterdam {
			installAmsterdamGas(&table)
		}
		return table
	case rules.IsAmsterdam:
		return NewAmsterdamJumpTable()
	case rules.IsOsaka:
		return NewOsakaJumpTable()
	case rules.IsPrague:
		return NewPragueJumpTable()
	case rules.IsCancun:
		return NewCancunJumpTable()
	case rules.IsShanghai:
		return NewShanghaiJumpTable()
	case rules.IsMerge:
		return NewMergeJumpTable()
	case rules.IsLondon:
		return NewLondonJumpTable()
	case rules.IsBerlin:
		return NewBerlinJumpTable()
	case rules.IsIstanbul:
		return NewIstanbulJumpTable()
	case rules.IsConstantinople:
		return NewConstantinopleJumpTable()
	case rules.IsByzantium:
		return NewByzantiumJumpTable()
	case rules.IsHomestead:
		return NewHomesteadJumpTable()
	default:
		return NewFrontierJumpTable()
	}
}

// Run executes the contract bytecode using the interpreter loop.
// Gas charging order follows go-ethereum: constant gas -> dynamic gas
// (which includes memory expansion cost) -> resize memory -> execute.
func (evm *EVM) Run(contract *Contract, input []byte) (result []byte, runError error) {
	previousContract := evm.activeContract
	evm.activeContract = contract
	defer func() { evm.activeContract = previousContract }()
	contract.Input = input
	if evm.frame8141 != nil {
		evm.returnData = nil
	}

	var (
		pc    uint64
		stack = NewStack()
		mem   = NewMemory()
		debug = evm.Config.Debug && evm.Config.Tracer != nil
	)

	var tracePC, traceGas, traceReservoir uint64
	var traceOp OpCode
	traceLogged := true
	if debug {
		defer func() {
			if runError != nil && !traceLogged {
				if observer, ok := evm.Config.Tracer.(OpcodeLoggerV2); ok {
					if context, ok := evm.Config.Tracer.(OpcodeContextLogger); ok {
						context.CaptureOpcodeContext(evm, contract, traceOp, stack)
					}
					observer.CaptureStateV2(tracePC, traceOp, traceGas, 0, 0, traceReservoir, stack, mem, evm.depth, runError)
				}
			}
		}()
	}
	var steps uint64
	cancellation := evm.Config.Context
	for {
		if cancellation != nil {
			if steps&1023 == 0 {
				if err := cancellation.Err(); err != nil {
					return nil, err
				}
			}
			steps++
		}
		op := contract.GetOp(pc)
		if debug {
			tracePC, traceOp, traceGas, traceReservoir, traceLogged = pc, op, contract.Gas, contract.StateGasReservoir, false
		}
		operation := evm.jumpTable[op]
		if operation == nil || operation.execute == nil {
			// INVALID has no dispatch entry, but is still a banned validation
			// opcode even if a parent catches the child's exceptional halt.
			if op == INVALID {
				if err := evm.checkFrameValidation(op, contract, stack, pc); err != nil {
					evm.frameValidationErr = err
				}
			}
			return nil, ErrInvalidOpCode
		}

		// Stack validation
		sLen := stack.Len()
		if sLen < operation.minStack {
			return nil, ErrStackUnderflow
		}
		if sLen > operation.maxStack {
			return nil, ErrStackOverflow
		}

		if err := evm.checkFrameValidation(op, contract, stack, pc); err != nil {
			evm.frameValidationErr = err
			return nil, err
		}

		// EIP-214 forbids the write before memory/access/state metering. CALLCODE
		// may carry value because it does not transfer value to another account.
		if evm.readOnly && (operation.writes || (op == CALL && stack.Back(2).Sign() != 0)) {
			return nil, ErrWriteProtection
		}

		// Calculate total gas cost for this step (for tracing).
		var stepCost uint64
		executionCost := operation.constantGas
		stateBefore, reservoirBefore := contract.StateGasUsed, contract.StateGasReservoir
		gasBefore := contract.Gas

		// Constant gas deduction
		if operation.constantGas > 0 {
			if !contract.UseGas(operation.constantGas) {
				return nil, ErrOutOfGas
			}
		}

		// Calculate required memory size (but don't resize yet).
		var memorySize uint64
		if operation.memorySize != nil {
			memSize, overflow := operation.memorySize(stack)
			if overflow {
				return nil, ErrOutOfGas
			}
			// Align to 32-byte words.
			if memSize > 0 {
				if memSize > math.MaxUint64-31 {
					return nil, ErrOutOfGas
				}
				memorySize = (memSize + 31) / 32 * 32
			}
		}

		// Dynamic gas: includes memory expansion cost + operation-specific costs.
		// This is charged BEFORE memory is actually resized.
		if operation.dynamicGas != nil {
			if evm.witnessGas != nil {
				evm.witnessGas.staging = true
			}
			cost, err := operation.dynamicGas(evm, contract, stack, mem, memorySize)
			paid := err == nil && contract.UseGas(cost)
			if paid {
				executionCost += cost
			}
			if evm.witnessGas != nil {
				evm.witnessGas.finishGasCalculation(paid)
				if !paid {
					contract.Gas = 0
				}
			}
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
			}
			if !paid {
				return nil, ErrOutOfGas
			}
		}

		// Compute the total cost for this step (difference before/after gas charging).
		stepCost = gasBefore - contract.Gas

		// Trace: capture state before executing the opcode.
		if debug {
			if context, ok := evm.Config.Tracer.(OpcodeContextLogger); ok {
				context.CaptureOpcodeContext(evm, contract, op, stack)
			}
			if observer, ok := evm.Config.Tracer.(OpcodeLoggerV2); ok {
				stateCost, reservoir := int64(0), uint64(0)
				if evm.forkRules.IsAmsterdam {
					stateCost = contract.StateGasUsed - stateBefore
					if op == SSTORE {
						stateCost += evm.pendingTraceStorage(contract, stack)
					}
					reservoir = reservoirBefore
					if op == CREATE || op == CREATE2 {
						stateCost += int64(evm.pendingTraceCreation(op, contract, stack, mem))
					}
				}
				observer.CaptureStateV2(pc, op, gasBefore, executionCost, stateCost, reservoir, stack, mem, evm.depth, nil)
			} else {
				evm.Config.Tracer.CaptureState(pc, op, gasBefore, stepCost, stack, mem, evm.depth, nil)
			}
		}

		// Execute the opcode
		if debug {
			if logger, ok := evm.Config.Tracer.(TraceFailureLogger); ok && logger.TraceError() != nil {
				return nil, logger.TraceError()
			}
			traceLogged = true
		}
		if memorySize > 0 && uint64(mem.Len()) < memorySize {
			mem.Resize(memorySize)
		}
		ret, err := operation.execute(&pc, evm, contract, mem, stack)
		if debug {
			if context, ok := evm.Config.Tracer.(OpcodeContextLogger); ok {
				context.CaptureOpcodeResult(evm, contract, op, err)
			}
		}

		if debug {
			if logger, ok := evm.Config.Tracer.(TraceFailureLogger); ok && logger.TraceError() != nil {
				return nil, logger.TraceError()
			}
		}

		if err != nil {
			if errors.Is(err, ErrExecutionReverted) {
				return ret, err
			}
			return nil, err
		}

		// Handle halting opcodes
		if operation.halts {
			return ret, nil
		}
		if operation.jumps {
			continue
		}

		pc++
	}
}

// Call executes a message call to the given address with the given input, gas, and value.
func (evm *EVM) Call(caller types.Address, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, gasLeft uint64, err error) {
	evm.returnData = nil // EIP-211: a new call-like operation starts with an empty buffer.

	if trace := evm.traceFrame(CALL, caller, addr, input, gas, value); trace != nil {
		defer func() { trace(ret, gasLeft, err) }()
	}
	if evm.depth > evm.Config.MaxCallDepth {
		return nil, gas, ErrMaxCallDepthExceeded
	}

	debug := evm.Config.Debug && evm.Config.Tracer != nil

	// Notify tracer at the top-level call (depth 0).
	if debug && evm.depth == 0 {
		evm.Config.Tracer.CaptureStart(caller, addr, false, input, gas, value)
	}

	// Check if the callee has sufficient balance for value transfer.
	transfersValue := value != nil && value.Sign() > 0
	if transfersValue && evm.StateDB != nil {
		callerBalance := evm.StateDB.GetBalance(caller)
		if callerBalance.Cmp(value) < 0 {
			if debug && evm.depth == 0 {
				evm.Config.Tracer.CaptureEnd(nil, 0, errors.New("insufficient balance for transfer"))
			}
			return nil, gas, errors.New("insufficient balance for transfer")
		}
	}

	if evm.StateDB == nil {
		return nil, gas, errors.New("no state database")
	}

	// Snapshot state for revert on failure.
	meter := evm.beginStateGasFrame(caller, addr, value, gas)
	if meter != nil {
		defer func() { evm.finishStateGasFrame(meter, &gasLeft, err) }()
	}
	snapshot := evm.StateDB.Snapshot()
	createdSnapshot := len(evm.createdOrder)

	if evm.frame8141 != nil && transfersValue && evm.depth > 0 && evm.StateDB.Empty(addr) {
		if err := evm.frame8141.Charge(frameNewAccountGas); err != nil {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
			return nil, gas, ErrFrameStateGas
		}
	}

	// Check for precompiled contract.
	p, isPrecompile := evm.precompile(addr)

	// Handle account creation / EIP-158 empty account rule.
	if !evm.StateDB.Exist(addr) {
		if !isPrecompile && evm.forkRules.IsEIP158 && !transfersValue {
			// EIP-158: do not create empty accounts for zero-value calls.
			if debug && evm.depth == 0 {
				evm.Config.Tracer.CaptureEnd(nil, 0, nil)
			}
			return nil, gas, nil
		}
		if evm.frame8141 == nil || !isPrecompile || transfersValue {
			evm.StateDB.CreateAccount(addr)
		}
	}

	// Transfer value (before running precompile or code).
	if transfersValue {
		if evm.readOnly {
			return nil, gas, ErrWriteProtection
		}
		// EIP-7928: capture pre-transfer balances for BAL tracking.
		var callerPreBal, addrPreBal *big.Int
		if evm.balTracker != nil {
			callerPreBal = new(big.Int).Set(evm.StateDB.GetBalance(caller))
			addrPreBal = new(big.Int).Set(evm.StateDB.GetBalance(addr))
		}
		evm.StateDB.SubBalance(caller, value)
		evm.StateDB.AddBalance(addr, value)
		// EIP-7928: record balance changes for BAL tracking.
		if evm.balTracker != nil {
			evm.balTracker.RecordBalanceChange(caller, callerPreBal, evm.StateDB.GetBalance(caller))
			if caller != addr {
				evm.balTracker.RecordBalanceChange(addr, addrPreBal, evm.StateDB.GetBalance(addr))
			}
		}

		// EIP-7708: emit transfer log for nonzero-value CALL to a different account.
		evm.recordETHTransfer(caller, addr, value)
	}

	if !transfersValue {
		evm.StateDB.AddBalance(addr, new(big.Int))
	}

	// Execute precompile or contract code.
	if isPrecompile {
		ret, gasLeft, err := runPrecompile(p, input, gas)
		if err != nil {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
		}
		if debug && evm.depth == 0 {
			evm.Config.Tracer.CaptureEnd(ret, gas-gasLeft, err)
		}
		return ret, gasLeft, err
	}

	// Get the code to execute.
	code := evm.StateDB.GetCode(addr)
	// EIP-7702: a delegated EOA's "code" is the 0xef0100 || delegate
	// designator. Resolve it and execute the delegate's code in the EOA's
	// execution context (storage/balance stay with the EOA). The delegation
	// is followed exactly once — resolved code is never re-resolved.
	if types.HasDelegationPrefix(code) {
		if delegate, ok := types.ParseDelegation(code); ok {
			if evm.frame8141 != nil && evm.depth == 0 {
				cost := uint64(100)
				if !evm.StateDB.AddressInAccessList(delegate) {
					cost = 3000
					evm.StateDB.AddAddressToAccessList(delegate)
				}
				if gas < cost {
					evm.revertCreationSnapshot(snapshot, createdSnapshot)
					return nil, 0, ErrOutOfGas
				}
				gas -= cost
			}
			if evm.balTracker != nil {
				evm.balTracker.RecordAddressTouch(delegate)
			}
			code = evm.StateDB.GetCode(delegate)
		}
	}
	if len(code) == 0 {
		// No code to execute, the call succeeds with no return data.
		if debug && evm.depth == 0 {
			evm.Config.Tracer.CaptureEnd(nil, 0, nil)
		}
		return nil, gas, nil
	}

	// EL-3.3: if code starts with RISC-V magic bytes and I+ fork is active,
	// route execution through the RISC-V CPU emulator instead of the EVM.
	if evm.forkRules.IsIPlus && IsRVCode(code) {
		ret, gasLeft, rvErr := runRVContract(code, input, gas)
		if rvErr != nil {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
		}
		if debug && evm.depth == 0 {
			evm.Config.Tracer.CaptureEnd(ret, gas-gasLeft, rvErr)
		}
		return ret, gasLeft, rvErr
	}

	// Create the contract for execution.
	contract := NewContract(caller, addr, value, gas)
	if meter != nil {
		contract = meter
	}
	contract.Code = code
	contract.CodeHash = evm.StateDB.GetCodeHash(addr)

	// GAP-1.2: forward parent's state-creation reservoir to child (full amount, not 63/64).
	if meter == nil {
		childReservoir := evm.callReservoirForward
		evm.callReservoirForward = 0
		contract.StateGasReservoir = childReservoir
	}

	// Execute.
	evm.depth++
	ret, err = evm.Run(contract, input)
	evm.depth--

	gasLeft = contract.Gas
	// GAP-1.2: return unused child reservoir to caller via callReservoirReturn.
	evm.callReservoirReturn = contract.StateGasReservoir
	if err != nil && !errors.Is(err, ErrExecutionReverted) {
		evm.callReservoirReturn = 0
	}

	if err != nil && !errors.Is(err, ErrExecutionReverted) {
		// On error (not revert), revert state changes and consume all gas.
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		gasLeft = 0
	} else if errors.Is(err, ErrExecutionReverted) {
		// On revert, revert state changes but return remaining gas.
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
	}

	if debug && evm.depth == 0 {
		evm.Config.Tracer.CaptureEnd(ret, gas-gasLeft, err)
	}

	return ret, gasLeft, err
}

// CallCode executes a CALLCODE operation. Runs the callee's code in the caller's context.
func (evm *EVM) CallCode(caller types.Address, addr types.Address, input []byte, gas uint64, value *big.Int) (ret []byte, gasLeft uint64, err error) {
	evm.returnData = nil // EIP-211: a new call-like operation starts with an empty buffer.

	if trace := evm.traceFrame(CALLCODE, caller, addr, input, gas, value); trace != nil {
		defer func() { trace(ret, gasLeft, err) }()
	}
	if evm.depth > evm.Config.MaxCallDepth {
		return nil, gas, ErrMaxCallDepthExceeded
	}

	if evm.StateDB != nil && value != nil && value.Sign() > 0 && evm.StateDB.GetBalance(caller).Cmp(value) < 0 {
		return nil, gas, ErrInsufficientBalance
	}
	meter := evm.beginStateGasFrame(caller, caller, value, gas)
	if meter != nil {
		defer func() { evm.finishStateGasFrame(meter, &gasLeft, err) }()
	}
	if p, ok := evm.precompile(addr); ok {
		ret, gasLeft, err := runPrecompile(p, input, gas)
		return ret, gasLeft, err
	}

	if evm.StateDB == nil {
		return nil, gas, errors.New("no state database")
	}

	snapshot := evm.StateDB.Snapshot()
	createdSnapshot := len(evm.createdOrder)

	// Get the code to execute from the target address
	code := evm.executionCode(addr)
	if len(code) == 0 {
		return nil, gas, nil
	}

	// CALLCODE executes the callee's code but in the caller's context
	// (caller's address is used for storage and msg.sender)
	contract := NewContract(caller, caller, value, gas)
	if meter != nil {
		contract = meter
	}
	contract.Code = code
	contract.CodeHash = evm.StateDB.GetCodeHash(addr)

	evm.depth++
	ret, err = evm.Run(contract, input)
	evm.depth--

	gasLeft = contract.Gas

	if err != nil && !errors.Is(err, ErrExecutionReverted) {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		gasLeft = 0
	} else if errors.Is(err, ErrExecutionReverted) {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
	}

	return ret, gasLeft, err
}

// DelegateCall executes a DELEGATECALL operation.
// Like CALLCODE but preserves the original caller and value.
func (evm *EVM) DelegateCall(caller types.Address, addr types.Address, input []byte, gas uint64) ([]byte, uint64, error) {
	return evm.delegateCall(caller, caller, nil, addr, input, gas)
}

func (evm *EVM) delegateCall(caller, executionAddr types.Address, value *big.Int, addr types.Address, input []byte, gas uint64) (ret []byte, gasLeft uint64, err error) {
	evm.returnData = nil // EIP-211: a new call-like operation starts with an empty buffer.

	if trace := evm.traceFrame(DELEGATECALL, executionAddr, addr, input, gas, value); trace != nil {
		defer func() { trace(ret, gasLeft, err) }()
	}
	if evm.depth > evm.Config.MaxCallDepth {
		return nil, gas, ErrMaxCallDepthExceeded
	}

	meter := evm.beginStateGasFrame(caller, executionAddr, value, gas)
	if meter != nil {
		defer func() { evm.finishStateGasFrame(meter, &gasLeft, err) }()
	}
	if p, ok := evm.precompile(addr); ok {
		ret, gasLeft, err := runPrecompile(p, input, gas)
		return ret, gasLeft, err
	}

	if evm.StateDB == nil {
		return nil, gas, errors.New("no state database")
	}

	snapshot := evm.StateDB.Snapshot()
	createdSnapshot := len(evm.createdOrder)

	code := evm.executionCode(addr)
	if len(code) == 0 {
		return nil, gas, nil
	}

	// DELEGATECALL preserves the caller (msg.sender) and value from the current context.
	// Storage operations happen in the parent contract's execution context.
	contract := NewContract(caller, executionAddr, value, gas)
	if meter != nil {
		contract = meter
	}
	contract.Code = code
	contract.CodeHash = evm.StateDB.GetCodeHash(addr)

	evm.depth++
	ret, err = evm.Run(contract, input)
	evm.depth--

	gasLeft = contract.Gas

	if err != nil && !errors.Is(err, ErrExecutionReverted) {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		gasLeft = 0
	} else if errors.Is(err, ErrExecutionReverted) {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
	}

	return ret, gasLeft, err
}

// StaticCall executes a read-only message call. Any state modifications will cause an error.
func (evm *EVM) StaticCall(caller types.Address, addr types.Address, input []byte, gas uint64) (ret []byte, gasLeft uint64, err error) {
	evm.returnData = nil // EIP-211: a new call-like operation starts with an empty buffer.

	if trace := evm.traceFrame(STATICCALL, caller, addr, input, gas, nil); trace != nil {
		defer func() { trace(ret, gasLeft, err) }()
	}
	if evm.depth > evm.Config.MaxCallDepth {
		return nil, gas, ErrMaxCallDepthExceeded
	}

	if evm.StateDB == nil {
		return nil, gas, errors.New("no state database")
	}

	// Set readOnly mode for the duration of this call.
	prevReadOnly := evm.readOnly
	evm.readOnly = true
	defer func() { evm.readOnly = prevReadOnly }()

	// We take a snapshot here. Even a staticcall is considered a 'touch'.
	// On mainnet, static calls were introduced after all empty accounts
	// were deleted, so this is not required. However, certain tests (e.g.
	// stRevertTest/RevertPrecompiledTouchExactOOG) require this behavior.
	meter := evm.beginStateGasFrame(caller, addr, new(big.Int), gas)
	if meter != nil {
		defer func() { evm.finishStateGasFrame(meter, &gasLeft, err) }()
	}
	snapshot := evm.StateDB.Snapshot()
	createdSnapshot := len(evm.createdOrder)

	evm.StateDB.AddBalance(addr, new(big.Int))

	// Check for precompiled contract.
	if p, ok := evm.precompile(addr); ok {
		ret, gasLeft, err := runPrecompile(p, input, gas)
		if err != nil {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
		}
		return ret, gasLeft, err
	}

	code := evm.executionCode(addr)
	if len(code) == 0 {
		return nil, gas, nil
	}

	contract := NewContract(caller, addr, new(big.Int), gas)
	if meter != nil {
		contract = meter
	}
	contract.Code = code
	contract.CodeHash = evm.StateDB.GetCodeHash(addr)

	evm.depth++
	ret, err = evm.Run(contract, input)
	evm.depth--

	gasLeft = contract.Gas

	if err != nil && !errors.Is(err, ErrExecutionReverted) {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		gasLeft = 0
	} else if errors.Is(err, ErrExecutionReverted) {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
	}

	return ret, gasLeft, err
}

// createAddress computes the address of a contract created with CREATE.
// Per the Yellow Paper: addr = keccak256(rlp([sender, nonce]))[12:]
func createAddress(caller types.Address, nonce uint64) types.Address {
	// Both fixed scalar types are supported by the canonical RLP encoder.
	encoded, _ := rlp.EncodeToBytes([]interface{}{caller, nonce})
	hash := crypto.Keccak256(encoded)
	return types.BytesToAddress(hash[12:])
}

// create2Address computes the address of a contract created with CREATE2.
func create2Address(caller types.Address, salt *big.Int, initCodeHash []byte) types.Address {
	// CREATE2 address = keccak256(0xff + caller + salt + keccak256(initCode))[12:]
	saltBytes := make([]byte, 32)
	if salt != nil {
		b := salt.Bytes()
		copy(saltBytes[32-len(b):], b)
	}
	data := make([]byte, 0, 85)
	data = append(data, 0xff)
	data = append(data, caller[:]...)
	data = append(data, saltBytes...)
	data = append(data, initCodeHash...)
	hash := crypto.Keccak256(data)
	return types.BytesToAddress(hash[12:])
}

// Create creates a new contract with the given code.
func (evm *EVM) Create(caller types.Address, code []byte, gas uint64, value *big.Int) (ret []byte, created types.Address, left uint64, err error) {
	evm.returnData = nil // EIP-211: a new call-like operation starts with an empty buffer.

	if !evm.amsterdamStateGasActive() && !evm.readOnly && evm.Config.Debug {
		if _, ok := evm.Config.Tracer.(FrameLogger); ok {
			address := types.Address{}
			if evm.StateDB != nil {
				address = createAddress(caller, evm.StateDB.GetNonce(caller))
			}
			if finish := evm.traceFrame(CREATE, caller, address, code, gas, value); finish != nil {
				defer func() { finish(ret, left, err) }()
			}
		}
	}
	if evm.depth > evm.Config.MaxCallDepth {
		return nil, types.Address{}, gas, ErrMaxCallDepthExceeded
	}
	if evm.readOnly {
		return nil, types.Address{}, gas, ErrWriteProtection
	}
	if evm.StateDB == nil {
		return nil, types.Address{}, gas, errors.New("no state database")
	}

	if value != nil && evm.StateDB.GetBalance(caller).Cmp(value) < 0 {
		return nil, types.Address{}, gas, ErrInsufficientBalance
	}
	// Compute the new contract address
	nonce := evm.StateDB.GetNonce(caller)
	if nonce == ^uint64(0) {
		return nil, types.Address{}, gas, errors.New("creator nonce overflow")
	}
	evm.StateDB.SetNonce(caller, nonce+1)
	evm.recordCreationNonce(caller, nonce, nonce+1)
	contractAddr := createAddress(caller, nonce)

	return evm.create(caller, code, gas, value, contractAddr, CREATE)
}

// Create2 creates a new contract using CREATE2 with the given salt.
func (evm *EVM) Create2(caller types.Address, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, created types.Address, left uint64, err error) {
	evm.returnData = nil // EIP-211: a new call-like operation starts with an empty buffer.

	if !evm.amsterdamStateGasActive() && !evm.readOnly && evm.Config.Debug {
		if _, ok := evm.Config.Tracer.(FrameLogger); ok {
			address := types.Address{}
			address = create2Address(caller, salt, crypto.Keccak256(code))
			if finish := evm.traceFrame(CREATE2, caller, address, code, gas, endowment); finish != nil {
				defer func() { finish(ret, left, err) }()
			}
		}
	}
	if evm.depth > evm.Config.MaxCallDepth {
		return nil, types.Address{}, gas, ErrMaxCallDepthExceeded
	}
	if evm.readOnly {
		return nil, types.Address{}, gas, ErrWriteProtection
	}
	if evm.StateDB == nil {
		return nil, types.Address{}, gas, errors.New("no state database")
	}

	if endowment != nil && evm.StateDB.GetBalance(caller).Cmp(endowment) < 0 {
		return nil, types.Address{}, gas, ErrInsufficientBalance
	}
	nonce := evm.StateDB.GetNonce(caller)
	if nonce == ^uint64(0) {
		return nil, types.Address{}, gas, errors.New("creator nonce overflow")
	}
	evm.StateDB.SetNonce(caller, nonce+1)
	evm.recordCreationNonce(caller, nonce, nonce+1)
	initCodeHash := crypto.Keccak256(code)
	contractAddr := create2Address(caller, salt, initCodeHash)

	return evm.create(caller, code, gas, endowment, contractAddr, CREATE2)
}

// PreWarmAccessList pre-warms the access list with the sender, recipient, and
// all precompile addresses (0x01-0x0a) per EIP-2929.
func (evm *EVM) PreWarmAccessList(sender types.Address, to *types.Address) {
	if evm.StateDB == nil {
		return
	}
	// Warm the sender.
	evm.StateDB.AddAddressToAccessList(sender)
	// Warm the recipient (if non-nil, i.e. not a contract creation).
	if to != nil {
		evm.StateDB.AddAddressToAccessList(*to)
	}
	// Warm all precompile addresses (0x01 through 0x13).
	// Includes: ecrecover(1), sha256(2), ripemd160(3), identity(4),
	// modexp(5), bn254add(6), bn254mul(7), bn254pairing(8),
	// blake2f(9), kzg(10), and EIP-2537 BLS12-381 (11-19).
	for i := 1; i <= 0x13; i++ {
		evm.StateDB.AddAddressToAccessList(types.BytesToAddress([]byte{byte(i)}))
	}
}

// gasEIP2929AccountCheck checks whether addr is warm. If cold, it warms the
// address and returns the extra cold gas (ColdAccountAccessCost - WarmStorageReadCost).
// If warm, it returns 0. The caller is expected to charge WarmStorageReadCost
// as the constant gas.
func gasEIP2929AccountCheck(evm *EVM, addr types.Address) uint64 {
	if evm.StateDB == nil {
		return 0
	}
	if evm.StateDB.AddressInAccessList(addr) {
		return 0
	}
	evm.StateDB.AddAddressToAccessList(addr)
	return ColdAccountAccessCost - WarmStorageReadCost
}

// gasEIP2929SlotCheck checks whether (addr, slot) is warm. If cold, it warms
// the slot and returns the extra cold gas (ColdSloadCost - WarmStorageReadCost).
// If warm, it returns 0. The caller is expected to charge WarmStorageReadCost
// as the constant gas.
func gasEIP2929SlotCheck(evm *EVM, addr types.Address, slot types.Hash) uint64 {
	if evm.StateDB == nil {
		return 0
	}
	_, slotWarm := evm.StateDB.SlotInAccessList(addr, slot)
	if slotWarm {
		return 0
	}
	evm.StateDB.AddSlotToAccessList(addr, slot)
	return ColdSloadCost - WarmStorageReadCost
}

// create is the shared implementation for Create and Create2.
func (evm *EVM) create(caller types.Address, code []byte, gas uint64, value *big.Int, contractAddr types.Address, kind OpCode) (ret []byte, created types.Address, left uint64, err error) {
	if evm.amsterdamStateGasActive() {
		return evm.createAmsterdam(caller, code, gas, value, contractAddr, kind)
	}
	// EIP-3860 / EIP-7954: max init code size check.
	maxInit := MaxInitCodeSizeForFork(evm.forkRules)
	if len(code) > maxInit {
		return nil, types.Address{}, gas, ErrMaxInitCodeSizeExceeded
	}

	// Only opcode-originated creations reserve parent gas. Top-level creation
	// consumes its entire supplied child budget on an exceptional halt.
	retained := uint64(0)
	if evm.activeContract != nil {
		retained = gas / CallGasFraction
		gas -= retained
	}
	defer func() { left += retained }()

	if evm.frame8141 != nil && evm.frame8141.PublicValidation && contractAddr != evm.frame8141.Tx.Sender {
		evm.frameValidationErr = errors.New("Frame deployment created an account other than sender")
		return nil, types.Address{}, gas, evm.frameValidationErr
	}
	// Collision check: fail if address already has non-zero nonce or non-empty code.
	// Per go-ethereum, all gas is consumed on collision.
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(contractAddr)
	}
	evm.StateDB.AddAddressToAccessList(contractAddr)
	contractHash := evm.StateDB.GetCodeHash(contractAddr)
	if evm.StateDB.GetNonce(contractAddr) != 0 ||
		(contractHash != (types.Hash{}) && contractHash != types.EmptyCodeHash) {
		return nil, types.Address{}, 0, errors.New("contract address collision")
	}

	// EIP-2929: warm the created contract address BEFORE taking snapshot.
	// Even if the creation fails, the access-list change should not be rolled back.
	evm.StateDB.AddAddressToAccessList(contractAddr)

	snapshot := evm.StateDB.Snapshot()
	createdSnapshot := len(evm.createdOrder)

	if evm.frame8141 != nil && evm.StateDB.Empty(contractAddr) {
		if err := evm.frame8141.Charge(frameNewAccountGas); err != nil {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
			return nil, types.Address{}, gas, ErrFrameStateGas
		}
	}

	// Reset an eligible existing account, retaining only its prefunding.
	previousBalance := evm.StateDB.GetBalance(contractAddr)
	evm.StateDB.CreateAccount(contractAddr)
	if previousBalance.Sign() > 0 {
		evm.StateDB.AddBalance(contractAddr, previousBalance)
	}

	evm.markContractCreated(contractAddr)
	// EIP-161: set contract nonce to 1 (post Spurious Dragon).
	evm.StateDB.SetNonce(contractAddr, 1)
	evm.recordCreationNonce(contractAddr, 0, 1)

	// Transfer value
	if value != nil && value.Sign() > 0 {
		callerBalance := evm.StateDB.GetBalance(caller)
		if callerBalance.Cmp(value) < 0 {
			return nil, types.Address{}, gas, errors.New("insufficient balance for transfer")
		}
		// EIP-7928: capture pre-transfer balances for BAL tracking.
		var creatorPreBal, contractPreBal *big.Int
		if evm.balTracker != nil {
			creatorPreBal = new(big.Int).Set(evm.StateDB.GetBalance(caller))
			contractPreBal = new(big.Int).Set(evm.StateDB.GetBalance(contractAddr))
		}
		evm.StateDB.SubBalance(caller, value)
		evm.StateDB.AddBalance(contractAddr, value)
		// EIP-7928: record balance changes for BAL tracking.
		if evm.balTracker != nil {
			evm.balTracker.RecordBalanceChange(caller, creatorPreBal, evm.StateDB.GetBalance(caller))
			evm.balTracker.RecordBalanceChange(contractAddr, contractPreBal, evm.StateDB.GetBalance(contractAddr))
		}

		// EIP-7708: emit transfer log for nonzero-value CREATE.
		evm.recordETHTransfer(caller, contractAddr, value)
	}

	// GasCreate and InitCodeWordGas are already charged by the jump table's
	// constantGas and dynamicGas functions. Do not charge them again here.

	// Parent retention remains outside the child, including code deposit.
	contract := NewContract(caller, contractAddr, value, gas)
	contract.Code = code

	// Execute init code
	evm.depth++
	ret, err = evm.Run(contract, nil)
	evm.depth--

	if err != nil {
		// On any error, revert state and return.
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		if !errors.Is(err, ErrExecutionReverted) {
			return nil, types.Address{}, 0, err
		}
		return ret, types.Address{}, contract.Gas, err
	}
	gas = contract.Gas
	if evm.forkRules.IsLondon && len(ret) > 0 && ret[0] == 0xef {
		evm.revertCreationSnapshot(snapshot, createdSnapshot)
		return nil, types.Address{}, 0, errors.New("invalid code prefix")
	}

	// Code deposit cost: 200 per byte of deployed code.
	if len(ret) > 0 {
		// EIP-170 / EIP-7954: max contract code size.
		maxCode := MaxCodeSizeForFork(evm.forkRules)
		if len(ret) > maxCode {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
			return nil, types.Address{}, 0, errors.New("max code size exceeded")
		}
		depositCost := uint64(len(ret)) * CreateDataGas
		if evm.frame8141 != nil {
			if err := evm.frame8141.Charge(uint64(len(ret)) * 1530); err != nil {
				evm.revertCreationSnapshot(snapshot, createdSnapshot)
				return nil, types.Address{}, 0, ErrOutOfGas
			}
			depositCost = 6 * toWordSize(uint64(len(ret)))
		}
		if gas < depositCost {
			evm.revertCreationSnapshot(snapshot, createdSnapshot)
			return nil, types.Address{}, 0, ErrOutOfGas
		}
		gas -= depositCost
		evm.StateDB.SetCode(contractAddr, ret)
		if tracker, ok := evm.balTracker.(interface {
			RecordCodeChange(types.Address, []byte, []byte)
		}); ok {
			tracker.RecordCodeChange(contractAddr, nil, ret)
		}
	}

	return ret, contractAddr, gas, nil
}
