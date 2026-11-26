package vm

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/eth2030/eth2030/core/framepay"
	"github.com/eth2030/eth2030/core/types"
)

// FrameStateGas tracks EIP-8141's explicit, isolated state pools. Snapshots
// are taken with the same StateDB snapshot that journals the corresponding writes.
type FrameStateGas struct {
	NonceProfile     *types.FrameTx8250Delta
	LegacyNonce      uint64
	Tx               *types.FrameTx8141
	Payment          *types.FramePayment
	Index            int
	Left             uint64
	Approved         bool
	Payer            *types.Address
	Results          []types.FrameResult8141
	owners           map[frameStorageKey]int
	PublicValidation bool
	Scopes           []uint8
	created          map[types.Address]bool
	bal              *frameBALJournal
}

type frameStorageKey struct {
	address types.Address
	slot    types.Hash
}

func (f *FrameStateGas) clone() FrameStateGas {
	copy := *f
	copy.Scopes = append([]uint8(nil), f.Scopes...)
	copy.created = make(map[types.Address]bool, len(f.created))
	for addr, created := range f.created {
		copy.created[addr] = created
	}
	copy.Results = append([]types.FrameResult8141(nil), f.Results...)
	copy.owners = make(map[frameStorageKey]int, len(f.owners))
	for key, index := range f.owners {
		copy.owners[key] = index
	}
	return copy
}

type frameJournalState struct {
	StateDB
	frame     *FrameStateGas
	snapshots map[int]FrameStateGas
}

func (s *frameJournalState) Snapshot() int {
	id := s.StateDB.Snapshot()
	s.snapshots[id] = s.frame.clone()
	return id
}

func (s *frameJournalState) RevertToSnapshot(id int) {
	s.StateDB.RevertToSnapshot(id)
	if frame, found := s.snapshots[id]; found {
		*s.frame = frame
	}
	for key := range s.snapshots {
		if key >= id {
			delete(s.snapshots, key)
		}
	}
}

func (f *FrameStateGas) Charge(amount uint64) error {
	if f.Left < amount {
		return ErrOutOfGas
	}
	f.Left -= amount
	f.Results[f.Index].GasUsed.State += amount
	return nil
}

var ErrFrameStateGas = errors.New("frame state gas exhausted before child call")
var ErrFrameNonceOverflow = errors.New("Frame sender nonce overflow")

const (
	frameNewAccountGas = 120 * 1530
	frameNewSlotGas    = 64 * 1530
)

