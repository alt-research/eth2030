package eiptest_test

import (
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/eiptest"
	"math/big"
	"testing"
)

func TestPublishedTransferLogsExcludeFeesAndSelfTransfers(t *testing.T) {
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkGlamsterdan, 0))
	for _, tc := range []struct {
		self  bool
		value int64
		want  int
	}{{false, 0, 0}, {true, 5, 0}, {false, 3, 1}} {
		to := s.Bob().Address
		if tc.self {
			to = s.Alice().Address
		}
		receipt := s.STF().Transfer(s.Alice(), to, big.NewInt(tc.value)).Receipt
		if len(receipt.Logs) != tc.want {
			t.Fatalf("value=%d self=%v logs=%d want %d", tc.value, tc.self, len(receipt.Logs), tc.want)
		}
		if tc.want == 1 {
			log := receipt.Logs[0]
			if log.Address != vm.SystemAddress || len(log.Topics) != 3 || log.Topics[0] != vm.TransferEventTopic || new(big.Int).SetBytes(log.Data).Int64() != tc.value {
				t.Fatal("incorrect canonical transfer event")
			}
		}
	}
}
func TestTransferLogsActivateAtCanonicalAmsterdam(t *testing.T) {
	s := eiptest.New(t, eiptest.WithGenesisTime(100), eiptest.ForkAt(eiptest.ForkAmsterdam, 24))
	first := s.STF().Transfer(s.Alice(), s.Bob().Address, eiptest.Wei(1)).Receipt
	second := s.STF().Transfer(s.Alice(), s.Bob().Address, eiptest.Wei(1)).Receipt
	if len(first.Logs) != 0 || len(second.Logs) != 1 {
		t.Fatalf("wrong Amsterdam activation: %d then %d", len(first.Logs), len(second.Logs))
	}
}
func TestTransferLogCreationOrderingAndRollback(t *testing.T) {
	target := types.HexToAddress("0x2222222222222222222222222222222222222222")
	s := eiptest.New(t, eiptest.ForkAt(eiptest.ForkGlamsterdan, 0), eiptest.FundCode(target, new(big.Int), []byte{0x60, 0, 0x60, 0, 0xfd}, 1))
	init := []byte{0x60, 0, 0x60, 0, 0xa0, 0}
	tx := eiptest.DynamicFee(s.Config().ChainID, s.Alice(), s.Nonce(s.Alice().Address), nil, big.NewInt(7), init, 100000, bigFeeCap, bigTipCap)
	receipt := s.Commit(tx).Tx(0).Receipt
	if receipt.Status != 1 || len(receipt.Logs) != 2 || receipt.Logs[0].Address != vm.SystemAddress || len(receipt.Logs[0].Topics) != 3 || receipt.Logs[1].Address != receipt.ContractAddress || len(receipt.Logs[1].Topics) != 0 {
		t.Fatal("creation logs include fees or violate transfer-before-constructor order")
	}
	tx = eiptest.DynamicFee(s.Config().ChainID, s.Alice(), s.Nonce(s.Alice().Address), &target, big.NewInt(3), nil, 100000, bigFeeCap, bigTipCap)
	failed := s.Commit(tx).Tx(0).Receipt
	if failed.Status != 0 || len(failed.Logs) != 0 {
		t.Fatalf("reverted transfer or fee leaked logs: status=%d logs=%d", failed.Status, len(failed.Logs))
	}
}
