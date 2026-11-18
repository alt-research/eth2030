package execution

import (
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	geth4844 "github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestNativeBlobFeeBurnReceiptAndOpcodeMatchIndependentPrice(t *testing.T) {
	cfg := *config.TestConfig
	cfg.AmsterdamTime = nil
	osaka := uint64(0)
	cfg.OsakaTime = &osaka
	excess := uint64(20_000_000)
	header := &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 30_000_000, BaseFee: big.NewInt(1), ExcessBlobGas: &excess}
	reference := *params.AllDevChainProtocolChanges
	reference.AmsterdamTime, reference.BogotaTime = nil, nil
	reference.BlobScheduleConfig = &params.BlobScheduleConfig{Cancun: params.DefaultCancunBlobConfig, Prague: params.DefaultPragueBlobConfig}
	price := geth4844.CalcBlobFee(&reference, &gethtypes.Header{Time: 12, ExcessBlobGas: &excess})
	sender, contract := types.Address{18: 0xaa, 19: 1}, types.Address{18: 0xbb, 19: 2}
	for _, test := range []struct {
		name         string
		cap, balance *big.Int
		invalid      bool
	}{
		{"priced", new(big.Int).Mul(price, big.NewInt(3)), big.NewInt(1_000_000_000_000_000_000), false},
		{"underpriced", new(big.Int).Sub(price, big.NewInt(1)), big.NewInt(1_000_000_000_000_000_000), true},
		{"cannot cover max blob fee", new(big.Int).Mul(price, big.NewInt(3)), new(big.Int).Add(big.NewInt(5_000_000), new(big.Int).Mul(price, big.NewInt(131072))), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := state.NewMemoryStateDB()
			db.AddBalance(sender, test.balance)
			db.SetNonce(contract, 1)
			db.SetCode(contract, []byte{0x4a, 0x5f, 0x55, 0x00}) // BLOBBASEFEE -> storage[0]
			initial := db.GetRoot()
			tx := types.NewTransaction(&types.BlobTx{Nonce: 0, To: contract, Gas: 100_000, GasFeeCap: big.NewInt(50), GasTipCap: big.NewInt(2), BlobFeeCap: test.cap, BlobHashes: []types.Hash{{0: 1}}, Value: new(big.Int)})
			tx.SetSender(sender)
			gp := new(gaspool.GasPool).AddGas(header.GasLimit)
			receipt, _, err := ApplyTransaction(&cfg, db, header, tx, gp)
			if test.invalid {
				if err == nil || db.GetRoot() != initial || gp.Gas() != header.GasLimit {
					t.Fatalf("invalid blob pricing changed state/gas: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if stored := db.GetState(contract, types.Hash{}); stored != types.BytesToHash(price.Bytes()) {
				t.Errorf("BLOBBASEFEE stored %s want %s", stored, price)
			}
			if receipt.BlobGasPrice == nil || receipt.BlobGasPrice.Cmp(price) != 0 {
				t.Errorf("receipt blob fee %v want %v", receipt.BlobGasPrice, price)
			}
			want := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), big.NewInt(3))
			want.Add(want, new(big.Int).Mul(price, big.NewInt(131072)))
			spent := new(big.Int).Sub(test.balance, db.GetBalance(sender))
			if spent.Cmp(want) != 0 {
				t.Errorf("sender paid %s want execution + burned blob fee %s", spent, want)
			}
		})
	}
}
