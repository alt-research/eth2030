package execution

import (
	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"math/big"
	"testing"
)

func TestAmsterdamActualSystemStateMargin(t *testing.T) {
	for _, slots := range []int{8, 16, 17} {
		db := state.NewMemoryStateDB()
		address := types.Address{0: 0xaa, 19: 0x21}
		var code []byte
		for i := 0; i < slots; i++ {
			code = append(code, 0x60, 1, 0x60, byte(i), 0x55)
		}
		code = append(code, 0x5a, 0x5f, 0x52, 0x60, 32, 0x5f, 0xf3)
		db.SetCode(address, code)
		db.SetNonce(address, 1)
		header := &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 30_000_000, BaseFee: big.NewInt(1)}
		cfg := *config.TestConfig
		evm, err := newSystemEVM(&cfg, nil, db, header)
		if err != nil {
			t.Fatal(err)
		}
		output, err := executeSystemCall(evm, db, address, nil, bal.NewTracker())
		if err != nil || len(output) != 32 {
			t.Fatalf("system result %x %v", output, err)
		}
		want := int64(30_000_000 - slots*12106 - 2)
		if slots > 16 {
			want -= int64(slots-16) * 97920
		}
		if new(big.Int).SetBytes(output).Int64() != want {
			t.Fatalf("%d slots returned GAS=%s want independent execution margin %d", slots, new(big.Int).SetBytes(output), want)
		}
		if db.Exist(types.SystemAddress) || header.GasUsed != 0 || db.GetRefund() != 0 || db.AddressInAccessList(address) {
			t.Fatal("system call leaked caller/block/refund/warm state")
		}
	}
}
