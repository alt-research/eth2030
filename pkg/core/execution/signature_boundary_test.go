package execution

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestInvalidSignatureCannotFallBackToFundedZeroSender(t *testing.T) {
	cfg := *config.TestConfig
	cfg.ChainID = big.NewInt(1)
	db := state.NewMemoryStateDB()
	db.AddBalance(types.Address{}, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
	target := types.Address{19: 0x42}
	db.SetNonce(target, 1)
	before, err := db.Commit()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := types.NewTransaction(&types.LegacyTx{Gas: 50000, GasPrice: big.NewInt(1), To: &target, Value: new(big.Int), V: big.NewInt(37), R: new(big.Int), S: new(big.Int)}).EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := types.DecodeTxRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	pool := new(gaspool.GasPool).AddGas(100000)
	_, _, err = ApplyTransaction(&cfg, db, &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 100000, BaseFee: big.NewInt(1), Difficulty: new(big.Int)}, tx, pool)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("invalid signature did not fail before zero-sender execution: %v", err)
	}
	after, e := db.Commit()
	if e != nil {
		t.Fatal(e)
	}
	if after != before || pool.Gas() != 100000 || db.GetNonce(types.Address{}) != 0 {
		t.Fatal("invalid signature changed state or reserved gas")
	}
}

func TestPreHomesteadHighSSignatureRetainsHistoricalRules(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	to := gethcommon.Address{19: 0x42}
	signed, err := gethtypes.SignTx(gethtypes.NewTransaction(0, to, new(big.Int), 50000, big.NewInt(1), nil), gethtypes.FrontierSigner{}, key)
	if err != nil {
		t.Fatal(err)
	}
	v, r, s := signed.RawSignatureValues()
	high := new(big.Int).Sub(gethcrypto.S256().Params().N, s)
	parity := new(big.Int).Set(v)
	if v.Uint64() == 27 {
		parity.SetUint64(28)
	} else {
		parity.SetUint64(27)
	}
	malleable := gethtypes.NewTx(&gethtypes.LegacyTx{Gas: 50000, GasPrice: big.NewInt(1), To: &to, Value: new(big.Int), V: parity, R: r, S: high})
	if _, err := gethtypes.Sender(gethtypes.FrontierSigner{}, malleable); err != nil {
		t.Fatal(err)
	}
	raw, err := malleable.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for _, homestead := range []bool{false, true} {
		cfg := *config.TestConfig
		cfg.ChainID = big.NewInt(1)
		future := big.NewInt(100)
		cfg.HomesteadBlock = future
		cfg.EIP150Block = future
		cfg.EIP155Block = future
		cfg.EIP158Block = future
		cfg.ByzantiumBlock = future
		cfg.ConstantinopleBlock = future
		cfg.PetersburgBlock = future
		cfg.IstanbulBlock = future
		cfg.BerlinBlock = future
		cfg.LondonBlock = future
		cfg.TerminalTotalDifficulty = nil
		cfg.ShanghaiTime = nil
		cfg.CancunTime = nil
		cfg.PragueTime = nil
		cfg.OsakaTime = nil
		cfg.AmsterdamTime = nil
		cfg.GlamsterdanTime = nil
		cfg.HogotaTime = nil
		cfg.IPlusTime = nil
		if homestead {
			cfg.HomesteadBlock = new(big.Int)
		}
		sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
		db := state.NewMemoryStateDB()
		db.AddBalance(sender, big.NewInt(1000000))
		db.SetNonce(types.Address(to), 1)
		if _, err := db.Commit(); err != nil {
			t.Fatal(err)
		}
		tx, err := types.DecodeTxRLP(raw)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = ApplyTransaction(&cfg, db, &types.Header{Number: big.NewInt(1), Time: 1, GasLimit: 100000, Difficulty: big.NewInt(1)}, tx, new(gaspool.GasPool).AddGas(100000))
		if homestead {
			if err == nil || !strings.Contains(err.Error(), "signature") {
				t.Fatalf("Homestead accepted high-S: %v", err)
			}
		} else if err != nil || db.GetNonce(sender) != 1 {
			t.Fatalf("Frontier historical signature behavior changed: nonce=%d err=%v", db.GetNonce(sender), err)
		}
	}
}
