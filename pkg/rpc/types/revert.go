package rpctypes

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// NewRevertError preserves the actual EVM payload and standard ABI Error/Panic
// reason at the RPC boundary. Reverts are application errors with code 3.
func NewRevertError(data []byte) *RPCError {
	message := "execution reverted"
	if reason, err := abi.UnpackRevert(data); err == nil {
		message += ": " + reason
	}
	return &RPCError{Code: 3, Message: message, Data: hexutil.Encode(data)}
}