// SetFrameTransaction selects the pinned draft's instructions only for a
// canonical Frame envelope. Older transaction formats keep their own dispatch.
func (evm *EVM) SetFrameTransaction(tx *types.FrameTx8141, payment *types.FramePayment) *FrameStateGas {
	ctx := &FrameStateGas{
		Tx: tx, Payment: payment, Results: make([]types.FrameResult8141, len(tx.Frames)),
		owners: make(map[frameStorageKey]int), Scopes: make([]uint8, len(tx.Frames)), created: make(map[types.Address]bool),
	}
	if evm.balTracker != nil {
		ctx.bal = newFrameBALJournal(evm.balTracker, evm.StateDB)
		evm.balTracker = ctx.bal
	}
	evm.frame8141 = ctx
	evm.forkRules.IsEIP7708 = true
	evm.StateDB = &frameJournalState{StateDB: evm.StateDB, frame: ctx, snapshots: make(map[int]FrameStateGas)}
	evm.jumpTable[APPROVE] = &operation{execute: opApprove8141, minStack: 3, maxStack: 1024,
		halts: true, memorySize: memoryApprove, dynamicGas: gasMemExpansion}
	evm.jumpTable[0xb0] = &operation{execute: opTxParam8141, constantGas: 2, minStack: 1, maxStack: 1024}
	evm.jumpTable[0xb1] = &operation{execute: opFrameDataLoad8141, constantGas: 3, minStack: 2, maxStack: 1024}
	evm.jumpTable[0xb2] = &operation{execute: opFrameDataCopy8141, constantGas: 3, minStack: 4, maxStack: 1024,
		memorySize: memoryCalldataCopy, dynamicGas: gasFrameCopy8141}
	evm.jumpTable[0xb3] = &operation{execute: opFrameParam8141, constantGas: 2, minStack: 2, maxStack: 1024}
	evm.jumpTable[0xb4] = &operation{execute: opSigParam8141, constantGas: 2, minStack: 2, maxStack: 1024}
	evm.jumpTable[0xb5] = &operation{execute: opSigDataCopy8141, constantGas: 3, minStack: 4, maxStack: 1024,
		memorySize: memoryCalldataCopy, dynamicGas: gasFrameCopy8141}
	for _, op := range []OpCode{CALL, CALLCODE, DELEGATECALL, STATICCALL} {
		call := *evm.jumpTable[op]
		call.constantGas = 0
		call.dynamicGas = func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, size uint64) (uint64, error) {
			return gasFrameCall8141(op, evm, contract, stack, mem, size)
		}
		evm.jumpTable[op] = &call
	}
	for _, op := range []OpCode{BALANCE, EXTCODESIZE, EXTCODEHASH, EXTCODECOPY} {
		operation := *evm.jumpTable[op]
		operation.constantGas = 0
		operation.dynamicGas = func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, size uint64) (uint64, error) {
			return gasFrameAccount8141(op, evm, contract, stack, mem, size)
		}
		evm.jumpTable[op] = &operation
	}
	if evm.jumpTable[SELFDESTRUCT] != nil {
		operation := *evm.jumpTable[SELFDESTRUCT]
		operation.execute, operation.dynamicGas = opFrameSelfdestruct8141, gasFrameSelfdestruct8141
		evm.jumpTable[SELFDESTRUCT] = &operation
	}
	create := *evm.jumpTable[CREATE]
	create.constantGas = 12000
	evm.jumpTable[CREATE] = &create
	create2 := *evm.jumpTable[CREATE2]
	create2.constantGas = 12000
	evm.jumpTable[CREATE2] = &create2
	evm.jumpTable[SSTORE] = &operation{execute: opFrameSstore8141, minStack: 2, maxStack: 1024, dynamicGas: gasFrameSstore8141, writes: true}
	return ctx
}

func (evm *EVM) approve8141(target types.Address, scope uint64) error {
	f := evm.frame8141
	frame := f.Tx.Frames[f.Index]
	if scope == 0 || scope > 3 || scope & ^uint64(frame.Flags&3) != 0 {
		return ErrInvalidApproveScope
	}
	execution := scope&uint64(types.FrameApproveExecution) != 0
	payment := scope&uint64(types.FrameApprovePayment) != 0
	if execution && (f.Approved || target != f.Tx.Sender) {
		return ErrAlreadyApproved
	}
	if payment {
		if f.Payer != nil || (!f.Approved && !execution) {
			return ErrSenderNotApproved
		}
		if evm.StateDB.GetBalance(target).Cmp(f.Payment.Escrow()) < 0 {
			return ErrInsufficientBalance
		}
		if err := evm.consumeFrameNonce(); err != nil {
			return err
		}
		evm.StateDB.SubBalance(target, f.Payment.Escrow())
		payer := target
		f.Payer = &payer
	}
	if execution {
		f.Approved = true
	}
	f.Scopes[f.Index] = uint8(scope)
	return nil
}

func opApprove8141(_ *uint64, evm *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, length, scope := stack.Pop(), stack.Pop(), stack.Pop()
	ctx := evm.frame8141
	target := ctx.Tx.Sender
	if frame := ctx.Tx.Frames[ctx.Index]; frame.Target != nil {
		target = *frame.Target
	}
	if contract.Address != target || scope.BitLen() > 64 {
		return nil, ErrExecutionReverted
	}
	if err := evm.approve8141(target, scope.Uint64()); err != nil {
		if errors.Is(err, ErrOutOfGas) || errors.Is(err, ErrFrameNonceOverflow) {
			return nil, err
		}
		return nil, ErrExecutionReverted
	}
	return memory.Get(int64(offset.Uint64()), int64(length.Uint64())), nil
}

