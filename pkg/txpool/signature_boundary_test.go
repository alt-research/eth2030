package txpool

import (
	"fmt"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"strings"
	"testing"
)

func TestRawPoolAdmissionRejectsHighSBeforeSenderCaching(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	to := gethcommon.Address{19: 0x42}
	chain := big.NewInt(1337)
	signed, err := gethtypes.SignTx(gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: chain, Gas: 50000, GasFeeCap: big.NewInt(2000000000), GasTipCap: big.NewInt(1000000000), To: &to}), gethtypes.LatestSignerForChainID(chain), key)
	if err != nil {
		t.Fatal(err)
	}
	v, r, s := signed.RawSignatureValues()
	bad := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: chain, Gas: signed.Gas(), GasFeeCap: signed.GasFeeCap(), GasTipCap: signed.GasTipCap(), To: &to, V: new(big.Int).Xor(v, big.NewInt(1)), R: r, S: new(big.Int).Sub(gethcrypto.S256().Params().N, s)})
	raw, err := bad.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	native, err := types.DecodePooledTransaction(raw)
	if err != nil {
		t.Fatal(err)
	}
	db := state.NewMemoryStateDB()
	db.AddBalance(types.Address(gethcrypto.PubkeyToAddress(key.PublicKey)), new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
	db.AddBalance(types.Address{}, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil))
	pool := New(DefaultConfig(), db)
	cfg := *config.TestConfig
	cfg.ChainID = chain
	pool.SetForkRules(&cfg, 12)
	if err := pool.AddLocal(native); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("raw pool accepted high-S: %v", err)
	}
	if native.Sender() != nil {
		t.Fatal("invalid raw signature populated trusted sender cache")
	}
}
