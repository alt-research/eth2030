package vm

import (
	"context"
	"errors"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestSimulationCancellationInterruptsExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	evm := NewEVMWithState(BlockContext{}, TxContext{}, Config{Context: ctx}, nil)
	contract := NewContract(types.Address{}, types.Address{}, new(big.Int), 10_000_000)
	contract.Code = []byte{0x5b, 0x5f, 0x56}
	if _, err := evm.Run(contract, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled simulation continued: %v", err)
	}
	if contract.Gas != 10_000_000 {
		t.Fatal("cancellation executed opcodes before stopping")
	}
}