// RunFrame8141 performs the entry access and resolved-target dispatch. It
// journals warmness and value creation before the ordinary EVM Call snapshot.
func (evm *EVM) RunFrame8141(index int) (gasLeft uint64, err error) {
	ctx := evm.frame8141
	frame := ctx.Tx.Frames[index]
	ctx.Index, ctx.Left = index, frame.Limits.State
	evm.returnData = nil
	snapshot := evm.StateDB.Snapshot()
	gasLeft = frame.Limits.Execution
	defer func() {
		if err != nil {
			evm.StateDB.RevertToSnapshot(snapshot)
			if !errors.Is(err, ErrExecutionReverted) {
				gasLeft = 0
			}
		}
	}()
	target := ctx.Tx.Sender
	if frame.Target != nil {
		target = *frame.Target
	}
	accessGas := uint64(100)
	if !evm.StateDB.AddressInAccessList(target) {
		accessGas = 3000
		evm.StateDB.AddAddressToAccessList(target)
	}
	if gasLeft < accessGas {
		return 0, ErrOutOfGas
	}
	gasLeft -= accessGas
	// Entry access is paid before dispatch, including explicit precompiles and
	// frames that subsequently fail. Underfunded entry does not access state.
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(target)
	}
	caller := types.EntryPointAddress
	if frame.Mode == types.ModeSender {
		if !ctx.Approved {
			return gasLeft, ErrSenderNotApproved
		}
		caller = ctx.Tx.Sender
	}
	value := frame.Value
	if value == nil {
		value = new(big.Int)
	}
	if value.Sign() > 0 && evm.StateDB.GetBalance(caller).Cmp(value) < 0 {
		return gasLeft, ErrExecutionReverted
	}
	if value.Sign() > 0 && evm.StateDB.Empty(target) {
		if err := ctx.Charge(frameNewAccountGas); err != nil {
			return 0, err
		}
	}
	evm.TxContext.Origin = caller
	previousStatic := evm.readOnly
	evm.readOnly = frame.Mode == types.ModeVerify
	defer func() { evm.readOnly = previousStatic }()
	if _, precompile := evm.precompile(target); !precompile && len(evm.StateDB.GetCode(target)) == 0 {
		if frame.Mode == types.ModeVerify {
			scope := frame.Flags & 3
			sigIndex := 0
			if scope&types.FrameApproveExecution == 0 {
				sigIndex = 1
			}
			if scope == 0 || sigIndex >= len(ctx.Tx.Signatures) {
				return gasLeft, ErrExecutionReverted
			}
			sig := ctx.Tx.Signatures[sigIndex]
			signer := ctx.Tx.Sender
			if sig.Signer != nil {
				signer = *sig.Signer
			}
			if sig.Scheme != types.FrameSignatureSecp256k1 || signer != target || len(sig.Message) != 0 {
				return gasLeft, ErrExecutionReverted
			}
			if err := evm.approve8141(target, uint64(scope)); err != nil {
				return gasLeft, err
			}
			return gasLeft, nil
		}
	}
	_, gasLeft, err = evm.Call(caller, target, frame.Data, gasLeft, value)
	return gasLeft, err
}

func gasFrameSstore8141(evm *EVM, contract *Contract, stack *Stack, _ *Memory, _ uint64) (uint64, error) {
	if contract.Gas <= CallStipend {
		return 0, ErrOutOfGas
	}
	key, value := bigToHash(stack.Back(0)), bigToHash(stack.Back(1))
	cold := uint64(0)
	if _, warm := evm.StateDB.SlotInAccessList(contract.Address, key); !warm {
		cold = 2100
		evm.StateDB.AddSlotToAccessList(contract.Address, key)
	}
	current := evm.StateDB.GetState(contract.Address, key)
	original := evm.StateDB.GetCommittedState(contract.Address, key)
	// This implicit read survives post-state execution/state gas failures.
	if evm.balTracker != nil {
		evm.balTracker.RecordStorageRead(contract.Address, key, current)
	}
	cost := uint64(100)
	if cold != 0 {
		cost = cold
	}
	if current != value && current == original {
		cost += 10000
	}
	return cost, nil
}

