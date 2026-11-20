package execution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type cancelOnHistoryCode struct {
	state.StateDB
	target types.Address
	cancel context.CancelFunc
	reads  int
}

func (s *cancelOnHistoryCode) GetCode(address types.Address) []byte {
	if address == s.target {
		s.reads++
		s.cancel()
	}
	return s.StateDB.GetCode(address)
}

func TestHistoricalProcessorCancellationReachesSignedTransactionAndSystemCall(t *testing.T) {
	for _, system := range []bool{false, true} {
		t.Run(fmt.Sprintf("system=%t", system), func(t *testing.T) {
			cfg := *config.TestConfig
			cfg.ChainID = big.NewInt(1)
			cfg.AmsterdamTime, cfg.GlamsterdanTime, cfg.HogotaTime, cfg.IPlusTime = nil, nil, nil, nil
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			target := types.Address{19: 0x42}
			if system {
				target = types.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02")
			}
			db := state.NewMemoryStateDB()
			// A genuine long execution, cancelled only after the EVM asks for code.
			db.SetCode(target, []byte{0x5b, 0x60, 0, 0x56})
			wrapped := &cancelOnHistoryCode{StateDB: db, target: target, cancel: cancel}
			header := newTestHeader()
			header.GasLimit = 1_000_000_000
			header.ParentBeaconRoot = new(types.Hash)
			var transactions []*types.Transaction
			if !system {
				key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
				if err != nil {
					t.Fatal(err)
				}
				db.AddBalance(types.Address(gethcrypto.PubkeyToAddress(key.PublicKey)), new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil))
				to := gethcommon.Address(target)
				tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1), Gas: 900_000_000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1), To: &to, Value: new(big.Int)}), gethtypes.LatestSignerForChainID(big.NewInt(1)), key)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := tx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				native, err := types.DecodeBlockTransaction(raw)
				if err != nil {
					t.Fatal(err)
				}
				transactions = []*types.Transaction{native}
			}
			original := NewStateProcessor(&cfg)
			request := original.WithContext(ctx)
			started := time.Now()
			_, err := request.Process(types.NewBlock(header, &types.Body{Transactions: transactions}), wrapped)
			if !errors.Is(err, context.Canceled) || wrapped.reads == 0 {
				t.Fatalf("execution did not reach and cancel its own EVM: reads=%d error=%v", wrapped.reads, err)
			}
			if time.Since(started) > time.Second {
				t.Fatal("cancelled execution consumed its normal gas budget")
			}
			if original.ctx != nil {
				t.Fatal("request cancellation changed the canonical processor")
			}
			if _, err := original.Process(types.NewBlock(newTestHeader(), &types.Body{}), newBuilderRequestTestState()); err != nil {
				t.Fatalf("independent processor inherited cancelled context: %v", err)
			}
		})
	}
}
