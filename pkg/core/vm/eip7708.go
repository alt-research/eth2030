package vm

import (
	"math/big"

	"github.com/eth2030/eth2030/core/types"
)

// EIP-7708: ETH transfers emit a log.
//
// Specified nonzero-value transfers (transaction, CALL, CREATE/CREATE2, SELFDESTRUCT)
// emit a LOG3 identical to an ERC-20 Transfer event. Fees and withdrawals
// are excluded. The emitter is SYSTEM_ADDRESS (EIP-4788).

var (
	// SystemAddress is the EIP-4788 system address used as the log emitter.
	SystemAddress = types.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")

	// TransferEventTopic is keccak256("Transfer(address,address,uint256)").
	TransferEventTopic = types.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	// BurnEventTopic is keccak256("Burn(address,uint256)").
	BurnEventTopic = types.HexToHash("0xcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5")
)

// EmitTransferLog emits an EIP-7708 ETH transfer log (LOG3).
// It is called whenever nonzero ETH value is transferred between accounts.
// The log is emitted from SYSTEM_ADDRESS with the ERC-20 Transfer event signature.
func EmitTransferLog(statedb StateDB, from, to types.Address, amount *big.Int) *types.Log {
	if statedb == nil || amount == nil || amount.Sign() <= 0 || from == to {
		return nil
	}

	// Encode amount as big-endian uint256 (32 bytes, left-padded).
	data := make([]byte, 32)
	amountBytes := amount.Bytes()
	copy(data[32-len(amountBytes):], amountBytes)

	log := &types.Log{
		Address: SystemAddress,
		Topics: []types.Hash{
			TransferEventTopic,
			addressToTopic(from),
			addressToTopic(to),
		},
		Data: data,
	}
	statedb.AddLog(log)
	return log
}

// EmitBurnLog is a legacy analytics helper, not EIP-7708 consensus emission.
// Current transfer-log rules do not emit fee or SELFDESTRUCT burn events.
func EmitBurnLog(statedb StateDB, addr types.Address, amount *big.Int) {
	if statedb == nil || amount == nil || amount.Sign() <= 0 {
		return
	}

	data := make([]byte, 32)
	amountBytes := amount.Bytes()
	copy(data[32-len(amountBytes):], amountBytes)

	statedb.AddLog(&types.Log{
		Address: SystemAddress,
		Topics: []types.Hash{
			BurnEventTopic,
			addressToTopic(addr),
		},
		Data: data,
	})
}

// addressToTopic converts an address to a 32-byte topic (zero-padded on the left).
func addressToTopic(addr types.Address) types.Hash {
	var topic types.Hash
	copy(topic[12:], addr[:])
	return topic
}

// recordETHTransfer also notifies an optional request-owned simulation observer.
// The observer does not change fork activation or add logs to real receipts.
func (evm *EVM) recordETHTransfer(from, to types.Address, amount *big.Int) {
	if evm.StateDB == nil || amount == nil || amount.Sign() <= 0 || from == to {
		return
	}
	if observer, ok := evm.StateDB.(interface {
		RecordETHTransfer(types.Address, types.Address, *big.Int)
	}); ok {
		observer.RecordETHTransfer(from, to, amount)
	}
	if evm.forkRules.IsEIP7708 {
		evm.traceLog(EmitTransferLog(evm.StateDB, from, to, amount))
	}
}