func opFrameSstore8141(_ *uint64, evm *EVM, contract *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	if evm.readOnly {
		return nil, ErrWriteProtection
	}
	ctx := evm.frame8141
	key, value := bigToHash(stack.Back(0)), bigToHash(stack.Back(1))
	current := evm.StateDB.GetState(contract.Address, key)
	original := evm.StateDB.GetCommittedState(contract.Address, key)
	ownerKey := frameStorageKey{contract.Address, key}
	if original.IsZero() && current.IsZero() && !value.IsZero() {
		if err := ctx.Charge(frameNewSlotGas); err != nil {
			return nil, err
		}
		ctx.owners[ownerKey] = ctx.Index
	}
	if original.IsZero() && !current.IsZero() && value.IsZero() {
		if owner, found := ctx.owners[ownerKey]; found {
			ctx.Results[owner].GasUsed.State -= frameNewSlotGas
			if owner == ctx.Index {
				ctx.Left += frameNewSlotGas
			}
			delete(ctx.owners, ownerKey)
		}
	}
	// EIP-8038 execution refunds are independent of the state-gas refill.
	if !original.IsZero() {
		if !current.IsZero() && value.IsZero() {
			evm.StateDB.AddRefund(11616)
		}
		if current.IsZero() && !value.IsZero() {
			evm.StateDB.SubRefund(11616)
		}
	}
	if current != value && value == original {
		evm.StateDB.AddRefund(10000)
	}
	stack.Pop()
	stack.Pop()
	evm.StateDB.SetState(contract.Address, key, value)
	if evm.balTracker != nil {
		if current == value {
			evm.balTracker.RecordStorageRead(contract.Address, key, current)
		} else {
			evm.balTracker.RecordStorageChange(contract.Address, key, current, value)
		}
	}
	return nil, nil
}

func opTxParam8141(_ *uint64, evm *EVM, _ *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	param := stack.Pop()
	if !param.IsUint64() {
		return nil, ErrInvalidTxParamIndex
	}
	ctx := evm.frame8141
	tx := ctx.Tx
	var value *big.Int
	switch param.Uint64() {
	case 0:
		value = big.NewInt(int64(types.FrameTxType))
	case 1:
		value = new(big.Int).SetUint64(tx.Nonce)
		if ctx.NonceProfile != nil {
			value.SetUint64(ctx.NonceProfile.NonceSeq)
		}
	case 2:
		value = new(big.Int).SetBytes(tx.Sender[:])
	case 3:
		value = new(big.Int)
		if tx.Fees.MaxPriorityFeePerGas != nil {
			value.Set(tx.Fees.MaxPriorityFeePerGas)
		}
	case 4:
		value = new(big.Int)
		if tx.Fees.MaxFee != nil {
			value.Set(tx.Fees.MaxFee)
		}
	case 7:
		value = new(big.Int).SetUint64(uint64(len(tx.BlobVersionedHashes)))
	case 8:
		hash, err := tx.SigningHash()
		if ctx.NonceProfile != nil {
			hash, err = ctx.NonceProfile.SigningHash()
		}
		if err != nil {
			return nil, err
		}
		value = new(big.Int).SetBytes(hash[:])
	case 9:
		value = new(big.Int).SetUint64(uint64(len(tx.Frames)))
	case 10:
		value = new(big.Int).SetUint64(uint64(ctx.Index))
	case 11:
		value = new(big.Int).SetUint64(uint64(len(tx.Signatures)))
	case 12:
		value = new(big.Int).SetUint64(ctx.Left)
	case 13, 14, 15, 16:
		if ctx.NonceProfile == nil {
			return nil, ErrInvalidTxParamIndex
		}
		keys := ctx.NonceProfile.NonceKeys
		value = new(big.Int)
		switch param.Uint64() {
		case 13:
			value.SetUint64(ctx.LegacyNonce)
		case 14:
			value.SetUint64(uint64(len(keys)))
		case 15:
			hash := types.NonceKeysHash8250(keys)
			value.SetBytes(hash[:])
		case 16:
			if len(keys) > 0 {
				value.Set(keys[0])
			}
		}
	default:
		return nil, ErrInvalidTxParamIndex
	}
	stack.Push(value)
	return nil, nil
}

func frameIndex8141(ctx *FrameStateGas, value *big.Int) (int, error) {
	if !value.IsUint64() || value.Uint64() >= uint64(len(ctx.Tx.Frames)) {
		return 0, ErrFrameIndexOutOfBounds
	}
	return int(value.Uint64()), nil
}

