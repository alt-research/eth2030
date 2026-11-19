package main

import (
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"math/big"
)

func main() {
	fmt.Printf("constants: txBase=%d cpsb=%d coldAccount=%d coldStorage=%d warm=%d codeLimit=%d initLimit=%d\n", vm.TxBaseGlamsterdam, vm.CostPerStateByte, vm.ColdAccountAccessGlamst, vm.ColdSloadGlamst, vm.WarmStorageReadGlamst, vm.MaxCodeSizeGlamsterdam, vm.MaxInitCodeSizeGlamsterdam)
	for _, clear := range []bool{false, true} {
		s := state.NewMemoryStateDB()
		sender := types.HexToAddress("0x1111111111111111111111111111111111111111")
		to := types.HexToAddress("0x2222222222222222222222222222222222222222")
		s.CreateAccount(sender)
		s.AddBalance(sender, new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil))
		s.CreateAccount(to)
		s.SetNonce(to, 1)
		if clear {
			s.SetCode(to, []byte{0x60, 0, 0x60, 0, 0x55, 0})
			s.SetState(to, types.Hash{}, types.HexToHash("0x01"))
		}
		h := &types.Header{Number: big.NewInt(100), GasLimit: 60000000, Time: 100, BaseFee: big.NewInt(1)}
		msg := &config.Message{From: sender, To: &to, GasLimit: 100000, GasPrice: big.NewInt(1), Value: big.NewInt(0)}
		out, err := execution.ApplyMessage(config.TestConfigGlamsterdan, nil, s, h, msg, new(gaspool.GasPool).AddGas(h.GasLimit))
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("clear=%v used=%d blockUsed=%d refund=%d logs=%d failed=%v\n", clear, out.UsedGas, out.BlockGasUsed, s.GetRefund(), len(s.GetLogs(types.Hash{})), out.Err)
	}
}
