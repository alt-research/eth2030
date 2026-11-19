package execution

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	bridge "github.com/eth2030/eth2030/geth"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestSignedNativePrecompileEmptyAccountMatchesIndependentRoot(t *testing.T) {
	cfg := *config.TestConfig
	cfg.ChainID = big.NewInt(1)
	zero := uint64(0)
	cfg.OsakaTime = &zero
	cfg.AmsterdamTime = nil
	cfg.GlamsterdanTime = nil
	cfg.HogotaTime = nil
	cfg.IPlusTime = nil
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	sender := types.Address(gethcrypto.PubkeyToAddress(key.PublicKey))
	identity := gethcommon.Address{19: 4}
	miner := types.Address{19: 0x88}
	tx, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(1), Gas: 100000, GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1), To: &identity, Value: new(big.Int), Data: []byte{7}}), gethtypes.LatestSignerForChainID(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	nativeTx, err := types.DecodeTxRLP(raw)
	if err != nil {
		t.Fatal(err)
	}
	initial := big.NewInt(1000000000000000000)
	db := state.NewMemoryStateDB()
	db.AddBalance(sender, initial)
	if _, err := db.Commit(); err != nil {
		t.Fatal(err)
	}
	header := &types.Header{Number: big.NewInt(1), Time: 12, GasLimit: 30000000, BaseFee: big.NewInt(1), Coinbase: miner, Difficulty: new(big.Int)}
	receipt, _, err := ApplyTransaction(&cfg, db, header, nativeTx, new(gaspool.GasPool).AddGas(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	nativeRoot, err := db.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if db.Exist(types.Address(identity)) {
		t.Fatal("native transition retained touched empty precompile account")
	}
	reference, err := bridge.MakePreState(map[string]bridge.PreAccount{sender.Hex(): {Balance: initial}})
	if err != nil {
		t.Fatal(err)
	}
	defer reference.Close()
	sdkConfig := bridge.ToGethChainConfig(&cfg)
	random := gethcommon.Hash{}
	evm := gethvm.NewEVM(gethvm.BlockContext{CanTransfer: gethcore.CanTransfer, Transfer: gethcore.Transfer, Coinbase: gethcommon.Address(miner), BlockNumber: big.NewInt(1), Time: 12, GasLimit: header.GasLimit, BaseFee: header.BaseFee, BlobBaseFee: big.NewInt(1), Difficulty: new(big.Int), Random: &random}, reference.StateDB, sdkConfig, gethvm.Config{})
	defer evm.Release()
	msg, err := gethcore.TransactionToMessage(tx, gethtypes.MakeSigner(sdkConfig, big.NewInt(1), 12), header.BaseFee)
	if err != nil {
		t.Fatal(err)
	}
	reference.StateDB.SetTxContext(tx.Hash(), 0, 1)
	result, err := gethcore.ApplyMessage(evm, msg, gethcore.NewGasPool(header.GasLimit))
	if err != nil {
		t.Fatal(err)
	}
	root := reference.StateDB.IntermediateRoot(evm.GetRules())
	if types.Hash(root) != nativeRoot || result.UsedGas != receipt.GasUsed {
		t.Fatalf("native gas/root %d/%s want unchanged SDK %d/%s", receipt.GasUsed, nativeRoot, result.UsedGas, root)
	}
	t.Logf("signed independent gas=%d root=%s", result.UsedGas, root)
}