func opFrameParam8141(_ *uint64, evm *EVM, _ *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	indexValue, paramValue := stack.Pop(), stack.Pop()
	ctx := evm.frame8141
	index, err := frameIndex8141(ctx, indexValue)
	if err != nil || !paramValue.IsUint64() {
		return nil, ErrFrameIndexOutOfBounds
	}
	frame := ctx.Tx.Frames[index]
	param := paramValue.Uint64()
	if (param == 5 || param == 10 || param == 11) && index >= ctx.Index {
		return nil, ErrFrameIndexOutOfBounds
	}
	value := new(big.Int)
	switch param {
	case 0:
		target := ctx.Tx.Sender
		if frame.Target != nil {
			target = *frame.Target
		}
		value.SetBytes(target[:])
	case 1:
		value.SetUint64(frame.Limits.Execution)
	case 2:
		value.SetUint64(uint64(frame.Mode))
	case 3:
		value.SetUint64(uint64(frame.Flags))
	case 4:
		value.SetUint64(uint64(len(frame.Data)))
	case 5:
		value.SetUint64(ctx.Results[index].Status)
	case 6:
		value.SetUint64(uint64(frame.Flags & 3))
	case 7:
		value.SetUint64(uint64(frame.Flags >> 2 & 1))
	case 8:
		if frame.Value != nil {
			value.Set(frame.Value)
		}
	case 9:
		value.SetUint64(frame.Limits.State)
	case 10:
		value.SetUint64(ctx.Results[index].GasUsed.Execution)
	case 11:
		value.SetUint64(ctx.Results[index].GasUsed.State)
	default:
		return nil, ErrInvalidTxParamIndex
	}
	stack.Push(value)
	return nil, nil
}

func signature8141(ctx *FrameStateGas, index *big.Int) (types.FrameSignature, error) {
	if !index.IsUint64() || index.Uint64() >= uint64(len(ctx.Tx.Signatures)) {
		return types.FrameSignature{}, ErrFrameIndexOutOfBounds
	}
	return ctx.Tx.Signatures[index.Uint64()], nil
}

func opSigParam8141(_ *uint64, evm *EVM, _ *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	index, param := stack.Pop(), stack.Pop()
	sig, err := signature8141(evm.frame8141, index)
	if err != nil || !param.IsUint64() {
		return nil, ErrInvalidTxParamIndex
	}
	value := new(big.Int)
	switch param.Uint64() {
	case 0:
		if sig.Scheme == types.FrameSignatureArbitrary {
			return nil, ErrInvalidTxParamIndex
		}
		signer := evm.frame8141.Tx.Sender
		if sig.Signer != nil {
			signer = *sig.Signer
		}
		value.SetBytes(signer[:])
	case 1:
		value.SetUint64(uint64(sig.Scheme))
	case 2:
		value.SetBytes(sig.Message)
	case 3:
		if sig.Scheme != types.FrameSignatureArbitrary {
			return nil, ErrInvalidTxParamIndex
		}
		value.SetUint64(uint64(len(sig.Signature)))
	default:
		return nil, ErrInvalidTxParamIndex
	}
	stack.Push(value)
	return nil, nil
}

func opFrameDataLoad8141(_ *uint64, evm *EVM, _ *Contract, _ *Memory, stack *Stack) ([]byte, error) {
	offset, index := stack.Pop(), stack.Pop()
	frame, err := frameIndex8141(evm.frame8141, index)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 32)
	data := evm.frame8141.Tx.Frames[frame].Data
	if offset.IsUint64() && offset.Uint64() < uint64(len(data)) {
		copy(out, data[offset.Uint64():])
	}
	stack.Push(new(big.Int).SetBytes(out))
	return nil, nil
}

func gasFrameCopy8141(evm *EVM, contract *Contract, stack *Stack, mem *Memory, size uint64) (uint64, error) {
	cost, err := gasMemExpansion(evm, contract, stack, mem, size)
	length := stack.Back(2)
	if !length.IsUint64() {
		return 0, ErrOutOfGas
	}
	return safeAdd(cost, safeMul(3, toWordSize(length.Uint64()))), err
}

func copyFrameData8141(memory *Memory, stack *Stack, data []byte) {
	memOffset, offset, length := stack.Pop(), stack.Pop(), stack.Pop()
	count := length.Uint64()
	// Memory expansion has already bounded count and the destination.
	out := make([]byte, count)
	if offset.IsUint64() && offset.Uint64() < uint64(len(data)) {
		copy(out, data[offset.Uint64():])
	}
	memory.Set(memOffset.Uint64(), count, out)
}

