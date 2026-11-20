package backend

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/eips"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	rpctypes "github.com/eth2030/eth2030/rpc/types"
)

func TestSimulationSystemDeadlinesKeepTimeoutCode(t *testing.T) {
	for _, address := range []types.Address{eips.BeaconRootAddress, types.BuilderDepositRequestAddress, types.BuilderExitRequestAddress} {
		t.Run(address.Hex(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			_, err := timeoutSystemBlock(ctx, address, []byte{0x5b, 0x5f, 0x56})
			var rpcError *rpctypes.RPCError
			if !errors.As(err, &rpcError) || rpcError.Code != -32016 {
				t.Fatalf("actual system execution deadline returned %T %v", err, err)
			}
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				t.Fatal("fixture did not reach actual context deadline")
			}
		})
	}
}

func TestSimulationSystemFailureIsNotTimeout(t *testing.T) {
	_, err := timeoutSystemBlock(context.Background(), types.BuilderExitRequestAddress, []byte{0x5f, 0x5f, 0xfd})
	if err == nil || !strings.Contains(err.Error(), "system call") {
		t.Fatalf("expected actual request system revert: %v", err)
	}
	var rpcError *rpctypes.RPCError
	if errors.As(err, &rpcError) && rpcError.Code == -32016 {
		t.Fatal("real system failure misclassified as deadline")
	}
}

func timeoutSystemBlock(ctx context.Context, target types.Address, code []byte) (rpctypes.SimulateBlockResult, error) {
	cfg := *config.TestConfig
	zero := uint64(0)
	cfg.OsakaTime, cfg.AmsterdamTime, cfg.GlamsterdanTime = &zero, &zero, nil
	db := state.NewMemoryStateDB()
	for _, address := range []types.Address{types.BuilderDepositRequestAddress, types.BuilderExitRequestAddress} {
		db.CreateAccount(address)
		db.SetNonce(address, 1)
		db.SetCode(address, []byte{0})
	}
	db.CreateAccount(target)
	db.SetNonce(target, 1)
	db.SetCode(target, code)
	header := &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 50000000, BaseFee: big.NewInt(1), ParentBeaconRoot: new(types.Hash), SlotNumber: new(uint64)}
	remaining := uint64(50000000)
	return new(RPCBackend).simulateBlock(&cfg, db, header, nil, nil, rpctypes.SimulateOptions{}, execution.EVMOverrides{Context: ctx}, nil, &remaining)
}
