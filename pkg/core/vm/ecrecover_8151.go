package vm

import "github.com/eth2030/eth2030/core/types"

// PermittedRecoveryCode8151 classifies raw code without following delegation.
func PermittedRecoveryCode8151(code []byte) bool {
	if len(code) == 0 {
		return true
	}
	_, ok := types.ParseDelegation(code)
	return ok
}

// runPrecompile keeps recovery pure until both the base and state-access
// charges have been paid. Read tracking deliberately survives enclosing reverts.
func (evm *EVM) runPrecompile(p PrecompiledContract, input []byte, gas uint64) ([]byte, uint64, error) {
	if _, ok := p.(*ecrecover); !ok || !evm.forkRules.IsEIP8151 {
		return runPrecompile(p, input, gas)
	}
	if gas < 3000 {
		return nil, 0, ErrOutOfGas
	}
	output, err := p.Run(input)
	gas -= 3000
	if err != nil {
		return nil, 0, err
	}
	if len(output) == 0 {
		return nil, gas, nil
	}
	address := types.BytesToAddress(output)
	cost := uint64(3100)
	if evm.StateDB.AddressInAccessList(address) {
		cost = 200
	}
	if gas < cost {
		return nil, 0, ErrOutOfGas
	}
	gas -= cost
	evm.StateDB.AddAddressToAccessList(address)
	if evm.balTracker != nil {
		evm.balTracker.RecordAddressTouch(address)
	}
	var code []byte
	if reader, ok := evm.StateDB.(interface{ FrameRecoveryCode8151(types.Address) []byte }); ok {
		code = reader.FrameRecoveryCode8151(address)
	} else {
		code = evm.StateDB.GetCode(address)
	}
	if callback := evm.Config.OnFrameRecovery8151; callback != nil && evm.frame8141 != nil && evm.frame8141.PublicValidation {
		ctx := evm.frame8141
		frame := ctx.Tx.Frames[ctx.Index]
		pay := frame.Mode == types.ModeVerify && frame.Flags == types.FrameApprovePayment
		payer := ctx.Tx.Sender
		if frame.Target != nil {
			payer = *frame.Target
		}
		digest := types.BytesToHash(padRight(input, 128)[:32])
		callback(address, digest, payer, pay)
	}
	if !PermittedRecoveryCode8151(code) {
		return nil, gas, nil
	}
	return output, gas, nil
}

// Preserve signer-class tracking through the Frame state journal while helper
// code accesses keep using ordinary hash dependencies on the same address.
func (s *frameJournalState) FrameRecoveryCode8151(address types.Address) []byte {
	if reader, ok := s.StateDB.(interface{ FrameRecoveryCode8151(types.Address) []byte }); ok {
		return reader.FrameRecoveryCode8151(address)
	}
	return s.StateDB.GetCode(address)
}