func opFrameDataCopy8141(_ *uint64, evm *EVM, _ *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	index, err := frameIndex8141(evm.frame8141, stack.Back(3))
	if err != nil {
		return nil, err
	}
	copyFrameData8141(memory, stack, evm.frame8141.Tx.Frames[index].Data)
	stack.Pop()
	return nil, nil
}

func opSigDataCopy8141(_ *uint64, evm *EVM, _ *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	sig, err := signature8141(evm.frame8141, stack.Back(3))
	if err != nil {
		return nil, err
	}
	if sig.Scheme != types.FrameSignatureArbitrary {
		return nil, ErrInvalidTxParamIndex
	}
	copyFrameData8141(memory, stack, sig.Signature)
	stack.Pop()
	return nil, nil
}

func (evm *EVM) FrameValidationError8141() error { return evm.frameValidationErr }

func (evm *EVM) CheckFrameApproval8141() (types.Address, error) {
	if evm.frame8141.Payer == nil {
		return types.Address{}, fmt.Errorf("frame transaction never approved payment")
	}
	return *evm.frame8141.Payer, nil
}

// gasFrameCall8141 moves conditional account creation into the state pool
// and computes EIP-150 forwarding after the remaining execution-only costs.
func gasFrameCall8141(op OpCode, evm *EVM, contract *Contract, stack *Stack, mem *Memory, size uint64) (uint64, error) {
	address := types.BytesToAddress(stack.Back(1).Bytes())
	cost := frameAccountAccess8141(evm, address)
	if (op == CALL || op == CALLCODE) && stack.Back(2).Sign() != 0 {
		cost += 9000 + CallStipend
	}
	memory, err := gasMemExpansion(evm, contract, stack, mem, size)
	if err != nil {
		return 0, err
	}
	cost = safeAdd(cost, memory)
	// Resolve and charge delegation in the parent, before EIP-150 forwarding
	// and before child balance/depth checks (EIP-7702 / EIP-7928).
	cost, err = gasDelegationResolution(evm, contract, address, cost)
	if err != nil {
		return 0, err
	}
	requested := ^uint64(0)
	if stack.Back(0).IsUint64() {
		requested = stack.Back(0).Uint64()
	}
	evm.callGasTemp = callGasEIP150(contract.Gas, cost, requested)
	return safeAdd(cost, evm.callGasTemp), nil
}

// checkFrameValidation applies the public prefix trace rules at every call depth.
func (evm *EVM) checkFrameValidation(op OpCode, contract *Contract, stack *Stack, pc uint64) error {
	f := evm.frame8141
	if f == nil || !f.PublicValidation {
		return nil
	}
	frame := f.Tx.Frames[f.Index]
	target := f.Tx.Sender
	if frame.Target != nil {
		target = *frame.Target
	}
	// Only the fixed PAY runtime receives the draft's canonical-code exception.
	// STATICCALL/write protection and exact-scope approval still apply normally.
	if frame.Mode == types.ModeVerify && frame.Flags == types.FrameApprovePayment && evm.Config.ExperimentalFramePaymasterV1 && framepay.IsV1(evm.StateDB.GetCode(target)) {
		return nil
	}
	deploy := f.Index == f.Tx.ValidationDeployIndex8141()
	rootVerifier := frame.IsRecentRootVerifier() && evm.depth == 1 &&
		contract.Address == types.RecentRootAddress && string(contract.Code) == string(types.RecentRootCode)
	if frame.IsRecentRootVerifier() && string(evm.StateDB.GetCode(types.RecentRootAddress)) != string(types.RecentRootCode) {
		return errors.New("Frame recent-root verifier runtime mismatch")
	}
	switch op {
	case GASPRICE, BLOCKHASH, COINBASE, NUMBER, PREVRANDAO, GASLIMIT, BASEFEE,
		BLOBBASEFEE, INVALID, SELFDESTRUCT, BALANCE, SELFBALANCE:
		return errors.New("Frame public validation used a block or balance dependency")
	case SLOTNUM:
		if !rootVerifier {
			return errors.New("Frame SLOTNUM outside top-level recent-root verifier")
		}
	case TIMESTAMP:
		if f.Index != 0 || target != types.FrameExpiryVerifier || frame.Mode != types.ModeVerify || frame.Flags != 0 || evm.depth != 1 || contract.Address != types.FrameExpiryVerifier ||
			string(contract.Code) != string(types.FrameExpiryCode) {
			return errors.New("Frame public validation timestamp is not the expiry verifier")
		}
	case GAS:
		next := contract.GetOp(pc + 1)
		if next != CALL && next != CALLCODE && next != DELEGATECALL && next != STATICCALL {
			return errors.New("Frame public validation observed GAS")
		}
	case SLOAD, SSTORE:
		if op == SLOAD && rootVerifier {
			refs, _ := types.DecodeRecentRootReferences(frame.Data)
			key := bigToHash(stack.Back(0))
			for _, ref := range refs {
				if ref.StorageKey() == key {
					return nil
				}
			}
			return errors.New("Frame recent-root verifier read an undeclared key")
		}
		if contract.Address != f.Tx.Sender || (op == SSTORE && !deploy) {
			return errors.New("Frame public validation accessed mutable third-party state")
		}
	case CREATE, CREATE2:
		if !deploy {
			return errors.New("Frame public validation created a contract outside deployment")
		}
	case CALL, CALLCODE, DELEGATECALL, STATICCALL, EXTCODESIZE, EXTCODEHASH, EXTCODECOPY:
		if (op == CALL || op == CALLCODE) && stack.Back(2).Sign() != 0 {
			return errors.New("Frame public validation transferred value outside deployment")
		}
		index := 0
		if op == CALL || op == CALLCODE || op == DELEGATECALL || op == STATICCALL {
			index = 1
		}
		address := types.BytesToAddress(stack.Back(index).Bytes())
		if _, precompile := evm.precompile(address); precompile {
			return nil
		}
		code := evm.StateDB.GetCode(address)
		if len(code) == 0 || types.HasDelegationPrefix(code) {
			return errors.New("Frame public validation called an empty or delegated third-party account")
		}
	}
	return nil
}

func frameAccountAccess8141(evm *EVM, address types.Address) uint64 {
	if evm.StateDB.AddressInAccessList(address) {
		return 100
	}
	evm.StateDB.AddAddressToAccessList(address)
	return 3000
}

func gasFrameAccount8141(op OpCode, evm *EVM, contract *Contract, stack *Stack, mem *Memory, size uint64) (uint64, error) {
	cost := frameAccountAccess8141(evm, types.BytesToAddress(stack.Back(0).Bytes()))
	if op == EXTCODESIZE || op == EXTCODECOPY {
		cost += 100
	}
	if op == EXTCODECOPY {
		memory, err := gasMemExpansion(evm, contract, stack, mem, size)
		return safeAdd(safeAdd(cost, memory), safeMul(3, toWordSize(stack.Back(3).Uint64()))), err
	}
	return cost, nil
}

func gasFrameSelfdestruct8141(evm *EVM, contract *Contract, stack *Stack, _ *Memory, _ uint64) (uint64, error) {
	beneficiary := types.BytesToAddress(stack.Back(0).Bytes())
	var cost uint64
	if !evm.StateDB.AddressInAccessList(beneficiary) {
		evm.StateDB.AddAddressToAccessList(beneficiary)
		cost = 3000
	}
	if contract.Gas < cost {
		return 0, ErrOutOfGas
	}
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(beneficiary)
	}
	if evm.StateDB.GetBalance(contract.Address).Sign() > 0 && evm.StateDB.Empty(beneficiary) {
		cost += 9000
	}
	return cost, nil
}

func opFrameSelfdestruct8141(pc *uint64, evm *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	if evm.readOnly {
		return nil, ErrWriteProtection
	}
	beneficiary := types.BytesToAddress(stack.Back(0).Bytes())
	if evm.StateDB.GetBalance(contract.Address).Sign() > 0 && evm.StateDB.Empty(beneficiary) {
		if err := evm.frame8141.Charge(frameNewAccountGas); err != nil {
			return nil, err
		}
	}
	return opSelfdestruct(pc, evm, contract, memory, stack)
}
